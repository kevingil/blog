use std::collections::BTreeMap;

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::{core::datasource::CrawledContent, error::AppError};

use super::{ContentTopicMatch, Insight, InsightTopic, UserInsightStatus};

#[async_trait]
pub trait InsightRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Insight, AppError>;
    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<Insight>, i64), AppError>;
    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError>;
    async fn find_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError>;
    async fn find_unread(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<Insight>, AppError>;
    async fn search_similar(&self, embedding: &[f32], limit: i64)
    -> Result<Vec<Insight>, AppError>;
    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Insight>, AppError>;
    async fn save(&self, insight: &mut Insight) -> Result<(), AppError>;
    async fn mark_as_read(&self, id: Uuid) -> Result<(), AppError>;
    async fn toggle_pinned(&self, id: Uuid) -> Result<(), AppError>;
    async fn mark_as_used_in_article(&self, id: Uuid) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
    async fn count_unread(&self, organization_id: Uuid) -> Result<i64, AppError>;
    async fn count_all_unread(&self) -> Result<i64, AppError>;
}

#[async_trait]
pub trait InsightTopicRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<InsightTopic, AppError>;
    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
    ) -> Result<Vec<InsightTopic>, AppError>;
    async fn find_all(&self) -> Result<Vec<InsightTopic>, AppError>;
    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
        threshold: f64,
    ) -> Result<(Vec<InsightTopic>, Vec<f64>), AppError>;
    async fn save(&self, topic: &mut InsightTopic) -> Result<(), AppError>;
    async fn update(&self, topic: &InsightTopic) -> Result<(), AppError>;
    async fn update_content_count(&self, id: Uuid, count: i32) -> Result<(), AppError>;
    async fn update_last_insight_at(
        &self,
        id: Uuid,
        timestamp: DateTime<Utc>,
    ) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}

#[async_trait]
pub trait UserInsightStatusRepository: Send + Sync {
    async fn find_by_user_and_insight(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<Option<UserInsightStatus>, AppError>;
    async fn mark_as_read(&self, user_id: Uuid, insight_id: Uuid) -> Result<(), AppError>;
    async fn toggle_pinned(&self, user_id: Uuid, insight_id: Uuid) -> Result<bool, AppError>;
    async fn mark_as_used_in_article(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<(), AppError>;
    async fn get_status_map_for_insights(
        &self,
        user_id: Uuid,
        insight_ids: &[Uuid],
    ) -> Result<BTreeMap<Uuid, UserInsightStatus>, AppError>;
    async fn count_unread_by_user_id(&self, user_id: Uuid) -> Result<i64, AppError>;
}

#[async_trait]
pub trait ContentTopicMatchRepository: Send + Sync {
    async fn save_batch(&self, matches: &mut [ContentTopicMatch]) -> Result<(), AppError>;
    async fn count_by_topic_id(&self, topic_id: Uuid) -> Result<i64, AppError>;
    async fn find_primary_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError>;
}

#[async_trait]
pub trait InsightContentRepository: Send + Sync {
    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError>;
    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError>;
    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError>;
    async fn find_recent_by_org(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError>;
}

#[async_trait]
pub trait EmbeddingPort: Send + Sync {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError>;
}
