use std::{
    error::Error,
    sync::{
        Arc, Mutex, MutexGuard,
        atomic::{AtomicI64, Ordering},
    },
    time::Duration,
};

use async_trait::async_trait;
use blog_backend::{
    api::{websocket::WorkerStatusProvider, worker::WorkerStatusAdapter},
    core::worker::{
        Clock, InsightGenerationPort, InsightWorker, ManagerConfig, PipelineWorker, RunMetadata,
        StatusService, Worker, WorkerContext, WorkerFailure, WorkerManager, WorkerManagerError,
        WorkerResult, WorkerState,
    },
};
use chrono::{TimeZone, Utc};
use tokio::time::timeout;
use tokio_util::sync::CancellationToken;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

struct FixtureClock {
    seconds: AtomicI64,
}

impl FixtureClock {
    fn new(seconds: i64) -> Self {
        Self {
            seconds: AtomicI64::new(seconds),
        }
    }
}

impl Clock for FixtureClock {
    fn now(&self) -> chrono::DateTime<Utc> {
        let seconds = self.seconds.fetch_add(1, Ordering::SeqCst);
        Utc.timestamp_opt(seconds, 0)
            .single()
            .unwrap_or_else(Utc::now)
    }
}

fn status() -> Arc<StatusService> {
    Arc::new(StatusService::new(Arc::new(FixtureClock::new(100))))
}

#[test]
fn status_service_has_deterministic_transitions_and_websocket_mapping() -> TestResult {
    let status = status();
    status.register_worker("crawl");
    status.start_worker("crawl", None);
    status.set_progress("crawl", 1, 4, "one");
    status.complete_worker("crawl", "done");

    let current = status.status("crawl").ok_or("crawl status missing")?;
    assert_eq!(current.state, WorkerState::Completed);
    assert_eq!(current.progress, 100);
    assert_eq!(current.items_done, 1);
    assert_eq!(current.started_at, Utc.timestamp_opt(100, 0).single());
    assert_eq!(current.completed_at, Utc.timestamp_opt(102, 0).single());

    let adapter = WorkerStatusAdapter::new(status.clone());
    let snapshot = adapter.snapshot();
    assert_eq!(snapshot.len(), 1);
    assert_eq!(snapshot[0].worker_name, "crawl");
    assert_eq!(snapshot[0].status.state, "completed");
    assert_eq!(snapshot[0].status.task_run_id, None);

    let mut updates = adapter.subscribe();
    status.reset_worker("crawl");
    let update = updates.try_recv()?;
    assert_eq!(update.worker_name, "crawl");
    assert_eq!(update.status.state, "idle");
    assert_eq!(
        update.timestamp,
        Utc.timestamp_opt(103, 0)
            .single()
            .ok_or("fixture timestamp missing")?
    );
    drop(updates);
    status.update_status("crawl", WorkerState::Running, 5, "again");
    assert_eq!(status.subscriber_count(), 0);
    Ok(())
}

struct BlockingWorker;

#[async_trait]
impl Worker for BlockingWorker {
    fn name(&self) -> &str {
        "blocking"
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        context.cancelled().await;
        Err(WorkerFailure::new("operation cancelled"))
    }
}

struct OrderedWorker {
    name: &'static str,
    order: Arc<Mutex<Vec<&'static str>>>,
}

#[async_trait]
impl Worker for OrderedWorker {
    fn name(&self) -> &str {
        self.name
    }

    async fn run(&self, _context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        lock(&self.order).push(self.name);
        Ok(WorkerResult::completed(format!("{} ok", self.name)))
    }
}

fn lock<T>(value: &Mutex<T>) -> MutexGuard<'_, T> {
    value
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

#[tokio::test]
async fn manager_claims_atomically_cancels_and_joins_owned_tasks() -> TestResult {
    let status = status();
    let manager = WorkerManager::new(
        status.clone(),
        None,
        CancellationToken::new(),
        ManagerConfig {
            shutdown_timeout: Duration::from_secs(1),
            finalization_timeout: Duration::from_secs(1),
        },
    )?;
    manager.register(Arc::new(BlockingWorker));
    manager.start()?;
    let mut updates = status.subscribe();
    assert_eq!(
        manager.run_now("blocking", RunMetadata::default()).await?,
        ""
    );
    assert!(matches!(
        manager.run_now("blocking", RunMetadata::default()).await,
        Err(WorkerManagerError::AlreadyRunning)
    ));
    assert_eq!(manager.running_workers(), vec!["blocking"]);
    manager.stop("blocking")?;

    let terminal = timeout(Duration::from_secs(1), async {
        loop {
            let update = updates.recv().await.ok_or("status stream closed")?;
            if update.worker_name == "blocking" && update.status.state == WorkerState::Failed {
                return TestResult::Ok(update);
            }
        }
    })
    .await??;
    assert!(matches!(
        terminal.status.error.as_str(),
        "Stopped by user" | "operation cancelled"
    ));
    manager.shutdown().await?;
    assert!(!manager.is_running());
    assert!(manager.running_workers().is_empty());
    Ok(())
}

#[tokio::test]
async fn pipeline_runs_crawl_then_insight_and_propagates_completion() -> TestResult {
    let status = status();
    let manager = WorkerManager::new(
        status.clone(),
        None,
        CancellationToken::new(),
        ManagerConfig::default(),
    )?;
    let order = Arc::new(Mutex::new(Vec::new()));
    manager.register(Arc::new(OrderedWorker {
        name: "crawl",
        order: order.clone(),
    }));
    manager.register(Arc::new(OrderedWorker {
        name: "insight",
        order: order.clone(),
    }));
    manager.register(Arc::new(PipelineWorker::new(
        WorkerManager::downgrade(&manager),
        status.clone(),
    )));
    let mut updates = status.subscribe();
    manager.run_now("pipeline", RunMetadata::default()).await?;

    timeout(Duration::from_secs(1), async {
        loop {
            let update = updates.recv().await.ok_or("status stream closed")?;
            if update.worker_name == "pipeline" && update.status.state == WorkerState::Completed {
                return TestResult::Ok(());
            }
        }
    })
    .await??;
    assert_eq!(lock(&order).as_slice(), ["crawl", "insight"]);
    manager.shutdown().await?;
    Ok(())
}

struct InvalidInsightProvider;

#[async_trait]
impl InsightGenerationPort for InvalidInsightProvider {
    fn is_configured(&self) -> bool {
        false
    }

    async fn topics(
        &self,
    ) -> Result<Vec<blog_backend::core::insight::InsightTopic>, WorkerFailure> {
        Ok(Vec::new())
    }

    async fn generate_for_topic(
        &self,
        _topic: &blog_backend::core::insight::InsightTopic,
        _cancellation: &CancellationToken,
    ) -> Result<blog_backend::core::worker::InsightTopicResult, WorkerFailure> {
        Err(WorkerFailure::new("not configured"))
    }
}

#[test]
fn insight_worker_rejects_an_invalid_configured_provider() {
    assert!(InsightWorker::new(status(), Some(Arc::new(InvalidInsightProvider))).is_err());
}
