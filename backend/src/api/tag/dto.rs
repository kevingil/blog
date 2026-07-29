use chrono::{DateTime, Utc};
use serde::Serialize;
use utoipa::ToSchema;

use crate::core::tag::Tag;

#[derive(Debug, Serialize, ToSchema)]
pub struct TagResponse {
    pub id: i32,
    pub name: String,
    pub created_at: Option<DateTime<Utc>>,
}

impl From<Tag> for TagResponse {
    fn from(value: Tag) -> Self {
        Self {
            id: value.id,
            name: value.name,
            created_at: value.created_at,
        }
    }
}
