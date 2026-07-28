use std::{
    collections::{BTreeMap, VecDeque},
    panic::AssertUnwindSafe,
    sync::{Arc, Mutex, MutexGuard, Weak},
    time::Duration,
};

use async_trait::async_trait;
use futures_util::FutureExt;
use serde_json::Map;
use tokio::{
    sync::{Mutex as AsyncMutex, watch},
    task::JoinHandle,
    time::{Instant, timeout, timeout_at},
};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{
    core::taskrun::{
        FinishRunInput, RecordEventInput, StartRunInput, TaskRunContext, TaskRunEventLevel,
        TaskRunKind, TaskRunService, TaskRunStatus, TaskRunTracker,
    },
    error::AppError,
};

use super::{
    PIPELINE_WORKER_NAME, RunMetadata, StatusService, WorkerContext, WorkerFailure, WorkerResult,
    WorkerResultStatus,
};

#[async_trait]
pub trait Worker: Send + Sync {
    fn name(&self) -> &str;
    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure>;
}

#[derive(Debug, Clone)]
pub struct ManagerConfig {
    pub shutdown_timeout: Duration,
    pub finalization_timeout: Duration,
}

impl Default for ManagerConfig {
    fn default() -> Self {
        Self {
            shutdown_timeout: Duration::from_secs(30),
            finalization_timeout: Duration::from_secs(5),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum WorkerManagerError {
    #[error("worker not found")]
    NotFound,
    #[error("worker is already running")]
    AlreadyRunning,
    #[error("worker is not running")]
    NotRunning,
    #[error("worker manager is shutting down")]
    ShuttingDown,
    #[error("worker manager duration settings must be greater than zero")]
    InvalidConfig,
    #[error("worker task-run persistence failed")]
    TaskRunPersistence,
}

#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[error("worker shutdown failed")]
pub struct WorkerShutdownError {
    pub timed_out: bool,
    pub task_failures: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum WorkerOutcome {
    Completed(WorkerResultStatus),
    Failed(String),
    Cancelled,
}

struct RunningWorker {
    cancellation: CancellationToken,
}

struct ManagerState {
    workers: BTreeMap<String, Arc<dyn Worker>>,
    running: BTreeMap<String, RunningWorker>,
    is_running: bool,
    accepting: bool,
    lifecycle_failures: Vec<String>,
}

impl Default for ManagerState {
    fn default() -> Self {
        Self {
            workers: BTreeMap::new(),
            running: BTreeMap::new(),
            is_running: false,
            accepting: true,
            lifecycle_failures: Vec::new(),
        }
    }
}

pub struct WorkerManager {
    state: Mutex<ManagerState>,
    tasks: Mutex<Vec<JoinHandle<()>>>,
    start_gate: AsyncMutex<()>,
    status: Arc<StatusService>,
    task_runs: Option<Arc<TaskRunService>>,
    root_cancellation: CancellationToken,
    config: ManagerConfig,
}

impl WorkerManager {
    pub fn new(
        status: Arc<StatusService>,
        task_runs: Option<Arc<TaskRunService>>,
        root_cancellation: CancellationToken,
        config: ManagerConfig,
    ) -> Result<Arc<Self>, WorkerManagerError> {
        if config.shutdown_timeout.is_zero() || config.finalization_timeout.is_zero() {
            return Err(WorkerManagerError::InvalidConfig);
        }
        Ok(Arc::new(Self {
            state: Mutex::new(ManagerState::default()),
            tasks: Mutex::new(Vec::new()),
            start_gate: AsyncMutex::new(()),
            status,
            task_runs,
            root_cancellation,
            config,
        }))
    }

    pub fn downgrade(manager: &Arc<Self>) -> Weak<Self> {
        Arc::downgrade(manager)
    }

    pub fn register(&self, worker: Arc<dyn Worker>) {
        let name = worker.name().to_owned();
        self.state().workers.insert(name.clone(), worker);
        self.status.register_worker(&name);
    }

    pub fn start(&self) -> Result<(), WorkerManagerError> {
        let mut state = self.state();
        if !state.accepting {
            return Err(WorkerManagerError::ShuttingDown);
        }
        state.is_running = true;
        Ok(())
    }

    pub fn is_running(&self) -> bool {
        self.state().is_running
    }

    pub fn registered_workers(&self) -> Vec<String> {
        self.state().workers.keys().cloned().collect()
    }

    pub fn running_workers(&self) -> Vec<String> {
        self.state().running.keys().cloned().collect()
    }

    pub fn is_worker_running(&self, name: &str) -> bool {
        self.state().running.contains_key(name)
    }

    pub async fn run_now(
        self: &Arc<Self>,
        name: &str,
        metadata: RunMetadata,
    ) -> Result<String, WorkerManagerError> {
        let started = self.start_worker(name, metadata).await?;
        Ok(started
            .task_run_id
            .map_or_else(String::new, |id| id.to_string()))
    }

    pub(crate) async fn run_and_wait(
        self: &Arc<Self>,
        name: &str,
        metadata: RunMetadata,
        cancellation: &CancellationToken,
    ) -> Result<(Option<Uuid>, WorkerOutcome), WorkerManagerError> {
        let mut started = self.start_worker(name, metadata).await?;
        let outcome = tokio::select! {
            biased;
            () = cancellation.cancelled() => {
                let _ = self.stop(name);
                started.completion.changed().await.ok();
                started.completion.borrow().clone()
            }
            changed = started.completion.changed() => {
                changed.ok();
                started.completion.borrow().clone()
            }
        }
        .ok_or(WorkerManagerError::TaskRunPersistence)?;
        Ok((started.task_run_id, outcome))
    }

    pub fn stop(&self, name: &str) -> Result<(), WorkerManagerError> {
        let cancellation = self
            .state()
            .running
            .get(name)
            .map(|worker| worker.cancellation.clone())
            .ok_or(WorkerManagerError::NotRunning)?;
        cancellation.cancel();
        self.status.set_error(name, "Stopped by user");
        Ok(())
    }

    pub async fn shutdown(&self) -> Result<(), WorkerShutdownError> {
        let _start_guard = self.start_gate.lock().await;
        let cancellations = {
            let mut state = self.state();
            state.accepting = false;
            state.is_running = false;
            state
                .running
                .values()
                .map(|worker| worker.cancellation.clone())
                .collect::<Vec<_>>()
        };
        self.root_cancellation.cancel();
        for cancellation in cancellations {
            cancellation.cancel();
        }

        let mut tasks = VecDeque::from(std::mem::take(&mut *self.tasks()));
        let deadline = Instant::now() + self.config.shutdown_timeout;
        let mut timed_out = false;
        let mut task_failures = Vec::new();
        while let Some(mut task) = tasks.pop_front() {
            match timeout_at(deadline, &mut task).await {
                Ok(Ok(())) => {}
                Ok(Err(error)) => task_failures.push(error.to_string()),
                Err(_) => {
                    timed_out = true;
                    task.abort();
                    for remaining in &tasks {
                        remaining.abort();
                    }
                    if let Err(error) = task.await
                        && !error.is_cancelled()
                    {
                        task_failures.push(error.to_string());
                    }
                    for remaining in tasks {
                        if let Err(error) = remaining.await
                            && !error.is_cancelled()
                        {
                            task_failures.push(error.to_string());
                        }
                    }
                    break;
                }
            }
        }
        task_failures.extend(std::mem::take(&mut self.state().lifecycle_failures));
        if timed_out || !task_failures.is_empty() {
            Err(WorkerShutdownError {
                timed_out,
                task_failures,
            })
        } else {
            Ok(())
        }
    }

    pub fn status_service(&self) -> &Arc<StatusService> {
        &self.status
    }

    async fn start_worker(
        self: &Arc<Self>,
        name: &str,
        mut metadata: RunMetadata,
    ) -> Result<StartedWorker, WorkerManagerError> {
        let _start_guard = self.start_gate.lock().await;
        self.reap_finished_tasks().await;
        let worker = {
            let mut state = self.state();
            if !state.accepting {
                return Err(WorkerManagerError::ShuttingDown);
            }
            let worker = state
                .workers
                .get(name)
                .cloned()
                .ok_or(WorkerManagerError::NotFound)?;
            if state.running.contains_key(name) {
                return Err(WorkerManagerError::AlreadyRunning);
            }
            let cancellation = self.root_cancellation.child_token();
            state.running.insert(
                name.to_owned(),
                RunningWorker {
                    cancellation: cancellation.clone(),
                },
            );
            (worker, cancellation)
        };
        if metadata.trigger_source.is_empty() {
            metadata.trigger_source = "manual".to_owned();
        }
        let run = match self.start_task_run(name, &metadata).await {
            Ok(run) => run,
            Err(error) => {
                self.state().running.remove(name);
                return Err(error);
            }
        };
        let task_run_id = run.as_ref().map(|run| run.id);
        let task_run_context = TaskRunContext::new(run.as_ref().and_then(|run| {
            self.task_runs
                .as_ref()
                .and_then(|service| TaskRunTracker::new(service.clone(), run.id))
        }));
        let (completion_tx, completion) = watch::channel(None);
        let manager = self.clone();
        let name = name.to_owned();
        let handle = tokio::spawn(async move {
            manager
                .execute_worker(
                    worker.0,
                    name,
                    worker.1,
                    task_run_context,
                    task_run_id,
                    completion_tx,
                )
                .await;
        });
        self.tasks().push(handle);
        Ok(StartedWorker {
            task_run_id,
            completion,
        })
    }

    async fn start_task_run(
        &self,
        name: &str,
        metadata: &RunMetadata,
    ) -> Result<Option<crate::core::taskrun::TaskRun>, WorkerManagerError> {
        let Some(service) = self.task_runs.as_ref() else {
            return Ok(None);
        };
        service
            .start_run(StartRunInput {
                kind: if name == PIPELINE_WORKER_NAME {
                    TaskRunKind::Workflow
                } else {
                    TaskRunKind::Worker
                },
                task_name: name.to_owned(),
                organization_id: metadata.organization_id,
                user_id: metadata.user_id,
                triggered_by_user_id: metadata.triggered_by_user_id,
                trigger_source: metadata.trigger_source.clone(),
                parent_run_id: metadata.parent_run_id,
                input_payload: Map::new(),
                summary: None,
            })
            .await
            .map(Some)
            .map_err(|_| WorkerManagerError::TaskRunPersistence)
    }

    async fn execute_worker(
        self: Arc<Self>,
        worker: Arc<dyn Worker>,
        name: String,
        cancellation: CancellationToken,
        task_run: TaskRunContext,
        task_run_id: Option<Uuid>,
        completion: watch::Sender<Option<WorkerOutcome>>,
    ) {
        self.status.start_worker(&name, task_run_id);
        let context = WorkerContext::new(cancellation.clone(), task_run);
        let result = tokio::select! {
            biased;
            () = cancellation.cancelled() => Err(WorkerFailure::new("operation cancelled")),
            result = AssertUnwindSafe(worker.run(context)).catch_unwind() => {
                match result {
                    Ok(result) => result,
                    Err(_) => {
                        self.state().lifecycle_failures.push(format!("{name}: worker task panicked"));
                        Err(WorkerFailure::new("worker task panicked"))
                    }
                }
            },
        };
        let outcome = if cancellation.is_cancelled() {
            self.status.set_error(&name, "operation cancelled");
            WorkerOutcome::Cancelled
        } else {
            match result {
                Ok(result) => {
                    let status = result.status;
                    if let Err(error) = self.finish_task_run(task_run_id, Ok(&result)).await {
                        self.record_lifecycle_failure(&name, error);
                    }
                    self.status.complete_worker(&name, "Completed successfully");
                    WorkerOutcome::Completed(status)
                }
                Err(error) => {
                    if let Err(finalization) = self.finish_task_run(task_run_id, Err(&error)).await
                    {
                        self.record_lifecycle_failure(&name, finalization);
                    }
                    self.status.set_error(&name, error.message());
                    WorkerOutcome::Failed(error.to_string())
                }
            }
        };
        if matches!(outcome, WorkerOutcome::Cancelled)
            && let Err(error) = self.finish_cancelled_task_run(task_run_id).await
        {
            self.record_lifecycle_failure(&name, error);
        }
        self.state().running.remove(&name);
        let _ = completion.send(Some(outcome));
    }

    async fn finish_task_run(
        &self,
        run_id: Option<Uuid>,
        result: Result<&WorkerResult, &WorkerFailure>,
    ) -> Result<(), AppError> {
        let (Some(service), Some(run_id)) = (self.task_runs.as_ref(), run_id) else {
            return Ok(());
        };
        let input = match &result {
            Ok(result) => FinishRunInput {
                run_id,
                status: if result.status == WorkerResultStatus::Warning {
                    TaskRunStatus::Warning
                } else {
                    TaskRunStatus::Completed
                },
                summary: Some(if result.summary.is_empty() {
                    "Completed successfully".to_owned()
                } else {
                    result.summary.clone()
                }),
                error_summary: None,
                output_summary: result.output_summary.clone().into_iter().collect(),
                metrics: result.metrics.clone().into_iter().collect(),
            },
            Err(error) => FinishRunInput {
                run_id,
                status: TaskRunStatus::Failed,
                summary: Some(error.to_string()),
                error_summary: Some(error.to_string()),
                output_summary: Map::new(),
                metrics: Map::new(),
            },
        };
        timeout(self.config.finalization_timeout, service.finish_run(input))
            .await
            .map_err(|_| AppError::Internal)??;
        if let Ok(result) = result {
            for warning in &result.warnings {
                timeout(
                    self.config.finalization_timeout,
                    service.record_event(RecordEventInput {
                        run_id,
                        step_key: None,
                        event_type: "warning".to_owned(),
                        level: TaskRunEventLevel::Warning,
                        message: warning.clone(),
                        meta_data: Map::new(),
                    }),
                )
                .await
                .map_err(|_| AppError::Internal)??;
            }
        }
        Ok(())
    }

    async fn finish_cancelled_task_run(&self, run_id: Option<Uuid>) -> Result<(), AppError> {
        let (Some(service), Some(run_id)) = (self.task_runs.as_ref(), run_id) else {
            return Ok(());
        };
        timeout(
            self.config.finalization_timeout,
            service.finish_run(FinishRunInput {
                run_id,
                status: TaskRunStatus::Cancelled,
                summary: Some("Run cancelled".to_owned()),
                error_summary: None,
                output_summary: Map::new(),
                metrics: Map::new(),
            }),
        )
        .await
        .map_err(|_| AppError::Internal)??;
        Ok(())
    }

    fn record_lifecycle_failure(&self, name: &str, error: AppError) {
        self.state()
            .lifecycle_failures
            .push(format!("{name}: {error}"));
    }

    async fn reap_finished_tasks(&self) {
        let finished = {
            let mut tasks = self.tasks();
            let mut finished = Vec::new();
            let mut index = 0;
            while index < tasks.len() {
                if tasks[index].is_finished() {
                    finished.push(tasks.swap_remove(index));
                } else {
                    index += 1;
                }
            }
            finished
        };
        for task in finished {
            if let Err(error) = task.await {
                self.state().lifecycle_failures.push(error.to_string());
            }
        }
    }

    fn state(&self) -> MutexGuard<'_, ManagerState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }

    fn tasks(&self) -> MutexGuard<'_, Vec<JoinHandle<()>>> {
        self.tasks
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

struct StartedWorker {
    task_run_id: Option<Uuid>,
    completion: watch::Receiver<Option<WorkerOutcome>>,
}
