use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::{task_run, task_run_event, task_run_step};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = task_run)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskRunRow {
    pub id: Uuid,
    pub kind: String,
    pub task_name: String,
    pub status: String,
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub triggered_by_user_id: Option<Uuid>,
    pub trigger_source: String,
    pub parent_run_id: Option<Uuid>,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub input_payload: Option<Value>,
    pub output_summary: Option<Value>,
    pub metrics: Option<Value>,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = task_run)]
pub struct NewTaskRunRow {
    pub id: Uuid,
    pub kind: String,
    pub task_name: String,
    pub status: String,
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub triggered_by_user_id: Option<Uuid>,
    pub trigger_source: String,
    pub parent_run_id: Option<Uuid>,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub input_payload: Value,
    pub output_summary: Value,
    pub metrics: Value,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = task_run)]
pub struct TaskRunChangeset {
    pub kind: String,
    pub task_name: String,
    pub status: String,
    pub organization_id: Option<Option<Uuid>>,
    pub user_id: Option<Option<Uuid>>,
    pub triggered_by_user_id: Option<Option<Uuid>>,
    pub trigger_source: String,
    pub parent_run_id: Option<Option<Uuid>>,
    pub summary: Option<Option<String>>,
    pub error_summary: Option<Option<String>>,
    pub input_payload: Option<Option<Value>>,
    pub output_summary: Option<Option<Value>>,
    pub metrics: Option<Option<Value>>,
    pub started_at: Option<Option<DateTime<Utc>>>,
    pub completed_at: Option<Option<DateTime<Utc>>>,
    pub created_at: Option<Option<DateTime<Utc>>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = task_run_step)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskRunStepRow {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub step_key: String,
    pub step_name: String,
    pub status: String,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub metrics: Option<Value>,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = task_run_step)]
pub struct NewTaskRunStepRow {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub step_key: String,
    pub step_name: String,
    pub status: String,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub metrics: Value,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = task_run_step)]
pub struct TaskRunStepChangeset {
    pub step_key: String,
    pub step_name: String,
    pub status: String,
    pub summary: Option<Option<String>>,
    pub error_summary: Option<Option<String>>,
    pub metrics: Option<Option<Value>>,
    pub started_at: Option<Option<DateTime<Utc>>>,
    pub completed_at: Option<Option<DateTime<Utc>>>,
    pub created_at: Option<Option<DateTime<Utc>>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = task_run_event)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskRunEventRow {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub task_run_step_id: Option<Uuid>,
    pub event_type: String,
    pub level: String,
    pub message: String,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = task_run_event)]
pub struct NewTaskRunEventRow {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub task_run_step_id: Option<Uuid>,
    pub event_type: String,
    pub level: String,
    pub message: String,
    pub meta_data: Value,
}
