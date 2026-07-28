use std::{
    collections::{BTreeMap, HashMap},
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::organization::{
        Organization, OrganizationAccountRepository, OrganizationCreateRequest,
        OrganizationRepository, OrganizationService, OrganizationUpdateRequest, generate_slug,
    },
    error::AppError,
};
use uuid::Uuid;

#[derive(Default)]
struct MemoryOrganizations {
    values: Mutex<HashMap<Uuid, Organization>>,
}

#[async_trait]
impl OrganizationRepository for MemoryOrganizations {
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
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        if !values.contains_key(&organization.id) {
            return Err(AppError::NotFound);
        }
        values.insert(organization.id, organization.clone());
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

#[derive(Default)]
struct MemoryOrganizationAccounts {
    values: Mutex<HashMap<Uuid, Option<Uuid>>>,
}

#[async_trait]
impl OrganizationAccountRepository for MemoryOrganizationAccounts {
    async fn set_organization(
        &self,
        account_id: Uuid,
        organization_id: Option<Uuid>,
    ) -> Result<bool, AppError> {
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        let Some(value) = values.get_mut(&account_id) else {
            return Ok(false);
        };
        *value = organization_id;
        Ok(true)
    }
}

fn organization(name: &str, slug: &str) -> Organization {
    Organization {
        id: Uuid::new_v4(),
        name: name.to_owned(),
        slug: slug.to_owned(),
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

fn service(
    organizations: Arc<MemoryOrganizations>,
    accounts: Arc<MemoryOrganizationAccounts>,
) -> OrganizationService {
    OrganizationService::new(organizations, accounts)
}

#[tokio::test]
async fn organization_get_by_id_and_slug_preserve_found_and_not_found_cases() {
    let organizations = Arc::new(MemoryOrganizations::default());
    let accounts = Arc::new(MemoryOrganizationAccounts::default());
    let value = organization("Test Org", "test-org");
    organizations
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = service(organizations, accounts);

    let by_id = service.get_by_id(value.id).await;
    assert!(matches!(by_id, Ok(ref response) if response.name == "Test Org"));
    let by_slug = service.get_by_slug("test-org").await;
    assert!(matches!(by_slug, Ok(ref response) if response.slug == "test-org"));
    assert!(matches!(
        service.get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    assert!(matches!(
        service.get_by_slug("nonexistent").await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn organization_list_converts_every_organization() {
    let organizations = Arc::new(MemoryOrganizations::default());
    let accounts = Arc::new(MemoryOrganizationAccounts::default());
    let first = organization("Org One", "org-one");
    let second = organization("Org Two", "org-two");
    {
        let mut values = organizations
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        values.insert(first.id, first);
        values.insert(second.id, second);
    }

    let result = service(organizations, accounts).list().await;
    assert!(matches!(result, Ok(ref values) if values.len() == 2));
}

#[tokio::test]
async fn organization_create_generates_or_preserves_slug_and_rejects_conflicts() {
    let organizations = Arc::new(MemoryOrganizations::default());
    let accounts = Arc::new(MemoryOrganizationAccounts::default());
    let service = service(organizations.clone(), accounts);
    let bio = "A test organization".to_owned();
    let generated = service
        .create(OrganizationCreateRequest {
            name: "New Organization".to_owned(),
            slug: String::new(),
            bio: Some(bio),
            logo_url: None,
            website_url: None,
            email_public: None,
            social_links: None,
            meta_description: None,
        })
        .await;
    assert!(
        matches!(generated, Ok(ref value) if value.slug == "new-organization" && value.bio == "A test organization")
    );

    let custom = service
        .create(OrganizationCreateRequest {
            name: "Another Organization".to_owned(),
            slug: "custom-slug".to_owned(),
            bio: None,
            logo_url: None,
            website_url: None,
            email_public: None,
            social_links: None,
            meta_description: None,
        })
        .await;
    assert!(matches!(custom, Ok(ref value) if value.slug == "custom-slug"));

    let conflict = service
        .create(OrganizationCreateRequest {
            name: "Existing Org".to_owned(),
            slug: "custom-slug".to_owned(),
            bio: None,
            logo_url: None,
            website_url: None,
            email_public: None,
            social_links: None,
            meta_description: None,
        })
        .await;
    assert!(matches!(conflict, Err(AppError::Conflict(_))));
}

#[tokio::test]
async fn organization_update_preserves_partial_slug_and_not_found_cases() {
    let organizations = Arc::new(MemoryOrganizations::default());
    let accounts = Arc::new(MemoryOrganizationAccounts::default());
    let original = organization("Original Name", "original-slug");
    let taken = organization("Other Org", "taken-slug");
    {
        let mut values = organizations
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        values.insert(original.id, original.clone());
        values.insert(taken.id, taken);
    }
    let service = service(organizations, accounts);
    let renamed = service
        .update(
            original.id,
            OrganizationUpdateRequest {
                name: Some("Updated Name".to_owned()),
                ..OrganizationUpdateRequest::default()
            },
        )
        .await;
    assert!(
        matches!(renamed, Ok(ref value) if value.name == "Updated Name" && value.slug == "original-slug")
    );

    let re_slugged = service
        .update(
            original.id,
            OrganizationUpdateRequest {
                slug: Some("new-slug".to_owned()),
                ..OrganizationUpdateRequest::default()
            },
        )
        .await;
    assert!(matches!(re_slugged, Ok(ref value) if value.slug == "new-slug"));

    let conflict = service
        .update(
            original.id,
            OrganizationUpdateRequest {
                slug: Some("taken-slug".to_owned()),
                ..OrganizationUpdateRequest::default()
            },
        )
        .await;
    assert!(matches!(conflict, Err(AppError::Conflict(_))));
    assert!(matches!(
        service
            .update(Uuid::new_v4(), OrganizationUpdateRequest::default())
            .await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn organization_delete_join_and_leave_preserve_not_found_behavior() {
    let organizations = Arc::new(MemoryOrganizations::default());
    let accounts = Arc::new(MemoryOrganizationAccounts::default());
    let value = organization("Test Org", "test-org");
    let account_id = Uuid::new_v4();
    organizations
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    accounts
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(account_id, None);
    let service = service(organizations, accounts.clone());

    assert!(
        service
            .join_organization(account_id, value.id)
            .await
            .is_ok()
    );
    assert_eq!(
        accounts
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner())
            .get(&account_id),
        Some(&Some(value.id))
    );
    assert!(service.leave_organization(account_id).await.is_ok());
    assert!(matches!(
        service.leave_organization(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    assert!(matches!(
        service.join_organization(account_id, Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    assert!(matches!(
        service.join_organization(Uuid::new_v4(), value.id).await,
        Err(AppError::NotFound)
    ));
    assert!(service.delete(value.id).await.is_ok());
    assert!(matches!(
        service.delete(value.id).await,
        Err(AppError::NotFound)
    ));
}

#[test]
fn organization_slug_generation_matches_every_go_helper_case() {
    let cases = [
        ("My Company", "my-company"),
        ("UPPERCASE", "uppercase"),
        ("hello world test", "hello-world-test"),
        ("Company! @#$%^&*()", "company"),
        ("hello    world", "hello-world"),
        ("  hello world  ", "hello-world"),
        ("Company 123", "company-123"),
        ("Café Münich", "caf-mnich"),
        ("", ""),
        ("!@#$%^&*()", ""),
        ("hello - - world", "hello-world"),
        ("-hello", "hello"),
        ("hello-", "hello"),
    ];
    for (input, expected) in cases {
        assert_eq!(generate_slug(input), expected);
    }

    let social_links = BTreeMap::from([("github".to_owned(), "kevin".to_owned())]);
    assert_eq!(
        social_links.get("github").map(String::as_str),
        Some("kevin")
    );
}
