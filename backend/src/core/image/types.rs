use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

pub const IMAGE_STATUS_PENDING: &str = "pending";
pub const IMAGE_STATUS_COMPLETED: &str = "completed";
pub const IMAGE_STATUS_FAILED: &str = "failed";

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ImageGeneration {
    pub id: Uuid,
    pub prompt: String,
    pub provider: String,
    pub model_name: String,
    pub request_id: String,
    pub status: String,
    pub output_url: String,
    pub file_index_id: Option<Uuid>,
    pub error_message: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub created_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct CreateImageRequest {
    pub prompt: String,
    pub provider: String,
    pub model_name: String,
    pub request_id: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
}
