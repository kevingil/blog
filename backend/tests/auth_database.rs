use std::{env, error::Error, io, sync::Arc};

use axum::{
    Router,
    body::Body,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::auth::{AuthState, routes},
    core::auth::{
        Account, AccountId, AccountRepository, AuthService, LoginInput, RegistrationInput,
    },
    database::{
        pool::{PgPool, create_pool},
        repository::account::DieselAccountRepository,
    },
    error::AppError,
    schema::{account, organization},
};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
const TEST_SECRET: &str = "database-auth-test-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the auth_database target; start the Docker test PostgreSQL service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("auth test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

fn router(auth: Arc<AuthService>) -> Router {
    routes::router::<AuthState>()
        .with_state(AuthState::new(auth))
        .into()
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    content_type: &str,
    body: String,
    authorization: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut builder = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, content_type);
    if let Some(authorization) = authorization {
        builder = builder.header("authorization", authorization);
    }
    let response = router.oneshot(builder.body(Body::from(body))?).await?;
    let status = response.status();
    let bytes = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&bytes)?))
}

fn multipart(fields: &[(&str, &str)]) -> (String, String) {
    let boundary = "auth-database-test-boundary";
    let mut body = String::new();
    for (name, value) in fields {
        body.push_str("--");
        body.push_str(boundary);
        body.push_str("\r\nContent-Disposition: form-data; name=\"");
        body.push_str(name);
        body.push_str("\"\r\n\r\n");
        body.push_str(value);
        body.push_str("\r\n");
    }
    body.push_str("--");
    body.push_str(boundary);
    body.push_str("--\r\n");
    (format!("multipart/form-data; boundary={boundary}"), body)
}

#[tokio::test]
async fn postgres_repository_and_http_auth_flow_preserve_atomic_fields() -> TestResult {
    let pool = test_pool()?;
    let repository = Arc::new(DieselAccountRepository::new(pool.clone()));
    let auth = Arc::new(AuthService::new(repository.clone(), TEST_SECRET)?);
    let account_id = AccountId(Uuid::new_v4());
    let email = format!("auth-db-{}@example.com", account_id.0);
    let password_hash = auth.hash_password("correctpassword").await?;
    let account_value = Account {
        id: account_id,
        name: "Database User".to_owned(),
        email: email.clone(),
        password_hash: password_hash.clone(),
        role: "user".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: None,
    };
    repository.create(&account_value).await?;

    let organization_id = Uuid::new_v4();
    let mut connection = pool.get().await?;
    diesel::insert_into(organization::table)
        .values((
            organization::id.eq(organization_id),
            organization::name.eq("Auth Database Test Organization"),
            organization::slug.eq(format!("auth-db-{organization_id}")),
        ))
        .execute(&mut connection)
        .await?;
    diesel::update(account::table.find(account_id.0))
        .set((
            account::bio.eq(Some("preserved bio")),
            account::profile_image.eq(Some("preserved.png")),
            account::email_public.eq(Some("public@example.com")),
            account::social_links.eq(Some(json!({"website": "https://example.com"}))),
            account::meta_description.eq(Some("preserved metadata")),
            account::organization_id.eq(Some(organization_id)),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    assert!(
        !repository
            .update_password_if_current(account_id, "stale-hash", "replacement-hash")
            .await?
    );
    assert!(
        !repository
            .delete_if_password_hash(account_id, "stale-hash")
            .await?
    );

    let login = auth
        .login(LoginInput {
            email: email.clone(),
            password: "correctpassword".to_owned(),
        })
        .await?;
    let bearer = format!("Bearer {}", login.token);
    let app = router(auth.clone());

    let updated_email = format!("updated-{email}");
    let (content_type, body) =
        multipart(&[("name", "Updated Database User"), ("email", &updated_email)]);
    let (status, response) = call(
        app.clone(),
        Method::PUT,
        "/auth/account",
        &content_type,
        body,
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(
        response,
        json!({"data": {"message": "Account updated successfully"}})
    );

    let updated = repository
        .find_by_id(account_id)
        .await?
        .ok_or_else(|| io::Error::other("updated account not found"))?;
    assert_eq!(updated.name, "Updated Database User");
    assert_eq!(updated.email, updated_email);
    assert_eq!(updated.password_hash, password_hash);
    assert_eq!(updated.bio.as_deref(), Some("preserved bio"));
    assert_eq!(updated.profile_image.as_deref(), Some("preserved.png"));
    assert_eq!(updated.email_public.as_deref(), Some("public@example.com"));
    assert_eq!(
        updated.meta_description.as_deref(),
        Some("preserved metadata")
    );
    assert_eq!(updated.organization_id, Some(organization_id));
    assert_eq!(
        updated
            .social_links
            .as_ref()
            .and_then(|links| links.get("website")),
        Some(&json!("https://example.com"))
    );

    let duplicate = Account {
        id: AccountId(Uuid::new_v4()),
        email: updated.email.clone(),
        ..updated.clone()
    };
    assert!(matches!(
        repository.create(&duplicate).await,
        Err(AppError::Conflict(_))
    ));

    let invalid_social_id = AccountId(Uuid::new_v4());
    let invalid_social = Account {
        id: invalid_social_id,
        email: format!("invalid-social-{}@example.com", invalid_social_id.0),
        ..updated
    };
    repository.create(&invalid_social).await?;
    let mut connection = pool.get().await?;
    diesel::update(account::table.find(invalid_social_id.0))
        .set(account::social_links.eq(Some(json!([]))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert!(matches!(
        repository.find_by_id(invalid_social_id).await,
        Err(AppError::Database)
    ));
    assert!(
        repository
            .delete_if_password_hash(invalid_social_id, &invalid_social.password_hash)
            .await?
    );

    let (content_type, body) = multipart(&[
        ("currentPassword", "correctpassword"),
        ("newPassword", "newpassword"),
    ]);
    let (status, _) = call(
        app.clone(),
        Method::PUT,
        "/auth/password",
        &content_type,
        body,
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);

    let relogin = auth
        .login(LoginInput {
            email: updated_email,
            password: "newpassword".to_owned(),
        })
        .await?;
    assert!(!relogin.token.is_empty());

    let (content_type, body) = multipart(&[("password", "newpassword")]);
    let (status, _) = call(
        app,
        Method::DELETE,
        "/auth/account",
        &content_type,
        body,
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK);
    assert!(repository.find_by_id(account_id).await?.is_none());

    let registration_email = format!("registration-{}@example.com", Uuid::new_v4());
    auth.register(RegistrationInput {
        name: "Registered User".to_owned(),
        email: registration_email.clone(),
        password: "registration-password".to_owned(),
    })
    .await?;
    let registered = repository
        .find_by_email(&registration_email)
        .await?
        .ok_or_else(|| io::Error::other("registered account not found"))?;
    assert!(registered.password_hash.starts_with("$2b$10$"));
    assert!(
        repository
            .delete_if_password_hash(registered.id, &registered.password_hash)
            .await?
    );
    let mut connection = pool.get().await?;
    diesel::delete(organization::table.find(organization_id))
        .execute(&mut connection)
        .await?;
    Ok(())
}
