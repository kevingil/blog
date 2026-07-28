use std::{env, error::Error, io};

use blog_backend::{
    core::{
        project::{Project, ProjectRepository},
        tag::{TagRepository, TagService},
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{project::DieselProjectRepository, tag::DieselTagRepository},
    },
    error::AppError,
    schema::{project, tag},
};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the tag_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("tag test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn tag_postgres_repository_preserves_case_insensitive_atomic_ensure_and_usage() -> TestResult
{
    let pool = test_pool()?;
    let repository = DieselTagRepository::new(pool.clone());
    let names = vec![" Go ".to_owned(), "go".to_owned(), "   ".to_owned()];
    let ids = repository.ensure_exists(&names).await?;
    assert_eq!(ids.len(), 2);
    assert_eq!(ids[0], ids[1]);
    let tag_id = i32::try_from(ids[0])?;
    assert_eq!(repository.find_by_name("GO").await?.id, tag_id);
    assert_eq!(repository.find_by_ids(&ids).await?.len(), 1);
    assert!(repository.list().await?.iter().any(|tag| tag.id == tag_id));

    let project_repository = DieselProjectRepository::new(pool.clone());
    let project_id = Uuid::new_v4();
    let mut project_value = Project {
        id: project_id,
        title: "Tag usage project".to_owned(),
        description: "A project used by the tag repository test".to_owned(),
        content: String::new(),
        tag_ids: vec![i64::from(tag_id)],
        image_url: String::new(),
        url: String::new(),
        created_at: None,
        updated_at: None,
    };
    project_repository.save(&mut project_value).await?;
    assert!(repository.is_used(tag_id).await?);
    assert!(
        TagService::new(std::sync::Arc::new(repository.clone()))
            .is_tag_used(tag_id)
            .await?
    );

    let mut connection = pool.get().await?;
    diesel::delete(project::table.find(project_id))
        .execute(&mut connection)
        .await?;
    repository.delete(tag_id).await?;
    assert!(matches!(
        repository.find_by_id(tag_id).await,
        Err(AppError::NotFound)
    ));
    diesel::delete(tag::table.filter(tag::name.eq("go")))
        .execute(&mut connection)
        .await?;
    Ok(())
}
