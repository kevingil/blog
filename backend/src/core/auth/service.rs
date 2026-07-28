use std::{sync::Arc, time::Duration};

use async_trait::async_trait;
use bcrypt::{non_truncating_hash, non_truncating_verify};
use chrono::Utc;
use jsonwebtoken::{
    Algorithm, DecodingKey, EncodingKey, Header, TokenData, Validation, decode, encode,
};
use serde::{Deserialize, Serialize};
use tokio::sync::Semaphore;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    Account, AccountId, AccountUpdate, LoginInput, LoginResult, PasswordUpdate, RegistrationInput,
    UserData,
};

const BCRYPT_COST: u32 = 10;
const BCRYPT_MAX_PASSWORD_BYTES: usize = 72;
const BCRYPT_PARALLELISM: usize = 4;
const TOKEN_LIFETIME: Duration = Duration::from_secs(24 * 60 * 60);

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub exp: usize,
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
}

#[derive(Clone)]
pub struct AuthService {
    accounts: Arc<dyn AccountRepository>,
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
            .map_err(|_| AppError::Internal)?;
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
            .map_err(|_| AppError::Internal)
        })
        .await
        .map_err(|_| AppError::Internal)?
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
            .map_err(|_| AppError::Internal)?;
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
        .map_err(|_| AppError::Internal)?
    }

    pub async fn login(&self, input: LoginInput) -> Result<LoginResult, AppError> {
        let account = self
            .accounts
            .find_by_email(&input.email)
            .await
            .map_err(|_| AppError::Unauthorized)?
            .ok_or(AppError::Unauthorized)?;

        if !self
            .verify_password(&input.password, &account.password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }

        let token = self.issue_token(account.id)?;
        Ok(LoginResult {
            token,
            user: UserData {
                id: account.id.0.to_string(),
                name: account.name,
                email: account.email,
                role: account.role,
            },
        })
    }

    pub async fn register(&self, input: RegistrationInput) -> Result<(), AppError> {
        if self.accounts.find_by_email(&input.email).await?.is_some() {
            return Err(AppError::Conflict("resource already exists".to_owned()));
        }

        let account = Account {
            id: AccountId(Uuid::new_v4()),
            name: input.name,
            email: input.email,
            password_hash: self.hash_password(&input.password).await?,
            role: "user".to_owned(),
            created_at: None,
            updated_at: None,
            bio: None,
            profile_image: None,
            email_public: None,
            social_links: None,
            meta_description: None,
            organization_id: None,
        };
        self.accounts.create(&account).await
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

    pub async fn update_password(
        &self,
        id: AccountId,
        update: PasswordUpdate,
    ) -> Result<(), AppError> {
        let account = self.get_account(id).await?;
        if !self
            .verify_password(&update.current_password, &account.password_hash)
            .await?
        {
            return Err(AppError::Unauthorized);
        }

        let new_password_hash = self.hash_password(&update.new_password).await?;
        if self
            .accounts
            .update_password_if_current(id, &account.password_hash, &new_password_hash)
            .await?
        {
            Ok(())
        } else {
            Err(AppError::Unauthorized)
        }
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

    pub fn issue_token(&self, id: AccountId) -> Result<String, AppError> {
        let now = Utc::now().timestamp();
        let lifetime = i64::try_from(TOKEN_LIFETIME.as_secs()).map_err(|_| AppError::Internal)?;
        let expires_at = now.checked_add(lifetime).ok_or(AppError::Internal)?;
        let exp = usize::try_from(expires_at).map_err(|_| AppError::Internal)?;
        let claims = Claims {
            sub: id.0.to_string(),
            exp,
        };

        encode(
            &Header::new(Algorithm::HS256),
            &claims,
            &EncodingKey::from_secret(&self.jwt_secret),
        )
        .map_err(|_| AppError::Internal)
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

    pub fn account_id_from_token(&self, token: &str) -> Result<AccountId, AppError> {
        let token = self.validate_token(token)?;
        Uuid::parse_str(&token.claims.sub)
            .map(AccountId)
            .map_err(|_| AppError::Unauthorized)
    }
}
