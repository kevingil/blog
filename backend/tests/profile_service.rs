use std::{
    collections::{BTreeMap, HashMap},
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        organization::{Organization, OrganizationRepository},
        profile::{
            ProfileAccount, ProfileAccountRepository, ProfileRepository, ProfileService,
            ProfileUpdateRequest, PublicProfile, SiteSettings, SiteSettingsRepository,
            SiteSettingsUpdateRequest,
        },
    },
    error::AppError,
};
use serde_json::Value;
use uuid::Uuid;

#[derive(Default)]
struct MemoryProfiles {
    accounts: Mutex<HashMap<Uuid, ProfileAccount>>,
    settings: Mutex<Option<SiteSettings>>,
    public_profile: Mutex<Option<PublicProfile>>,
    admin: Mutex<HashMap<Uuid, bool>>,
}

#[async_trait]
impl ProfileRepository for MemoryProfiles {
    async fn get_public_profile(&self) -> Result<PublicProfile, AppError> {
        self.public_profile
            .lock()
            .map_err(|_| AppError::Internal)?
            .clone()
            .ok_or(AppError::NotFound)
    }

    async fn is_user_admin(&self, user_id: Uuid) -> Result<bool, AppError> {
        self.admin
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&user_id)
            .copied()
            .ok_or(AppError::NotFound)
    }
}

#[async_trait]
impl ProfileAccountRepository for MemoryProfiles {
    async fn find_profile_account(&self, id: Uuid) -> Result<ProfileAccount, AppError> {
        self.accounts
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn update_profile_account(&self, account: &ProfileAccount) -> Result<(), AppError> {
        let mut accounts = self.accounts.lock().map_err(|_| AppError::Internal)?;
        if !accounts.contains_key(&account.id) {
            return Err(AppError::NotFound);
        }
        accounts.insert(account.id, account.clone());
        Ok(())
    }
}

#[async_trait]
impl SiteSettingsRepository for MemoryProfiles {
    async fn get(&self) -> Result<SiteSettings, AppError> {
        self.settings
            .lock()
            .map_err(|_| AppError::Internal)?
            .clone()
            .ok_or(AppError::NotFound)
    }

    async fn save(&self, settings: &mut SiteSettings) -> Result<(), AppError> {
        settings.id = 1;
        *self.settings.lock().map_err(|_| AppError::Internal)? = Some(settings.clone());
        Ok(())
    }
}

#[derive(Default)]
struct MemoryProfileOrganizations {
    values: Mutex<HashMap<Uuid, Organization>>,
}

#[async_trait]
impl OrganizationRepository for MemoryProfileOrganizations {
    async fn find_by_id(&self, id: Uuid) -> Result<Organization, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Organization, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .find(|organization| organization.slug == slug)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list(&self) -> Result<Vec<Organization>, AppError> {
        Ok(self
            .values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .cloned()
            .collect())
    }

    async fn save(&self, organization: &mut Organization) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .insert(organization.id, organization.clone());
        Ok(())
    }

    async fn update(&self, organization: &Organization) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .insert(organization.id, organization.clone());
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .remove(&id)
            .map(|_| ())
            .ok_or(AppError::NotFound)
    }
}

fn service(
    profiles: Arc<MemoryProfiles>,
    organizations: Arc<MemoryProfileOrganizations>,
) -> ProfileService {
    ProfileService::new(profiles.clone(), profiles.clone(), profiles, organizations)
}

fn profile_account(id: Uuid) -> ProfileAccount {
    ProfileAccount {
        id,
        name: "Test User".to_owned(),
        bio: Some("User bio".to_owned()),
        profile_image: Some("https://example.com/profile.jpg".to_owned()),
        email_public: Some("user@example.com".to_owned()),
        social_links: Some(BTreeMap::from([(
            "github".to_owned(),
            Value::String("testuser".to_owned()),
        )])),
        meta_description: Some("User meta description".to_owned()),
        organization_id: Some(Uuid::new_v4()),
    }
}

fn organization(id: Uuid) -> Organization {
    Organization {
        id,
        name: "Test Org".to_owned(),
        slug: "test-org".to_owned(),
        bio: None,
        logo_url: None,
        website_url: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        created_at: None,
        updated_at: None,
    }
}

#[tokio::test]
async fn profile_public_response_preserves_values_and_zero_id() {
    let profiles = Arc::new(MemoryProfiles::default());
    let organizations = Arc::new(MemoryProfileOrganizations::default());
    *profiles
        .public_profile
        .lock()
        .unwrap_or_else(|error| error.into_inner()) = Some(PublicProfile {
        profile_type: "user".to_owned(),
        name: "Test User".to_owned(),
        bio: Some("Test bio".to_owned()),
        image_url: Some("https://example.com/image.jpg".to_owned()),
        website_url: Some("https://example.com".to_owned()),
        email_public: Some("test@example.com".to_owned()),
        social_links: Some(BTreeMap::from([(
            "twitter".to_owned(),
            Value::String("testuser".to_owned()),
        )])),
        meta_description: Some("Meta description".to_owned()),
    });
    let result = service(profiles, organizations).get_public_profile().await;
    assert!(
        matches!(result, Ok(ref response) if response.id.is_nil() && response.name == "Test User" && response.social_links.get("twitter").map(String::as_str) == Some("testuser"))
    );
}

#[tokio::test]
async fn profile_user_get_and_update_preserve_partial_fields_and_not_found() {
    let profiles = Arc::new(MemoryProfiles::default());
    let organizations = Arc::new(MemoryProfileOrganizations::default());
    let account_id = Uuid::new_v4();
    profiles
        .accounts
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(account_id, profile_account(account_id));
    let service = service(profiles, organizations);
    let found = service.get_user_profile(account_id).await;
    assert!(
        matches!(found, Ok(ref response) if response.name == "Test User" && response.bio == "User bio" && response.social_links.get("github").map(String::as_str) == Some("testuser"))
    );

    let updated = service
        .update_user_profile(
            account_id,
            ProfileUpdateRequest {
                name: Some("Updated Name".to_owned()),
                bio: Some("Updated bio".to_owned()),
                profile_image: Some("https://example.com/updated.jpg".to_owned()),
                email_public: Some("updated@example.com".to_owned()),
                social_links: Some(BTreeMap::from([
                    ("twitter".to_owned(), "updated".to_owned()),
                    ("github".to_owned(), "newaccount".to_owned()),
                ])),
                meta_description: Some("Updated meta".to_owned()),
            },
        )
        .await;
    assert!(
        matches!(updated, Ok(ref response) if response.name == "Updated Name" && response.social_links.get("github").map(String::as_str) == Some("newaccount"))
    );
    assert!(matches!(
        service.get_user_profile(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn profile_site_settings_preserve_existing_and_default_cases() {
    let profiles = Arc::new(MemoryProfiles::default());
    let organizations = Arc::new(MemoryProfileOrganizations::default());
    let service = service(profiles.clone(), organizations);
    let defaults = service.get_site_settings().await;
    assert!(
        matches!(defaults, Ok(ref response) if response.public_profile_type == "user" && response.public_user_id.is_none())
    );

    let user_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    *profiles
        .settings
        .lock()
        .unwrap_or_else(|error| error.into_inner()) = Some(SiteSettings {
        id: 1,
        public_profile_type: "organization".to_owned(),
        public_user_id: Some(user_id),
        public_organization_id: Some(organization_id),
        created_at: None,
        updated_at: None,
    });
    let existing = service.get_site_settings().await;
    assert!(
        matches!(existing, Ok(ref response) if response.public_profile_type == "organization" && response.public_user_id == Some(user_id) && response.public_organization_id == Some(organization_id))
    );
}

#[tokio::test]
async fn profile_site_settings_update_validates_references_and_creates_defaults() {
    let profiles = Arc::new(MemoryProfiles::default());
    let organizations = Arc::new(MemoryProfileOrganizations::default());
    let user_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    profiles
        .accounts
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(user_id, profile_account(user_id));
    organizations
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(organization_id, organization(organization_id));
    let service = service(profiles.clone(), organizations);

    let created = service
        .update_site_settings(SiteSettingsUpdateRequest {
            public_profile_type: Some("organization".to_owned()),
            public_user_id: Some(user_id),
            public_organization_id: Some(organization_id),
        })
        .await;
    assert!(
        matches!(created, Ok(ref response) if response.public_profile_type == "organization" && response.public_user_id == Some(user_id))
    );

    let invalid_type = service
        .update_site_settings(SiteSettingsUpdateRequest {
            public_profile_type: Some("team".to_owned()),
            ..SiteSettingsUpdateRequest::default()
        })
        .await;
    assert!(matches!(invalid_type, Err(AppError::InvalidInput(_))));
    let missing_user = service
        .update_site_settings(SiteSettingsUpdateRequest {
            public_user_id: Some(Uuid::new_v4()),
            ..SiteSettingsUpdateRequest::default()
        })
        .await;
    assert!(matches!(missing_user, Err(AppError::NotFound)));

    *profiles
        .settings
        .lock()
        .unwrap_or_else(|error| error.into_inner()) = None;
    let defaults = service
        .update_site_settings(SiteSettingsUpdateRequest {
            public_profile_type: Some("user".to_owned()),
            ..SiteSettingsUpdateRequest::default()
        })
        .await;
    assert!(matches!(defaults, Ok(ref response) if response.public_profile_type == "user"));
}

#[tokio::test]
async fn profile_admin_check_preserves_true_and_false_cases() {
    let profiles = Arc::new(MemoryProfiles::default());
    let organizations = Arc::new(MemoryProfileOrganizations::default());
    let admin_id = Uuid::new_v4();
    let user_id = Uuid::new_v4();
    {
        let mut roles = profiles
            .admin
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        roles.insert(admin_id, true);
        roles.insert(user_id, false);
    }
    let service = service(profiles, organizations);
    assert!(service.is_user_admin(admin_id).await.unwrap_or(false));
    assert!(!service.is_user_admin(user_id).await.unwrap_or(true));
}
