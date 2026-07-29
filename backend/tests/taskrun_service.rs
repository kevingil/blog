use std::sync::{Arc, Mutex, MutexGuard};

use async_trait::async_trait;
use blog_backend::{
    core::taskrun::{
        FinishRunInput, FinishStepInput, JsonObject, RecordEventInput, StartRunInput,
        StartStepInput, TaskRun, TaskRunContext, TaskRunEvent, TaskRunEventLevel, TaskRunFilter,
        TaskRunKind, TaskRunRepository, TaskRunService, TaskRunStatus, TaskRunStep, TaskRunTracker,
    },
    error::AppError,
};
use chrono::Utc;
use serde_json::{Value, json};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

#[derive(Default)]
struct TaskRunState {
    runs: Vec<TaskRun>,
    steps: Vec<TaskRunStep>,
    events: Vec<TaskRunEvent>,
    fail_next_event: bool,
    fail_step_lookup: bool,
    calls: Vec<&'static str>,
}

#[derive(Default)]
struct MemoryTaskRunRepository {
    state: Mutex<TaskRunState>,
}

impl MemoryTaskRunRepository {
    fn state(&self) -> MutexGuard<'_, TaskRunState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl TaskRunRepository for MemoryTaskRunRepository {
    async fn create_run(&self, run: &mut TaskRun) -> Result<(), AppError> {
        let mut state = self.state();
        state.calls.push("create_run");
        run.id = Uuid::new_v4();
        run.created_at = Some(Utc::now());
        run.updated_at = run.created_at;
        state.runs.push(run.clone());
        Ok(())
    }

    async fn update_run(&self, run: &TaskRun) -> Result<(), AppError> {
        let mut state = self.state();
        state.calls.push("update_run");
        let stored = state
            .runs
            .iter_mut()
            .find(|stored| stored.id == run.id)
            .ok_or(AppError::NotFound)?;
        *stored = run.clone();
        Ok(())
    }

    async fn find_run_by_id(&self, id: Uuid) -> Result<TaskRun, AppError> {
        self.state()
            .runs
            .iter()
            .find(|run| run.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_runs(&self, _filter: TaskRunFilter) -> Result<Vec<TaskRun>, AppError> {
        Ok(self.state().runs.clone())
    }

    async fn create_step(&self, step: &mut TaskRunStep) -> Result<(), AppError> {
        let mut state = self.state();
        state.calls.push("create_step");
        step.id = Uuid::new_v4();
        step.created_at = Some(Utc::now());
        step.updated_at = step.created_at;
        state.steps.push(step.clone());
        Ok(())
    }

    async fn update_step(&self, step: &TaskRunStep) -> Result<(), AppError> {
        let mut state = self.state();
        state.calls.push("update_step");
        let stored = state
            .steps
            .iter_mut()
            .find(|stored| stored.id == step.id)
            .ok_or(AppError::NotFound)?;
        *stored = step.clone();
        Ok(())
    }

    async fn find_step_by_run_and_key(
        &self,
        run_id: Uuid,
        step_key: &str,
    ) -> Result<TaskRunStep, AppError> {
        let state = self.state();
        if state.fail_step_lookup {
            return Err(AppError::Database);
        }
        state
            .steps
            .iter()
            .find(|step| step.task_run_id == run_id && step.step_key == step_key)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_steps_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunStep>, AppError> {
        Ok(self
            .state()
            .steps
            .iter()
            .filter(|step| step.task_run_id == run_id)
            .cloned()
            .collect())
    }

    async fn create_event(&self, event: &mut TaskRunEvent) -> Result<(), AppError> {
        let mut state = self.state();
        state.calls.push("create_event");
        if state.fail_next_event {
            state.fail_next_event = false;
            return Err(AppError::Database);
        }
        event.id = Uuid::new_v4();
        event.created_at = Some(Utc::now());
        state.events.push(event.clone());
        Ok(())
    }

    async fn list_events_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunEvent>, AppError> {
        Ok(self
            .state()
            .events
            .iter()
            .filter(|event| event.task_run_id == run_id)
            .cloned()
            .collect())
    }
}

fn service(
    repository: Arc<MemoryTaskRunRepository>,
    cancellation: CancellationToken,
) -> Arc<TaskRunService> {
    Arc::new(TaskRunService::new(repository, cancellation))
}

fn start_input() -> StartRunInput {
    StartRunInput {
        kind: TaskRunKind::Worker,
        task_name: "crawler".to_owned(),
        organization_id: None,
        user_id: Some(Uuid::new_v4()),
        triggered_by_user_id: None,
        trigger_source: String::new(),
        parent_run_id: None,
        input_payload: JsonObject::new(),
        summary: None,
    }
}

#[tokio::test]
async fn start_run_preserves_default_and_non_transactional_event_boundary() {
    let repository = Arc::new(MemoryTaskRunRepository::default());
    repository.state().fail_next_event = true;
    let service = service(repository.clone(), CancellationToken::new());

    let result = service.start_run(start_input()).await;
    assert!(matches!(result, Err(AppError::Database)));
    let state = repository.state();
    assert_eq!(state.runs.len(), 1);
    assert!(state.events.is_empty());
    assert_eq!(state.calls, vec!["create_run", "create_event"]);
    assert_eq!(state.runs[0].trigger_source, "manual");
    assert_eq!(state.runs[0].status, TaskRunStatus::Running);
    assert_eq!(state.runs[0].output_summary, JsonObject::new());
}

#[tokio::test]
async fn run_and_step_events_keep_status_mapping_messages_and_order() {
    let repository = Arc::new(MemoryTaskRunRepository::default());
    let service = service(repository.clone(), CancellationToken::new());
    let run = service.start_run(start_input()).await;
    assert!(run.is_ok());
    let run_id = run.map(|run| run.id).unwrap_or_else(|_| Uuid::nil());
    let step = service
        .start_step(StartStepInput {
            run_id,
            step_key: "crawl".to_owned(),
            step_name: "Crawler".to_owned(),
            summary: None,
        })
        .await;
    assert!(step.is_ok());
    let mut metrics = JsonObject::new();
    metrics.insert("records".to_owned(), json!(3));
    let finish_step = service
        .finish_step(FinishStepInput {
            run_id,
            step_key: "crawl".to_owned(),
            status: TaskRunStatus::Warning,
            summary: None,
            error_summary: None,
            metrics: metrics.clone(),
        })
        .await;
    assert!(finish_step.is_ok());
    let finish_run = service
        .finish_run(FinishRunInput {
            run_id,
            status: TaskRunStatus::Failed,
            summary: Some("provider failed".to_owned()),
            error_summary: Some("timeout".to_owned()),
            output_summary: JsonObject::new(),
            metrics: metrics.clone(),
        })
        .await;
    assert!(finish_run.is_ok());

    let state = repository.state();
    assert_eq!(
        state
            .events
            .iter()
            .map(|event| event.event_type.as_str())
            .collect::<Vec<_>>(),
        vec!["run_started", "step_started", "step_warning", "run_failed"]
    );
    assert_eq!(state.events[1].task_run_step_id, Some(state.steps[0].id));
    assert_eq!(state.events[2].message, "Crawler completed with warnings");
    assert_eq!(state.events[2].meta_data, metrics);
    assert_eq!(state.events[3].message, "provider failed");
    assert_eq!(state.events[3].level, TaskRunEventLevel::Error);
    assert_eq!(
        state.events[3].meta_data.get("status"),
        Some(&Value::String("failed".to_owned()))
    );
}

#[tokio::test]
async fn record_event_ignores_step_lookup_failures_like_go() {
    let repository = Arc::new(MemoryTaskRunRepository::default());
    repository.state().fail_step_lookup = true;
    let service = service(repository.clone(), CancellationToken::new());
    let run_id = Uuid::new_v4();
    let result = service
        .record_event(RecordEventInput {
            run_id,
            step_key: Some("missing".to_owned()),
            event_type: "warning".to_owned(),
            level: TaskRunEventLevel::Warning,
            message: "step lookup did not block event".to_owned(),
            meta_data: JsonObject::new(),
        })
        .await;
    assert!(result.is_ok());
    assert_eq!(repository.state().events[0].task_run_step_id, None);
}

#[tokio::test]
async fn tracker_context_is_explicit_and_disabled_context_is_a_noop() {
    let repository = Arc::new(MemoryTaskRunRepository::default());
    let service = service(repository.clone(), CancellationToken::new());
    assert!(TaskRunTracker::new(service.clone(), Uuid::nil()).is_none());

    let disabled = TaskRunContext::default();
    let result = disabled
        .record_event(
            None,
            "ignored",
            "unexpected-level",
            "ignored",
            JsonObject::new(),
        )
        .await;
    assert!(result.is_ok());
    assert!(repository.state().calls.is_empty());

    let run_id = Uuid::new_v4();
    let tracker = TaskRunTracker::new(service, run_id);
    assert!(tracker.is_some());
    let context = TaskRunContext::new(tracker);
    let result = context
        .record_event(
            None,
            "info",
            "unexpected-level",
            "mapped to info",
            JsonObject::new(),
        )
        .await;
    assert!(result.is_ok());
    assert_eq!(context.run_id(), Some(run_id));
    assert_eq!(repository.state().events[0].level, TaskRunEventLevel::Info);
}

#[tokio::test]
async fn cancellation_prevents_repository_admission() {
    let repository = Arc::new(MemoryTaskRunRepository::default());
    let cancellation = CancellationToken::new();
    cancellation.cancel();
    let service = service(repository.clone(), cancellation);
    let result = service.start_run(start_input()).await;
    assert!(matches!(result, Err(AppError::Internal)));
    assert!(repository.state().calls.is_empty());
}
