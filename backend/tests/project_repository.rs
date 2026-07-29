use std::{env, error::Error, io};

use blog_backend::{
    core::project::{Project, ProjectListOptions, ProjectRepository},
    database::{
        pool::{PgPool, create_pool},
        repository::project::DieselProjectRepository,
    },
    error::AppError,
    schema::project,
};
use diesel::{Connection, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the project_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("project test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn project_postgres_repository_preserves_arrays_empty_strings_order_and_updates() -> TestResult
{
    let pool = test_pool()?;
    let repository = DieselProjectRepository::new(pool.clone());
    let id = Uuid::new_v4();
    let mut value = Project {
        id,
        title: "Repository project".to_owned(),
        description: "Repository project description".to_owned(),
        content: String::new(),
        tag_ids: vec![1, 2],
        image_url: String::new(),
        url: String::new(),
        created_at: None,
        updated_at: None,
    };
    repository.save(&mut value).await?;
    let inserted = repository.find_by_id(id).await?;
    assert_eq!(inserted.tag_ids, vec![1, 2]);
    assert_eq!(inserted.content, "");
    assert_eq!(inserted.image_url, "");

    value.title = "Updated repository project".to_owned();
    value.tag_ids.clear();
    repository.update(&value).await?;
    let updated = repository.find_by_id(id).await?;
    assert_eq!(updated.title, "Updated repository project");
    assert!(updated.tag_ids.is_empty());

    let (projects, total) = repository
        .list(ProjectListOptions {
            page: 1,
            per_page: 20,
        })
        .await?;
    assert!(total >= 1);
    assert!(projects.iter().any(|project| project.id == id));
    repository.delete(id).await?;
    assert!(matches!(
        repository.find_by_id(id).await,
        Err(AppError::NotFound)
    ));
    let mut connection = pool.get().await?;
    diesel::delete(project::table.find(id))
        .execute(&mut connection)
        .await?;
    Ok(())
}
