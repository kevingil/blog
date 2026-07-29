use async_trait::async_trait;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper, dsl::sql,
    result::DatabaseErrorKind, sql_types::Bool,
};
use diesel_async::RunQueryDsl;
use serde_json::{Value, json};
use uuid::Uuid;

use crate::{
    core::chat::{ChatMessage, ChatMessageRepository},
    database::{
        models::chat_message::{ChatMessageChangeset, ChatMessageRow, NewChatMessageRow},
        pool::PgPool,
    },
    error::AppError,
    schema::chat_message,
};

#[derive(Clone)]
pub struct DieselChatMessageRepository {
    pool: PgPool,
}

impl DieselChatMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }
}

#[async_trait]
impl ChatMessageRepository for DieselChatMessageRepository {
    async fn create(&self, message: &mut ChatMessage) -> Result<(), AppError> {
        if message.id.is_nil() {
            message.id = Uuid::new_v4();
        }
        let row = NewChatMessageRow {
            id: message.id,
            article_id: message.article_id,
            role: message.role.clone(),
            content: message.content.clone(),
            meta_data: message.meta_data.clone().unwrap_or_else(|| json!({})),
        };
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(chat_message::table)
            .values(row)
            .returning(ChatMessageRow::as_returning())
            .get_result::<ChatMessageRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        *message = inserted.into();
        Ok(())
    }

    async fn find_by_id(&self, id: Uuid) -> Result<ChatMessage, AppError> {
        let mut connection = self.connection().await?;
        chat_message::table
            .find(id)
            .select(ChatMessageRow::as_select())
            .first::<ChatMessageRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Into::into)
            .ok_or(AppError::NotFound)
    }

    async fn list_by_article(
        &self,
        article_id: Uuid,
        limit: i64,
    ) -> Result<Vec<ChatMessage>, AppError> {
        let mut connection = self.connection().await?;
        chat_message::table
            .filter(chat_message::article_id.eq(article_id))
            .order(chat_message::created_at.desc())
            .limit(limit)
            .select(ChatMessageRow::as_select())
            .load::<ChatMessageRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Into::into).collect())
            .map_err(map_diesel_error)
    }

    async fn list_pending_artifacts(&self, article_id: Uuid) -> Result<Vec<ChatMessage>, AppError> {
        let mut connection = self.connection().await?;
        chat_message::table
            .filter(chat_message::article_id.eq(article_id))
            .filter(sql::<Bool>("meta_data->'artifact'->>'status' = 'pending'"))
            .order(chat_message::created_at.desc())
            .select(ChatMessageRow::as_select())
            .load::<ChatMessageRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Into::into).collect())
            .map_err(map_diesel_error)
    }

    async fn update(&self, message: &ChatMessage) -> Result<(), AppError> {
        let changes = ChatMessageChangeset {
            article_id: message.article_id,
            role: message.role.clone(),
            content: message.content.clone(),
            meta_data: Some(message.meta_data.clone()),
            created_at: Some(message.created_at),
        };
        let mut connection = self.connection().await?;
        let rows = diesel::update(chat_message::table.find(message.id))
            .set(changes)
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if rows == 0 {
            return Err(AppError::NotFound);
        }
        Ok(())
    }

    async fn update_metadata(&self, id: Uuid, metadata: Value) -> Result<u64, AppError> {
        let mut connection = self.connection().await?;
        let rows = diesel::update(chat_message::table.find(id))
            .set(chat_message::meta_data.eq(metadata))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        u64::try_from(rows).map_err(|_| AppError::Internal)
    }

    async fn delete_by_article(&self, article_id: Uuid) -> Result<u64, AppError> {
        let mut connection = self.connection().await?;
        let rows =
            diesel::delete(chat_message::table.filter(chat_message::article_id.eq(article_id)))
                .execute(&mut connection)
                .await
                .map_err(map_diesel_error)?;
        u64::try_from(rows).map_err(|_| AppError::Internal)
    }
}

impl From<ChatMessageRow> for ChatMessage {
    fn from(row: ChatMessageRow) -> Self {
        Self {
            id: row.id,
            article_id: row.article_id,
            role: row.role,
            content: row.content,
            meta_data: row.meta_data,
            created_at: row.created_at,
        }
    }
}

fn map_diesel_error(error: diesel::result::Error) -> AppError {
    match error {
        diesel::result::Error::NotFound => AppError::NotFound,
        diesel::result::Error::DatabaseError(
            DatabaseErrorKind::UniqueViolation
            | DatabaseErrorKind::ForeignKeyViolation
            | DatabaseErrorKind::CheckViolation,
            _,
        ) => AppError::Conflict("database constraint violation".to_owned()),
        _ => AppError::Database,
    }
}
