use async_trait::async_trait;
use diesel::{ExpressionMethods, QueryDsl, SelectableHelper, dsl::count_star};
use diesel_async::RunQueryDsl;
use uuid::Uuid;

use crate::{
    core::insight::{ContentTopicMatch, ContentTopicMatchRepository},
    database::{
        models::content_topic_match::{ContentTopicMatchRow, NewContentTopicMatchRow},
        pool::PgPool,
    },
    error::AppError,
    schema::content_topic_match,
};

#[derive(Clone)]
pub struct DieselContentTopicMatchRepository {
    pool: PgPool,
}

impl DieselContentTopicMatchRepository {
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

    pub async fn find_by_content_id(
        &self,
        content_id: Uuid,
    ) -> Result<Vec<ContentTopicMatch>, AppError> {
        let mut connection = self.connection().await?;
        content_topic_match::table
            .filter(content_topic_match::content_id.eq(content_id))
            .order(content_topic_match::similarity_score.desc())
            .select(ContentTopicMatchRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<ContentTopicMatchRow>| rows.into_iter().map(Into::into).collect())
            .map_err(|_| AppError::Database)
    }

    pub async fn find_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        self.find_by_topic_id_with_primary(topic_id, offset, limit, None)
            .await
    }

    pub async fn save(&self, value: &mut ContentTopicMatch) -> Result<(), AppError> {
        self.save_batch(std::slice::from_mut(value)).await
    }

    pub async fn delete_by_content_id(&self, content_id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(
            content_topic_match::table.filter(content_topic_match::content_id.eq(content_id)),
        )
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    pub async fn delete_by_topic_id(&self, topic_id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(
            content_topic_match::table.filter(content_topic_match::topic_id.eq(topic_id)),
        )
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(content_topic_match::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }

    pub async fn update_primary_status(
        &self,
        content_id: Uuid,
        topic_id: Uuid,
        is_primary: bool,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(
            content_topic_match::table
                .filter(content_topic_match::content_id.eq(content_id))
                .filter(content_topic_match::topic_id.eq(topic_id)),
        )
        .set(content_topic_match::is_primary.eq(is_primary))
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    pub async fn clear_primary_for_content(&self, content_id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(
            content_topic_match::table.filter(content_topic_match::content_id.eq(content_id)),
        )
        .set(content_topic_match::is_primary.eq(false))
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    async fn find_by_topic_id_with_primary(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
        primary: Option<bool>,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        let mut connection = self.connection().await?;
        let mut count_query = content_topic_match::table
            .filter(content_topic_match::topic_id.eq(topic_id))
            .into_boxed();
        if let Some(primary) = primary {
            count_query = count_query.filter(content_topic_match::is_primary.eq(primary));
        }
        let total = count_query
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;

        let mut query = content_topic_match::table
            .filter(content_topic_match::topic_id.eq(topic_id))
            .order(content_topic_match::similarity_score.desc())
            .offset(offset)
            .into_boxed();
        if let Some(primary) = primary {
            query = query.filter(content_topic_match::is_primary.eq(primary));
        }
        if limit >= 0 {
            query = query.limit(limit);
        }
        let rows = query
            .select(ContentTopicMatchRow::as_select())
            .load(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        Ok((rows.into_iter().map(Into::into).collect(), total))
    }
}

#[async_trait]
impl ContentTopicMatchRepository for DieselContentTopicMatchRepository {
    async fn save_batch(&self, matches: &mut [ContentTopicMatch]) -> Result<(), AppError> {
        if matches.is_empty() {
            return Ok(());
        }
        let now = chrono::Utc::now();
        for value in matches.iter_mut() {
            if value.id.is_nil() {
                value.id = Uuid::new_v4();
            }
        }
        let rows: Vec<_> = matches
            .iter()
            .map(|value| NewContentTopicMatchRow {
                id: value.id,
                content_id: value.content_id,
                topic_id: value.topic_id,
                similarity_score: value.similarity_score,
                is_primary: value.is_primary,
                created_at: value.created_at.unwrap_or(now),
            })
            .collect();
        let mut connection = self.connection().await?;
        diesel::insert_into(content_topic_match::table)
            .values(rows)
            .on_conflict((
                content_topic_match::content_id,
                content_topic_match::topic_id,
            ))
            .do_update()
            .set((
                content_topic_match::similarity_score.eq(diesel::upsert::excluded(
                    content_topic_match::similarity_score,
                )),
                content_topic_match::is_primary
                    .eq(diesel::upsert::excluded(content_topic_match::is_primary)),
            ))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn count_by_topic_id(&self, topic_id: Uuid) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        content_topic_match::table
            .filter(content_topic_match::topic_id.eq(topic_id))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)
    }

    async fn find_primary_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        self.find_by_topic_id_with_primary(topic_id, offset, limit, Some(true))
            .await
    }
}

impl From<ContentTopicMatchRow> for ContentTopicMatch {
    fn from(row: ContentTopicMatchRow) -> Self {
        Self {
            id: row.id,
            content_id: row.content_id,
            topic_id: row.topic_id,
            similarity_score: row.similarity_score,
            is_primary: row.is_primary.unwrap_or_default(),
            created_at: row.created_at,
        }
    }
}
