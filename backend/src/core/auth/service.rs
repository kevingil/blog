use std::{sync::Arc, time::Duration};

use async_trait::async_trait;
use base64::{Engine, engine::general_purpose::URL_SAFE_NO_PAD};
use bcrypt::{non_truncating_hash, non_truncating_verify};
use chrono::Utc;
use jsonwebtoken::{
    Algorithm, DecodingKey, EncodingKey, Header, TokenData, Validation, decode, encode,
};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use tokio::sync::Semaphore;
use uuid::Uuid;

use crate::{
    error::AppError,
    setup::{SUPERUSER_ROLE, registration_role},
};

use super::{
    Account, AccountId, AccountUpdate, AuthorizationCode, IssuedMcpAccessToken, LoginInput,
    LoginResult, McpAccessToken, NewAuthorizationCode, NewMcpAccessToken, NewOauthClient,
    OauthClient, OauthTokenRotation, PasswordUpdate, RegistrationInput, UserData,
};

/// Distinguishes an MCP access token from a JWT, which always contains `.`.
pub(in crate::core::auth) const MCP_ACCESS_TOKEN_PREFIX: &str = "mcp_";
/// Characters of the plaintext shown in lists. Includes `mcp_` plus 8 hex chars.
pub(in crate::core::auth) const MCP_ACCESS_TOKEN_PREFIX_CHARS: usize = 12;
const MCP_ACCESS_TOKEN_RANDOM_HEX: usize = 64;
const MCP_ACCESS_TOKEN_LEN: usize = MCP_ACCESS_TOKEN_PREFIX.len() + MCP_ACCESS_TOKEN_RANDOM_HEX;
/// How long `last_used_at` may sit before authentication writes it again.
const MCP_TOKEN_LAST_USED_REFRESH_SECONDS: i64 = 60;

const BCRYPT_COST: u32 = 10;
const BCRYPT_MAX_PASSWORD_BYTES: usize = 72;
const BCRYPT_PARALLELISM: usize = 4;
const TOKEN_LIFETIME: Duration = Duration::from_secs(30 * 24 * 60 * 60);
/// Verified when no account matches the email, so a miss costs the same bcrypt
/// work as a wrong password and timing does not reveal which emails exist.
const UNKNOWN_ACCOUNT_HASH: &str = "$2b$10$jtQO49smyDNuKhbtFroZ5.x/dqeeUoR4iqDX4MEKn8azE7cs5LWrm";

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub exp: usize,
    pub iat: usize,
    /// Digest of the password hash the token was issued against. Changing the
    /// password changes it, which signs out every earlier session.
    pub sv: String,
}

#[async_trait]
pub trait AccountRepository: Send + Sync {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError>;
    async fn find_by_email(&self, email: &str) -> Result<Option<Account>, AppError>;
    async fn create(&self, account: &Account) -> Result<(), AppError>;
    async fn update_identity(
        &self,
        id: AccountId,
        name: &str,
        email: &str,
    ) -> Result<bool, AppError>;
    async fn update_password_if_current(
        &self,
        id: AccountId,
        expected_password_hash: &str,
        new_password_hash: &str,
    ) -> Result<bool, AppError>;
    async fn delete_if_password_hash(
        &self,
        id: AccountId,
        expected_password_hash: &str,
    ) -> Result<bool, AppError>;

    async fn count(&self) -> Result<i64, AppError> {
        Ok(0)
    }

    /// Point site settings at this account when no public author is configured.
    async fn assign_public_author(&self, _id: AccountId) -> Result<(), AppError> {
        Ok(())
    }

    async fn insert_mcp_token(&self, _token: &NewMcpAccessToken) -> Result<(), AppError> {
        Err(AppError::internal("mcp access tokens are not available"))
    }

    async fn list_mcp_tokens(&self, _owner: AccountId) -> Result<Vec<McpAccessToken>, AppError> {
        Ok(Vec::new())
    }

    async fn find_active_mcp_token_by_hash(
        &self,
        _token_hash: &str,
    ) -> Result<Option<McpAccessToken>, AppError> {
        Ok(None)
    }

    async fn touch_mcp_token_last_used(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }

    /// Revokes the owner's token. Returns the row when it belongs to `owner`,
    /// including when it was already revoked.
    async fn revoke_mcp_token(
        &self,
        _owner: AccountId,
        _id: Uuid,
    ) -> Result<Option<McpAccessToken>, AppError> {
        Ok(None)
    }

    async fn insert_oauth_client(&self, _client: &NewOauthClient) -> Result<(), AppError> {
        Err(AppError::internal("oauth clients are not available"))
    }

    async fn find_oauth_client(&self, _client_id: &str) -> Result<Option<OauthClient>, AppError> {
        Ok(None)
    }

    async fn insert_authorization_code(
        &self,
        _code: &NewAuthorizationCode,
    ) -> Result<(), AppError> {
        Err(AppError::internal(
            "oauth authorization codes are not available",
        ))
    }

    async fn consume_authorization_code(
        &self,
        _code_hash: &str,
    ) -> Result<Option<AuthorizationCode>, AppError> {
        Ok(None)
    }

    async fn rotate_oauth_refresh_token(
        &self,
        _refresh_hash: &str,
        _client_id: &str,
        _rotation: &OauthTokenRotation,
    ) -> Result<Option<McpAccessToken>, AppError> {
        Ok(None)
    }
}

#[derive(Clone)]
pub struct AuthService {
    pub(in crate::core::auth) accounts: Arc<dyn AccountRepository>,
    jwt_secret: Arc<[u8]>,
    bcrypt_slots: Arc<Semaphore>,
}

impl AuthService {
    pub fn new(
        accounts: Arc<dyn AccountRepository>,
        jwt_secret: impl AsRef<[u8]>,
    ) -> Result<Self, AppError> {
        if jwt_secret.as_ref().is_empty() {
            return Err(AppError::InvalidInput(
                "JWT secret must not be empty".to_owned(),
            ));
        }

        Ok(Self {
            accounts,
            jwt_secret: Arc::from(jwt_secret.as_ref()),
            bcrypt_slots: Arc::new(Semaphore::new(BCRYPT_PARALLELISM)),
        })
    }

    pub async fn hash_password(&self, password: &str) -> Result<String, AppError> {
        if password.len() > BCRYPT_MAX_PASSWORD_BYTES {
            return Err(AppError::InvalidInput(format!(
                "Password must be at most {BCRYPT_MAX_PASSWORD_BYTES} bytes"
            )));
        }

        let password = password.to_owned();
        let permit = self
            .bcrypt_slots
            .clone()
            .acquire_owned()
            .await
            .map_err(AppError::internal)?;
        tokio::task::spawn_blocking(move || {
            let _permit = permit;
            // bcrypt's non-truncating API reserves one byte for a C-style terminator
            // and rejects an otherwise Go-compatible 72-byte password. The explicit
            // length check above prevents truncation; this branch preserves Go's
            // accepted 72-byte boundary.
            if password.len() == BCRYPT_MAX_PASSWORD_BYTES {
                bcrypt::hash(password, BCRYPT_COST)
            } else {
                non_truncating_hash(password, BCRYPT_COST)
            }
            .map_err(AppError::internal)
        })
        .await
        .map_err(AppError::internal)?
    }

    async fn verify_password(
        &self,
        plain_text_password: &str,
        hashed_password: &str,
    ) -> Result<bool, AppError> {
        if plain_text_password.len() > BCRYPT_MAX_PASSWORD_BYTES {
            return Ok(false);
        }

        let password = plain_text_password.to_owned();
        let password_hash = hashed_password.to_owned();
        let permit = self
            .bcrypt_slots
            .clone()
            .acquire_owned()
            .await
            .map_err(AppError::internal)?;
        tokio::task::spawn_blocking(move || {
            let _permit = permit;
            let verified = if password.len() == BCRYPT_MAX_PASSWORD_BYTES {
                bcrypt::verify(password, &password_hash)
            } else {
                non_truncating_verify(password, &password_hash)
            };
            match verified {
                Ok(matches) => Ok(matches),
                Err(_) => Ok(false),
            }
        })
        .await
        .map_err(AppError::internal)?
    }

    /// Checks the email and password. A missing account does the same bcrypt work
    /// as a wrong password.
    pub async fn authenticate_credentials(&self, input: LoginInput) -> Result<Account, AppError> {
        let account = self
            .accounts
            .find_by_email(&input.email)
            .await
            .map_err(|_| AppError::Unauthorized)?;
        let Some(account) = account else {
            self.verify_password(&input.password, UNKNOWN_ACCOUNT_HASH)
                .await?;
            return Err(AppError::Unauthorized);
        };

        if !self
            .verify_password(&input.password, &account.password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }
        Ok(account)
    }

    pub async fn login(&self, input: LoginInput) -> Result<LoginResult, AppError> {
        let account = self.authenticate_credentials(input).await?;
        self.issue_session(account)
    }

    /// Anyone may register while the database has no accounts; that first
    /// account becomes the admin. After that only an admin can add accounts.
    pub async fn register(
        &self,
        input: RegistrationInput,
        requested_by: Option<&Account>,
    ) -> Result<(), AppError> {
        let existing_accounts = self.accounts.count().await?;
        let requested_by_admin = requested_by.is_some_and(|account| account.role == SUPERUSER_ROLE);
        if existing_accounts > 0 && !requested_by_admin {
            return Err(AppError::Forbidden);
        }

        if self.accounts.find_by_email(&input.email).await?.is_some() {
            return Err(AppError::Conflict("resource already exists".to_owned()));
        }

        let account = Account {
            id: AccountId(Uuid::new_v4()),
            name: input.name,
            email: input.email,
            password_hash: self.hash_password(&input.password).await?,
            role: registration_role(existing_accounts).to_owned(),
            created_at: None,
            updated_at: None,
            bio: None,
            profile_image: None,
            email_public: None,
            social_links: None,
            meta_description: None,
            organization_id: None,
        };
        self.accounts.create(&account).await?;
        if existing_accounts == 0 {
            self.accounts.assign_public_author(account.id).await?;
        }
        Ok(())
    }

    pub async fn get_account(&self, id: AccountId) -> Result<Account, AppError> {
        self.accounts
            .find_by_id(id)
            .await?
            .ok_or(AppError::NotFound)
    }

    pub async fn update_account(
        &self,
        id: AccountId,
        update: AccountUpdate,
    ) -> Result<(), AppError> {
        let account = self.get_account(id).await?;

        if update.email != account.email
            && let Some(existing) = self.accounts.find_by_email(&update.email).await?
            && existing.id != id
        {
            return Err(AppError::Conflict("resource already exists".to_owned()));
        }

        if self
            .accounts
            .update_identity(id, &update.name, &update.email)
            .await?
        {
            Ok(())
        } else {
            Err(AppError::NotFound)
        }
    }

    /// Changes the password and returns a token for the new password. Tokens
    /// issued before the change stop working.
    pub async fn update_password(
        &self,
        id: AccountId,
        update: PasswordUpdate,
    ) -> Result<String, AppError> {
        let account = self.get_account(id).await?;
        if !self
            .verify_password(&update.current_password, &account.password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }

        let new_password_hash = self.hash_password(&update.new_password).await?;
        if !self
            .accounts
            .update_password_if_current(id, &account.password_hash, &new_password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }
        self.issue_token(&Account {
            password_hash: new_password_hash,
            ..account
        })
    }

    pub async fn delete_account(&self, id: AccountId, password: &str) -> Result<(), AppError> {
        let account = self.get_account(id).await?;
        if !self
            .verify_password(password, &account.password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }

        if self
            .accounts
            .delete_if_password_hash(id, &account.password_hash)
            .await?
        {
            Ok(())
        } else {
            Err(AppError::Unauthorized)
        }
    }

    pub fn issue_session(&self, account: Account) -> Result<LoginResult, AppError> {
        Ok(LoginResult {
            token: self.issue_token(&account)?,
            user: UserData {
                id: account.id.0.to_string(),
                name: account.name,
                email: account.email,
                role: account.role,
            },
        })
    }

    pub fn issue_token(&self, account: &Account) -> Result<String, AppError> {
        let now = Utc::now().timestamp();
        let lifetime = i64::try_from(TOKEN_LIFETIME.as_secs()).map_err(AppError::internal)?;
        let expires_at = now
            .checked_add(lifetime)
            .ok_or(AppError::internal("token expiry overflowed"))?;
        let claims = Claims {
            sub: account.id.0.to_string(),
            exp: usize::try_from(expires_at).map_err(AppError::internal)?,
            iat: usize::try_from(now).map_err(AppError::internal)?,
            sv: session_version(&account.password_hash),
        };

        encode(
            &Header::new(Algorithm::HS256),
            &claims,
            &EncodingKey::from_secret(&self.jwt_secret),
        )
        .map_err(AppError::internal)
    }

    pub fn validate_token(&self, token: &str) -> Result<TokenData<Claims>, AppError> {
        let mut validation = Validation::new(Algorithm::HS256);
        validation.leeway = 0;
        validation.set_required_spec_claims(&["exp"]);
        decode::<Claims>(
            token,
            &DecodingKey::from_secret(&self.jwt_secret),
            &validation,
        )
        .map_err(|_| AppError::Unauthorized)
    }

    /// Resolves a JWT to its account. Rejects tokens that are invalid or
    /// expired, whose account was deleted, or that predate a password change.
    /// MCP access tokens are not accepted here.
    pub async fn authenticate(&self, token: &str) -> Result<Account, AppError> {
        let claims = self.validate_token(token)?.claims;
        let id = Uuid::parse_str(&claims.sub)
            .map(AccountId)
            .map_err(|_| AppError::Unauthorized)?;
        let account = self
            .accounts
            .find_by_id(id)
            .await?
            .ok_or(AppError::Unauthorized)?;
        if claims.sv != session_version(&account.password_hash) {
            return Err(AppError::Unauthorized);
        }
        Ok(account)
    }

    /// Resolves either a JWT session or a live MCP access token. Password
    /// changes invalidate the JWT and leave the MCP token in place.
    pub async fn authenticate_bearer(&self, token: &str) -> Result<Account, AppError> {
        if token.starts_with(MCP_ACCESS_TOKEN_PREFIX) {
            return self.authenticate_mcp_token(token).await;
        }
        self.authenticate(token).await
    }

    pub async fn create_mcp_token(
        &self,
        owner: AccountId,
        name: &str,
    ) -> Result<IssuedMcpAccessToken, AppError> {
        let name = name.trim();
        if name.is_empty() || name.chars().count() > 100 {
            return Err(AppError::InvalidInput(
                "token name must be 1 to 100 characters".to_owned(),
            ));
        }
        let plaintext = generate_mcp_access_token();
        let token_prefix = plaintext[..MCP_ACCESS_TOKEN_PREFIX_CHARS].to_owned();
        let created_at = Utc::now();
        let record = McpAccessToken {
            id: Uuid::new_v4(),
            account_id: owner,
            name: name.to_owned(),
            token_prefix: token_prefix.clone(),
            created_at,
            last_used_at: None,
            revoked_at: None,
            client_name: None,
            expires_at: None,
        };
        self.accounts
            .insert_mcp_token(&NewMcpAccessToken {
                id: record.id,
                account_id: owner,
                name: name.to_owned(),
                token_hash: mcp_token_hash(&plaintext),
                token_prefix,
                created_at,
                oauth_client_id: None,
                client_name: None,
                expires_at: None,
                refresh_token_hash: None,
                refresh_expires_at: None,
            })
            .await?;
        Ok(IssuedMcpAccessToken { record, plaintext })
    }

    pub async fn list_mcp_tokens(&self, owner: AccountId) -> Result<Vec<McpAccessToken>, AppError> {
        self.accounts.list_mcp_tokens(owner).await
    }

    pub async fn revoke_mcp_token(
        &self,
        owner: AccountId,
        id: Uuid,
    ) -> Result<McpAccessToken, AppError> {
        self.accounts
            .revoke_mcp_token(owner, id)
            .await?
            .ok_or(AppError::NotFound)
    }

    async fn authenticate_mcp_token(&self, token: &str) -> Result<Account, AppError> {
        if token.len() != MCP_ACCESS_TOKEN_LEN {
            return Err(AppError::Unauthorized);
        }
        let Some(record) = self
            .accounts
            .find_active_mcp_token_by_hash(&mcp_token_hash(token))
            .await?
        else {
            return Err(AppError::Unauthorized);
        };
        self.note_mcp_token_use(&record).await;
        self.accounts
            .find_by_id(record.account_id)
            .await?
            .ok_or(AppError::Unauthorized)
    }

    /// Writes `last_used_at` on first use and again once it is at least a
    /// minute old. A failed write does not fail the request.
    async fn note_mcp_token_use(&self, record: &McpAccessToken) {
        let stale = match record.last_used_at {
            None => true,
            Some(seen) => {
                Utc::now().signed_duration_since(seen)
                    >= chrono::Duration::seconds(MCP_TOKEN_LAST_USED_REFRESH_SECONDS)
            }
        };
        if !stale {
            return;
        }
        if let Err(error) = self.accounts.touch_mcp_token_last_used(record.id).await {
            tracing::warn!(
                error = %error,
                token_id = %record.id,
                "mcp access token last_used_at update failed"
            );
        }
    }
}

fn session_version(password_hash: &str) -> String {
    let digest = Sha256::digest(password_hash.as_bytes());
    URL_SAFE_NO_PAD.encode(&digest[..16])
}

/// Hex SHA-256 of the bearer plaintext. This is the only form stored.
pub fn mcp_token_hash(plaintext: &str) -> String {
    hex::encode(Sha256::digest(plaintext.as_bytes()))
}

/// `mcp_` plus 64 hex characters from two v4 UUIDs (256 bits, OS CSPRNG).
fn generate_mcp_access_token() -> String {
    random_bearer(MCP_ACCESS_TOKEN_PREFIX)
}

/// `prefix` plus 64 hex characters from two v4 UUIDs.
pub(in crate::core::auth) fn random_bearer(prefix: &str) -> String {
    let mut token = String::with_capacity(prefix.len() + MCP_ACCESS_TOKEN_RANDOM_HEX);
    token.push_str(prefix);
    token.push_str(&Uuid::new_v4().simple().to_string());
    token.push_str(&Uuid::new_v4().simple().to_string());
    token
}
