use std::sync::Arc;

use chrono::Utc;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    CreateImageRequest, IMAGE_STATUS_COMPLETED, IMAGE_STATUS_FAILED, IMAGE_STATUS_PENDING,
    ImageGeneration, ImageRepository,
};

#[derive(Clone)]
pub struct ImageService {
    repository: Arc<dyn ImageRepository>,
}

impl ImageService {
    pub fn new(repository: Arc<dyn ImageRepository>) -> Self {
        Self { repository }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<ImageGeneration, AppError> {
        self.repository.find_by_id(id).await
    }

    pub async fn get_by_request_id(&self, request_id: &str) -> Result<ImageGeneration, AppError> {
        self.repository.find_by_request_id(request_id).await
    }

    pub async fn create(&self, request: CreateImageRequest) -> Result<ImageGeneration, AppError> {
        let mut image = ImageGeneration {
            id: Uuid::new_v4(),
            prompt: request.prompt,
            provider: request.provider,
            model_name: request.model_name,
            request_id: request.request_id,
            status: IMAGE_STATUS_PENDING.to_owned(),
            output_url: String::new(),
            file_index_id: None,
            error_message: String::new(),
            meta_data: request.meta_data,
            created_at: None,
            completed_at: None,
        };
        self.repository.save(&mut image).await?;
        Ok(image)
    }

    pub async fn mark_completed(
        &self,
        id: Uuid,
        output_url: String,
        file_index_id: Option<Uuid>,
    ) -> Result<(), AppError> {
        let mut image = self.repository.find_by_id(id).await?;
        image.status = IMAGE_STATUS_COMPLETED.to_owned();
        image.output_url = output_url;
        image.file_index_id = file_index_id;
        image.completed_at = Some(Utc::now());
        self.repository.update(&image).await
    }

    pub async fn mark_failed(&self, id: Uuid, error_message: String) -> Result<(), AppError> {
        let mut image = self.repository.find_by_id(id).await?;
        image.status = IMAGE_STATUS_FAILED.to_owned();
        image.error_message = error_message;
        image.completed_at = Some(Utc::now());
        self.repository.update(&image).await
    }

    pub async fn get_status(&self, id: Uuid) -> Result<String, AppError> {
        self.repository
            .find_by_id(id)
            .await
            .map(|image| image.status)
    }
}
