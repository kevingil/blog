use std::{collections::BTreeMap, sync::Arc};

use chrono::{SecondsFormat, Utc};
use serde_json::{Map, Value};
use uuid::Uuid;

use crate::error::AppError;

use super::{
    AgentResourceSelection, ArticleLookupPort, CreateSourceRequest, EmbeddingPort,
    FetchExtractPort, Source, SourceListOptions, SourceListResponse, SourceRepository,
    UpdateSourceRequest,
};

#[derive(Clone)]
pub struct SourceService {
    sources: Arc<dyn SourceRepository>,
    articles: Arc<dyn ArticleLookupPort>,
    embeddings: Arc<dyn EmbeddingPort>,
    fetch_extract: Arc<dyn FetchExtractPort>,
}

impl SourceService {
    pub fn new(
        sources: Arc<dyn SourceRepository>,
        articles: Arc<dyn ArticleLookupPort>,
        embeddings: Arc<dyn EmbeddingPort>,
        fetch_extract: Arc<dyn FetchExtractPort>,
    ) -> Self {
        Self {
            sources,
            articles,
            embeddings,
            fetch_extract,
        }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<Source, AppError> {
        self.sources.find_by_id(id).await
    }

    pub async fn get_by_article_id(&self, article_id: Uuid) -> Result<Vec<Source>, AppError> {
        self.sources.find_by_article_id(article_id).await
    }

    pub async fn list(&self, page: i64, limit: i64) -> Result<SourceListResponse, AppError> {
        let page = page.max(1);
        let limit = if !(1..=100).contains(&limit) {
            20
        } else {
            limit
        };
        let (sources, total) = self
            .sources
            .list(SourceListOptions {
                page,
                per_page: limit,
            })
            .await?;
        Ok(SourceListResponse {
            sources,
            total_pages: (total + limit - 1) / limit,
            page,
        })
    }

    pub async fn create(&self, request: CreateSourceRequest) -> Result<Source, AppError> {
        self.articles.ensure_exists(request.article_id).await?;
        let embedding = self
            .embeddings
            .generate_embedding(&request.content)
            .await
            .map_err(|_| AppError::External)?;
        let source_type = if request.source_type.is_empty() {
            if request.url.is_empty() {
                "manual".to_owned()
            } else {
                "web".to_owned()
            }
        } else {
            request.source_type
        };
        let mut source = Source {
            id: Uuid::new_v4(),
            article_id: request.article_id,
            title: request.title,
            content: request.content,
            url: request.url,
            source_type,
            embedding: Some(embedding),
            meta_data: request.meta_data,
            created_at: Some(Utc::now()),
        };
        self.sources.save(&mut source).await?;
        Ok(source)
    }

    pub async fn scrape_and_create(
        &self,
        article_id: Uuid,
        target_url: &str,
    ) -> Result<Source, AppError> {
        let scraped = self
            .fetch_extract
            .fetch_extract(target_url)
            .await
            .map_err(|_| AppError::External)?;
        self.create(CreateSourceRequest {
            article_id,
            title: scraped.title,
            content: scraped.content,
            url: scraped.url,
            source_type: if is_pdf_url(target_url) {
                "pdf".to_owned()
            } else {
                "web".to_owned()
            },
            meta_data: None,
        })
        .await
    }

    pub async fn update(&self, id: Uuid, request: UpdateSourceRequest) -> Result<Source, AppError> {
        let mut source = self.sources.find_by_id(id).await?;
        if let Some(title) = request.title {
            source.title = title;
        }
        if let Some(content) = request.content {
            source.content = content;
            source.embedding = Some(
                self.embeddings
                    .generate_embedding(&source.content)
                    .await
                    .map_err(|_| AppError::External)?,
            );
        }
        if let Some(url) = request.url {
            source.url = url;
        }
        if let Some(source_type) = request.source_type {
            source.source_type = source_type;
        }
        if let Some(meta_data) = request.meta_data {
            source.meta_data = Some(meta_data);
        }
        self.sources.update(&source).await?;
        Ok(source)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.sources.delete(id).await
    }

    pub async fn search_similar(
        &self,
        article_id: Uuid,
        query: &str,
        limit: i64,
    ) -> Result<Vec<Source>, AppError> {
        let embedding = self
            .embeddings
            .generate_embedding(query)
            .await
            .map_err(|_| AppError::External)?;
        self.sources
            .search_similar(article_id, &embedding, limit)
            .await
    }

    pub async fn upsert_agent_resource(
        &self,
        request: AgentResourceSelection,
    ) -> Result<Source, AppError> {
        if request.article_id.is_nil() {
            return Err(AppError::InvalidInput("article_id is required".to_owned()));
        }
        if request.selected_excerpt.is_empty() && request.content.is_empty() {
            return Err(AppError::InvalidInput(
                "selected excerpt or content is required".to_owned(),
            ));
        }

        let mut existing = if request.source_id.is_some_and(|id| !id.is_nil()) {
            match self
                .sources
                .find_by_id(request.source_id.unwrap_or_default())
                .await
            {
                Ok(source) => {
                    if source.article_id != request.article_id {
                        return Err(AppError::InvalidInput(
                            "source does not belong to article".to_owned(),
                        ));
                    }
                    Some(source)
                }
                Err(AppError::NotFound) => None,
                Err(error) => return Err(error),
            }
        } else {
            None
        };

        if existing.is_none() && !request.url.is_empty() {
            existing = self
                .sources
                .find_by_article_id(request.article_id)
                .await?
                .into_iter()
                .find(|source| {
                    !source.url.is_empty()
                        && source.url.trim().eq_ignore_ascii_case(request.url.trim())
                });
        }

        if let Some(mut source) = existing {
            if !request.title.is_empty() {
                source.title = request.title.clone();
            }
            if !request.source_type.is_empty() {
                source.source_type = request.source_type.clone();
            }
            if !request.content.is_empty() && source.content.trim().is_empty() {
                source.content = request.content.clone();
            }
            source.meta_data = Some(agent_resource_meta(source.meta_data.take(), &request));
            self.sources.update(&source).await?;
            return Ok(source);
        }

        self.create(CreateSourceRequest {
            article_id: request.article_id,
            title: request.title.clone(),
            content: if request.content.is_empty() {
                request.selected_excerpt.clone()
            } else {
                request.content.clone()
            },
            url: request.url.clone(),
            source_type: if request.source_type.is_empty() {
                "web".to_owned()
            } else {
                request.source_type.clone()
            },
            meta_data: Some(agent_resource_meta(None, &request)),
        })
        .await
    }
}

fn agent_resource_meta(
    existing: Option<BTreeMap<String, Value>>,
    request: &AgentResourceSelection,
) -> BTreeMap<String, Value> {
    let mut result = existing.unwrap_or_default();
    let mut resource = result
        .remove("resource")
        .and_then(|value| value.as_object().cloned())
        .unwrap_or_default();
    insert_nonempty(&mut resource, "origin_tool", &request.origin_tool);
    insert_nonempty(&mut resource, "origin_query", &request.origin_query);
    insert_nonempty(&mut resource, "origin_question", &request.origin_question);
    insert_nonempty(&mut resource, "author", &request.author);
    insert_nonempty(&mut resource, "published_date", &request.published_date);
    insert_nonempty(&mut resource, "selected_excerpt", &request.selected_excerpt);
    insert_nonempty(
        &mut resource,
        "selected_excerpt_id",
        &request.selected_excerpt_id,
    );
    resource.insert(
        "usage_status".to_owned(),
        Value::String(if request.usage_status.is_empty() {
            "used".to_owned()
        } else {
            request.usage_status.clone()
        }),
    );
    let now = Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true);
    resource.insert("selected_at".to_owned(), Value::String(now.clone()));
    resource.insert("last_used_at".to_owned(), Value::String(now));
    insert_nonempty(&mut resource, "last_used_in_turn", &request.request_id);
    result.insert("resource".to_owned(), Value::Object(resource));
    result
}

fn insert_nonempty(target: &mut Map<String, Value>, key: &str, value: &str) {
    if !value.is_empty() {
        target.insert(key.to_owned(), Value::String(value.to_owned()));
    }
}

fn is_pdf_url(target: &str) -> bool {
    let lower = target.to_lowercase();
    lower.ends_with(".pdf")
        || lower.contains(".pdf")
        || lower.contains("/pdf/")
        || lower.contains("content-type=application/pdf")
}
