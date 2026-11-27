use async_trait::async_trait;
use chrono::Utc;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper,
    result::{DatabaseErrorKind, Error as DieselError},
    sql_types::Uuid as SqlUuid,
};
use diesel_async::RunQueryDsl;
use uuid::Uuid;

use crate::{
    core::auth::{Account, AccountId, AccountRepository, McpAccessToken, NewMcpAccessToken},
    database::{
        models::{
            account::{AccountIdentityChangeset, AccountRow, NewAccountRow},
            mcp_access_token::{McpAccessTokenRow, NewMcpAccessTokenRow},
        },
        pool::PgPool,
    },
    error::AppError,
    schema::{account, mcp_access_token},
};

#[derive(Clone)]
pub struct DieselAccountRepository {
    pool: PgPool,
}

impl DieselAccountRepository {
    pub const fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(AppError::database)
    }
}

#[async_trait]
impl AccountRepository for DieselAccountRepository {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        let mut connection = self.connection().await?;
        account::table
            .find(id.0)
            .select(AccountRow::as_select())
            .first::<AccountRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Account::try_from)
            .transpose()
    }

    async fn find_by_email(&self, email: &str) -> Result<Option<Account>, AppError> {
        let mut connection = self.connection().await?;
        account::table
            .filter(account::email.eq(email))
            .select(AccountRow::as_select())
            .first::<AccountRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Account::try_from)
            .transpose()
    }

    async fn create(&self, account: &Account) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::insert_into(account::table)
            .values(NewAccountRow::from(account))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_diesel_error)
    }

    async fn update_identity(
        &self,
        id: AccountId,
        name: &str,
        email: &str,
    ) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        diesel::update(account::table.find(id.0))
            .set(AccountIdentityChangeset::new(name, email))
            .execute(&mut connection)
            .await
            .map(|affected| affected > 0)
            .map_err(map_diesel_error)
    }

    async fn update_password_if_current(
        &self,
        id: AccountId,
        expected_password_hash: &str,
        new_password_hash: &str,
    ) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        diesel::update(
            account::table
                .filter(account::id.eq(id.0))
                .filter(account::password_hash.eq(expected_password_hash)),
        )
        .set((
            account::password_hash.eq(new_password_hash),
            account::updated_at.eq(chrono::Utc::now()),
        ))
        .execute(&mut connection)
        .await
        .map(|affected| affected > 0)
        .map_err(map_diesel_error)
    }

    async fn delete_if_password_hash(
        &self,
        id: AccountId,
        expected_password_hash: &str,
    ) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(
            account::table
                .filter(account::id.eq(id.0))
                .filter(account::password_hash.eq(expected_password_hash)),
        )
        .execute(&mut connection)
        .await
        .map(|affected| affected > 0)
        .map_err(map_diesel_error)
    }

    async fn count(&self) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        account::table
            .count()
            .get_result(&mut connection)
            .await
            .map_err(map_diesel_error)
    }

    async fn assign_public_author(&self, id: AccountId) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query(
            "UPDATE account SET email_public = email, updated_at = NOW() WHERE id = $1 AND email_public IS NULL",
        )
        .bind::<SqlUuid, _>(id.0)
        .execute(&mut connection)
        .await
        .map_err(map_diesel_error)?;
        diesel::sql_query(
            "UPDATE site_settings SET public_profile_type = 'user', public_user_id = $1, updated_at = NOW() WHERE id = 1 AND public_user_id IS NULL",
        )
        .bind::<SqlUuid, _>(id.0)
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(map_diesel_error)
    }

    async fn insert_mcp_token(&self, token: &NewMcpAccessToken) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::insert_into(mcp_access_token::table)
            .values(NewMcpAccessTokenRow::from(token))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_diesel_error)
    }

    async fn list_mcp_tokens(&self, owner: AccountId) -> Result<Vec<McpAccessToken>, AppError> {
        let mut connection = self.connection().await?;
        mcp_access_token::table
            .filter(mcp_access_token::account_id.eq(owner.0))
            .order(mcp_access_token::created_at.desc())
            .select(McpAccessTokenRow::as_select())
            .load::<McpAccessTokenRow>(&mut connection)
            .await
            .map_err(map_diesel_error)
            .map(|rows| rows.into_iter().map(McpAccessToken::from).collect())
    }

    async fn find_active_mcp_token_by_hash(
        &self,
        token_hash: &str,
    ) -> Result<Option<McpAccessToken>, AppError> {
        let mut connection = self.connection().await?;
        mcp_access_token::table
            .filter(mcp_access_token::token_hash.eq(token_hash))
            .filter(mcp_access_token::revoked_at.is_null())
            .select(McpAccessTokenRow::as_select())
            .first::<McpAccessTokenRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)
            .map(|row| row.map(McpAccessToken::from))
    }

    async fn touch_mcp_token_last_used(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(
            mcp_access_token::table
                .filter(mcp_access_token::id.eq(id))
                .filter(mcp_access_token::revoked_at.is_null()),
        )
        .set(mcp_access_token::last_used_at.eq(Utc::now()))
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(map_diesel_error)
    }

    async fn revoke_mcp_token(
        &self,
        owner: AccountId,
        id: Uuid,
    ) -> Result<Option<McpAccessToken>, AppError> {
        let mut connection = self.connection().await?;
        let revoked = diesel::update(
            mcp_access_token::table
                .filter(mcp_access_token::id.eq(id))
                .filter(mcp_access_token::account_id.eq(owner.0))
                .filter(mcp_access_token::revoked_at.is_null()),
        )
        .set(mcp_access_token::revoked_at.eq(Utc::now()))
        .returning(McpAccessTokenRow::as_returning())
        .get_result::<McpAccessTokenRow>(&mut connection)
        .await
        .optional()
        .map_err(map_diesel_error)?;
        if let Some(row) = revoked {
            return Ok(Some(row.into()));
        }

        mcp_access_token::table
            .filter(mcp_access_token::id.eq(id))
            .filter(mcp_access_token::account_id.eq(owner.0))
            .select(McpAccessTokenRow::as_select())
            .first::<McpAccessTokenRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)
            .map(|row| row.map(McpAccessToken::from))
    }
}

fn map_diesel_error(error: DieselError) -> AppError {
    match error {
        DieselError::NotFound => AppError::NotFound,
        DieselError::DatabaseError(DatabaseErrorKind::UniqueViolation, _) => {
            AppError::Conflict("resource already exists".to_owned())
        }
        other => AppError::database(other),
    }
}
