use std::sync::Arc;

use chrono::{Duration, Utc};
use uuid::Uuid;

use crate::error::AppError;

use super::{
    CrawledContentRepository, CrawledContentResponse, DataSource, DataSourceCreateRequest,
    DataSourceRepository, DataSourceResponse, DataSourceUpdateRequest,
};

#[derive(Clone)]
pub struct DataSourceService {
    data_sources: Arc<dyn DataSourceRepository>,
    crawled_content: Arc<dyn CrawledContentRepository>,
}

impl DataSourceService {
    pub fn new(
        data_sources: Arc<dyn DataSourceRepository>,
        crawled_content: Arc<dyn CrawledContentRepository>,
    ) -> Self {
        Self {
            data_sources,
            crawled_content,
        }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<DataSourceResponse, AppError> {
        self.data_sources.find_by_id(id).await.map(Into::into)
    }

    pub async fn list(&self, organization_id: Uuid) -> Result<Vec<DataSourceResponse>, AppError> {
        self.data_sources
            .find_by_organization_id(organization_id)
            .await
            .map(|items| items.into_iter().map(Into::into).collect())
    }

    pub async fn list_by_user_id(
        &self,
        user_id: Uuid,
    ) -> Result<Vec<DataSourceResponse>, AppError> {
        self.data_sources
            .find_by_user_id(user_id)
            .await
            .map(|items| items.into_iter().map(Into::into).collect())
    }

    pub async fn list_all(
        &self,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<DataSourceResponse>, i64), AppError> {
        let (page, limit) = normalize_pagination(page, limit);
        let (items, total) = self.data_sources.list((page - 1) * limit, limit).await?;
        Ok((items.into_iter().map(Into::into).collect(), total))
    }

    pub async fn create(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
        request: DataSourceCreateRequest,
    ) -> Result<DataSourceResponse, AppError> {
        require_owner(organization_id, user_id)?;
        if self.data_sources.find_by_url(&request.url).await?.is_some() {
            return Err(AppError::Conflict("resource already exists".to_owned()));
        }

        let frequency = default_if_empty(request.crawl_frequency, "daily");
        let mut source = DataSource {
            id: Uuid::new_v4(),
            organization_id,
            user_id,
            name: request.name,
            url: request.url,
            feed_url: request.feed_url,
            source_type: default_if_empty(request.source_type, "blog"),
            crawl_frequency: frequency.clone(),
            is_enabled: request.is_enabled.unwrap_or(true),
            is_discovered: false,
            discovered_from_id: None,
            last_crawled_at: None,
            next_crawl_at: Some(next_crawl_time(&frequency)),
            crawl_status: "pending".to_owned(),
            error_message: None,
            content_count: 0,
            subscriber_count: 1,
            meta_data: None,
            created_at: None,
            updated_at: None,
        };
        self.data_sources.save(&mut source).await?;
        Ok(source.into())
    }

    pub async fn update(
        &self,
        id: Uuid,
        request: DataSourceUpdateRequest,
    ) -> Result<DataSourceResponse, AppError> {
        let mut source = self.data_sources.find_by_id(id).await?;
        if let Some(url) = request.url
            && url != source.url
        {
            if self
                .data_sources
                .find_by_url(&url)
                .await?
                .is_some_and(|existing| existing.id != id)
            {
                return Err(AppError::Conflict("resource already exists".to_owned()));
            }
            source.url = url;
        }
        if let Some(name) = request.name {
            source.name = name;
        }
        if let Some(feed_url) = request.feed_url {
            source.feed_url = Some(feed_url);
        }
        if let Some(source_type) = request.source_type {
            source.source_type = source_type;
        }
        if let Some(frequency) = request.crawl_frequency {
            source.next_crawl_at = Some(next_crawl_time(&frequency));
            source.crawl_frequency = frequency;
        }
        if let Some(enabled) = request.is_enabled {
            source.is_enabled = enabled;
        }
        self.data_sources.update(&source).await?;
        Ok(source.into())
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.data_sources.delete(id).await
    }

    pub async fn trigger_crawl(&self, id: Uuid) -> Result<(), AppError> {
        let mut source = self.data_sources.find_by_id(id).await?;
        source.crawl_status = "pending".to_owned();
        source.next_crawl_at = Some(Utc::now());
        self.data_sources.update(&source).await
    }

    pub async fn get_content(
        &self,
        data_source_id: Uuid,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<CrawledContentResponse>, i64), AppError> {
        let (page, limit) = normalize_pagination(page, limit);
        let (items, total) = self
            .crawled_content
            .find_by_data_source_id(data_source_id, (page - 1) * limit, limit)
            .await?;
        Ok((items.into_iter().map(Into::into).collect(), total))
    }

    pub async fn get_due_to_crawl(&self, limit: i64) -> Result<Vec<DataSource>, AppError> {
        self.data_sources.find_due_to_crawl(limit).await
    }

    pub async fn update_crawl_status(
        &self,
        id: Uuid,
        status: &str,
        error_message: Option<&str>,
    ) -> Result<(), AppError> {
        self.data_sources
            .update_crawl_status(id, status, error_message)
            .await
    }

    pub async fn set_next_crawl_time(&self, id: Uuid, frequency: &str) -> Result<(), AppError> {
        self.data_sources
            .update_next_crawl_at(id, next_crawl_time(frequency))
            .await
    }

    pub async fn create_discovered_source(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
        discovered_from_id: Uuid,
        name: String,
        url: String,
    ) -> Result<DataSourceResponse, AppError> {
        require_owner(organization_id, user_id)?;
        if self.data_sources.find_by_url(&url).await?.is_some() {
            return Err(AppError::Conflict("resource already exists".to_owned()));
        }
        let mut source = DataSource {
            id: Uuid::new_v4(),
            organization_id,
            user_id,
            name,
            url,
            feed_url: None,
            source_type: "blog".to_owned(),
            crawl_frequency: "daily".to_owned(),
            is_enabled: false,
            is_discovered: true,
            discovered_from_id: Some(discovered_from_id),
            last_crawled_at: None,
            next_crawl_at: Some(next_crawl_time("daily")),
            crawl_status: "pending".to_owned(),
            error_message: None,
            content_count: 0,
            subscriber_count: 1,
            meta_data: None,
            created_at: None,
            updated_at: None,
        };
        self.data_sources.save(&mut source).await?;
        Ok(source.into())
    }
}

fn normalize_pagination(page: i64, limit: i64) -> (i64, i64) {
    let page = if page < 1 { 1 } else { page };
    let limit = if !(1..=100).contains(&limit) {
        20
    } else {
        limit
    };
    (page, limit)
}

fn require_owner(organization_id: Option<Uuid>, user_id: Option<Uuid>) -> Result<(), AppError> {
    if organization_id.is_none() && user_id.is_none() {
        Err(AppError::InvalidInput(
            "Either organization_id or user_id must be provided".to_owned(),
        ))
    } else {
        Ok(())
    }
}

fn default_if_empty(value: String, fallback: &str) -> String {
    if value.is_empty() {
        fallback.to_owned()
    } else {
        value
    }
}

fn next_crawl_time(frequency: &str) -> chrono::DateTime<Utc> {
    let duration = match frequency {
        "hourly" => Duration::hours(1),
        "weekly" => Duration::days(7),
        _ => Duration::days(1),
    };
    Utc::now() + duration
}
