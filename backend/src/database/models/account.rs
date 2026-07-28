use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::auth::{Account, AccountId},
    schema::account,
};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = account)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct AccountRow {
    pub id: Uuid,
    pub name: String,
    pub email: String,
    pub password_hash: String,
    pub role: String,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub bio: Option<String>,
    pub profile_image: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<Value>,
    pub meta_description: Option<String>,
    pub organization_id: Option<Uuid>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = account)]
pub struct NewAccountRow<'a> {
    pub id: Uuid,
    pub name: &'a str,
    pub email: &'a str,
    pub password_hash: &'a str,
    pub role: &'a str,
}

#[derive(Debug, AsChangeset)]
#[diesel(table_name = account)]
pub struct AccountIdentityChangeset<'a> {
    pub name: &'a str,
    pub email: &'a str,
    pub updated_at: DateTime<Utc>,
}

impl<'a> From<&'a Account> for NewAccountRow<'a> {
    fn from(account: &'a Account) -> Self {
        Self {
            id: account.id.0,
            name: &account.name,
            email: &account.email,
            password_hash: &account.password_hash,
            role: &account.role,
        }
    }
}

impl TryFrom<AccountRow> for Account {
    type Error = crate::error::AppError;

    fn try_from(row: AccountRow) -> Result<Self, Self::Error> {
        let social_links = match row.social_links {
            Some(Value::Object(object)) => Some(object.into_iter().collect()),
            Some(_) => return Err(crate::error::AppError::Database),
            None => None,
        };
        Ok(Self {
            id: AccountId(row.id),
            name: row.name,
            email: row.email,
            password_hash: row.password_hash,
            role: row.role,
            created_at: row.created_at,
            updated_at: row.updated_at,
            bio: row.bio,
            profile_image: row.profile_image,
            email_public: row.email_public,
            social_links,
            meta_description: row.meta_description,
            organization_id: row.organization_id,
        })
    }
}

impl<'a> AccountIdentityChangeset<'a> {
    pub fn new(name: &'a str, email: &'a str) -> Self {
        Self {
            name,
            email,
            updated_at: Utc::now(),
        }
    }
}
