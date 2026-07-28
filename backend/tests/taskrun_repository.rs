use std::{env, error::Error, io, sync::Arc};

use blog_backend::{
    core::taskrun::{
        JsonObject, TaskRun, TaskRunEvent, TaskRunEventLevel, TaskRunFilter, TaskRunKind,
        TaskRunRepository, TaskRunStatus, TaskRunStep,
    },
    database::{
        pool::{PgPool, create_pool},
        repository::task_run::DieselTaskRunRepository,
    },
    error::AppError,
    schema::{account, organization, task_run, task_run_event, task_run_step},
};
use chrono::{Duration, Utc};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use serde_json::{Value, json};
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the taskrun_repository target; start the Docker PostgreSQL service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("task-run test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

fn run_fixture(task_name: &str, organization_id: Option<Uuid>, user_id: Option<Uuid>) -> TaskRun {
    let mut input = JsonObject::new();
    input.insert("source".to_owned(), json!("test"));
    TaskRun {
        id: Uuid::nil(),
        kind: TaskRunKind::Worker,
        task_name: task_name.to_owned(),
        status: TaskRunStatus::Running,
        organization_id,
        user_id,
        triggered_by_user_id: None,
        trigger_source: "manual".to_owned(),
        parent_run_id: None,
        summary: None,
        error_summary: None,
        input_payload: input,
        output_summary: JsonObject::new(),
        metrics: JsonObject::new(),
        started_at: Some(Utc::now()),
        completed_at: None,
        created_at: None,
        updated_at: None,
    }
}

#[tokio::test]
async fn postgres_taskrun_repository_preserves_filters_json_order_and_constraints() -> TestResult {
    let pool = test_pool()?;
    let repository = Arc::new(DieselTaskRunRepository::new(pool.clone()));
    let user_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let mut connection = pool.get().await?;
    diesel::insert_into(organization::table)
        .values((
            organization::id.eq(organization_id),
            organization::name.eq("Task Run Test Organization"),
            organization::slug.eq(format!("task-run-{organization_id}")),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(user_id),
            account::name.eq("Task Run User"),
            account::email.eq(format!("{user_id}@example.test")),
            account::password_hash.eq("not-used"),
            account::role.eq("user"),
            account::organization_id.eq(Some(organization_id)),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let mut first = run_fixture("crawler", Some(organization_id), Some(user_id));
    let mut second = run_fixture("crawler", None, Some(user_id));
    let mut third = run_fixture("pipeline", Some(organization_id), Some(user_id));
    repository.create_run(&mut first).await?;
    repository.create_run(&mut second).await?;
    repository.create_run(&mut third).await?;
    assert!(first.created_at.is_some());
    assert!(first.updated_at.is_some());

    let now = Utc::now();
    let mut connection = pool.get().await?;
    diesel::update(task_run::table.find(first.id))
        .set(task_run::created_at.eq(Some(now - Duration::seconds(3))))
        .execute(&mut connection)
        .await?;
    diesel::update(task_run::table.find(second.id))
        .set(task_run::created_at.eq(Some(now - Duration::seconds(2))))
        .execute(&mut connection)
        .await?;
    diesel::update(task_run::table.find(third.id))
        .set(task_run::created_at.eq(Some(now - Duration::seconds(1))))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let organization_rows = repository
        .list_runs(TaskRunFilter {
            organization_id: Some(organization_id),
            user_id: Some(user_id),
            task_name: String::new(),
            status: " running ".to_owned(),
            kind: " worker ".to_owned(),
            limit: 50,
        })
        .await?;
    assert_eq!(
        organization_rows
            .iter()
            .map(|run| run.id)
            .collect::<Vec<_>>(),
        vec![third.id, first.id]
    );
    assert!(!organization_rows.iter().any(|run| run.id == second.id));
    let newest = repository
        .list_runs(TaskRunFilter {
            organization_id: None,
            user_id: Some(user_id),
            task_name: " crawler ".to_owned(),
            status: String::new(),
            kind: String::new(),
            limit: 1,
        })
        .await?;
    assert_eq!(newest.len(), 1);
    assert_eq!(newest[0].id, second.id);

    let mut connection = pool.get().await?;
    diesel::update(task_run::table.find(first.id))
        .set(task_run::input_payload.eq(Some(json!(["not", "an", "object"]))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert!(
        repository
            .find_run_by_id(first.id)
            .await?
            .input_payload
            .is_empty()
    );

    first.summary = Some("finished".to_owned());
    first.error_summary = None;
    first.output_summary.insert("count".to_owned(), json!(3));
    first.completed_at = Some(Utc::now());
    first.status = TaskRunStatus::Warning;
    repository.update_run(&first).await?;
    let reloaded = repository.find_run_by_id(first.id).await?;
    assert_eq!(reloaded.summary.as_deref(), Some("finished"));
    assert_eq!(reloaded.status, TaskRunStatus::Warning);
    assert_eq!(reloaded.output_summary.get("count"), Some(&json!(3)));

    let mut step_one = TaskRunStep {
        id: Uuid::nil(),
        task_run_id: first.id,
        step_key: "crawl".to_owned(),
        step_name: "Crawl".to_owned(),
        status: TaskRunStatus::Running,
        summary: None,
        error_summary: None,
        metrics: JsonObject::new(),
        started_at: Some(Utc::now()),
        completed_at: None,
        created_at: None,
        updated_at: None,
    };
    repository.create_step(&mut step_one).await?;
    let mut duplicate_step = step_one.clone();
    duplicate_step.id = Uuid::nil();
    assert!(matches!(
        repository.create_step(&mut duplicate_step).await,
        Err(AppError::Conflict(_))
    ));
    let mut step_two = TaskRunStep {
        step_key: "index".to_owned(),
        step_name: "Index".to_owned(),
        ..step_one.clone()
    };
    step_two.id = Uuid::nil();
    repository.create_step(&mut step_two).await?;
    let mut connection = pool.get().await?;
    diesel::update(task_run_step::table.find(step_one.id))
        .set(task_run_step::created_at.eq(Some(now - Duration::seconds(2))))
        .execute(&mut connection)
        .await?;
    diesel::update(task_run_step::table.find(step_two.id))
        .set(task_run_step::created_at.eq(Some(now - Duration::seconds(1))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert_eq!(
        repository
            .list_steps_by_run_id(first.id)
            .await?
            .iter()
            .map(|step| step.step_key.as_str())
            .collect::<Vec<_>>(),
        vec!["crawl", "index"]
    );

    let mut first_event = TaskRunEvent {
        id: Uuid::nil(),
        task_run_id: first.id,
        task_run_step_id: Some(step_one.id),
        event_type: "step_started".to_owned(),
        level: TaskRunEventLevel::Info,
        message: "Crawl".to_owned(),
        meta_data: JsonObject::new(),
        created_at: None,
    };
    let mut second_event = TaskRunEvent {
        event_type: "step_warning".to_owned(),
        level: TaskRunEventLevel::Warning,
        message: "warning".to_owned(),
        ..first_event.clone()
    };
    second_event.id = Uuid::nil();
    repository.create_event(&mut first_event).await?;
    repository.create_event(&mut second_event).await?;
    let mut connection = pool.get().await?;
    diesel::update(task_run_event::table.find(first_event.id))
        .set(task_run_event::created_at.eq(Some(now - Duration::seconds(2))))
        .execute(&mut connection)
        .await?;
    diesel::update(task_run_event::table.find(second_event.id))
        .set(task_run_event::created_at.eq(Some(now - Duration::seconds(1))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert_eq!(
        repository
            .list_events_by_run_id(first.id)
            .await?
            .iter()
            .map(|event| event.event_type.as_str())
            .collect::<Vec<_>>(),
        vec!["step_started", "step_warning"]
    );

    let mut connection = pool.get().await?;
    diesel::delete(task_run::table.filter(task_run::id.eq_any([first.id, second.id, third.id])))
        .execute(&mut connection)
        .await?;
    diesel::delete(account::table.find(user_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(organization::table.find(organization_id))
        .execute(&mut connection)
        .await?;
    assert_eq!(
        task_run_step::table
            .filter(task_run_step::task_run_id.eq(first.id))
            .count()
            .get_result::<i64>(&mut connection)
            .await?,
        0
    );
    assert_eq!(
        task_run_event::table
            .filter(task_run_event::task_run_id.eq(first.id))
            .count()
            .get_result::<i64>(&mut connection)
            .await?,
        0
    );
    Ok(())
}

#[test]
fn taskrun_string_types_preserve_unknown_database_values() {
    let kind = TaskRunKind::from("custom-kind".to_owned());
    let status = TaskRunStatus::from("custom-status".to_owned());
    let level = TaskRunEventLevel::from("custom-level".to_owned());
    assert_eq!(kind.as_str(), "custom-kind");
    assert_eq!(status.as_str(), "custom-status");
    assert_eq!(level.as_str(), "custom-level");
    assert_eq!(
        serde_json::to_value(status).unwrap_or(Value::Null),
        json!("custom-status")
    );
}
