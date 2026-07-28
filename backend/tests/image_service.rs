use std::{
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::image::{
        CreateImageRequest, IMAGE_STATUS_COMPLETED, IMAGE_STATUS_FAILED, IMAGE_STATUS_PENDING,
        ImageGeneration, ImageRepository, ImageService,
    },
    error::AppError,
};
use uuid::Uuid;

type TestResult = Result<(), Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct Store {
    values: Mutex<Vec<ImageGeneration>>,
}

fn lock<T>(value: &Mutex<T>) -> Result<MutexGuard<'_, T>, AppError> {
    value.lock().map_err(|_| AppError::Internal)
}

fn image(id: Uuid, request_id: &str) -> ImageGeneration {
    ImageGeneration {
        id,
        prompt: "A landscape".to_owned(),
        provider: "provider".to_owned(),
        model_name: "model".to_owned(),
        request_id: request_id.to_owned(),
        status: IMAGE_STATUS_PENDING.to_owned(),
        output_url: String::new(),
        file_index_id: None,
        error_message: String::new(),
        meta_data: None,
        created_at: None,
        completed_at: None,
    }
}

#[async_trait]
impl ImageRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<ImageGeneration, AppError> {
        lock(&self.values)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_request_id(&self, request_id: &str) -> Result<ImageGeneration, AppError> {
        lock(&self.values)?
            .iter()
            .find(|value| value.request_id == request_id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn save(&self, image: &mut ImageGeneration) -> Result<(), AppError> {
        lock(&self.values)?.push(image.clone());
        Ok(())
    }

    async fn update(&self, image: &ImageGeneration) -> Result<(), AppError> {
        let mut values = lock(&self.values)?;
        let current = values
            .iter_mut()
            .find(|value| value.id == image.id)
            .ok_or(AppError::NotFound)?;
        *current = image.clone();
        Ok(())
    }
}

#[tokio::test]
async fn test_service_get_by_id() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.values)?.push(image(id, "request"));
    assert_eq!(ImageService::new(store.clone()).get_by_id(id).await?.id, id);
    assert!(matches!(
        ImageService::new(store).get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_get_by_request_id() -> TestResult {
    let store = Arc::new(Store::default());
    lock(&store.values)?.push(image(Uuid::new_v4(), "request"));
    assert_eq!(
        ImageService::new(store)
            .get_by_request_id("request")
            .await?
            .request_id,
        "request"
    );
    Ok(())
}

#[tokio::test]
async fn test_service_create() -> TestResult {
    let store = Arc::new(Store::default());
    let value = ImageService::new(store.clone())
        .create(CreateImageRequest {
            prompt: "Prompt".to_owned(),
            provider: "provider".to_owned(),
            model_name: "model".to_owned(),
            request_id: "request".to_owned(),
            meta_data: None,
        })
        .await?;
    assert_eq!(value.status, IMAGE_STATUS_PENDING);
    assert_eq!(lock(&store.values)?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn test_service_mark_completed() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    let file_id = Uuid::new_v4();
    lock(&store.values)?.push(image(id, "request"));
    ImageService::new(store.clone())
        .mark_completed(
            id,
            "https://example.com/image.png".to_owned(),
            Some(file_id),
        )
        .await?;
    let value = lock(&store.values)?[0].clone();
    assert_eq!(value.status, IMAGE_STATUS_COMPLETED);
    assert_eq!(value.file_index_id, Some(file_id));
    assert!(value.completed_at.is_some());
    Ok(())
}

#[tokio::test]
async fn test_service_mark_failed() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.values)?.push(image(id, "request"));
    ImageService::new(store.clone())
        .mark_failed(id, "provider failed".to_owned())
        .await?;
    let value = lock(&store.values)?[0].clone();
    assert_eq!(value.status, IMAGE_STATUS_FAILED);
    assert_eq!(value.error_message, "provider failed");
    assert!(value.completed_at.is_some());
    Ok(())
}

#[tokio::test]
async fn test_service_get_status() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.values)?.push(image(id, "request"));
    assert_eq!(
        ImageService::new(store.clone()).get_status(id).await?,
        IMAGE_STATUS_PENDING
    );
    assert!(matches!(
        ImageService::new(store).get_status(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}
