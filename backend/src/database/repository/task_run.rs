use async_trait::async_trait;
use chrono::Utc;
use diesel::{ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper};
use diesel_async::RunQueryDsl;
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::taskrun::{
        JsonObject, TaskRun, TaskRunEvent, TaskRunFilter, TaskRunRepository, TaskRunStep,
    },
    database::{
        models::task_run::{
            NewTaskRunEventRow, NewTaskRunRow, NewTaskRunStepRow, TaskRunChangeset,
            TaskRunEventRow, TaskRunRow, TaskRunStepChangeset, TaskRunStepRow,
        },
        pool::PgPool,
    },
    error::AppError,
    schema::{task_run, task_run_event, task_run_step},
};

#[derive(Clone)]
pub struct DieselTaskRunRepository {
    pool: PgPool,
}

impl DieselTaskRunRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }
}

#[async_trait]
impl TaskRunRepository for DieselTaskRunRepository {
    async fn create_run(&self, run: &mut TaskRun) -> Result<(), AppError> {
        if run.id.is_nil() {
            run.id = Uuid::new_v4();
        }
        let row = NewTaskRunRow {
            id: run.id,
            kind: run.kind.as_str().to_owned(),
            task_name: run.task_name.clone(),
            status: run.status.as_str().to_owned(),
            organization_id: run.organization_id,
            user_id: run.user_id,
            triggered_by_user_id: run.triggered_by_user_id,
            trigger_source: run.trigger_source.clone(),
            parent_run_id: run.parent_run_id,
            summary: run.summary.clone(),
            error_summary: run.error_summary.clone(),
            input_payload: object_value(&run.input_payload),
            output_summary: object_value(&run.output_summary),
            metrics: object_value(&run.metrics),
            started_at: run.started_at,
            completed_at: run.completed_at,
        };
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(task_run::table)
            .values(row)
            .returning(TaskRunRow::as_returning())
            .get_result::<TaskRunRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        *run = inserted.into();
        Ok(())
    }

    async fn update_run(&self, run: &TaskRun) -> Result<(), AppError> {
        let changes = TaskRunChangeset {
            kind: run.kind.as_str().to_owned(),
            task_name: run.task_name.clone(),
            status: run.status.as_str().to_owned(),
            organization_id: Some(run.organization_id),
            user_id: Some(run.user_id),
            triggered_by_user_id: Some(run.triggered_by_user_id),
            trigger_source: run.trigger_source.clone(),
            parent_run_id: Some(run.parent_run_id),
            summary: Some(run.summary.clone()),
            error_summary: Some(run.error_summary.clone()),
            input_payload: Some(Some(object_value(&run.input_payload))),
            output_summary: Some(Some(object_value(&run.output_summary))),
            metrics: Some(Some(object_value(&run.metrics))),
            started_at: Some(run.started_at),
            completed_at: Some(run.completed_at),
            created_at: Some(run.created_at),
            updated_at: Some(Utc::now()),
        };
        let mut connection = self.connection().await?;
        let rows = diesel::update(task_run::table.find(run.id))
            .set(changes)
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if rows == 0 {
            return Err(AppError::NotFound);
        }
        Ok(())
    }

    async fn find_run_by_id(&self, id: Uuid) -> Result<TaskRun, AppError> {
        let mut connection = self.connection().await?;
        task_run::table
            .find(id)
            .select(TaskRunRow::as_select())
            .first::<TaskRunRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Into::into)
            .ok_or(AppError::NotFound)
    }

    async fn list_runs(&self, filter: TaskRunFilter) -> Result<Vec<TaskRun>, AppError> {
        let mut query = task_run::table.into_boxed();
        if let Some(organization_id) = filter.organization_id {
            query = query.filter(task_run::organization_id.eq(organization_id));
        } else if let Some(user_id) = filter.user_id {
            query = query.filter(task_run::user_id.eq(user_id));
        }
        let task_name = filter.task_name.trim();
        if !task_name.is_empty() {
            query = query.filter(task_run::task_name.eq(task_name));
        }
        let status = filter.status.trim();
        if !status.is_empty() {
            query = query.filter(task_run::status.eq(status));
        }
        let kind = filter.kind.trim();
        if !kind.is_empty() {
            query = query.filter(task_run::kind.eq(kind));
        }
        let limit = if filter.limit <= 0 || filter.limit > 100 {
            50
        } else {
            filter.limit
        };
        let mut connection = self.connection().await?;
        query
            .order(task_run::created_at.desc())
            .limit(limit)
            .select(TaskRunRow::as_select())
            .load::<TaskRunRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Into::into).collect())
            .map_err(map_diesel_error)
    }

    async fn create_step(&self, step: &mut TaskRunStep) -> Result<(), AppError> {
        if step.id.is_nil() {
            step.id = Uuid::new_v4();
        }
        let row = NewTaskRunStepRow {
            id: step.id,
            task_run_id: step.task_run_id,
            step_key: step.step_key.clone(),
            step_name: step.step_name.clone(),
            status: step.status.as_str().to_owned(),
            summary: step.summary.clone(),
            error_summary: step.error_summary.clone(),
            metrics: object_value(&step.metrics),
            started_at: step.started_at,
            completed_at: step.completed_at,
        };
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(task_run_step::table)
            .values(row)
            .returning(TaskRunStepRow::as_returning())
            .get_result::<TaskRunStepRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        *step = inserted.into();
        Ok(())
    }

    async fn update_step(&self, step: &TaskRunStep) -> Result<(), AppError> {
        let changes = TaskRunStepChangeset {
            step_key: step.step_key.clone(),
            step_name: step.step_name.clone(),
            status: step.status.as_str().to_owned(),
            summary: Some(step.summary.clone()),
            error_summary: Some(step.error_summary.clone()),
            metrics: Some(Some(object_value(&step.metrics))),
            started_at: Some(step.started_at),
            completed_at: Some(step.completed_at),
            created_at: Some(step.created_at),
            updated_at: Some(Utc::now()),
        };
        let mut connection = self.connection().await?;
        let rows = diesel::update(task_run_step::table.find(step.id))
            .set(changes)
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if rows == 0 {
            return Err(AppError::NotFound);
        }
        Ok(())
    }

    async fn find_step_by_run_and_key(
        &self,
        run_id: Uuid,
        step_key: &str,
    ) -> Result<TaskRunStep, AppError> {
        let mut connection = self.connection().await?;
        task_run_step::table
            .filter(task_run_step::task_run_id.eq(run_id))
            .filter(task_run_step::step_key.eq(step_key))
            .select(TaskRunStepRow::as_select())
            .first::<TaskRunStepRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Into::into)
            .ok_or(AppError::NotFound)
    }

    async fn list_steps_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunStep>, AppError> {
        let mut connection = self.connection().await?;
        task_run_step::table
            .filter(task_run_step::task_run_id.eq(run_id))
            .order(task_run_step::created_at.asc())
            .select(TaskRunStepRow::as_select())
            .load::<TaskRunStepRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Into::into).collect())
            .map_err(map_diesel_error)
    }

    async fn create_event(&self, event: &mut TaskRunEvent) -> Result<(), AppError> {
        if event.id.is_nil() {
            event.id = Uuid::new_v4();
        }
        let row = NewTaskRunEventRow {
            id: event.id,
            task_run_id: event.task_run_id,
            task_run_step_id: event.task_run_step_id,
            event_type: event.event_type.clone(),
            level: event.level.as_str().to_owned(),
            message: event.message.clone(),
            meta_data: object_value(&event.meta_data),
        };
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(task_run_event::table)
            .values(row)
            .returning(TaskRunEventRow::as_returning())
            .get_result::<TaskRunEventRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        *event = inserted.into();
        Ok(())
    }

    async fn list_events_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunEvent>, AppError> {
        let mut connection = self.connection().await?;
        task_run_event::table
            .filter(task_run_event::task_run_id.eq(run_id))
            .order(task_run_event::created_at.asc())
            .select(TaskRunEventRow::as_select())
            .load::<TaskRunEventRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Into::into).collect())
            .map_err(map_diesel_error)
    }
}

impl From<TaskRunRow> for TaskRun {
    fn from(row: TaskRunRow) -> Self {
        Self {
            id: row.id,
            kind: row.kind.into(),
            task_name: row.task_name,
            status: row.status.into(),
            organization_id: row.organization_id,
            user_id: row.user_id,
            triggered_by_user_id: row.triggered_by_user_id,
            trigger_source: row.trigger_source,
            parent_run_id: row.parent_run_id,
            summary: row.summary,
            error_summary: row.error_summary,
            input_payload: value_object(row.input_payload),
            output_summary: value_object(row.output_summary),
            metrics: value_object(row.metrics),
            started_at: row.started_at,
            completed_at: row.completed_at,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

impl From<TaskRunStepRow> for TaskRunStep {
    fn from(row: TaskRunStepRow) -> Self {
        Self {
            id: row.id,
            task_run_id: row.task_run_id,
            step_key: row.step_key,
            step_name: row.step_name,
            status: row.status.into(),
            summary: row.summary,
            error_summary: row.error_summary,
            metrics: value_object(row.metrics),
            started_at: row.started_at,
            completed_at: row.completed_at,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

impl From<TaskRunEventRow> for TaskRunEvent {
    fn from(row: TaskRunEventRow) -> Self {
        Self {
            id: row.id,
            task_run_id: row.task_run_id,
            task_run_step_id: row.task_run_step_id,
            event_type: row.event_type,
            level: row.level.into(),
            message: row.message,
            meta_data: value_object(row.meta_data),
            created_at: row.created_at,
        }
    }
}

fn object_value(value: &JsonObject) -> Value {
    Value::Object(value.clone())
}

fn value_object(value: Option<Value>) -> JsonObject {
    value
        .and_then(|value| value.as_object().cloned())
        .unwrap_or_default()
}

fn map_diesel_error(error: diesel::result::Error) -> AppError {
    match error {
        diesel::result::Error::NotFound => AppError::NotFound,
        diesel::result::Error::DatabaseError(
            diesel::result::DatabaseErrorKind::UniqueViolation,
            _,
        ) => AppError::Conflict("task run record already exists".to_owned()),
        _ => AppError::Database,
    }
}
