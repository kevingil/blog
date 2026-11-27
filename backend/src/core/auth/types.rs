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

/// Stored MCP credential. The plaintext and its hash are not on this value.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct McpAccessToken {
    pub id: Uuid,
    pub account_id: AccountId,
    pub name: String,
    pub token_prefix: String,
    pub created_at: DateTime<Utc>,
    pub last_used_at: Option<DateTime<Utc>>,
    pub revoked_at: Option<DateTime<Utc>>,
    /// Registered OAuth client name. Absent for a token minted at `/auth/mcp-tokens`.
    pub client_name: Option<String>,
    /// Absent when the token does not expire. OAuth access tokens set this.
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewMcpAccessToken {
    pub id: Uuid,
    pub account_id: AccountId,
    pub name: String,
    pub token_hash: String,
    pub token_prefix: String,
    pub created_at: DateTime<Utc>,
    pub oauth_client_id: Option<String>,
    pub client_name: Option<String>,
    pub expires_at: Option<DateTime<Utc>>,
    pub refresh_token_hash: Option<String>,
    pub refresh_expires_at: Option<DateTime<Utc>>,
}

/// Replacement secrets written when an OAuth refresh token is rotated.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OauthTokenRotation {
    pub token_hash: String,
    pub token_prefix: String,
    pub expires_at: DateTime<Utc>,
    pub refresh_token_hash: String,
    pub refresh_expires_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OauthClient {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewOauthClient {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub created_at: DateTime<Utc>,
}

/// Authorization code after it has been consumed. The plaintext code is not stored.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AuthorizationCode {
    pub client_id: String,
    pub account_id: AccountId,
    pub redirect_uri: String,
    pub code_challenge: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewAuthorizationCode {
    pub code_hash: String,
    pub client_id: String,
    pub account_id: AccountId,
    pub redirect_uri: String,
    pub code_challenge: String,
    pub expires_at: DateTime<Utc>,
}

/// `plaintext` is the bearer secret. It is returned to the caller once and is
/// not stored.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IssuedMcpAccessToken {
    pub record: McpAccessToken,
    pub plaintext: String,
}
