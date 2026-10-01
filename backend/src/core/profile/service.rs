use std::{collections::BTreeMap, sync::Arc};

use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::{organization::OrganizationRepository, storage::UploadRepository},
    error::AppError,
};

use super::{
    ProfileAccount, ProfileAccountRepository, ProfileRepository, ProfileUpdateRequest,
    PublicProfileResponse, SiteSettings, SiteSettingsRepository, SiteSettingsResponse,
    SiteSettingsUpdateRequest, UserProfileResponse,
};

#[derive(Clone)]
pub struct ProfileService {
    profiles: Arc<dyn ProfileRepository>,
    settings: Arc<dyn SiteSettingsRepository>,
    accounts: Arc<dyn ProfileAccountRepository>,
    organizations: Arc<dyn OrganizationRepository>,
    uploads: Option<Arc<dyn UploadRepository>>,
}

impl ProfileService {
    pub fn new(
        profiles: Arc<dyn ProfileRepository>,
        settings: Arc<dyn SiteSettingsRepository>,
        accounts: Arc<dyn ProfileAccountRepository>,
        organizations: Arc<dyn OrganizationRepository>,
    ) -> Self {
        Self {
            profiles,
            settings,
            accounts,
            organizations,
            uploads: None,
        }
    }

    pub fn with_uploads(mut self, uploads: Arc<dyn UploadRepository>) -> Self {
        self.uploads = Some(uploads);
        self
    }

    pub async fn get_public_profile(&self) -> Result<PublicProfileResponse, AppError> {
        let mut profile = self.profiles.get_public_profile().await?;
        profile.image = self.image_asset(profile.image_upload_file_id).await?;
        Ok(PublicProfileResponse {
            profile_type: profile.profile_type,
            // The Go service never assigns the DTO's ID field, so its encoded
            // value is UUID's all-zero value even when a backing row exists.
            id: Uuid::nil(),
            name: profile.name,
            bio: profile.bio.unwrap_or_default(),
            image_url: profile.image_url.unwrap_or_default(),
            email_public: profile.email_public.unwrap_or_default(),
            social_links: string_values(profile.social_links),
            meta_description: profile.meta_description.unwrap_or_default(),
            website_url: profile.website_url,
            image: profile.image,
        })
    }

    pub async fn get_user_profile(
        &self,
        account_id: Uuid,
    ) -> Result<UserProfileResponse, AppError> {
        let mut account = self.accounts.find_profile_account(account_id).await?;
        account.image = self.image_asset(account.profile_upload_file_id).await?;
        Ok(account_response(account))
    }

    pub async fn update_user_profile(
        &self,
        account_id: Uuid,
        request: ProfileUpdateRequest,
    ) -> Result<UserProfileResponse, AppError> {
        let mut account = self.accounts.find_profile_account(account_id).await?;
        if let Some(name) = request.name {
            account.name = name;
        }
        if let Some(bio) = request.bio {
            account.bio = Some(bio);
        }
        if let Some(profile_upload_file_id) = request.profile_upload_file_id {
            let image = self.image_asset(Some(profile_upload_file_id)).await?;
            let Some(image) = image else {
                return Err(AppError::InvalidInput("image was not found".to_owned()));
            };
            account.profile_upload_file_id = Some(profile_upload_file_id);
            account.profile_image = Some(image.url.clone());
            account.image = Some(image);
        } else if let Some(profile_image) = request.profile_image {
            account.profile_image = Some(profile_image);
            account.profile_upload_file_id = None;
            account.image = None;
        }
        if let Some(email_public) = request.email_public {
            account.email_public = Some(email_public);
        }
        if let Some(meta_description) = request.meta_description {
            account.meta_description = Some(meta_description);
        }
        if let Some(social_links) = request.social_links {
            account.social_links = Some(
                social_links
                    .into_iter()
                    .map(|(key, value)| (key, Value::String(value)))
                    .collect(),
            );
        }
        self.accounts.update_profile_account(&account).await?;
        account.image = self.image_asset(account.profile_upload_file_id).await?;
        Ok(account_response(account))
    }

    pub async fn get_site_settings(&self) -> Result<SiteSettingsResponse, AppError> {
        match self.settings.get().await {
            Ok(settings) => Ok(settings_response(settings)),
            Err(AppError::NotFound) => Ok(SiteSettingsResponse {
                public_profile_type: "user".to_owned(),
                public_user_id: None,
                public_organization_id: None,
            }),
            Err(error) => Err(error),
        }
    }

    pub async fn update_site_settings(
        &self,
        request: SiteSettingsUpdateRequest,
    ) -> Result<SiteSettingsResponse, AppError> {
        let mut settings = match self.settings.get().await {
            Ok(settings) => settings,
            Err(AppError::NotFound) => SiteSettings {
                id: 1,
                public_profile_type: "user".to_owned(),
                public_user_id: None,
                public_organization_id: None,
                created_at: None,
                updated_at: None,
            },
            Err(error) => return Err(error),
        };

        if let Some(profile_type) = request.public_profile_type {
            if profile_type != "user" && profile_type != "organization" {
                return Err(AppError::InvalidInput("validation failed".to_owned()));
            }
            settings.public_profile_type = profile_type;
        }
        if let Some(user_id) = request.public_user_id {
            self.accounts.find_profile_account(user_id).await?;
            settings.public_user_id = Some(user_id);
        }
        if let Some(organization_id) = request.public_organization_id {
            self.organizations.find_by_id(organization_id).await?;
            settings.public_organization_id = Some(organization_id);
        }
        self.settings.save(&mut settings).await?;
        Ok(settings_response(settings))
    }

    pub async fn is_user_admin(&self, user_id: Uuid) -> Result<bool, AppError> {
        self.profiles.is_user_admin(user_id).await
    }
}

fn account_response(account: ProfileAccount) -> UserProfileResponse {
    UserProfileResponse {
        id: account.id,
        name: account.name,
        bio: account.bio.unwrap_or_default(),
        profile_image: account.profile_image.unwrap_or_default(),
        email_public: account.email_public.unwrap_or_default(),
        social_links: string_values(account.social_links),
        meta_description: account.meta_description.unwrap_or_default(),
        organization_id: account.organization_id,
        image: account.image,
    }
}

impl ProfileService {
    async fn image_asset(
        &self,
        id: Option<Uuid>,
    ) -> Result<Option<crate::core::storage::ImageAsset>, AppError> {
        let Some(id) = id else {
            return Ok(None);
        };
        let Some(uploads) = &self.uploads else {
            return Ok(None);
        };
        Ok(uploads.find_by_id(id).await?.map(|file| file.asset()))
    }
}

fn settings_response(settings: SiteSettings) -> SiteSettingsResponse {
    SiteSettingsResponse {
        public_profile_type: settings.public_profile_type,
        public_user_id: settings.public_user_id,
        public_organization_id: settings.public_organization_id,
    }
}

fn string_values(values: Option<BTreeMap<String, Value>>) -> BTreeMap<String, String> {
    values
        .unwrap_or_default()
        .into_iter()
        .filter_map(|(key, value)| value.as_str().map(|value| (key, value.to_owned())))
        .collect()
}
