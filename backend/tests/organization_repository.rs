use std::{collections::BTreeMap, env, error::Error, io};

use blog_backend::{
    core::organization::{Organization, OrganizationAccountRepository, OrganizationRepository},
    database::{
        pool::{PgPool, create_pool},
        repository::organization::DieselOrganizationRepository,
    },
    error::AppError,
    schema::{account, organization},
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
            "TEST_DATABASE_URL is required for the organization_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| {
            io::Error::other(format!("organization test migration failed: {error}"))
        })?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn organization_postgres_repository_preserves_json_uniqueness_membership_and_delete()
-> TestResult {
    let pool = test_pool()?;
    let repository = DieselOrganizationRepository::new(pool.clone());
    let account_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let slug = format!("organization-repository-{organization_id}");
    let mut connection = pool.get().await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(account_id),
            account::name.eq("Organization repository user"),
            account::email.eq(format!("organization-{account_id}@example.com")),
            account::password_hash.eq("not-used"),
            account::role.eq("user"),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let mut value = Organization {
        id: organization_id,
        name: "Repository Organization".to_owned(),
        slug: slug.clone(),
        bio: None,
        logo_url: None,
        website_url: None,
        email_public: None,
        social_links: Some(BTreeMap::from([
            ("github".to_owned(), json!("kevin")),
            ("followers".to_owned(), json!(10)),
        ])),
        meta_description: None,
        created_at: None,
        updated_at: None,
    };
    repository.save(&mut value).await?;
    let loaded = repository.find_by_slug(&slug).await?;
    assert_eq!(
        loaded
            .social_links
            .as_ref()
            .and_then(|links| links.get("github")),
        Some(&json!("kevin"))
    );
    assert!(
        repository
            .list()
            .await?
            .iter()
            .any(|organization| organization.id == organization_id)
    );

    let mut duplicate = value.clone();
    duplicate.id = Uuid::new_v4();
    assert!(matches!(
        repository.save(&mut duplicate).await,
        Err(AppError::Conflict(_))
    ));
    assert!(
        repository
            .set_organization(account_id, Some(organization_id))
            .await?
    );
    let mut connection = pool.get().await?;
    assert_eq!(
        account::table
            .find(account_id)
            .select(account::organization_id)
            .first::<Option<Uuid>>(&mut connection)
            .await?,
        Some(organization_id)
    );
    drop(connection);

    value.name = "Updated Repository Organization".to_owned();
    repository.update(&value).await?;
    assert_eq!(
        repository.find_by_id(organization_id).await?.name,
        "Updated Repository Organization"
    );
    repository.delete(organization_id).await?;
    let mut connection = pool.get().await?;
    assert_eq!(
        account::table
            .find(account_id)
            .select(account::organization_id)
            .first::<Option<Uuid>>(&mut connection)
            .await?,
        None
    );
    diesel::delete(account::table.find(account_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(organization::table.find(duplicate.id))
        .execute(&mut connection)
        .await?;
    Ok(())
}
