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
        auth::AuthState,
        image::{ImageGenerationJob, ImageGenerationQueue, ImageState, router as image_router},
    },
    core::{
        auth::{Account, AccountId, AccountRepository, AuthService},
        image::{
            IMAGE_STATUS_COMPLETED, IMAGE_STATUS_FAILED, ImageGeneration, ImageRepository,
            ImageService,
        },
    },
    error::AppError,
};
use http_body_util::BodyExt;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const TEST_SECRET: &str = "image-http-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn lock<T>(value: &Mutex<T>) -> MutexGuard<'_, T> {
    value
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

#[derive(Clone)]
struct TestState {
    auth: AuthState,
    image: ImageState,
}

impl FromRef<TestState> for AuthState {
    fn from_ref(state: &TestState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<TestState> for ImageState {
    fn from_ref(state: &TestState) -> Self {
        state.image.clone()
    }
}

#[derive(Default)]
struct Accounts;

#[async_trait]
impl AccountRepository for Accounts {
    async fn find_by_id(&self, _id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(None)
    }

    async fn find_by_email(&self, _email: &str) -> Result<Option<Account>, AppError> {
        Ok(None)
    }

    async fn create(&self, _account: &Account) -> Result<(), AppError> {
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
struct Images {
    values: Mutex<Vec<ImageGeneration>>,
}

#[async_trait]
impl ImageRepository for Images {
    async fn find_by_id(&self, id: Uuid) -> Result<ImageGeneration, AppError> {
        lock(&self.values)
            .iter()
            .find(|image| image.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_request_id(&self, request_id: &str) -> Result<ImageGeneration, AppError> {
        lock(&self.values)
            .iter()
            .find(|image| image.request_id == request_id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn save(&self, image: &mut ImageGeneration) -> Result<(), AppError> {
        lock(&self.values).push(image.clone());
        Ok(())
    }

    async fn update(&self, image: &ImageGeneration) -> Result<(), AppError> {
        let mut values = lock(&self.values);
        let current = values
            .iter_mut()
            .find(|current| current.id == image.id)
            .ok_or(AppError::NotFound)?;
        *current = image.clone();
        Ok(())
    }
}

#[derive(Default)]
struct Queue {
    jobs: Mutex<Vec<ImageGenerationJob>>,
    fail: Mutex<bool>,
}

#[async_trait]
impl ImageGenerationQueue for Queue {
    fn provider(&self) -> &str {
        "test-provider"
    }

    fn model_name(&self) -> &str {
        "test-model"
    }

    async fn enqueue(&self, job: ImageGenerationJob) -> Result<(), AppError> {
        if *lock(&self.fail) {
            return Err(AppError::External);
        }
        lock(&self.jobs).push(job);
        Ok(())
    }
}

struct Fixture {
    router: Router,
    bearer: String,
    images: Arc<Images>,
    queue: Arc<Queue>,
}

fn fixture() -> TestResult<Fixture> {
    let auth_service = Arc::new(AuthService::new(Arc::new(Accounts), TEST_SECRET)?);
    let account_id = AccountId(Uuid::new_v4());
    let bearer = format!("Bearer {}", auth_service.issue_token(account_id)?);
    let images = Arc::new(Images::default());
    let queue = Arc::new(Queue::default());
    let state = TestState {
        auth: AuthState::new(auth_service),
        image: ImageState::new(Arc::new(ImageService::new(images.clone())), queue.clone()),
    };
    Ok(Fixture {
        router: image_router::<TestState>().with_state(state).into(),
        bearer,
        images,
        queue,
    })
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    body: Value,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut request = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, "application/json");
    if let Some(bearer) = bearer {
        request = request.header("authorization", bearer);
    }
    let response = router
        .oneshot(request.body(Body::from(serde_json::to_vec(&body)?))?)
        .await?;
    let status = response.status();
    let bytes = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&bytes)?))
}

#[tokio::test]
async fn image_routes_require_bearer_authentication() -> TestResult {
    let fixture = fixture()?;
    for (method, path, body) in [
        (
            Method::POST,
            "/images/generate",
            json!({
                "prompt": "A city at dawn",
                "article_id": Uuid::new_v4(),
                "generate_prompt": false
            }),
        ),
        (Method::GET, "/images/request-1", json!(null)),
        (Method::GET, "/images/request-1/status", json!(null)),
    ] {
        let (status, response) = call(fixture.router.clone(), method, path, body, None).await?;
        assert_eq!(status, StatusCode::UNAUTHORIZED, "{response}");
        assert_eq!(response["code"], "UNAUTHORIZED");
    }
    Ok(())
}

#[tokio::test]
async fn generate_persists_pending_record_before_queueing_exact_job() -> TestResult {
    let fixture = fixture()?;
    let article_id = Uuid::new_v4();
    let (status, response) = call(
        fixture.router,
        Method::POST,
        "/images/generate",
        json!({
            "prompt": "A city at dawn",
            "article_id": article_id,
            "generate_prompt": true
        }),
        Some(&fixture.bearer),
    )
    .await?;

    assert_eq!(status, StatusCode::ACCEPTED, "{response}");
    let request_id = response["data"]["request_id"]
        .as_str()
        .ok_or("missing request_id")?;
    Uuid::parse_str(request_id)?;
    let images = lock(&fixture.images.values);
    assert_eq!(images.len(), 1);
    assert_eq!(images[0].request_id, request_id);
    assert_eq!(images[0].provider, "test-provider");
    assert_eq!(images[0].model_name, "test-model");
    assert_eq!(images[0].status, "pending");
    assert_eq!(
        images[0]
            .meta_data
            .as_ref()
            .and_then(|value| value.get("article_id")),
        Some(&json!(article_id))
    );
    drop(images);
    assert_eq!(
        lock(&fixture.queue.jobs).as_slice(),
        &[ImageGenerationJob {
            image_id: lock(&fixture.images.values)[0].id,
            request_id: request_id.to_owned(),
            article_id,
            prompt: "A city at dawn".to_owned(),
            generate_prompt: true,
        }]
    );
    Ok(())
}

#[tokio::test]
async fn generate_rejects_empty_prompt_and_non_uuid_article() -> TestResult {
    let fixture = fixture()?;
    let (status, empty) = call(
        fixture.router.clone(),
        Method::POST,
        "/images/generate",
        json!({
            "prompt": "   ",
            "article_id": Uuid::new_v4(),
            "generate_prompt": false
        }),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{empty}");
    assert_eq!(empty["code"], "VALIDATION_ERROR");

    let (status, invalid) = call(
        fixture.router,
        Method::POST,
        "/images/generate",
        json!({
            "prompt": "prompt",
            "article_id": "not-a-uuid",
            "generate_prompt": false
        }),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{invalid}");
    assert_eq!(invalid["code"], "INVALID_INPUT");
    assert_eq!(invalid["error"], "Invalid request body");
    assert!(lock(&fixture.images.values).is_empty());
    assert!(lock(&fixture.queue.jobs).is_empty());
    Ok(())
}

#[tokio::test]
async fn enqueue_failure_marks_the_persisted_request_failed() -> TestResult {
    let fixture = fixture()?;
    *lock(&fixture.queue.fail) = true;
    let (status, response) = call(
        fixture.router,
        Method::POST,
        "/images/generate",
        json!({
            "prompt": "A city at dawn",
            "article_id": Uuid::new_v4(),
            "generate_prompt": false
        }),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_GATEWAY, "{response}");
    assert_eq!(response["code"], "EXTERNAL_SERVICE_ERROR");
    let images = lock(&fixture.images.values);
    assert_eq!(images.len(), 1);
    assert_eq!(images[0].status, IMAGE_STATUS_FAILED);
    assert_eq!(
        images[0].error_message,
        "failed to enqueue image generation"
    );
    assert!(images[0].completed_at.is_some());
    Ok(())
}

#[tokio::test]
async fn get_and_status_use_request_id_and_frontend_compatible_fields() -> TestResult {
    let fixture = fixture()?;
    let request_id = "request-123";
    lock(&fixture.images.values).push(ImageGeneration {
        id: Uuid::new_v4(),
        prompt: "A city at dawn".to_owned(),
        provider: "test-provider".to_owned(),
        model_name: "test-model".to_owned(),
        request_id: request_id.to_owned(),
        status: IMAGE_STATUS_COMPLETED.to_owned(),
        output_url: "https://cdn.example.test/image.png".to_owned(),
        file_index_id: None,
        error_message: String::new(),
        meta_data: None,
        created_at: None,
        completed_at: None,
    });

    let (status, image) = call(
        fixture.router.clone(),
        Method::GET,
        &format!("/images/{request_id}"),
        json!(null),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{image}");
    assert_eq!(image["data"]["request_id"], request_id);
    assert_eq!(
        image["data"]["output_url"],
        "https://cdn.example.test/image.png"
    );
    assert_eq!(image["data"]["model"], "test-model");

    let (status, response) = call(
        fixture.router,
        Method::GET,
        &format!("/images/{request_id}/status"),
        json!(null),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{response}");
    assert_eq!(response["data"]["accepted"], true);
    assert_eq!(response["data"]["requestId"], request_id);
    assert_eq!(response["data"]["request_id"], request_id);
    assert_eq!(
        response["data"]["outputUrl"],
        "https://cdn.example.test/image.png"
    );
    assert_eq!(
        response["data"]["output_url"],
        "https://cdn.example.test/image.png"
    );
    Ok(())
}

#[tokio::test]
async fn unknown_request_id_returns_not_found() -> TestResult {
    let fixture = fixture()?;
    let (status, response) = call(
        fixture.router,
        Method::GET,
        "/images/unknown/status",
        json!(null),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::NOT_FOUND, "{response}");
    assert_eq!(response["code"], "NOT_FOUND");
    Ok(())
}

#[test]
fn image_openapi_describes_the_frontend_contract_and_compatibility_fields() -> TestResult {
    let (_, document) = image_router::<TestState>().split_for_parts();
    let document = serde_json::to_value(document)?;
    assert_eq!(
        document["paths"]["/images/generate"]["post"]["operationId"],
        "generateImage"
    );
    assert_eq!(
        document["paths"]["/images/{requestId}"]["get"]["operationId"],
        "getImageGeneration"
    );
    assert_eq!(
        document["paths"]["/images/{requestId}/status"]["get"]["operationId"],
        "getImageGenerationStatus"
    );
    let status_properties =
        &document["components"]["schemas"]["ImageGenerationStatus"]["properties"];
    for field in [
        "accepted",
        "requestId",
        "outputUrl",
        "request_id",
        "output_url",
    ] {
        assert!(
            status_properties.get(field).is_some(),
            "missing status field {field}: {status_properties}"
        );
    }
    assert_eq!(
        document["components"]["schemas"]["GenerateImageRequest"]["properties"]["prompt"]["minLength"],
        1
    );
    Ok(())
}
