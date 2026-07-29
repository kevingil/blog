use std::{env, error::Error, io, sync::Arc};

use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::{
        auth::AuthState,
        organization::{OrganizationState, router as organization_router},
        page::{PageState, router as page_router},
        profile::{ProfileState, router as profile_router},
        project::{ProjectState, router as project_router},
    },
    core::{
        auth::{Account, AccountId, AccountRepository, AuthService},
        organization::OrganizationService,
        page::PageService,
        profile::ProfileService,
        project::ProjectService,
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            account::DieselAccountRepository, organization::DieselOrganizationRepository,
            page::DieselPageRepository, project::DieselProjectRepository,
            site_settings::DieselSiteSettingsRepository, tag::DieselTagRepository,
        },
    },
};
use diesel::{Connection, PgConnection};
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
const TEST_SECRET: &str = "content-crud-http-database-test-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone)]
struct TestState {
    auth: AuthState,
    organization: OrganizationState,
    page: PageState,
    profile: ProfileState,
    project: ProjectState,
}

impl FromRef<TestState> for AuthState {
    fn from_ref(state: &TestState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<TestState> for OrganizationState {
    fn from_ref(state: &TestState) -> Self {
        state.organization.clone()
    }
}

impl FromRef<TestState> for PageState {
    fn from_ref(state: &TestState) -> Self {
        state.page.clone()
    }
}

impl FromRef<TestState> for ProfileState {
    fn from_ref(state: &TestState) -> Self {
        state.profile.clone()
    }
}

impl FromRef<TestState> for ProjectState {
    fn from_ref(state: &TestState) -> Self {
        state.project.clone()
    }
}

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for content_crud_http_database; start the Docker test PostgreSQL service and provide its URL",
        )
    })?;
    let mut connection = PgConnection::establish(&database_url)?;
    connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("content CRUD migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    body: impl Into<Body>,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut builder = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, "application/json");
    if let Some(bearer) = bearer {
        builder = builder.header("authorization", bearer);
    }
    let response = router.oneshot(builder.body(body.into())?).await?;
    let status = response.status();
    let bytes = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&bytes)?))
}

#[tokio::test]
async fn content_crud_routes_use_constructor_injected_postgres_services() -> TestResult {
    let pool = test_pool()?;
    let accounts = Arc::new(DieselAccountRepository::new(pool.clone()));
    let organizations = Arc::new(DieselOrganizationRepository::new(pool.clone()));
    let pages = Arc::new(DieselPageRepository::new(pool.clone()));
    let projects = Arc::new(DieselProjectRepository::new(pool.clone()));
    let settings = Arc::new(DieselSiteSettingsRepository::new(pool.clone()));
    let tags = Arc::new(DieselTagRepository::new(pool));
    let auth_service = Arc::new(AuthService::new(accounts.clone(), TEST_SECRET)?);
    let state = TestState {
        auth: AuthState::new(auth_service.clone()),
        organization: OrganizationState::new(Arc::new(OrganizationService::new(
            organizations.clone(),
            organizations.clone(),
        ))),
        page: PageState::new(Arc::new(PageService::new(pages))),
        profile: ProfileState::new(Arc::new(ProfileService::new(
            settings.clone(),
            settings.clone(),
            settings,
            organizations,
        ))),
        project: ProjectState::new(Arc::new(ProjectService::new(projects, tags))),
    };
    let router: Router = organization_router::<TestState>()
        .merge(page_router())
        .merge(profile_router())
        .merge(project_router())
        .with_state(state)
        .into();

    let account_id = AccountId::new(Uuid::new_v4());
    let password_hash = auth_service.hash_password("test-password").await?;
    accounts
        .create(&Account {
            id: account_id,
            name: "Content CRUD Admin".to_owned(),
            email: format!("content-crud-{}@example.com", account_id.into_inner()),
            password_hash,
            role: "admin".to_owned(),
            created_at: None,
            updated_at: None,
            bio: None,
            profile_image: None,
            email_public: None,
            social_links: None,
            meta_description: None,
            organization_id: None,
        })
        .await?;
    let bearer = format!("Bearer {}", auth_service.issue_token(account_id)?);

    let (status, unauthorized) = call(
        router.clone(),
        Method::POST,
        "/organizations",
        "{malformed",
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized}");
    assert_eq!(unauthorized["error"], "Not authenticated");
    assert_eq!(unauthorized["code"], "UNAUTHORIZED");

    let (status, invalid) = call(
        router.clone(),
        Method::POST,
        "/organizations",
        json!({"name": null, "slug": "x"}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{invalid}");
    assert_eq!(invalid["code"], "VALIDATION_ERROR");
    assert_eq!(invalid["details"]["Name"], "Name is required");
    assert_eq!(
        invalid["details"]["Slug"],
        "Slug must be at least 2 characters"
    );

    let suffix = Uuid::new_v4().simple().to_string();
    let slug = format!("content-org-{suffix}");
    let (status, created) = call(
        router.clone(),
        Method::POST,
        "/organizations",
        json!({
            "name": "Content Organization",
            "slug": slug,
            "social_links": {"github": "https://github.com/example"}
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{created}");
    assert_eq!(created["data"]["name"], "Content Organization");
    assert_eq!(
        created["data"]["social_links"]["github"],
        "https://github.com/example"
    );
    let organization_id = created["data"]["id"]
        .as_str()
        .ok_or_else(|| io::Error::other("organization response omitted id"))?;

    let (status, listed) = call(
        router.clone(),
        Method::GET,
        "/organizations",
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed}");
    assert!(listed["data"].as_array().is_some_and(|items| {
        items
            .iter()
            .any(|item| item["id"].as_str() == Some(organization_id))
    }));

    let (status, fetched) = call(
        router.clone(),
        Method::GET,
        &format!("/organizations/{organization_id}"),
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{fetched}");
    assert_eq!(fetched["data"]["slug"], slug);

    let (status, updated) = call(
        router.clone(),
        Method::PUT,
        &format!("/organizations/{organization_id}"),
        json!({"bio": "Updated organization bio"}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated}");
    assert_eq!(updated["data"]["bio"], "Updated organization bio");

    for path in [
        format!("/organizations/{organization_id}/join"),
        "/organizations/leave".to_owned(),
    ] {
        let (status, response) = call(
            router.clone(),
            Method::POST,
            &path,
            Body::empty(),
            Some(&bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::OK, "{response}");
        assert_eq!(response, json!({"data": {"success": true}}));
    }

    let page_slug = format!("content-page-{suffix}");
    let (status, page) = call(
        router.clone(),
        Method::POST,
        "/dashboard/pages",
        json!({
            "slug": page_slug,
            "title": "Content Page",
            "content": "Content page body",
            "description": "Page description",
            "is_published": true
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{page}");
    let page_id = page["data"]["id"]
        .as_str()
        .ok_or_else(|| io::Error::other("page response omitted id"))?;

    let (status, public_page) = call(
        router.clone(),
        Method::GET,
        &format!("/pages/{page_slug}"),
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{public_page}");
    assert_eq!(public_page["data"]["title"], "Content Page");

    let (status, pages) = call(
        router.clone(),
        Method::GET,
        "/dashboard/pages?page=bad&perPage=bad&isPublished=bad",
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{pages}");
    assert_eq!(pages["data"]["page"], 1);
    assert_eq!(pages["data"]["per_page"], 20);

    let (status, page_by_id) = call(
        router.clone(),
        Method::GET,
        &format!("/dashboard/pages/{page_id}"),
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{page_by_id}");

    let (status, updated_page) = call(
        router.clone(),
        Method::PUT,
        &format!("/dashboard/pages/{page_id}"),
        json!({"title": "Updated Content Page"}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated_page}");
    assert_eq!(updated_page["data"]["slug"], page_slug);

    let (status, my_profile) = call(
        router.clone(),
        Method::GET,
        "/profile",
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{my_profile}");
    assert_eq!(
        my_profile["data"]["id"],
        account_id.into_inner().to_string()
    );

    let (status, updated_profile) = call(
        router.clone(),
        Method::PUT,
        "/profile",
        json!({
            "bio": "Updated profile bio",
            "social_links": {"github": "https://github.com/example"}
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated_profile}");
    assert_eq!(updated_profile["data"]["bio"], "Updated profile bio");

    let (status, settings) = call(
        router.clone(),
        Method::PUT,
        "/profile/settings",
        json!({
            "public_profile_type": "user",
            "public_user_id": account_id.into_inner()
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{settings}");
    assert_eq!(settings["data"]["public_profile_type"], "user");

    let (status, fetched_settings) = call(
        router.clone(),
        Method::GET,
        "/profile/settings",
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{fetched_settings}");
    assert_eq!(
        fetched_settings["data"]["public_user_id"],
        account_id.into_inner().to_string()
    );

    let (status, public_profile) = call(
        router.clone(),
        Method::GET,
        "/profile/public",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{public_profile}");
    assert_eq!(public_profile["data"]["type"], "user");
    assert_eq!(
        public_profile["data"]["id"],
        Uuid::nil().to_string(),
        "the Go service returns the nil UUID in this DTO"
    );

    let (status, invalid_project) = call(
        router.clone(),
        Method::POST,
        "/projects",
        json!({"title": "", "description": "valid"}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{invalid_project}");
    assert_eq!(invalid_project["code"], "VALIDATION_ERROR");

    let (status, project) = call(
        router.clone(),
        Method::POST,
        "/projects",
        json!({
            "title": "Content Project",
            "description": "Content project description",
            "content": "Project body",
            "tags": ["rust", format!("content-{suffix}")],
            "image_url": "https://example.com/image.png",
            "url": "https://example.com/project"
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{project}");
    let project_id = project["data"]["id"]
        .as_str()
        .ok_or_else(|| io::Error::other("project response omitted id"))?;

    let (status, projects) = call(
        router.clone(),
        Method::GET,
        "/projects?page=bad&perPage=bad",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{projects}");
    assert_eq!(projects["data"]["current_page"], 1);
    assert!(projects["data"].get("total_pages").is_none());

    let (status, detail) = call(
        router.clone(),
        Method::GET,
        &format!("/projects/{project_id}"),
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{detail}");
    assert_eq!(detail["data"]["project"]["title"], "Content Project");
    assert!(
        detail["data"]["tags"]
            .as_array()
            .is_some_and(|tags| tags.iter().any(|tag| tag == "rust"))
    );

    let (status, updated_project) = call(
        router.clone(),
        Method::PUT,
        &format!("/projects/{project_id}"),
        json!({"title": "Updated Content Project"}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated_project}");
    assert_eq!(updated_project["data"]["title"], "Updated Content Project");

    for (path, protected) in [
        (format!("/dashboard/pages/{page_id}"), true),
        (format!("/projects/{project_id}"), true),
        (format!("/organizations/{organization_id}"), true),
    ] {
        let (status, response) = call(
            router.clone(),
            Method::DELETE,
            &path,
            Body::empty(),
            protected.then_some(bearer.as_str()),
        )
        .await?;
        assert_eq!(status, StatusCode::OK, "{response}");
        assert_eq!(response, json!({"data": {"success": true}}));
    }

    let (status, invalid_id) = call(
        router,
        Method::GET,
        "/projects/not-a-uuid",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{invalid_id}");
    assert_eq!(invalid_id["error"], "Invalid project ID");
    assert_eq!(invalid_id["code"], "INVALID_INPUT");

    Ok(())
}
