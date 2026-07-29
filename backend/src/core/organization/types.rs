use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq)]
pub struct Organization {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub bio: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, Value>>,
    pub meta_description: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct OrganizationCreateRequest {
    pub name: String,
    #[serde(default)]
    pub slug: String,
    pub bio: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, String>>,
    pub meta_description: Option<String>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct OrganizationUpdateRequest {
    pub name: Option<String>,
    pub slug: Option<String>,
    pub bio: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, String>>,
    pub meta_description: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct OrganizationResponse {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub bio: String,
    pub logo_url: String,
    pub website_url: String,
    pub email_public: String,
    pub social_links: BTreeMap<String, String>,
    pub meta_description: String,
}
