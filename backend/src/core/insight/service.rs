use std::sync::Arc;

use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::{core::datasource::CrawledContentResponse, error::AppError};

use super::{
    ContentTopicMatch, ContentTopicMatchRepository, EmbeddingPort, Insight,
    InsightContentRepository, InsightRepository, InsightResponse, InsightSearchRequest,
    InsightTopic, InsightTopicCreateRequest, InsightTopicRepository, InsightTopicResponse,
    InsightTopicUpdateRequest, InsightWithSources, InsightWithUserStatus, UserInsightStatus,
    UserInsightStatusRepository,
};

#[derive(Clone)]
pub struct InsightService {
    insights: Arc<dyn InsightRepository>,
    topics: Arc<dyn InsightTopicRepository>,
    user_statuses: Arc<dyn UserInsightStatusRepository>,
    contents: Arc<dyn InsightContentRepository>,
    matches: Arc<dyn ContentTopicMatchRepository>,
    embeddings: Arc<dyn EmbeddingPort>,
}

impl InsightService {
    pub fn new(
        insights: Arc<dyn InsightRepository>,
        topics: Arc<dyn InsightTopicRepository>,
        user_statuses: Arc<dyn UserInsightStatusRepository>,
        contents: Arc<dyn InsightContentRepository>,
        matches: Arc<dyn ContentTopicMatchRepository>,
        embeddings: Arc<dyn EmbeddingPort>,
    ) -> Self {
        Self {
            insights,
            topics,
            user_statuses,
            contents,
            matches,
            embeddings,
        }
    }

    pub async fn get_insight_by_id(&self, id: Uuid) -> Result<InsightResponse, AppError> {
        self.insights.find_by_id(id).await.map(Into::into)
    }

    pub async fn get_insight_with_sources(&self, id: Uuid) -> Result<InsightWithSources, AppError> {
        let insight = self.insights.find_by_id(id).await?;
        let source_contents = if !insight.source_content_ids.is_empty() {
            self.contents
                .find_by_ids(&insight.source_content_ids)
                .await?
                .into_iter()
                .map(CrawledContentResponse::from)
                .collect()
        } else {
            Vec::new()
        };
        let topic = match insight.topic_id {
            Some(topic_id) => self
                .topics
                .find_by_id(topic_id)
                .await
                .ok()
                .map(InsightTopicResponse::from),
            None => None,
        };
        Ok(InsightWithSources {
            insight: insight.into(),
            source_contents,
            topic,
        })
    }

    pub async fn list_insights(
        &self,
        organization_id: Uuid,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<InsightResponse>, i64), AppError> {
        let (page, limit) = pagination(page, limit);
        map_insight_page(
            self.insights
                .find_by_organization_id(organization_id, (page - 1) * limit, limit)
                .await?,
        )
    }

    pub async fn list_all_insights(
        &self,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<InsightResponse>, i64), AppError> {
        let (page, limit) = pagination(page, limit);
        map_insight_page(self.insights.list((page - 1) * limit, limit).await?)
    }

    pub async fn list_insights_by_topic(
        &self,
        topic_id: Uuid,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<InsightResponse>, i64), AppError> {
        let (page, limit) = pagination(page, limit);
        map_insight_page(
            self.insights
                .find_by_topic_id(topic_id, (page - 1) * limit, limit)
                .await?,
        )
    }

    pub async fn list_unread_insights(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<InsightResponse>, AppError> {
        self.insights
            .find_unread(organization_id, limit)
            .await
            .map(map_insights)
    }

    pub async fn search_insights(
        &self,
        request: InsightSearchRequest,
    ) -> Result<Vec<InsightResponse>, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(&request.query)
            .await
            .map_err(|_| AppError::External)?;
        self.insights
            .search_similar(&embedding, semantic_limit(request.limit))
            .await
            .map(map_insights)
    }

    pub async fn search_insights_by_org(
        &self,
        organization_id: Uuid,
        query: &str,
        limit: i64,
    ) -> Result<Vec<InsightResponse>, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(query)
            .await
            .map_err(|_| AppError::External)?;
        self.insights
            .search_similar_by_org(organization_id, &embedding, semantic_limit(limit))
            .await
            .map(map_insights)
    }

    #[allow(clippy::too_many_arguments)]
    pub async fn create_insight(
        &self,
        organization_id: Option<Uuid>,
        topic_id: Option<Uuid>,
        title: String,
        summary: String,
        content: String,
        key_points: Option<Vec<String>>,
        source_content_ids: Option<Vec<Uuid>>,
        period_start: Option<DateTime<Utc>>,
        period_end: Option<DateTime<Utc>>,
    ) -> Result<InsightResponse, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(&format!("{title} {summary}"))
            .await
            .map_err(|_| AppError::External)?;
        let mut insight = Insight {
            id: Uuid::new_v4(),
            organization_id,
            topic_id,
            title,
            summary,
            content: Some(content),
            key_points,
            source_content_ids: source_content_ids.unwrap_or_default(),
            embedding: Some(embedding),
            generated_at: Some(Utc::now()),
            period_start,
            period_end,
            is_read: false,
            is_pinned: false,
            is_used_in_article: false,
            meta_data: None,
        };
        self.insights.save(&mut insight).await?;
        if let Some(topic_id) = insight.topic_id {
            let _ = self
                .topics
                .update_last_insight_at(topic_id, Utc::now())
                .await;
        }
        Ok(insight.into())
    }

    pub async fn mark_insight_as_read(&self, id: Uuid) -> Result<(), AppError> {
        self.insights.mark_as_read(id).await
    }

    pub async fn mark_insight_as_read_for_user(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<(), AppError> {
        self.user_statuses.mark_as_read(user_id, insight_id).await
    }

    pub async fn toggle_insight_pinned_for_user(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<bool, AppError> {
        self.user_statuses.toggle_pinned(user_id, insight_id).await
    }

    pub async fn mark_insight_as_used_in_article_for_user(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<(), AppError> {
        self.user_statuses
            .mark_as_used_in_article(user_id, insight_id)
            .await
    }

    pub async fn get_user_insight_status(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<Option<UserInsightStatus>, AppError> {
        self.user_statuses
            .find_by_user_and_insight(user_id, insight_id)
            .await
    }

    pub async fn get_insight_with_user_status(
        &self,
        user_id: Uuid,
        insight_id: Uuid,
    ) -> Result<InsightWithUserStatus, AppError> {
        let insight = self.insights.find_by_id(insight_id).await?;
        let status = self
            .user_statuses
            .find_by_user_and_insight(user_id, insight_id)
            .await?;
        Ok(InsightWithUserStatus {
            insight: insight.into(),
            user_status: status.map(Into::into),
        })
    }

    pub async fn list_insights_with_user_status(
        &self,
        user_id: Uuid,
        page: i64,
        limit: i64,
    ) -> Result<(Vec<InsightWithUserStatus>, i64), AppError> {
        let (page, limit) = pagination(page, limit);
        let (insights, total) = self.insights.list((page - 1) * limit, limit).await?;
        let ids: Vec<_> = insights.iter().map(|insight| insight.id).collect();
        let statuses = self
            .user_statuses
            .get_status_map_for_insights(user_id, &ids)
            .await?;
        Ok((
            insights
                .into_iter()
                .map(|insight| InsightWithUserStatus {
                    user_status: statuses.get(&insight.id).cloned().map(Into::into),
                    insight: insight.into(),
                })
                .collect(),
            total,
        ))
    }

    pub async fn count_unread_insights_for_user(&self, user_id: Uuid) -> Result<i64, AppError> {
        let (_, total) = self.insights.list(0, 1).await?;
        let unread_statuses = self.user_statuses.count_unread_by_user_id(user_id).await?;
        Ok(total - unread_statuses)
    }

    pub async fn toggle_insight_pinned(&self, id: Uuid) -> Result<(), AppError> {
        self.insights.toggle_pinned(id).await
    }

    pub async fn mark_insight_as_used_in_article(&self, id: Uuid) -> Result<(), AppError> {
        self.insights.mark_as_used_in_article(id).await
    }

    pub async fn delete_insight(&self, id: Uuid) -> Result<(), AppError> {
        self.insights.delete(id).await
    }

    pub async fn count_unread_insights(&self, organization_id: Uuid) -> Result<i64, AppError> {
        self.insights.count_unread(organization_id).await
    }

    pub async fn count_all_unread_insights(&self) -> Result<i64, AppError> {
        self.insights.count_all_unread().await
    }

    pub async fn get_topic_by_id(&self, id: Uuid) -> Result<InsightTopicResponse, AppError> {
        self.topics.find_by_id(id).await.map(Into::into)
    }

    pub async fn list_topics(
        &self,
        organization_id: Uuid,
    ) -> Result<Vec<InsightTopicResponse>, AppError> {
        self.topics
            .find_by_organization_id(organization_id)
            .await
            .map(|items| items.into_iter().map(Into::into).collect())
    }

    pub async fn list_all_topics(&self) -> Result<Vec<InsightTopicResponse>, AppError> {
        self.topics
            .find_all()
            .await
            .map(|items| items.into_iter().map(Into::into).collect())
    }

    pub async fn create_topic(
        &self,
        organization_id: Option<Uuid>,
        request: InsightTopicCreateRequest,
    ) -> Result<InsightTopicResponse, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(&topic_embedding_text(
                &request.name,
                request.description.as_deref(),
                request.keywords.as_deref(),
            ))
            .await
            .map_err(|_| AppError::External)?;
        let mut topic = InsightTopic {
            id: Uuid::new_v4(),
            organization_id,
            name: request.name,
            description: request.description,
            keywords: request.keywords,
            embedding: Some(embedding),
            is_auto_generated: false,
            content_count: 0,
            last_insight_at: None,
            color: request.color,
            icon: request.icon,
            created_at: None,
            updated_at: None,
        };
        self.topics.save(&mut topic).await?;
        Ok(topic.into())
    }

    pub async fn update_topic(
        &self,
        id: Uuid,
        request: InsightTopicUpdateRequest,
    ) -> Result<InsightTopicResponse, AppError> {
        let mut topic = self.topics.find_by_id(id).await?;
        let mut update_embedding = false;
        if let Some(name) = request.name {
            topic.name = name;
            update_embedding = true;
        }
        if let Some(description) = request.description {
            topic.description = Some(description);
            update_embedding = true;
        }
        if let Some(keywords) = request.keywords {
            topic.keywords = Some(keywords);
            update_embedding = true;
        }
        if let Some(color) = request.color {
            topic.color = Some(color);
        }
        if let Some(icon) = request.icon {
            topic.icon = Some(icon);
        }
        if update_embedding {
            topic.embedding = Some(
                self.embeddings
                    .generate_embedding(&topic_embedding_text(
                        &topic.name,
                        topic.description.as_deref(),
                        topic.keywords.as_deref(),
                    ))
                    .await
                    .map_err(|_| AppError::External)?,
            );
        }
        self.topics.update(&topic).await?;
        Ok(topic.into())
    }

    pub async fn delete_topic(&self, id: Uuid) -> Result<(), AppError> {
        self.topics.delete(id).await
    }

    pub async fn match_content_to_topics(
        &self,
        content_id: Uuid,
        embedding: &[f32],
        threshold: f64,
    ) -> Result<Vec<ContentTopicMatch>, AppError> {
        let (topics, scores) = self.topics.search_similar(embedding, 10, threshold).await?;
        if topics.len() != scores.len() {
            tracing::error!(
                topic_count = topics.len(),
                score_count = scores.len(),
                "insight topic similarity result is internally inconsistent"
            );
            return Err(AppError::Internal);
        }
        let mut matches: Vec<_> = topics
            .iter()
            .zip(scores)
            .enumerate()
            .map(|(index, (topic, score))| ContentTopicMatch {
                id: Uuid::new_v4(),
                content_id,
                topic_id: topic.id,
                similarity_score: score,
                is_primary: index == 0,
                created_at: None,
            })
            .collect();
        if matches.is_empty() {
            return Ok(matches);
        }
        self.matches.save_batch(&mut matches).await?;
        for topic in topics {
            if let Ok(count) = self.matches.count_by_topic_id(topic.id).await
                && let Ok(count) = i32::try_from(count)
            {
                let _ = self.topics.update_content_count(topic.id, count).await;
            }
        }
        Ok(matches)
    }

    pub async fn search_crawled_content(
        &self,
        query: &str,
        limit: i64,
    ) -> Result<Vec<CrawledContentResponse>, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(query)
            .await
            .map_err(|_| AppError::External)?;
        self.contents
            .search_similar(&embedding, semantic_limit(limit))
            .await
            .map(map_contents)
    }

    pub async fn search_crawled_content_by_org(
        &self,
        organization_id: Uuid,
        query: &str,
        limit: i64,
    ) -> Result<Vec<CrawledContentResponse>, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(query)
            .await
            .map_err(|_| AppError::External)?;
        self.contents
            .search_similar_by_org(organization_id, &embedding, semantic_limit(limit))
            .await
            .map(map_contents)
    }

    pub async fn get_recent_crawled_content(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<CrawledContentResponse>, AppError> {
        self.contents
            .find_recent_by_org(organization_id, limit)
            .await
            .map(map_contents)
    }
}

fn pagination(page: i64, limit: i64) -> (i64, i64) {
    (
        page.max(1),
        if !(1..=100).contains(&limit) {
            20
        } else {
            limit
        },
    )
}

fn semantic_limit(limit: i64) -> i64 {
    if !(1..=50).contains(&limit) {
        10
    } else {
        limit
    }
}

fn map_insights(values: Vec<Insight>) -> Vec<InsightResponse> {
    values.into_iter().map(Into::into).collect()
}

fn map_insight_page(page: (Vec<Insight>, i64)) -> Result<(Vec<InsightResponse>, i64), AppError> {
    Ok((map_insights(page.0), page.1))
}

fn map_contents(
    values: Vec<crate::core::datasource::CrawledContent>,
) -> Vec<CrawledContentResponse> {
    values.into_iter().map(Into::into).collect()
}

fn topic_embedding_text(
    name: &str,
    description: Option<&str>,
    keywords: Option<&[String]>,
) -> String {
    let mut parts = vec![name.to_owned()];
    if let Some(description) = description.filter(|value| !value.is_empty()) {
        parts.push(description.to_owned());
    }
    if let Some(keywords) = keywords.filter(|values| !values.is_empty()) {
        parts.push(keywords.join(" "));
    }
    parts.join(" ")
}
