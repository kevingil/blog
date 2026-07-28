use std::collections::BTreeMap;

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use diesel::{
    ExpressionMethods, Identifiable, Insertable, OptionalExtension, QueryDsl, Queryable,
    QueryableByName, Selectable, SelectableHelper, dsl::count_star, sql_types::Bool,
};
use diesel_async::RunQueryDsl;
use uuid::Uuid;

use crate::{
    core::insight::{UserInsightStatus, UserInsightStatusRepository},
    database::pool::PgPool,
    error::AppError,
    schema::user_insight_status,
};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = user_insight_status)]
#[diesel(check_for_backend(diesel::pg::Pg))]
struct UserInsightStatusRow {
    id: Uuid,
    user_id: Uuid,
    insight_id: Uuid,
    is_read: Option<bool>,
    is_pinned: Option<bool>,
    is_used_in_article: Option<bool>,
    read_at: Option<DateTime<Utc>>,
    created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = user_insight_status)]
struct NewUserInsightStatusRow {
    id: Uuid,
    user_id: Uuid,
    insight_id: Uuid,
    is_read: bool,
    is_pinned: bool,
    is_used_in_article: bool,
    read_at: Option<DateTime<Utc>>,
    created_at: DateTime<Utc>,
}

#[derive(QueryableByName)]
struct PinnedResult {
    #[diesel(sql_type = Bool)]
    is_pinned: bool,
}

#[derive(Clone)]
pub struct DieselUserInsightStatusRepository {
    pool: PgPool,
}

impl DieselUserInsightStatusRepository {
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

    pub async fn upsert(&self, status: &mut UserInsightStatus) -> Result<(), AppError> {
        if status.id.is_nil() {
            status.id = Uuid::new_v4();
        }
        let row = NewUserInsightStatusRow {
            id: status.id,
            user_id: status.user_id,
            insight_id: status.insight_id,
            is_read: status.is_read,
            is_pinned: status.is_pinned,
            is_used_in_article: status.is_used_in_article,
            read_at: status.read_at,
            created_at: status.created_at.unwrap_or_else(Utc::now),
        };
        let mut connection = self.connection().await?;
        diesel::insert_into(user_insight_status::table)
            .values(row)
            .on_conflict((
                user_insight_status::user_id,
                user_insight_status::insight_id,
            ))
            .do_update()
            .set((
                user_insight_status::is_read
                    .eq(diesel::upsert::excluded(user_insight_status::is_read)),
                user_insight_status::is_pinned
                    .eq(diesel::upsert::excluded(user_insight_status::is_pinned)),
                user_insight_status::is_used_in_article.eq(diesel::upsert::excluded(
                    user_insight_status::is_used_in_article,
                )),
                user_insight_status::read_at
                    .eq(diesel::upsert::excluded(user_insight_status::read_at)),
            ))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    pub async fn find_by_user_id(&self, user_id: Uuid) -> Result<Vec<UserInsightStatus>, AppError> {
        self.find_by_user_filter(user_id, None, None).await
    }

    pub async fn find_unread_by_user_id(
        &self,
        user_id: Uuid,
    ) -> Result<Vec<UserInsightStatus>, AppError> {
        self.find_by_user_filter(user_id, Some(false), None).await
    }

    pub async fn find_pinned_by_user_id(
        &self,
        user_id: Uuid,
    ) -> Result<Vec<UserInsightStatus>, AppError> {
        self.find_by_user_filter(user_id, None, Some(true)).await
    }

    pub async fn delete(&self, user_id: Uuid, insight_id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(
            user_insight_status::table
                .filter(user_insight_status::user_id.eq(user_id))
                .filter(user_insight_status::insight_id.eq(insight_id)),
        )
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    async fn find_by_user_filter(
        &self,
        user_id: Uuid,
        is_read: Option<bool>,
        is_pinned: Option<bool>,
    ) -> Result<Vec<UserInsightStatus>, AppError> {
        let mut connection = self.connection().await?;
        let mut query = user_insight_status::table
            .filter(user_insight_status::user_id.eq(user_id))
            .into_boxed();
        if let Some(is_read) = is_read {
            query = query.filter(user_insight_status::is_read.eq(is_read));
        }
        if let Some(is_pinned) = is_pinned {
            query = query.filter(user_insight_status::is_pinned.eq(is_pinned));
        }
        query
            .select(UserInsightStatusRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<UserInsightStatusRow>| rows.into_iter().map(Into::into).collect())
            .map_err(|_| AppError::Database)
    }
}

#[async_trait]
impl UserInsightStatusRepository for DieselUserInsightStatusRepository {
    async fn find_by_user_and_insight(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<Option<UserInsightStatus>, AppError> {
        let mut connection = self.connection().await?;
        user_insight_status::table
            .filter(user_insight_status::user_id.eq(user_id))
            .filter(user_insight_status::insight_id.eq(insight_id))
            .select(UserInsightStatusRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map(|row| row.map(Into::into))
            .map_err(|_| AppError::Database)
    }

    async fn mark_as_read(&self, user_id: Uuid, insight_id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query(
            "INSERT INTO user_insight_status \
             (id, user_id, insight_id, is_read, is_pinned, is_used_in_article, read_at, created_at) \
             VALUES ($1, $2, $3, TRUE, FALSE, FALSE, NOW(), NOW()) \
             ON CONFLICT (user_id, insight_id) DO UPDATE \
             SET is_read = TRUE, read_at = EXCLUDED.read_at",
        )
        .bind::<diesel::sql_types::Uuid, _>(Uuid::new_v4())
        .bind::<diesel::sql_types::Uuid, _>(user_id)
        .bind::<diesel::sql_types::Uuid, _>(insight_id)
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    async fn toggle_pinned(&self, user_id: Uuid, insight_id: Uuid) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query(
            "INSERT INTO user_insight_status \
             (id, user_id, insight_id, is_read, is_pinned, is_used_in_article, created_at) \
             VALUES ($1, $2, $3, FALSE, TRUE, FALSE, NOW()) \
             ON CONFLICT (user_id, insight_id) DO UPDATE \
             SET is_pinned = NOT COALESCE(user_insight_status.is_pinned, FALSE) \
             RETURNING COALESCE(is_pinned, FALSE) AS is_pinned",
        )
        .bind::<diesel::sql_types::Uuid, _>(Uuid::new_v4())
        .bind::<diesel::sql_types::Uuid, _>(user_id)
        .bind::<diesel::sql_types::Uuid, _>(insight_id)
        .get_result::<PinnedResult>(&mut connection)
        .await
        .map(|result| result.is_pinned)
        .map_err(|_| AppError::Database)
    }

    async fn mark_as_used_in_article(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query(
            "INSERT INTO user_insight_status \
             (id, user_id, insight_id, is_read, is_pinned, is_used_in_article, created_at) \
             VALUES ($1, $2, $3, FALSE, FALSE, TRUE, NOW()) \
             ON CONFLICT (user_id, insight_id) DO UPDATE \
             SET is_used_in_article = TRUE",
        )
        .bind::<diesel::sql_types::Uuid, _>(Uuid::new_v4())
        .bind::<diesel::sql_types::Uuid, _>(user_id)
        .bind::<diesel::sql_types::Uuid, _>(insight_id)
        .execute(&mut connection)
        .await
        .map(|_| ())
        .map_err(|_| AppError::Database)
    }

    async fn get_status_map_for_insights(
        &self,
        user_id: Uuid,
        insight_ids: &[Uuid],
    ) -> Result<BTreeMap<Uuid, UserInsightStatus>, AppError> {
        if insight_ids.is_empty() {
            return Ok(BTreeMap::new());
        }
        let mut connection = self.connection().await?;
        user_insight_status::table
            .filter(user_insight_status::user_id.eq(user_id))
            .filter(user_insight_status::insight_id.eq_any(insight_ids))
            .select(UserInsightStatusRow::as_select())
            .load::<UserInsightStatusRow>(&mut connection)
            .await
            .map(|rows| {
                rows.into_iter()
                    .map(|row| {
                        let status = UserInsightStatus::from(row);
                        (status.insight_id, status)
                    })
                    .collect()
            })
            .map_err(|_| AppError::Database)
    }

    async fn count_unread_by_user_id(&self, user_id: Uuid) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        user_insight_status::table
            .filter(user_insight_status::user_id.eq(user_id))
            .filter(user_insight_status::is_read.eq(false))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)
    }
}

impl From<UserInsightStatusRow> for UserInsightStatus {
    fn from(row: UserInsightStatusRow) -> Self {
        Self {
            id: row.id,
            user_id: row.user_id,
            insight_id: row.insight_id,
            is_read: row.is_read.unwrap_or_default(),
            is_pinned: row.is_pinned.unwrap_or_default(),
            is_used_in_article: row.is_used_in_article.unwrap_or_default(),
            read_at: row.read_at,
            created_at: row.created_at,
        }
    }
}
