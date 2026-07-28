use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::ToSchema;
use uuid::Uuid;

use crate::{
    core::image::{CreateImageRequest, IMAGE_STATUS_FAILED, ImageGeneration},
    error::AppError,
};

#[derive(Debug, Clone, Deserialize, ToSchema)]
pub struct GenerateImageRequest {
    #[schema(min_length = 1)]
    pub prompt: String,
    pub article_id: Uuid,
    #[serde(default)]
    pub generate_prompt: bool,
}

impl GenerateImageRequest {
    pub fn validate(self) -> Result<Self, AppError> {
        if self.prompt.trim().is_empty() {
            return Err(AppError::InvalidInput("Prompt is required".to_owned()));
        }
        Ok(self)
    }

    pub fn persistence_request(
        &self,
        provider: &str,
        model_name: &str,
        request_id: String,
    ) -> CreateImageRequest {
        let mut meta_data = BTreeMap::new();
        meta_data.insert(
            "article_id".to_owned(),
            Value::String(self.article_id.to_string()),
        );
        meta_data.insert(
            "generate_prompt".to_owned(),
            Value::Bool(self.generate_prompt),
        );
        CreateImageRequest {
            prompt: self.prompt.clone(),
            provider: provider.to_owned(),
            model_name: model_name.to_owned(),
            request_id,
            meta_data: Some(meta_data),
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct GenerateImageResponse {
    pub request_id: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct ImageGenerationResponse {
    pub id: Uuid,
    pub prompt: String,
    pub provider: String,
    pub model: String,
    pub request_id: String,
    pub status: String,
    pub output_url: String,
    pub file_index_id: Option<Uuid>,
    pub error_message: String,
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub created_at: Option<String>,
    pub completed_at: Option<String>,
}

impl From<ImageGeneration> for ImageGenerationResponse {
    fn from(value: ImageGeneration) -> Self {
        Self {
            id: value.id,
            prompt: value.prompt,
            provider: value.provider,
            model: value.model_name,
            request_id: value.request_id,
            status: value.status,
            output_url: value.output_url,
            file_index_id: value.file_index_id,
            error_message: value.error_message,
            meta_data: value.meta_data,
            created_at: value.created_at.map(|timestamp| timestamp.to_rfc3339()),
            completed_at: value.completed_at.map(|timestamp| timestamp.to_rfc3339()),
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ImageGenerationStatus {
    pub accepted: bool,
    pub request_id: String,
    pub output_url: String,
    #[serde(rename = "request_id")]
    #[schema(rename = "request_id")]
    pub request_id_compat: String,
    #[serde(rename = "output_url")]
    #[schema(rename = "output_url")]
    pub output_url_compat: String,
}

impl From<ImageGeneration> for ImageGenerationStatus {
    fn from(value: ImageGeneration) -> Self {
        let request_id = value.request_id;
        let output_url = value.output_url;
        Self {
            accepted: value.status != IMAGE_STATUS_FAILED,
            request_id: request_id.clone(),
            output_url: output_url.clone(),
            request_id_compat: request_id,
            output_url_compat: output_url,
        }
    }
}
