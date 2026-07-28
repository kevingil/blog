use std::{collections::BTreeMap, env, error::Error, io};

use blog_backend::{
    core::page::{Page, PageListOptions, PageRepository},
    database::{
        pool::{PgPool, create_pool},
        repository::page::DieselPageRepository,
    },
    error::AppError,
    schema::page,
};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use serde_json::json;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the page_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("page test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn page_postgres_repository_preserves_defaults_json_filters_and_full_update() -> TestResult {
    let pool = test_pool()?;
    let repository = DieselPageRepository::new(pool.clone());
    let page_id = Uuid::new_v4();
    let slug = format!("page-repo-{}", &page_id.simple().to_string()[..20]);
    let mut value = Page {
        id: page_id,
        slug: slug.clone(),
        title: "Repository page".to_owned(),
        content: "Repository page content".to_owned(),
        description: String::new(),
        image_url: String::new(),
        meta_data: None,
        is_published: false,
        created_at: None,
        updated_at: None,
    };
    repository.save(&mut value).await?;

    // GORM applies its `default:true` tag for a false bool on insert.
    let inserted = repository.find_by_id(page_id).await?;
    assert!(inserted.is_published);
    assert_eq!(inserted.meta_data, Some(BTreeMap::new()));

    value.is_published = false;
    value.description = "Updated description".to_owned();
    value.meta_data = Some(BTreeMap::from([("kind".to_owned(), json!("test"))]));
    repository.save(&mut value).await?;
    let updated = repository.find_by_slug(&slug).await?;
    assert!(!updated.is_published);
    assert_eq!(updated.description, "Updated description");
    assert_eq!(
        updated
            .meta_data
            .as_ref()
            .and_then(|values| values.get("kind")),
        Some(&json!("test"))
    );
    let (drafts, total) = repository
        .list(PageListOptions {
            page: 1,
            per_page: 20,
            is_published: Some(false),
        })
        .await?;
    assert!(total >= 1);
    assert!(drafts.iter().any(|page| page.id == page_id));

    repository.delete(page_id).await?;
    assert!(matches!(
        repository.find_by_id(page_id).await,
        Err(AppError::NotFound)
    ));
    let mut connection = pool.get().await?;
    diesel::delete(page::table.filter(page::slug.eq(slug)))
        .execute(&mut connection)
        .await?;
    Ok(())
}
