use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::{core::organization::Organization, error::AppError, schema::organization};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = organization)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct OrganizationRow {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub bio: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<Value>,
    pub meta_description: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = organization)]
pub struct NewOrganizationRow<'a> {
    pub id: Uuid,
    pub name: &'a str,
    pub slug: &'a str,
    pub bio: Option<&'a str>,
    pub logo_url: Option<&'a str>,
    pub website_url: Option<&'a str>,
    pub email_public: Option<&'a str>,
    pub social_links: Option<Value>,
    pub meta_description: Option<&'a str>,
}

impl TryFrom<OrganizationRow> for Organization {
    type Error = AppError;

    fn try_from(row: OrganizationRow) -> Result<Self, Self::Error> {
        let social_links = match row.social_links {
            Some(Value::Object(values)) => Some(values.into_iter().collect()),
            Some(_) => None,
            None => None,
        };
        Ok(Self {
            id: row.id,
            name: row.name,
            slug: row.slug,
            bio: row.bio,
            logo_url: row.logo_url,
            website_url: row.website_url,
            email_public: row.email_public,
            social_links,
            meta_description: row.meta_description,
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}
