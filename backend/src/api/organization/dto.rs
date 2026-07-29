use std::collections::BTreeMap;

use serde::{Deserialize, Deserializer, Serialize};
use utoipa::ToSchema;
use uuid::Uuid;

use crate::core::organization::{
    OrganizationCreateRequest as CoreCreateRequest,
    OrganizationResponse as CoreOrganizationResponse,
    OrganizationUpdateRequest as CoreUpdateRequest,
};

use super::error::OrganizationApiError;

#[derive(Debug, Deserialize, ToSchema)]
pub struct OrganizationCreateRequest {
    #[serde(default, deserialize_with = "null_default")]
    pub name: String,
    #[serde(default, deserialize_with = "null_default")]
    pub slug: String,
    pub bio: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, String>>,
    pub meta_description: Option<String>,
}

impl OrganizationCreateRequest {
    pub fn validate(self) -> Result<CoreCreateRequest, OrganizationApiError> {
        let mut issues = Vec::new();
        if let Some(message) = required_length_issue("Name", &self.name, 2, 255) {
            issues.push(("Name", message));
        }
        if !self.slug.is_empty()
            && let Some(message) = length_issue("Slug", &self.slug, 2, 100)
        {
            issues.push(("Slug", message));
        }
        if !issues.is_empty() {
            return Err(OrganizationApiError::validations(issues));
        }
        Ok(CoreCreateRequest {
            name: self.name,
            slug: self.slug,
            bio: self.bio,
            logo_url: self.logo_url,
            website_url: self.website_url,
            email_public: self.email_public,
            social_links: self.social_links,
            meta_description: self.meta_description,
        })
    }
}

#[derive(Debug, Default, Deserialize, ToSchema)]
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

impl From<OrganizationUpdateRequest> for CoreUpdateRequest {
    fn from(value: OrganizationUpdateRequest) -> Self {
        Self {
            name: value.name,
            slug: value.slug,
            bio: value.bio,
            logo_url: value.logo_url,
            website_url: value.website_url,
            email_public: value.email_public,
            social_links: value.social_links,
            meta_description: value.meta_description,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
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

impl From<CoreOrganizationResponse> for OrganizationResponse {
    fn from(value: CoreOrganizationResponse) -> Self {
        Self {
            id: value.id,
            name: value.name,
            slug: value.slug,
            bio: value.bio,
            logo_url: value.logo_url,
            website_url: value.website_url,
            email_public: value.email_public,
            social_links: value.social_links,
            meta_description: value.meta_description,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessFlag {
    pub success: bool,
}

fn required_length_issue(
    field: &'static str,
    value: &str,
    minimum: usize,
    maximum: usize,
) -> Option<String> {
    if value.is_empty() {
        return Some(format!("{field} is required"));
    }
    length_issue(field, value, minimum, maximum)
}

fn length_issue(
    field: &'static str,
    value: &str,
    minimum: usize,
    maximum: usize,
) -> Option<String> {
    let length = value.chars().count();
    if length < minimum {
        return Some(format!("{field} must be at least {minimum} characters"));
    }
    if length > maximum {
        return Some(format!("{field} must be at most {maximum} characters"));
    }
    None
}

fn null_default<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    Option::<T>::deserialize(deserializer).map(Option::unwrap_or_default)
}
