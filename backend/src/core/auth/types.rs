use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct AccountId(pub Uuid);

impl AccountId {
    pub const fn new(id: Uuid) -> Self {
        Self(id)
    }

    pub const fn into_inner(self) -> Uuid {
        self.0
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct Account {
    pub id: AccountId,
    pub name: String,
    pub email: String,
    pub password_hash: String,
    pub role: String,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub bio: Option<String>,
    pub profile_image: Option<String>,
    pub email_public: Option<String>,
    pub social_links: Option<BTreeMap<String, Value>>,
    pub meta_description: Option<String>,
    pub organization_id: Option<Uuid>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LoginInput {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RegistrationInput {
    pub name: String,
    pub email: String,
    pub password: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountUpdate {
    pub name: String,
    pub email: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PasswordUpdate {
    pub current_password: String,
    pub new_password: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LoginResult {
    pub token: String,
    pub user: UserData,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct UserData {
    pub id: String,
    pub name: String,
    pub email: String,
    pub role: String,
}
