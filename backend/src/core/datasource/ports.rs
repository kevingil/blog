use async_trait::async_trait;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::error::AppError;

use super::{CrawledContent, DataSource};

#[async_trait]
pub trait DataSourceRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<DataSource, AppError>;
    async fn find_by_organization_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError>;
    async fn find_by_user_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError>;
    async fn find_by_url(&self, url: &str) -> Result<Option<DataSource>, AppError>;
    async fn find_due_to_crawl(&self, limit: i64) -> Result<Vec<DataSource>, AppError>;
    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<DataSource>, i64), AppError>;
    async fn save(&self, source: &mut DataSource) -> Result<(), AppError>;
    async fn update(&self, source: &DataSource) -> Result<(), AppError>;
    async fn update_crawl_status(
        &self,
        id: Uuid,
        status: &str,
        error_message: Option<&str>,
    ) -> Result<(), AppError>;
    async fn update_next_crawl_at(
        &self,
        id: Uuid,
        next_crawl_at: DateTime<Utc>,
    ) -> Result<(), AppError>;
    async fn increment_content_count(&self, id: Uuid, delta: i32) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}

#[async_trait]
pub trait CrawledContentRepository: Send + Sync {
    async fn find_by_data_source_id(
        &self,
        id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<CrawledContent>, i64), AppError>;
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
    async fn find_by_id(&self, id: Uuid) -> Result<CrawledContent, AppError>;
    async fn find_by_url(
        &self,
        data_source_id: Uuid,
        url: &str,
    ) -> Result<Option<CrawledContent>, AppError>;
    async fn save(&self, content: &mut CrawledContent) -> Result<(), AppError>;
    async fn update(&self, content: &CrawledContent) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
    async fn delete_by_data_source_id(&self, id: Uuid) -> Result<(), AppError>;
    async fn count_by_data_source_id(&self, id: Uuid) -> Result<i64, AppError>;
}

#[derive(Debug, Clone, PartialEq, Default)]
pub struct SearchOptions {
    pub num_results: i32,
    pub include_domains: Vec<String>,
    pub exclude_domains: Vec<String>,
    pub use_autoprompt: bool,
    pub include_text: bool,
    pub include_highlights: bool,
    pub include_summary: bool,
    pub start_date: String,
    pub end_date: String,
}

#[derive(Debug, Clone, PartialEq, Default)]
pub struct SimilarOptions {
    pub num_results: i32,
    pub include_domains: Vec<String>,
    pub exclude_domains: Vec<String>,
    pub exclude_source_domain: bool,
    pub include_text: bool,
    pub include_highlights: bool,
    pub include_summary: bool,
}

#[derive(Debug, Clone, PartialEq, Default)]
pub struct SearchResult {
    pub id: String,
    pub title: String,
    pub url: String,
    pub published_date: String,
    pub author: String,
    pub score: f64,
    pub text: String,
    pub highlights: Vec<String>,
    pub summary: String,
    pub image: String,
    pub favicon: String,
    pub extras: serde_json::Map<String, serde_json::Value>,
}

#[derive(Debug, Clone, PartialEq, Default)]
pub struct SearchResponse {
    pub results: Vec<SearchResult>,
}

#[async_trait]
pub trait RecommendationSearchPort: Send + Sync {
    async fn search(&self, query: &str, options: SearchOptions)
    -> Result<SearchResponse, AppError>;
    async fn find_similar(
        &self,
        url: &str,
        options: SimilarOptions,
    ) -> Result<SearchResponse, AppError>;
    fn is_configured(&self) -> bool;
}
