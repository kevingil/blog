use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use uuid::Uuid;

use crate::core::profile::{
    ProfileUpdateRequest as CoreProfileUpdateRequest,
    PublicProfileResponse as CorePublicProfileResponse,
    SiteSettingsResponse as CoreSiteSettingsResponse,
    SiteSettingsUpdateRequest as CoreSiteSettingsUpdateRequest,
    UserProfileResponse as CoreUserProfileResponse,
};

#[derive(Debug, Default, Deserialize, ToSchema)]
pub struct ProfileUpdateRequest {
    pub name: Option<String>,
    pub bio: Option<String>,
    pub profile_image: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, String>>,
    pub meta_description: Option<String>,
}

impl From<ProfileUpdateRequest> for CoreProfileUpdateRequest {
    fn from(value: ProfileUpdateRequest) -> Self {
        Self {
            name: value.name,
            bio: value.bio,
            profile_image: value.profile_image,
            email_public: value.email_public,
            social_links: value.social_links,
            meta_description: value.meta_description,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
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

impl From<CorePublicProfileResponse> for PublicProfileResponse {
    fn from(value: CorePublicProfileResponse) -> Self {
        Self {
            profile_type: value.profile_type,
            id: value.id,
            name: value.name,
            bio: value.bio,
            image_url: value.image_url,
            email_public: value.email_public,
            social_links: value.social_links,
            meta_description: value.meta_description,
            website_url: value.website_url,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
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

impl From<CoreUserProfileResponse> for UserProfileResponse {
    fn from(value: CoreUserProfileResponse) -> Self {
        Self {
            id: value.id,
            name: value.name,
            bio: value.bio,
            profile_image: value.profile_image,
            email_public: value.email_public,
            social_links: value.social_links,
            meta_description: value.meta_description,
            organization_id: value.organization_id,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SiteSettingsResponse {
    pub public_profile_type: String,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
}

impl From<CoreSiteSettingsResponse> for SiteSettingsResponse {
    fn from(value: CoreSiteSettingsResponse) -> Self {
        Self {
            public_profile_type: value.public_profile_type,
            public_user_id: value.public_user_id,
            public_organization_id: value.public_organization_id,
        }
    }
}

#[derive(Debug, Default, Deserialize, ToSchema)]
pub struct SiteSettingsUpdateRequest {
    pub public_profile_type: Option<String>,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
}

impl From<SiteSettingsUpdateRequest> for CoreSiteSettingsUpdateRequest {
    fn from(value: SiteSettingsUpdateRequest) -> Self {
        Self {
            public_profile_type: value.public_profile_type,
            public_user_id: value.public_user_id,
            public_organization_id: value.public_organization_id,
        }
    }
}
