use async_trait::async_trait;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper, dsl::count_star,
    sql_types::Uuid as SqlUuid,
};
use diesel_async::RunQueryDsl;
use pgvector::{Vector, VectorExpressionMethods};
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::insight::{Insight, InsightRepository, MetaData},
    database::{
        models::insight::{InsightRow, NewInsightRow},
        pool::PgPool,
    },
    error::AppError,
    schema::insight,
};

#[derive(Clone)]
pub struct DieselInsightRepository {
    pool: PgPool,
}

impl DieselInsightRepository {
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

    async fn list_filtered(
        &self,
        organization_id: Option<Uuid>,
        topic_id: Option<Uuid>,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError> {
        let mut connection = self.connection().await?;
        let mut count_query = insight::table.into_boxed();
        if let Some(id) = organization_id {
            count_query = count_query.filter(insight::organization_id.eq(id));
        }
        if let Some(id) = topic_id {
            count_query = count_query.filter(insight::topic_id.eq(id));
        }
        let total = count_query
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;

        let mut query = insight::table.into_boxed();
        if let Some(id) = organization_id {
            query = query.filter(insight::organization_id.eq(id));
        }
        if let Some(id) = topic_id {
            query = query.filter(insight::topic_id.eq(id));
        }
        query = query.order(insight::generated_at.desc()).offset(offset);
        if limit >= 0 {
            query = query.limit(limit);
        }
        let rows = query
            .select(InsightRow::as_select())
            .load(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        Ok((rows_into_domain(rows), total))
    }
}

#[async_trait]
impl InsightRepository for DieselInsightRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Insight, AppError> {
        let mut connection = self.connection().await?;
        insight::table
            .find(id)
            .select(InsightRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<Insight>, i64), AppError> {
        self.list_filtered(None, None, offset, limit).await
    }

    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError> {
        self.list_filtered(Some(organization_id), None, offset, limit)
            .await
    }

    async fn find_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError> {
        self.list_filtered(None, Some(topic_id), offset, limit)
            .await
    }

    async fn find_unread(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        let mut connection = self.connection().await?;
        let mut query = insight::table
            .filter(insight::organization_id.eq(organization_id))
            .filter(insight::is_read.eq(false))
            .order(insight::generated_at.desc())
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        query
            .select(InsightRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(|_| AppError::Database)
    }

    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        similar(&self.pool, None, embedding, limit).await
    }

    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        similar(&self.pool, Some(organization_id), embedding, limit).await
    }

    async fn save(&self, value: &mut Insight) -> Result<(), AppError> {
        if value.id.is_nil() {
            value.id = Uuid::new_v4();
        }
        let mut connection = self.connection().await?;
        diesel::insert_into(insight::table)
            .values(new_row(value))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn mark_as_read(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(insight::table.find(id))
            .set(insight::is_read.eq(true))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn toggle_pinned(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query("UPDATE insight SET is_pinned = NOT is_pinned WHERE id = $1")
            .bind::<SqlUuid, _>(id)
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn mark_as_used_in_article(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(insight::table.find(id))
            .set(insight::is_used_in_article.eq(true))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(insight::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }

    async fn count_unread(&self, organization_id: Uuid) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        insight::table
            .filter(insight::organization_id.eq(organization_id))
            .filter(insight::is_read.eq(false))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)
    }

    async fn count_all_unread(&self) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        insight::table
            .filter(insight::is_read.eq(false))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)
    }
}

async fn similar(
    pool: &PgPool,
    organization_id: Option<Uuid>,
    embedding: &[f32],
    limit: i64,
) -> Result<Vec<Insight>, AppError> {
    let mut connection = pool.get().await.map_err(|_| AppError::Database)?;
    let mut query = insight::table
        .filter(insight::embedding.is_not_null())
        .order(insight::embedding.cosine_distance(Vector::from(embedding.to_vec())))
        .into_boxed();
    if let Some(id) = organization_id {
        query = query.filter(insight::organization_id.eq(id));
    }
    if limit >= 0 {
        query = query.limit(limit);
    }
    query
        .select(InsightRow::as_select())
        .load(&mut connection)
        .await
        .map(rows_into_domain)
        .map_err(|_| AppError::Database)
}

impl From<InsightRow> for Insight {
    fn from(row: InsightRow) -> Self {
        Self {
            id: row.id,
            organization_id: row.organization_id,
            topic_id: row.topic_id,
            title: row.title,
            summary: row.summary,
            content: row.content,
            key_points: row.key_points.and_then(json_string_array),
            source_content_ids: row
                .source_content_ids
                .unwrap_or_default()
                .into_iter()
                .flatten()
                .collect(),
            embedding: row.embedding.map(|value| value.to_vec()),
            generated_at: row.generated_at,
            period_start: row.period_start,
            period_end: row.period_end,
            is_read: row.is_read.unwrap_or_default(),
            is_pinned: row.is_pinned.unwrap_or_default(),
            is_used_in_article: row.is_used_in_article.unwrap_or_default(),
            meta_data: row.meta_data.and_then(metadata),
        }
    }
}

fn new_row(value: &Insight) -> NewInsightRow {
    NewInsightRow {
        id: value.id,
        organization_id: value.organization_id,
        topic_id: value.topic_id,
        title: value.title.clone(),
        summary: value.summary.clone(),
        content: value.content.clone(),
        key_points: serde_json::to_value(value.key_points.as_deref().unwrap_or_default())
            .unwrap_or(Value::Array(Vec::new())),
        source_content_ids: value.source_content_ids.iter().copied().map(Some).collect(),
        embedding: value.embedding.clone().map(Vector::from),
        generated_at: value.generated_at.unwrap_or_else(chrono::Utc::now),
        period_start: value.period_start,
        period_end: value.period_end,
        is_read: value.is_read,
        is_pinned: value.is_pinned,
        is_used_in_article: value.is_used_in_article,
        meta_data: metadata_value(value.meta_data.as_ref()),
    }
}

fn rows_into_domain(rows: Vec<InsightRow>) -> Vec<Insight> {
    rows.into_iter().map(Into::into).collect()
}

fn json_string_array(value: Value) -> Option<Vec<String>> {
    serde_json::from_value(value).ok()
}

fn metadata(value: Value) -> Option<MetaData> {
    match value {
        Value::Object(map) => Some(map.into_iter().collect()),
        _ => None,
    }
}

fn metadata_value(value: Option<&MetaData>) -> Value {
    Value::Object(
        value
            .map(|map| map.clone().into_iter().collect())
            .unwrap_or_default(),
    )
}
