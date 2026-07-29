use std::{
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::{
        agent::{self, AgentRequestQueue, AgentState, ChatRequest},
        auth::AuthState,
        storage::{self, StorageState},
        taskrun::{self, TaskRunState as TaskRunApiState},
    },
    core::{
        auth::{Account, AccountId, AccountRepository, AuthService},
        chat::{ChatMessage, ChatMessageRepository, ChatMessageService},
        storage::{ObjectListing, ObjectStore, StorageService},
        taskrun::{
            TaskRun, TaskRunEvent, TaskRunFilter, TaskRunRepository, TaskRunService, TaskRunStep,
        },
    },
    error::AppError,
};
use chrono::{TimeZone, Utc};
use http_body_util::BodyExt;
use serde_json::{Value, json};
use tokio_util::sync::CancellationToken;
use tower::ServiceExt;
use uuid::Uuid;

const TEST_SECRET: &str = "support-http-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone)]
struct HttpState {
    agent: AgentState,
    auth: AuthState,
    storage: StorageState,
    taskrun: TaskRunApiState,
}

impl FromRef<HttpState> for AgentState {
    fn from_ref(state: &HttpState) -> Self {
        state.agent.clone()
    }
}

impl FromRef<HttpState> for AuthState {
    fn from_ref(state: &HttpState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<HttpState> for StorageState {
    fn from_ref(state: &HttpState) -> Self {
        state.storage.clone()
    }
}

impl FromRef<HttpState> for TaskRunApiState {
    fn from_ref(state: &HttpState) -> Self {
        state.taskrun.clone()
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

#[derive(Default)]
struct ChatRepository {
    values: Mutex<Vec<ChatMessage>>,
}

impl ChatRepository {
    fn state(&self) -> MutexGuard<'_, Vec<ChatMessage>> {
        self.values
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl ChatMessageRepository for ChatRepository {
    async fn create(&self, message: &mut ChatMessage) -> Result<(), AppError> {
        self.state().push(message.clone());
        Ok(())
    }

    async fn find_by_id(&self, id: Uuid) -> Result<ChatMessage, AppError> {
        self.state()
            .iter()
            .find(|message| message.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_by_article(
        &self,
        article_id: Uuid,
        limit: i64,
    ) -> Result<Vec<ChatMessage>, AppError> {
        let mut messages = self
            .state()
            .iter()
            .filter(|message| message.article_id == article_id)
            .cloned()
            .collect::<Vec<_>>();
        messages.sort_by(|left, right| right.created_at.cmp(&left.created_at));
        messages.truncate(usize::try_from(limit).map_err(|_| AppError::Internal)?);
        Ok(messages)
    }

    async fn list_pending_artifacts(&self, article_id: Uuid) -> Result<Vec<ChatMessage>, AppError> {
        Ok(self
            .state()
            .iter()
            .filter(|message| {
                message.article_id == article_id
                    && message
                        .meta_data
                        .as_ref()
                        .and_then(|metadata| metadata.pointer("/artifact/status"))
                        == Some(&json!("pending"))
            })
            .cloned()
            .collect())
    }

    async fn update(&self, _message: &ChatMessage) -> Result<(), AppError> {
        Ok(())
    }

    async fn update_metadata(&self, id: Uuid, metadata: Value) -> Result<u64, AppError> {
        let mut state = self.state();
        let Some(message) = state.iter_mut().find(|message| message.id == id) else {
            return Ok(0);
        };
        message.meta_data = Some(metadata);
        Ok(1)
    }

    async fn delete_by_article(&self, article_id: Uuid) -> Result<u64, AppError> {
        let mut state = self.state();
        let before = state.len();
        state.retain(|message| message.article_id != article_id);
        u64::try_from(before - state.len()).map_err(|_| AppError::Internal)
    }
}

#[derive(Default)]
struct Requests {
    values: Mutex<Vec<ChatRequest>>,
}

#[async_trait]
impl AgentRequestQueue for Requests {
    async fn submit(&self, request: ChatRequest) -> Result<String, AppError> {
        self.values
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .push(request);
        Ok("request-123".to_owned())
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
enum StorageOperation {
    List(String, Option<String>),
    Put(String, Vec<u8>),
    Delete(String),
    Copy(String, String),
}

#[derive(Default)]
struct Store {
    operations: Mutex<Vec<StorageOperation>>,
}

impl Store {
    fn state(&self) -> MutexGuard<'_, Vec<StorageOperation>> {
        self.operations
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl ObjectStore for Store {
    async fn list(&self, prefix: &str, delimiter: Option<&str>) -> Result<ObjectListing, AppError> {
        self.state().push(StorageOperation::List(
            prefix.to_owned(),
            delimiter.map(str::to_owned),
        ));
        Ok(ObjectListing::default())
    }

    async fn put(&self, key: &str, data: Vec<u8>) -> Result<(), AppError> {
        self.state()
            .push(StorageOperation::Put(key.to_owned(), data));
        Ok(())
    }

    async fn delete(&self, key: &str) -> Result<(), AppError> {
        self.state().push(StorageOperation::Delete(key.to_owned()));
        Ok(())
    }

    async fn copy(&self, source_key: &str, destination_key: &str) -> Result<(), AppError> {
        self.state().push(StorageOperation::Copy(
            source_key.to_owned(),
            destination_key.to_owned(),
        ));
        Ok(())
    }
}

#[derive(Default)]
struct Runs {
    runs: Mutex<Vec<TaskRun>>,
    steps: Mutex<Vec<TaskRunStep>>,
    events: Mutex<Vec<TaskRunEvent>>,
    filters: Mutex<Vec<TaskRunFilter>>,
}

fn lock<T>(value: &Mutex<T>) -> MutexGuard<'_, T> {
    value
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

#[async_trait]
impl TaskRunRepository for Runs {
    async fn create_run(&self, _run: &mut TaskRun) -> Result<(), AppError> {
        Err(AppError::Internal)
    }

    async fn update_run(&self, _run: &TaskRun) -> Result<(), AppError> {
        Err(AppError::Internal)
    }

    async fn find_run_by_id(&self, id: Uuid) -> Result<TaskRun, AppError> {
        lock(&self.runs)
            .iter()
            .find(|run| run.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_runs(&self, filter: TaskRunFilter) -> Result<Vec<TaskRun>, AppError> {
        lock(&self.filters).push(filter.clone());
        Ok(lock(&self.runs)
            .iter()
            .filter(|run| match (filter.organization_id, filter.user_id) {
                (Some(id), _) => run.organization_id == Some(id),
                (None, Some(id)) => run.user_id == Some(id),
                (None, None) => true,
            })
            .cloned()
            .collect())
    }

    async fn create_step(&self, _step: &mut TaskRunStep) -> Result<(), AppError> {
        Err(AppError::Internal)
    }

    async fn update_step(&self, _step: &TaskRunStep) -> Result<(), AppError> {
        Err(AppError::Internal)
    }

    async fn find_step_by_run_and_key(
        &self,
        run_id: Uuid,
        step_key: &str,
    ) -> Result<TaskRunStep, AppError> {
        lock(&self.steps)
            .iter()
            .find(|step| step.task_run_id == run_id && step.step_key == step_key)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_steps_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunStep>, AppError> {
        Ok(lock(&self.steps)
            .iter()
            .filter(|step| step.task_run_id == run_id)
            .cloned()
            .collect())
    }

    async fn create_event(&self, _event: &mut TaskRunEvent) -> Result<(), AppError> {
        Err(AppError::Internal)
    }

    async fn list_events_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunEvent>, AppError> {
        Ok(lock(&self.events)
            .iter()
            .filter(|event| event.task_run_id == run_id)
            .cloned()
            .collect())
    }
}

struct Fixture {
    router: Router,
    bearer: String,
    article_id: Uuid,
    message_id: Uuid,
    chat: Arc<ChatRepository>,
    requests: Arc<Requests>,
    store: Arc<Store>,
    runs: Arc<Runs>,
    run_id: Uuid,
    organization_id: Uuid,
}

fn fixture() -> TestResult<Fixture> {
    let account_id = AccountId(Uuid::new_v4());
    let organization_id = Uuid::new_v4();
    let accounts = Arc::new(Accounts::default());
    accounts.state().push(Account {
        id: account_id,
        name: "Support User".to_owned(),
        email: "support@example.com".to_owned(),
        password_hash: String::new(),
        role: "user".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: Some(organization_id),
    });
    let auth_service = Arc::new(AuthService::new(accounts, TEST_SECRET)?);
    let bearer = format!("Bearer {}", auth_service.issue_token(account_id)?);

    let article_id = Uuid::new_v4();
    let message_id = Uuid::new_v4();
    let chat = Arc::new(ChatRepository::default());
    chat.state().extend([
        ChatMessage {
            id: Uuid::new_v4(),
            article_id,
            role: "user".to_owned(),
            content: "first".to_owned(),
            meta_data: Some(json!({})),
            created_at: Utc.timestamp_opt(10, 0).single(),
        },
        ChatMessage {
            id: message_id,
            article_id,
            role: "assistant".to_owned(),
            content: "second".to_owned(),
            meta_data: Some(json!({
                "artifact": {
                    "id": "artifact-1",
                    "type": "rewrite",
                    "status": "pending",
                    "content": "replacement",
                    "diff_preview": "-old\\n+new",
                    "title": "Rewrite",
                    "description": "Replace draft"
                }
            })),
            created_at: Utc.timestamp_opt(20, 0).single(),
        },
    ]);
    let requests = Arc::new(Requests::default());
    let store = Arc::new(Store::default());
    let runs = Arc::new(Runs::default());
    let run_id = Uuid::new_v4();
    let step_id = Uuid::new_v4();
    lock(&runs.runs).extend([
        TaskRun {
            id: run_id,
            kind: "worker".to_owned().into(),
            task_name: "crawl".to_owned(),
            status: "completed".to_owned().into(),
            organization_id: Some(organization_id),
            user_id: None,
            triggered_by_user_id: Some(account_id.into_inner()),
            trigger_source: "manual".to_owned(),
            parent_run_id: None,
            summary: None,
            error_summary: None,
            input_payload: Default::default(),
            output_summary: Default::default(),
            metrics: Default::default(),
            started_at: Utc.timestamp_opt(100, 0).single(),
            completed_at: Utc.timestamp_opt(102, 250_000_000).single(),
            created_at: Utc.timestamp_opt(100, 0).single(),
            updated_at: Utc.timestamp_opt(102, 0).single(),
        },
        TaskRun {
            id: Uuid::new_v4(),
            kind: "worker".to_owned().into(),
            task_name: "other".to_owned(),
            status: "running".to_owned().into(),
            organization_id: Some(Uuid::new_v4()),
            user_id: None,
            triggered_by_user_id: None,
            trigger_source: "manual".to_owned(),
            parent_run_id: None,
            summary: None,
            error_summary: None,
            input_payload: Default::default(),
            output_summary: Default::default(),
            metrics: Default::default(),
            started_at: None,
            completed_at: None,
            created_at: None,
            updated_at: None,
        },
    ]);
    lock(&runs.steps).push(TaskRunStep {
        id: step_id,
        task_run_id: run_id,
        step_key: "fetch".to_owned(),
        step_name: "Fetch".to_owned(),
        status: "completed".to_owned().into(),
        summary: None,
        error_summary: None,
        metrics: Default::default(),
        started_at: Utc.timestamp_opt(100, 0).single(),
        completed_at: Utc.timestamp_opt(101, 0).single(),
        created_at: Utc.timestamp_opt(100, 0).single(),
        updated_at: Utc.timestamp_opt(101, 0).single(),
    });
    lock(&runs.events).push(TaskRunEvent {
        id: Uuid::new_v4(),
        task_run_id: run_id,
        task_run_step_id: Some(step_id),
        event_type: "step_completed".to_owned(),
        level: "info".to_owned().into(),
        message: "Fetched".to_owned(),
        meta_data: Default::default(),
        created_at: Utc.timestamp_opt(101, 0).single(),
    });

    let state = HttpState {
        agent: AgentState::new(
            Arc::new(ChatMessageService::new(
                chat.clone(),
                CancellationToken::new(),
            )),
            requests.clone(),
        ),
        auth: AuthState::new(auth_service),
        storage: StorageState::new(Arc::new(StorageService::new(
            store.clone(),
            "https://cdn.example.test",
            CancellationToken::new(),
        ))),
        taskrun: TaskRunApiState::new(Arc::new(TaskRunService::new(
            runs.clone(),
            CancellationToken::new(),
        ))),
    };
    let router: Router = agent::router::<HttpState>()
        .merge(storage::router::<HttpState>())
        .merge(taskrun::router::<HttpState>())
        .with_state(state)
        .into();
    Ok(Fixture {
        router,
        bearer,
        article_id,
        message_id,
        chat,
        requests,
        store,
        runs,
        run_id,
        organization_id,
    })
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    content_type: Option<&str>,
    body: impl Into<Body>,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut builder = Request::builder().method(method).uri(path);
    if let Some(content_type) = content_type {
        builder = builder.header(CONTENT_TYPE, content_type);
    }
    if let Some(bearer) = bearer {
        builder = builder.header("authorization", bearer);
    }
    let response = router.oneshot(builder.body(body.into())?).await?;
    let status = response.status();
    let body = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&body)?))
}

#[tokio::test]
async fn agent_routes_preserve_submission_history_and_artifact_contracts() -> TestResult {
    let fixture = fixture()?;
    let (status, unauthorized) = call(
        fixture.router.clone(),
        Method::POST,
        "/agent",
        Some("application/json"),
        json!({"message":"hi","articleId":fixture.article_id}).to_string(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized}");

    let (status, submitted) = call(
        fixture.router.clone(),
        Method::POST,
        "/agent",
        Some("application/json"),
        json!({
            "message":"improve this",
            "documentContent":"html",
            "documentMarkdown":"markdown",
            "articleId":fixture.article_id
        })
        .to_string(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{submitted}");
    assert_eq!(submitted["data"]["requestId"], "request-123");
    assert_eq!(submitted["data"]["status"], "processing");
    {
        let queued = lock(&fixture.requests.values);
        assert_eq!(queued.len(), 1);
        assert_eq!(queued[0].document_markdown, "markdown");
    }

    let (status, history) = call(
        fixture.router.clone(),
        Method::GET,
        &format!(
            "/agent/conversations/{}?limit=not-a-number",
            fixture.article_id
        ),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{history}");
    assert_eq!(
        history["data"]["article_id"],
        fixture.article_id.to_string()
    );
    assert_eq!(history["data"]["total"], 2);
    assert_eq!(history["data"]["messages"][0]["content"], "first");
    assert_eq!(history["data"]["messages"][1]["content"], "second");

    let (status, pending) = call(
        fixture.router.clone(),
        Method::GET,
        &format!("/agent/artifacts/{}/pending", fixture.article_id),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{pending}");
    assert_eq!(
        pending["data"]["artifacts"][0]["id"],
        fixture.message_id.to_string()
    );

    let (status, accepted) = call(
        fixture.router.clone(),
        Method::POST,
        &format!("/agent/artifacts/{}/accept", fixture.message_id),
        Some("application/json"),
        json!({"feedback":"looks good"}).to_string(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{accepted}");
    assert_eq!(accepted["data"]["success"], true);
    let message = fixture
        .chat
        .state()
        .iter()
        .find(|message| message.id == fixture.message_id)
        .cloned()
        .ok_or("accepted message missing")?;
    assert_eq!(
        message
            .meta_data
            .as_ref()
            .and_then(|metadata| metadata.pointer("/artifact/status")),
        Some(&json!("accepted"))
    );
    assert_eq!(
        message
            .meta_data
            .as_ref()
            .and_then(|metadata| metadata.pointer("/user_action/feedback")),
        Some(&json!("looks good"))
    );

    let (status, cleared) = call(
        fixture.router,
        Method::DELETE,
        &format!("/agent/conversations/{}", fixture.article_id),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{cleared}");
    assert!(fixture.chat.state().is_empty());
    Ok(())
}

#[tokio::test]
async fn storage_routes_preserve_multipart_keys_urls_and_folder_methods() -> TestResult {
    let fixture = fixture()?;
    let (status, listed) = call(
        fixture.router.clone(),
        Method::GET,
        "/storage/files?prefix=images%2F",
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed}");
    assert_eq!(listed["data"]["files"], json!([]));
    assert_eq!(listed["data"]["folders"], json!([]));

    let boundary = "support-http-boundary";
    let body = format!(
        "--{boundary}\r\nContent-Disposition: form-data; name=\"key\"\r\n\r\nimages/post.txt\r\n\
         --{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"post.txt\"\r\n\
         Content-Type: text/plain\r\n\r\nhello\r\n--{boundary}--\r\n"
    );
    let (status, uploaded) = call(
        fixture.router.clone(),
        Method::POST,
        "/storage/upload",
        Some(&format!("multipart/form-data; boundary={boundary}")),
        body,
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{uploaded}");
    assert_eq!(uploaded["data"]["key"], "images/post.txt");
    assert_eq!(
        uploaded["data"]["url"],
        "https://cdn.example.test/images/post.txt"
    );

    for (method, path, body) in [
        (
            Method::POST,
            "/storage/folders",
            json!({"path":"drafts"}).to_string(),
        ),
        (
            Method::PUT,
            "/storage/folders",
            json!({"oldPath":"drafts/","newPath":"published/"}).to_string(),
        ),
        (Method::DELETE, "/storage/post.txt", String::new()),
    ] {
        let (status, response) = call(
            fixture.router.clone(),
            method,
            path,
            Some("application/json"),
            body,
            Some(&fixture.bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::OK, "{response}");
        assert_eq!(response["data"]["success"], true);
    }
    assert_eq!(
        fixture.store.state().as_slice(),
        [
            StorageOperation::List("images/".to_owned(), Some("/".to_owned())),
            StorageOperation::Put("images/post.txt".to_owned(), b"hello".to_vec()),
            StorageOperation::Put("drafts/".to_owned(), Vec::new()),
            StorageOperation::List("drafts/".to_owned(), None),
            StorageOperation::Delete("post.txt".to_owned()),
        ]
    );
    Ok(())
}

#[tokio::test]
async fn taskrun_routes_scope_by_organization_and_preserve_detail_event_json() -> TestResult {
    let fixture = fixture()?;
    let (status, listed) = call(
        fixture.router.clone(),
        Method::GET,
        "/task-runs?task_name=crawl&status=completed&kind=worker&limit=25",
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed}");
    assert_eq!(listed["data"]["runs"].as_array().map(Vec::len), Some(1));
    assert_eq!(listed["data"]["runs"][0]["id"], fixture.run_id.to_string());
    assert_eq!(listed["data"]["runs"][0]["duration_ms"], 2250);
    assert!(listed["data"]["runs"][0].get("summary").is_none());
    assert!(listed["data"]["runs"][0].get("output_summary").is_none());
    {
        let filters = lock(&fixture.runs.filters);
        assert_eq!(filters[0].organization_id, Some(fixture.organization_id));
        assert_eq!(filters[0].user_id, None);
        assert_eq!(filters[0].task_name, "crawl");
        assert_eq!(filters[0].limit, 25);
    }

    let (status, detail) = call(
        fixture.router.clone(),
        Method::GET,
        &format!("/task-runs/{}", fixture.run_id),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{detail}");
    assert_eq!(detail["data"]["steps"][0]["step_key"], "fetch");
    assert_eq!(detail["data"]["events"][0]["step_key"], "fetch");
    assert_eq!(
        detail["data"]["events"][0]["created_at"],
        "1970-01-01T00:01:41Z"
    );
    assert!(detail["data"]["events"][0].get("meta_data").is_none());

    let (status, events) = call(
        fixture.router.clone(),
        Method::GET,
        &format!("/task-runs/{}/events", fixture.run_id),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{events}");
    assert_eq!(events["data"]["events"][0]["event_type"], "step_completed");

    let inaccessible_id = lock(&fixture.runs.runs)
        .iter()
        .find(|run| run.organization_id != Some(fixture.organization_id))
        .map(|run| run.id)
        .ok_or("inaccessible run missing")?;
    let (status, not_found) = call(
        fixture.router,
        Method::GET,
        &format!("/task-runs/{inaccessible_id}"),
        None,
        Body::empty(),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::NOT_FOUND, "{not_found}");
    assert_eq!(not_found["code"], "NOT_FOUND");
    Ok(())
}

#[test]
fn support_openapi_has_stable_operations_security_and_multipart_contract() -> TestResult {
    let (_, document) = agent::router::<HttpState>()
        .merge(storage::router::<HttpState>())
        .merge(taskrun::router::<HttpState>())
        .split_for_parts();
    let document = serde_json::to_value(document)?;
    let operations = [
        ("/agent", "post", "submitAgentRequest"),
        (
            "/agent/conversations/{articleId}",
            "get",
            "getConversationHistory",
        ),
        (
            "/agent/conversations/{articleId}",
            "delete",
            "clearConversationHistory",
        ),
        (
            "/agent/artifacts/{articleId}/pending",
            "get",
            "getPendingArtifacts",
        ),
        (
            "/agent/artifacts/{messageId}/accept",
            "post",
            "acceptArtifact",
        ),
        (
            "/agent/artifacts/{messageId}/reject",
            "post",
            "rejectArtifact",
        ),
        ("/storage/files", "get", "listStorageFiles"),
        ("/storage/upload", "post", "uploadStorageFile"),
        ("/storage/{key}", "delete", "deleteStorageFile"),
        ("/storage/folders", "post", "createStorageFolder"),
        ("/storage/folders", "put", "updateStorageFolder"),
        ("/task-runs", "get", "listTaskRuns"),
        ("/task-runs/{id}", "get", "getTaskRun"),
        ("/task-runs/{id}/events", "get", "listTaskRunEvents"),
    ];
    for (path, method, operation_id) in operations {
        let operation = &document["paths"][path][method];
        assert_eq!(operation["operationId"], operation_id);
        assert_eq!(operation["security"][0]["bearerAuth"], json!([]));
    }
    assert!(
        document["paths"]["/storage/upload"]["post"]["requestBody"]["content"]
            ["multipart/form-data"]
            .is_object()
    );
    Ok(())
}
