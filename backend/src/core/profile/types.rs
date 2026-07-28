use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SiteSettings {
    pub id: i32,
    pub public_profile_type: String,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct PublicProfile {
    pub profile_type: String,
    pub name: String,
    pub bio: Option<String>,
    pub image_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, Value>>,
    pub meta_description: Option<String>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ProfileAccount {
    pub id: Uuid,
    pub name: String,
    pub bio: Option<String>,
    pub profile_image: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, Value>>,
    pub meta_description: Option<String>,
    pub organization_id: Option<Uuid>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProfileUpdateRequest {
    pub name: Option<String>,
    pub bio: Option<String>,
    pub profile_image: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, String>>,
    pub meta_description: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PublicProfileResponse {
    #[serde(rename = "type")]
    pub profile_type: String,
    pub id: Uuid,
    pub name: String,
    pub bio: String,
    pub image_url: String,
    pub email_public: String,
    pub social_links: BTreeMap<String, String>,
    pub meta_description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub website_url: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SiteSettingsResponse {
    pub public_profile_type: String,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct SiteSettingsUpdateRequest {
    pub public_profile_type: Option<String>,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct UserProfileResponse {
    pub id: Uuid,
    pub name: String,
    pub bio: String,
    pub profile_image: String,
    pub email_public: String,
    pub social_links: BTreeMap<String, String>,
    pub meta_description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<Uuid>,
}
