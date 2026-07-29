use std::{
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{Method, Request, StatusCode},
};
use blog_backend::{
    api::{
        auth::AuthState,
        worker::{self, WorkerState as WorkerApiState},
    },
    core::{
        auth::{Account, AccountId, AccountRepository, AuthService},
        worker::{
            ManagerConfig, StatusService, SystemClock, Worker, WorkerContext, WorkerFailure,
            WorkerManager, WorkerResult, WorkerState as CoreWorkerState,
        },
    },
    error::AppError,
};
use http_body_util::BodyExt;
use serde_json::{Value, json};
use tokio::time::{Duration, timeout};
use tokio_util::sync::CancellationToken;
use tower::ServiceExt;
use uuid::Uuid;

const TEST_SECRET: &str = "worker-http-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone)]
struct HttpState {
    auth: AuthState,
    worker: WorkerApiState,
}

impl FromRef<HttpState> for AuthState {
    fn from_ref(state: &HttpState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<HttpState> for WorkerApiState {
    fn from_ref(state: &HttpState) -> Self {
        state.worker.clone()
    }
}

#[derive(Default)]
struct Accounts {
    values: Mutex<Vec<Account>>,
}

impl Accounts {
    fn state(&self) -> MutexGuard<'_, Vec<Account>> {
        self.values
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl AccountRepository for Accounts {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(self
            .state()
            .iter()
            .find(|account| account.id == id)
            .cloned())
    }

    async fn find_by_email(&self, email: &str) -> Result<Option<Account>, AppError> {
        Ok(self
            .state()
            .iter()
            .find(|account| account.email == email)
            .cloned())
    }

    async fn create(&self, account: &Account) -> Result<(), AppError> {
        self.state().push(account.clone());
        Ok(())
    }

    async fn update_identity(
        &self,
        _id: AccountId,
        _name: &str,
        _email: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn update_password_if_current(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
        _new_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn delete_if_password_hash(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }
}

struct BlockingWorker;

#[async_trait]
impl Worker for BlockingWorker {
    fn name(&self) -> &str {
        "crawl"
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        context.cancelled().await;
        Err(WorkerFailure::new("operation cancelled"))
    }
}

struct Fixture {
    router: Router,
    bearer: String,
    manager: Arc<WorkerManager>,
    status: Arc<StatusService>,
}

fn fixture() -> TestResult<Fixture> {
    let account_id = AccountId(Uuid::new_v4());
    let accounts = Arc::new(Accounts::default());
    accounts.state().push(Account {
        id: account_id,
        name: "Worker Operator".to_owned(),
        email: "worker@example.com".to_owned(),
        password_hash: String::new(),
        role: "user".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: Some(Uuid::new_v4()),
    });
    let auth_service = Arc::new(AuthService::new(accounts, TEST_SECRET)?);
    let bearer = format!("Bearer {}", auth_service.issue_token(account_id)?);
    let status = Arc::new(StatusService::new(Arc::new(SystemClock)));
    let manager = WorkerManager::new(
        status.clone(),
        None,
        CancellationToken::new(),
        ManagerConfig::default(),
    )?;
    manager.register(Arc::new(BlockingWorker));
    manager.start()?;
    let state = HttpState {
        auth: AuthState::new(auth_service),
        worker: WorkerApiState::new(manager.clone(), status.clone()),
    };
    let router: Router = worker::router::<HttpState>().with_state(state).into();
    Ok(Fixture {
        router,
        bearer,
        manager,
        status,
    })
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut request = Request::builder().method(method).uri(path);
    if let Some(bearer) = bearer {
        request = request.header("authorization", bearer);
    }
    let response = router.oneshot(request.body(Body::empty())?).await?;
    let status = response.status();
    let body = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&body)?))
}

#[tokio::test]
async fn worker_http_routes_preserve_auth_status_run_stop_and_error_contracts() -> TestResult {
    let fixture = fixture()?;
    let (status, unauthorized) =
        call(fixture.router.clone(), Method::GET, "/workers/status", None).await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized}");

    let (status, all) = call(
        fixture.router.clone(),
        Method::GET,
        "/workers/status",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{all}");
    assert_eq!(all["data"]["is_running"], true);
    assert_eq!(all["data"]["workers"][0]["name"], "crawl");
    assert_eq!(all["data"]["workers"][0]["state"], "idle");
    assert!(all["data"]["workers"][0].get("task_run_id").is_none());
    assert!(all["data"]["workers"][0].get("error").is_none());

    let mut updates = fixture.status.subscribe();
    let (status, started) = call(
        fixture.router.clone(),
        Method::POST,
        "/workers/crawl/run",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{started}");
    assert_eq!(started["data"]["started"], true);
    assert_eq!(started["data"]["message"], "Worker started successfully");
    assert_eq!(started["data"]["task_run_id"], "");
    timeout(Duration::from_secs(1), async {
        loop {
            let update = updates.recv().await.ok_or("status stream closed")?;
            if update.worker_name == "crawl" && update.status.state == CoreWorkerState::Running {
                return TestResult::Ok(());
            }
        }
    })
    .await??;

    let (status, running) = call(
        fixture.router.clone(),
        Method::GET,
        "/workers/running",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{running}");
    assert_eq!(running["data"]["workers"], json!(["crawl"]));

    let (status, current) = call(
        fixture.router.clone(),
        Method::GET,
        "/workers/crawl/status",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{current}");
    assert_eq!(current["data"]["state"], "running");
    assert!(current["data"]["started_at"].is_string());

    let (status, duplicate) = call(
        fixture.router.clone(),
        Method::POST,
        "/workers/crawl/run",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{duplicate}");
    assert_eq!(duplicate["code"], "INVALID_INPUT");

    let (status, stopped) = call(
        fixture.router.clone(),
        Method::POST,
        "/workers/crawl/stop",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{stopped}");
    assert_eq!(stopped["data"]["stopped"], true);
    assert_eq!(stopped["data"]["message"], "Worker stopped successfully");
    fixture.manager.shutdown().await?;

    let (status, not_running) = call(
        fixture.router.clone(),
        Method::POST,
        "/workers/crawl/stop",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{not_running}");

    let (status, missing) = call(
        fixture.router,
        Method::GET,
        "/workers/missing/status",
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::NOT_FOUND, "{missing}");
    Ok(())
}

#[test]
fn worker_openapi_has_all_stable_operations_and_bearer_security() -> TestResult {
    let (_, document) = worker::router::<HttpState>().split_for_parts();
    let document = serde_json::to_value(document)?;
    for (path, method, operation_id) in [
        ("/workers/status", "get", "getAllWorkerStatus"),
        ("/workers/running", "get", "getRunningWorkers"),
        ("/workers/{name}/status", "get", "getWorkerStatus"),
        ("/workers/{name}/run", "post", "runWorker"),
        ("/workers/{name}/stop", "post", "stopWorker"),
    ] {
        let operation = &document["paths"][path][method];
        assert_eq!(operation["operationId"], operation_id);
        assert_eq!(operation["security"][0]["bearerAuth"], json!([]));
    }
    Ok(())
}
