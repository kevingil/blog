use std::{collections::BTreeMap, env, error::Error, io};

use blog_backend::{
    core::{
        organization::{Organization, OrganizationRepository},
        profile::{
            ProfileAccountRepository, ProfileRepository, SiteSettings, SiteSettingsRepository,
        },
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            organization::DieselOrganizationRepository, site_settings::DieselSiteSettingsRepository,
        },
    },
    schema::{account, site_settings},
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
            "TEST_DATABASE_URL is required for the profile_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("profile test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn profile_postgres_repository_preserves_public_selection_settings_and_updates() -> TestResult
{
    let pool = test_pool()?;
    let profiles = DieselSiteSettingsRepository::new(pool.clone());
    let organizations = DieselOrganizationRepository::new(pool.clone());
    let user_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let mut connection = pool.get().await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(user_id),
            account::name.eq("Public User"),
            account::email.eq(format!("profile-{user_id}@example.com")),
            account::password_hash.eq("not-used"),
            account::role.eq("admin"),
            account::bio.eq("User bio"),
            account::social_links.eq(json!({"github": "user", "followers": 3})),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let mut organization = Organization {
        id: organization_id,
        name: "Public Organization".to_owned(),
        slug: format!("profile-organization-{organization_id}"),
        bio: Some("Organization bio".to_owned()),
        logo_url: Some("https://example.com/logo.png".to_owned()),
        website_url: Some("https://example.com".to_owned()),
        email_public: None,
        social_links: Some(BTreeMap::from([(
            "github".to_owned(),
            json!("organization"),
        )])),
        meta_description: None,
        created_at: None,
        updated_at: None,
    };
    organizations.save(&mut organization).await?;

    let mut settings = SiteSettings {
        id: 99,
        public_profile_type: "organization".to_owned(),
        public_user_id: Some(user_id),
        public_organization_id: Some(organization_id),
        created_at: None,
        updated_at: None,
    };
    profiles.save(&mut settings).await?;
    assert_eq!(settings.id, 1);
    assert_eq!(profiles.get().await?.public_profile_type, "organization");
    let public_organization = profiles.get_public_profile().await?;
    assert_eq!(public_organization.name, "Public Organization");
    assert_eq!(
        public_organization
            .social_links
            .as_ref()
            .and_then(|links| links.get("github")),
        Some(&json!("organization"))
    );

    settings.public_profile_type = "user".to_owned();
    profiles.save(&mut settings).await?;
    let public_user = profiles.get_public_profile().await?;
    assert_eq!(public_user.name, "Public User");
    assert!(profiles.is_user_admin(user_id).await?);
    let mut account_profile = profiles.find_profile_account(user_id).await?;
    account_profile.name = "Updated Public User".to_owned();
    account_profile.social_links = Some(BTreeMap::from([(
        "github".to_owned(),
        json!("updated-user"),
    )]));
    profiles.update_profile_account(&account_profile).await?;
    assert_eq!(
        profiles.find_profile_account(user_id).await?.name,
        "Updated Public User"
    );

    let mut reset = SiteSettings {
        id: 1,
        public_profile_type: "user".to_owned(),
        public_user_id: None,
        public_organization_id: None,
        created_at: None,
        updated_at: None,
    };
    profiles.save(&mut reset).await?;
    organizations.delete(organization_id).await?;
    let mut connection = pool.get().await?;
    diesel::delete(account::table.find(user_id))
        .execute(&mut connection)
        .await?;
    diesel::update(site_settings::table.find(1))
        .set((
            site_settings::public_user_id.eq::<Option<Uuid>>(None),
            site_settings::public_organization_id.eq::<Option<Uuid>>(None),
        ))
        .execute(&mut connection)
        .await?;
    Ok(())
}
