use async_trait::async_trait;
use chrono::{DateTime, Utc};
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper, result::Error as DieselError,
};
use diesel_async::RunQueryDsl;
use pgvector::{Vector, VectorExpressionMethods};
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::insight::{InsightTopic, InsightTopicRepository},
    database::{
        models::insight_topic::{InsightTopicChangeset, InsightTopicRow, NewInsightTopicRow},
        pool::PgPool,
    },
    error::AppError,
    schema::insight_topic,
};

#[derive(Clone)]
pub struct DieselInsightTopicRepository {
    pool: PgPool,
}

impl DieselInsightTopicRepository {
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
impl InsightTopicRepository for DieselInsightTopicRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<InsightTopic, AppError> {
        let mut connection = self.connection().await?;
        insight_topic::table
            .find(id)
            .select(InsightTopicRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(map_error)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
    ) -> Result<Vec<InsightTopic>, AppError> {
        let mut connection = self.connection().await?;
        insight_topic::table
            .filter(insight_topic::organization_id.eq(organization_id))
            .order(insight_topic::name.asc())
            .select(InsightTopicRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(map_error)
    }

    async fn find_all(&self) -> Result<Vec<InsightTopic>, AppError> {
        let mut connection = self.connection().await?;
        insight_topic::table
            .order(insight_topic::name.asc())
            .select(InsightTopicRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(map_error)
    }

    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
        threshold: f64,
    ) -> Result<(Vec<InsightTopic>, Vec<f64>), AppError> {
        let mut connection = self.connection().await?;
        let vector = Vector::from(embedding.to_vec());
        let distance = insight_topic::embedding.cosine_distance(vector.clone());
        let mut query = insight_topic::table
            .filter(insight_topic::embedding.is_not_null())
            .filter(distance.lt(1.0 - threshold))
            .order(insight_topic::embedding.cosine_distance(vector))
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        let rows = query
            .select((
                InsightTopicRow::as_select(),
                insight_topic::embedding.cosine_distance(Vector::from(embedding.to_vec())),
            ))
            .load::<(InsightTopicRow, Option<f64>)>(&mut connection)
            .await
            .map_err(map_error)?;
        let mut topics = Vec::with_capacity(rows.len());
        let mut scores = Vec::with_capacity(rows.len());
        for (row, distance) in rows {
            topics.push(row.into());
            scores.push(1.0 - distance.unwrap_or(1.0));
        }
        Ok((topics, scores))
    }

    async fn save(&self, topic: &mut InsightTopic) -> Result<(), AppError> {
        if topic.id.is_nil() {
            topic.id = Uuid::new_v4();
        }
        let mut connection = self.connection().await?;
        diesel::insert_into(insight_topic::table)
            .values(new_row(topic))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn update(&self, topic: &InsightTopic) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::update(insight_topic::table.find(topic.id))
            .set(changeset(topic))
            .execute(&mut connection)
            .await
            .map_err(map_error)?;
        if affected == 0 {
            diesel::insert_into(insight_topic::table)
                .values(new_row(topic))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        }
        Ok(())
    }

    async fn update_content_count(&self, id: Uuid, count: i32) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(insight_topic::table.find(id))
            .set(insight_topic::content_count.eq(count))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn update_last_insight_at(
        &self,
        id: Uuid,
        timestamp: DateTime<Utc>,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(insight_topic::table.find(id))
            .set(insight_topic::last_insight_at.eq(timestamp))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(insight_topic::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_error)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }
}

impl From<InsightTopicRow> for InsightTopic {
    fn from(row: InsightTopicRow) -> Self {
        Self {
            id: row.id,
            organization_id: row.organization_id,
            name: row.name,
            description: row.description,
            keywords: row
                .keywords
                .and_then(|value| serde_json::from_value(value).ok()),
            embedding: row.embedding.map(|value| value.to_vec()),
            is_auto_generated: row.is_auto_generated.unwrap_or_default(),
            content_count: row.content_count.unwrap_or_default(),
            last_insight_at: row.last_insight_at,
            color: row.color,
            icon: row.icon,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

fn new_row(topic: &InsightTopic) -> NewInsightTopicRow {
    let now = Utc::now();
    NewInsightTopicRow {
        id: topic.id,
        organization_id: topic.organization_id,
        name: topic.name.clone(),
        description: topic.description.clone(),
        keywords: serde_json::to_value(topic.keywords.as_deref().unwrap_or_default())
            .unwrap_or(Value::Array(Vec::new())),
        embedding: topic.embedding.clone().map(Vector::from),
        is_auto_generated: topic.is_auto_generated,
        content_count: topic.content_count,
        last_insight_at: topic.last_insight_at,
        color: topic.color.clone(),
        icon: topic.icon.clone(),
        created_at: topic.created_at.unwrap_or(now),
        updated_at: topic.updated_at.unwrap_or(now),
    }
}

fn changeset(topic: &InsightTopic) -> InsightTopicChangeset {
    InsightTopicChangeset {
        organization_id: topic.organization_id,
        name: topic.name.clone(),
        description: topic.description.clone(),
        keywords: topic
            .keywords
            .as_deref()
            .and_then(|value| serde_json::to_value(value).ok()),
        embedding: topic.embedding.clone().map(Vector::from),
        is_auto_generated: Some(topic.is_auto_generated),
        content_count: Some(topic.content_count),
        last_insight_at: topic.last_insight_at,
        color: topic.color.clone(),
        icon: topic.icon.clone(),
        updated_at: Some(Utc::now()),
    }
}

fn rows_into_domain(rows: Vec<InsightTopicRow>) -> Vec<InsightTopic> {
    rows.into_iter().map(Into::into).collect()
}

fn map_error(_: DieselError) -> AppError {
    AppError::Database
}
