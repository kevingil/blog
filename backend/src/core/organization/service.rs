use std::{collections::BTreeMap, sync::Arc};

use serde_json::Value;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    Organization, OrganizationAccountRepository, OrganizationCreateRequest, OrganizationRepository,
    OrganizationResponse, OrganizationUpdateRequest,
};

#[derive(Clone)]
pub struct OrganizationService {
    organizations: Arc<dyn OrganizationRepository>,
    accounts: Arc<dyn OrganizationAccountRepository>,
}

impl OrganizationService {
    pub fn new(
        organizations: Arc<dyn OrganizationRepository>,
        accounts: Arc<dyn OrganizationAccountRepository>,
    ) -> Self {
        Self {
            organizations,
            accounts,
        }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<OrganizationResponse, AppError> {
        self.organizations.find_by_id(id).await.map(to_response)
    }

    pub async fn get_by_slug(&self, slug: &str) -> Result<OrganizationResponse, AppError> {
        self.organizations.find_by_slug(slug).await.map(to_response)
    }

    pub async fn list(&self) -> Result<Vec<OrganizationResponse>, AppError> {
        self.organizations
            .list()
            .await
            .map(|organizations| organizations.into_iter().map(to_response).collect())
    }

    pub async fn create(
        &self,
        request: OrganizationCreateRequest,
    ) -> Result<OrganizationResponse, AppError> {
        let slug = if request.slug.is_empty() {
            generate_slug(&request.name)
        } else {
            request.slug
        };
        match self.organizations.find_by_slug(&slug).await {
            Ok(_) => {
                return Err(AppError::Conflict("resource already exists".to_owned()));
            }
            Err(AppError::NotFound) => {}
            Err(error) => return Err(error),
        }

        let social_links = request.social_links.map(string_social_links);
        let mut organization = Organization {
            id: Uuid::new_v4(),
            name: request.name,
            slug,
            bio: request.bio,
            logo_url: request.logo_url,
            website_url: request.website_url,
            email_public: request.email_public,
            social_links,
            meta_description: request.meta_description,
            created_at: None,
            updated_at: None,
        };
        self.organizations.save(&mut organization).await?;
        Ok(to_response(organization))
    }

    pub async fn update(
        &self,
        id: Uuid,
        request: OrganizationUpdateRequest,
    ) -> Result<OrganizationResponse, AppError> {
        let mut organization = self.organizations.find_by_id(id).await?;
        if let Some(slug) = request.slug
            && slug != organization.slug
        {
            match self.organizations.find_by_slug(&slug).await {
                Ok(_) => {
                    return Err(AppError::Conflict("resource already exists".to_owned()));
                }
                Err(AppError::NotFound) => {}
                Err(error) => return Err(error),
            }
            organization.slug = slug;
        }
        if let Some(name) = request.name {
            organization.name = name;
        }
        if let Some(bio) = request.bio {
            organization.bio = Some(bio);
        }
        if let Some(logo_url) = request.logo_url {
            organization.logo_url = Some(logo_url);
        }
        if let Some(website_url) = request.website_url {
            organization.website_url = Some(website_url);
        }
        if let Some(email_public) = request.email_public {
            organization.email_public = Some(email_public);
        }
        if let Some(social_links) = request.social_links {
            organization.social_links = Some(string_social_links(social_links));
        }
        if let Some(meta_description) = request.meta_description {
            organization.meta_description = Some(meta_description);
        }
        self.organizations.update(&organization).await?;
        Ok(to_response(organization))
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.organizations.delete(id).await
    }

    pub async fn join_organization(
        &self,
        account_id: Uuid,
        organization_id: Uuid,
    ) -> Result<(), AppError> {
        self.organizations.find_by_id(organization_id).await?;
        if self
            .accounts
            .set_organization(account_id, Some(organization_id))
            .await?
        {
            Ok(())
        } else {
            Err(AppError::NotFound)
        }
    }

    pub async fn leave_organization(&self, account_id: Uuid) -> Result<(), AppError> {
        if self.accounts.set_organization(account_id, None).await? {
            Ok(())
        } else {
            Err(AppError::NotFound)
        }
    }
}

pub fn generate_slug(name: &str) -> String {
    let mut slug = String::with_capacity(name.len());
    for character in name.to_lowercase().chars() {
        let character = if character == ' ' { '-' } else { character };
        if (character.is_ascii_lowercase() || character.is_ascii_digit() || character == '-')
            && (character != '-' || !slug.ends_with('-'))
        {
            slug.push(character);
        }
    }
    slug.trim_matches('-').to_owned()
}

fn string_social_links(values: BTreeMap<String, String>) -> BTreeMap<String, Value> {
    values
        .into_iter()
        .map(|(key, value)| (key, Value::String(value)))
        .collect()
}

fn to_response(organization: Organization) -> OrganizationResponse {
    OrganizationResponse {
        id: organization.id,
        name: organization.name,
        slug: organization.slug,
        bio: organization.bio.unwrap_or_default(),
        logo_url: organization.logo_url.unwrap_or_default(),
        website_url: organization.website_url.unwrap_or_default(),
        email_public: organization.email_public.unwrap_or_default(),
        social_links: organization
            .social_links
            .unwrap_or_default()
            .into_iter()
            .filter_map(|(key, value)| value.as_str().map(|value| (key, value.to_owned())))
            .collect(),
        meta_description: organization.meta_description.unwrap_or_default(),
    }
}
