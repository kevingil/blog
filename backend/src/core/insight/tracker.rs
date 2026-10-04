use std::sync::Arc;

use chrono::{DateTime, Duration, Utc};
use uuid::Uuid;

use crate::{
    core::datasource::{DataSourceCreateRequest, DataSourceRepository, DataSourceService},
    error::AppError,
};

use super::{InsightService, InsightTopicRepository};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TrackerKind {
    Domain,
    Subject,
}

impl TrackerKind {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Domain => "domain",
            Self::Subject => "subject",
        }
    }

    pub fn parse(value: &str) -> Result<Self, AppError> {
        match value {
            "domain" => Ok(Self::Domain),
            "subject" => Ok(Self::Subject),
            _ => Err(AppError::InvalidInput(
                "tracker kind must be domain or subject".to_owned(),
            )),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Tracker {
    pub id: Uuid,
    pub kind: TrackerKind,
    pub name: String,
    pub target: String,
    pub frequency: String,
    pub enabled: bool,
    pub last_checked_at: Option<DateTime<Utc>>,
    pub next_check_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CreateTracker {
    pub target: String,
    pub frequency: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct UpdateTracker {
    pub frequency: Option<String>,
    pub enabled: Option<bool>,
}

pub struct TrackerCatalog {
    topics: Arc<dyn InsightTopicRepository>,
    sources: Arc<dyn DataSourceRepository>,
    insights: Arc<InsightService>,
    data_sources: Arc<DataSourceService>,
}

impl TrackerCatalog {
    pub fn new(
        topics: Arc<dyn InsightTopicRepository>,
        sources: Arc<dyn DataSourceRepository>,
        insights: Arc<InsightService>,
        data_sources: Arc<DataSourceService>,
    ) -> Self {
        Self {
            topics,
            sources,
            insights,
            data_sources,
        }
    }

    pub async fn list(
        &self,
        organization_id: Option<Uuid>,
        user_id: Uuid,
    ) -> Result<Vec<Tracker>, AppError> {
        let topics = match organization_id {
            Some(organization_id) => self.topics.find_by_organization_id(organization_id).await?,
            None => self.topics.find_all().await?,
        };
        let sources = match organization_id {
            Some(organization_id) => {
                self.sources
                    .find_by_organization_id(organization_id)
                    .await?
            }
            None => self.sources.find_by_user_id(user_id).await?,
        };
        let mut trackers = Vec::with_capacity(topics.len() + sources.len());
        for topic in topics {
            trackers.push(Tracker {
                id: topic.id,
                kind: TrackerKind::Subject,
                name: topic.name.clone(),
                target: topic.name,
                frequency: topic.check_frequency,
                enabled: topic.is_enabled,
                last_checked_at: topic.last_insight_at,
                next_check_at: topic.next_check_at,
            });
        }
        for source in sources {
            if source.is_discovered {
                continue;
            }
            trackers.push(Tracker {
                id: source.id,
                kind: TrackerKind::Domain,
                name: source.name,
                target: source.url,
                frequency: source.crawl_frequency,
                enabled: source.is_enabled,
                last_checked_at: source.last_crawled_at,
                next_check_at: source.next_crawl_at,
            });
        }
        trackers.sort_by(|left, right| left.name.cmp(&right.name));
        Ok(trackers)
    }

    pub async fn create(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
        request: CreateTracker,
    ) -> Result<Tracker, AppError> {
        let frequency = normalize_frequency(&request.frequency)?;
        let target = request.target.trim();
        if target.is_empty() {
            return Err(AppError::InvalidInput("target is required".to_owned()));
        }
        if let Some(url) = domain_url(target) {
            let host = url
                .host_str()
                .ok_or_else(|| AppError::InvalidInput("target URL must include a host".to_owned()))?
                .to_owned();
            let created = self
                .data_sources
                .create(
                    organization_id,
                    user_id,
                    DataSourceCreateRequest {
                        name: host,
                        url: url.to_string(),
                        feed_url: None,
                        source_type: "blog".to_owned(),
                        crawl_frequency: frequency,
                        is_enabled: Some(true),
                    },
                )
                .await?;
            self.sources
                .update_next_crawl_at(created.id, Utc::now())
                .await?;
            return self.get(TrackerKind::Domain, created.id).await;
        }
        let created = self
            .insights
            .create_topic(
                organization_id,
                super::InsightTopicCreateRequest {
                    name: target.to_owned(),
                    description: None,
                    keywords: None,
                    color: None,
                    icon: None,
                },
            )
            .await?;
        if frequency != "daily" {
            let mut topic = self.topics.find_by_id(created.id).await?;
            topic.check_frequency = frequency;
            self.topics.update(&topic).await?;
        }
        self.get(TrackerKind::Subject, created.id).await
    }

    pub async fn update(
        &self,
        kind: TrackerKind,
        id: Uuid,
        request: UpdateTracker,
    ) -> Result<Tracker, AppError> {
        if let Some(frequency) = request.frequency.as_deref() {
            normalize_frequency(frequency)?;
        }
        match kind {
            TrackerKind::Subject => {
                let mut topic = self.topics.find_by_id(id).await?;
                if let Some(frequency) = request.frequency {
                    topic.check_frequency = normalize_frequency(&frequency)?;
                    topic.next_check_at = Some(Utc::now() + check_interval(&topic.check_frequency));
                }
                if let Some(enabled) = request.enabled {
                    topic.is_enabled = enabled;
                }
                self.topics.update(&topic).await?;
            }
            TrackerKind::Domain => {
                let mut source = self.sources.find_by_id(id).await?;
                if let Some(frequency) = request.frequency {
                    source.crawl_frequency = normalize_frequency(&frequency)?;
                    source.next_crawl_at =
                        Some(Utc::now() + check_interval(&source.crawl_frequency));
                }
                if let Some(enabled) = request.enabled {
                    source.is_enabled = enabled;
                }
                self.sources.update(&source).await?;
            }
        }
        self.get(kind, id).await
    }

    pub async fn delete(&self, kind: TrackerKind, id: Uuid) -> Result<(), AppError> {
        match kind {
            TrackerKind::Subject => self.insights.delete_topic(id).await,
            TrackerKind::Domain => self.data_sources.delete(id).await,
        }
    }

    pub async fn mark_due(&self, kind: TrackerKind, id: Uuid) -> Result<Tracker, AppError> {
        match kind {
            TrackerKind::Subject => {
                let mut topic = self.topics.find_by_id(id).await?;
                topic.is_enabled = true;
                topic.next_check_at = Some(Utc::now());
                self.topics.update(&topic).await?;
            }
            TrackerKind::Domain => {
                let mut source = self.sources.find_by_id(id).await?;
                source.is_enabled = true;
                source.next_crawl_at = Some(Utc::now());
                self.sources.update(&source).await?;
            }
        }
        self.get(kind, id).await
    }

    async fn get(&self, kind: TrackerKind, id: Uuid) -> Result<Tracker, AppError> {
        match kind {
            TrackerKind::Subject => {
                let topic = self.topics.find_by_id(id).await?;
                Ok(Tracker {
                    id: topic.id,
                    kind,
                    name: topic.name.clone(),
                    target: topic.name,
                    frequency: topic.check_frequency,
                    enabled: topic.is_enabled,
                    last_checked_at: topic.last_insight_at,
                    next_check_at: topic.next_check_at,
                })
            }
            TrackerKind::Domain => {
                let source = self.sources.find_by_id(id).await?;
                Ok(Tracker {
                    id: source.id,
                    kind,
                    name: source.name,
                    target: source.url,
                    frequency: source.crawl_frequency,
                    enabled: source.is_enabled,
                    last_checked_at: source.last_crawled_at,
                    next_check_at: source.next_crawl_at,
                })
            }
        }
    }
}

fn domain_url(target: &str) -> Option<reqwest::Url> {
    let url = reqwest::Url::parse(target).ok()?;
    if matches!(url.scheme(), "http" | "https") && url.host_str().is_some() {
        Some(url)
    } else {
        None
    }
}

fn normalize_frequency(frequency: &str) -> Result<String, AppError> {
    match frequency {
        "hourly" | "daily" | "weekly" => Ok(frequency.to_owned()),
        _ => Err(AppError::InvalidInput(
            "frequency must be hourly, daily, or weekly".to_owned(),
        )),
    }
}

fn check_interval(frequency: &str) -> Duration {
    match frequency {
        "hourly" => Duration::hours(1),
        "weekly" => Duration::days(7),
        _ => Duration::days(1),
    }
}
