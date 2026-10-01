use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, TimeZone, Utc};
use reqwest::Url;
use serde::{Deserialize, Deserializer, Serialize};
use utoipa::ToSchema;
use uuid::Uuid;

use crate::{
    core::{
        auth::{AccountId, AccountRepository},
        storage::{UploadRepository, resolve_image},
        tag::TagRepository,
    },
    error::AppError,
};

use super::{
    Article, ArticleListOptions, ArticleRepository, ArticleSearchOptions, ArticleVersion,
    ExternalImageCache, ExternalPage, ExternalPagePort,
};

pub const ITEMS_PER_PAGE: i64 = 6;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct AuthorData {
    pub id: Uuid,
    pub name: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct TagData {
    pub article_id: Uuid,
    pub tag_id: i32,
    pub name: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct ArticleListItem {
    pub article: Article,
    pub author: AuthorData,
    pub tags: Vec<TagData>,
}

pub type ArticleData = ArticleListItem;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct RecommendedArticle {
    pub id: Uuid,
    pub title: String,
    pub slug: String,
    pub image_url: Option<String>,
    pub published_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
    pub author: Option<String>,
    #[serde(default)]
    pub external_url: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image: Option<crate::core::storage::ImageAsset>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct ArticleListResponse {
    pub articles: Vec<ArticleListItem>,
    pub total_pages: i64,
    pub include_drafts: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct ArticleVersionResponse {
    pub id: Uuid,
    pub article_id: Uuid,
    pub version_number: i32,
    pub status: String,
    pub title: String,
    pub content: String,
    pub image_url: String,
    pub created_at: Option<DateTime<Utc>>,
}

impl From<ArticleVersion> for ArticleVersionResponse {
    fn from(version: ArticleVersion) -> Self {
        Self {
            id: version.id,
            article_id: version.article_id,
            version_number: version.version_number,
            status: version.status,
            title: version.title,
            content: version.content,
            image_url: version.image_url,
            created_at: version.created_at,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct ArticleVersionListResponse {
    pub versions: Vec<ArticleVersionResponse>,
    pub total: usize,
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, ToSchema)]
pub struct CreateArticle {
    pub title: String,
    pub content: String,
    #[serde(default)]
    pub image_url: String,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(default)]
    pub publish: bool,
    #[serde(rename = "authorId")]
    pub author_id: Uuid,
    /// When set, the public article opens this link. Empty keeps it on this site.
    #[serde(default)]
    pub external_url: Option<String>,
    #[serde(default)]
    pub image_upload_id: Option<Uuid>,
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, ToSchema)]
pub struct UpdateArticle {
    pub title: String,
    pub content: String,
    #[serde(default)]
    pub image_url: String,
    #[serde(default)]
    pub tags: Vec<String>,
    pub published_at: Option<i64>,
    /// `null` or `""` removes the external link. Omit the field to leave it unchanged.
    #[serde(default, deserialize_with = "deserialize_optional_update")]
    #[schema(value_type = Option<String>)]
    pub external_url: Option<Option<String>>,
    #[serde(default)]
    pub image_upload_id: Option<Uuid>,
}

fn deserialize_optional_update<'de, D>(deserializer: D) -> Result<Option<Option<String>>, D::Error>
where
    D: Deserializer<'de>,
{
    Ok(Some(Option::<String>::deserialize(deserializer)?))
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, ToSchema)]
pub struct CreateExternalArticle {
    pub url: String,
}

#[async_trait]
pub trait ArticleEmbeddingProvider: Send + Sync {
    async fn generate_embedding(&self, content: &str) -> Result<Vec<f32>, AppError>;
}

#[async_trait]
pub trait ArticleContextWriter: Send + Sync {
    async fn update_with_context(&self, article: &Article) -> Result<String, AppError>;
}

#[derive(Clone)]
pub struct ArticleService {
    articles: Arc<dyn ArticleRepository>,
    accounts: Arc<dyn AccountRepository>,
    tags: Arc<dyn TagRepository>,
    embeddings: Option<Arc<dyn ArticleEmbeddingProvider>>,
    context_writer: Option<Arc<dyn ArticleContextWriter>>,
    external_pages: Option<Arc<dyn ExternalPagePort>>,
    image_cache: Option<Arc<dyn ExternalImageCache>>,
    uploads: Option<Arc<dyn UploadRepository>>,
}

impl ArticleService {
    pub fn new(
        articles: Arc<dyn ArticleRepository>,
        accounts: Arc<dyn AccountRepository>,
        tags: Arc<dyn TagRepository>,
    ) -> Self {
        Self {
            articles,
            accounts,
            tags,
            embeddings: None,
            context_writer: None,
            external_pages: None,
            image_cache: None,
            uploads: None,
        }
    }

    pub fn with_uploads(mut self, uploads: Arc<dyn UploadRepository>) -> Self {
        self.uploads = Some(uploads);
        self
    }

    pub fn with_embedding_provider(mut self, provider: Arc<dyn ArticleEmbeddingProvider>) -> Self {
        self.embeddings = Some(provider);
        self
    }

    pub fn with_context_writer(mut self, writer: Arc<dyn ArticleContextWriter>) -> Self {
        self.context_writer = Some(writer);
        self
    }

    pub fn with_external_blogs(
        mut self,
        pages: Arc<dyn ExternalPagePort>,
        images: Arc<dyn ExternalImageCache>,
    ) -> Self {
        self.external_pages = Some(pages);
        self.image_cache = Some(images);
        self
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<ArticleListItem, AppError> {
        let article = self.articles.find_by_id(id).await?;
        self.enrich(article).await
    }

    pub async fn get_by_slug(&self, slug: &str) -> Result<ArticleData, AppError> {
        let article = self.articles.find_by_slug(slug).await?;
        self.enrich(article).await
    }

    pub async fn get_id_by_slug(&self, slug: &str) -> Result<Uuid, AppError> {
        Ok(self.articles.find_by_slug(slug).await?.id)
    }

    pub async fn list(
        &self,
        page: i64,
        tag_name: &str,
        status: &str,
        articles_per_page: i64,
        sort_by: &str,
        sort_order: &str,
    ) -> Result<ArticleListResponse, AppError> {
        let per_page = if articles_per_page <= 0 {
            ITEMS_PER_PAGE
        } else {
            articles_per_page
        };
        let tag_id = if tag_name.is_empty() {
            None
        } else {
            self.tags
                .find_by_name(tag_name)
                .await
                .ok()
                .map(|tag| i64::from(tag.id))
        };
        let (articles, total) = self
            .articles
            .list(ArticleListOptions {
                page,
                per_page,
                published_only: status != "all" && status != "drafts",
                author_id: None,
                tag_id,
                sort_by: sort_by.to_owned(),
                sort_order: sort_order.to_owned(),
            })
            .await?;
        self.enrich_page(articles, total, per_page, status).await
    }

    pub async fn search(
        &self,
        query: &str,
        page: i64,
        status: &str,
    ) -> Result<ArticleListResponse, AppError> {
        let (articles, total) = self
            .articles
            .search(ArticleSearchOptions {
                query: query.to_owned(),
                page,
                per_page: ITEMS_PER_PAGE,
                published_only: status != "all" && status != "drafts",
            })
            .await?;
        self.enrich_page(articles, total, ITEMS_PER_PAGE, status)
            .await
    }

    pub async fn get_popular_tags(&self) -> Result<Vec<String>, AppError> {
        let ids = self.articles.get_popular_tags(10).await?;
        Ok(self
            .tags
            .find_by_ids(&ids)
            .await?
            .into_iter()
            .map(|tag| tag.name)
            .collect())
    }

    pub async fn get_recommended(
        &self,
        current_article_id: Uuid,
    ) -> Result<Vec<RecommendedArticle>, AppError> {
        let (articles, _) = self
            .articles
            .list(ArticleListOptions {
                page: 1,
                per_page: 4,
                published_only: true,
                ..ArticleListOptions::default()
            })
            .await?;

        let mut recommended = Vec::with_capacity(3);
        for article in articles {
            if article.id == current_article_id {
                continue;
            }
            if recommended.len() == 3 {
                break;
            }
            let author = self
                .accounts
                .find_by_id(AccountId(article.author_id))
                .await
                .ok()
                .flatten()
                .map(|account| account.name);
            recommended.push(RecommendedArticle {
                id: article.id,
                title: article
                    .published_title
                    .clone()
                    .unwrap_or_else(|| article.draft_title.clone()),
                slug: article.slug.clone(),
                image_url: article
                    .published_image_url
                    .filter(|url| !url.is_empty())
                    .or_else(|| {
                        (!article.draft_image_url.is_empty()).then(|| article.draft_image_url.clone())
                    }),
                published_at: article.published_at,
                created_at: article.created_at,
                author,
                external_url: article.external_url.clone(),
                image: article
                    .published_image
                    .clone()
                    .or_else(|| article.draft_image.clone()),
            });
        }
        Ok(recommended)
    }

    pub async fn create_draft_shell(
        &self,
        title: &str,
        author_id: Uuid,
    ) -> Result<Article, AppError> {
        let now = Utc::now();
        let mut article = new_article(
            self.unique_slug(title, None).await?,
            title.to_owned(),
            String::new(),
            String::new(),
            author_id,
            Vec::new(),
            now,
        );
        article.tag_ids = None;
        self.articles.save(&mut article).await?;
        Ok(article)
    }

    pub async fn update_generated_draft(
        &self,
        article_id: Uuid,
        content: &str,
    ) -> Result<(), AppError> {
        if content.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "generated article content cannot be empty".to_owned(),
            ));
        }
        self.articles
            .update_draft_content(article_id, content)
            .await
    }

    pub async fn apply_generated_image(
        &self,
        article_id: Uuid,
        image_request_id: Uuid,
        output_url: &str,
    ) -> Result<(), AppError> {
        if output_url.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "generated image URL cannot be empty".to_owned(),
            ));
        }
        let mut article = self.articles.find_by_id(article_id).await?;
        article.imagen_request_id = Some(image_request_id);
        article.draft_image_url = output_url.to_owned();
        article.draft_upload_file_id = self.lookup_upload_id(output_url).await?;
        article.updated_at = Some(Utc::now());
        self.articles.save_draft(&mut article).await
    }

    pub async fn create(&self, mut request: CreateArticle) -> Result<ArticleListItem, AppError> {
        let page = self
            .resolve_external_link(None, request.external_url.as_deref().unwrap_or(""))
            .await?;
        if let Some(page) = &page {
            fill_missing_copy(&mut request.title, &mut request.content, page);
        }
        validate_create(&request)?;
        let tag_ids = self.tags.ensure_exists(&request.tags).await?;
        let now = Utc::now();
        let mut article = new_article(
            self.unique_slug(&request.title, None).await?,
            request.title.clone(),
            request.content.clone(),
            request.image_url.clone(),
            request.author_id,
            tag_ids,
            now,
        );
        if let Some(page) = &page {
            article.external_url = Some(page.url.clone());
            if article.draft_image_url.is_empty() {
                if let Some(stored) = self.cache_page_image(article.id, page).await? {
                    article.draft_upload_file_id = self.lookup_upload_id(&stored).await?;
                    article.draft_image_url = stored;
                }
            }
        }
        if let Some(id) = request.image_upload_id {
            let resolved = resolve_image(self.uploads.as_deref(), Some(id), "", "", None).await?;
            article.draft_image_url = resolved.url;
            article.draft_upload_file_id = resolved.upload_file_id;
            article.draft_image = resolved.image;
        }
        if request.publish {
            article.published_title = Some(request.title);
            article.published_content = Some(request.content);
            article.published_image_url = Some(article.draft_image_url.clone());
            article.published_upload_file_id = article.draft_upload_file_id;
            article.published_image = article.draft_image.clone();
            article.published_at = Some(now);
        }
        let id = article.id;
        self.articles.save(&mut article).await?;
        self.get_by_id(id).await
    }

    pub async fn create_external(
        &self,
        author_id: Uuid,
        request: CreateExternalArticle,
    ) -> Result<ArticleListItem, AppError> {
        let pages = self
            .external_pages
            .as_ref()
            .ok_or_else(|| AppError::InvalidInput("external blogs are unavailable".to_owned()))?;
        let images = self
            .image_cache
            .as_ref()
            .ok_or_else(|| AppError::InvalidInput("external blogs are unavailable".to_owned()))?;
        let url = request.url.trim();
        if url.is_empty() {
            return Err(AppError::InvalidInput("url is required".to_owned()));
        }
        let page = pages.load(url).await?;
        if self.articles.external_url_exists(&page.url, None).await? {
            return Err(AppError::Conflict(
                "an external blog for this link already exists".to_owned(),
            ));
        }
        let title = fit_title(&page.title);
        let excerpt = fit_excerpt(&page.excerpt);
        validate_article_fields(&title, &excerpt, "", &[])?;
        let now = Utc::now();
        let mut article = new_article(
            self.unique_slug(&title, None).await?,
            title.clone(),
            excerpt.clone(),
            String::new(),
            author_id,
            Vec::new(),
            now,
        );
        article.external_url = Some(page.url);
        article.published_title = Some(title);
        article.published_content = Some(excerpt);
        article.published_at = Some(page.published_at.unwrap_or(now));
        if let Some(image) = page.image {
            let key = format!("external-blogs/{}.{}", article.id, image.extension);
            let stored = images.store(&key, &image.content_type, image.bytes).await?;
            article.draft_upload_file_id = self.lookup_upload_id(&stored).await?;
            article.draft_image_url = stored.clone();
            article.published_upload_file_id = article.draft_upload_file_id;
            article.published_image_url = Some(stored);
        }
        let id = article.id;
        self.articles.save(&mut article).await?;
        self.get_by_id(id).await
    }

    pub async fn update(
        &self,
        article_id: Uuid,
        mut request: UpdateArticle,
    ) -> Result<ArticleListItem, AppError> {
        let mut article = self.articles.find_by_id(article_id).await?;
        let link_change = self
            .external_link_change(article_id, &article, &request)
            .await?;
        if let ExternalLinkChange::Set(page) = &link_change {
            fill_missing_copy(&mut request.title, &mut request.content, page);
            if should_replace_external_image(&request.image_url) {
                if let Some(stored) = self.cache_page_image(article_id, page).await? {
                    request.image_url = stored;
                }
            }
        }
        validate_update(&request)?;
        let tag_ids = self.tags.ensure_exists(&request.tags).await?;
        if article.draft_title != request.title {
            article.slug = self.unique_slug(&request.title, Some(article_id)).await?;
        }
        let resolved = resolve_image(
            self.uploads.as_deref(),
            request.image_upload_id,
            &request.image_url,
            &article.draft_image_url,
            article.draft_upload_file_id,
        )
        .await?;
        let draft_changed = article.draft_title != request.title
            || article.draft_content != request.content
            || article.draft_image_url != resolved.url
            || article.draft_upload_file_id != resolved.upload_file_id;
        if draft_changed {
            self.articles.create_draft_snapshot(article_id).await?;
        }
        article.draft_title = request.title;
        article.draft_content = request.content.clone();
        article.draft_image_url = resolved.url;
        article.draft_upload_file_id = resolved.upload_file_id;
        article.draft_image = resolved.image;
        article.tag_ids = Some(tag_ids);
        article.updated_at = Some(Utc::now());
        if let Some(timestamp) = request.published_at {
            article.published_at = Some(timestamp_to_utc(timestamp)?);
        }
        match link_change {
            ExternalLinkChange::Unchanged => {}
            ExternalLinkChange::Clear => article.external_url = None,
            ExternalLinkChange::Set(page) => {
                article.external_url = Some(page.url);
                if article.published_at.is_some() {
                    if article
                        .published_image_url
                        .as_deref()
                        .is_none_or(should_replace_external_image)
                        && !article.draft_image_url.is_empty()
                    {
                        article.published_upload_file_id = article.draft_upload_file_id;
                        article.published_image = article.draft_image.clone();
                        article.published_image_url = Some(article.draft_image_url.clone());
                    }
                    if article
                        .published_content
                        .as_deref()
                        .unwrap_or("")
                        .chars()
                        .count()
                        < 10
                    {
                        article.published_content = Some(article.draft_content.clone());
                        article.published_title = Some(article.draft_title.clone());
                    }
                }
            }
        }
        self.articles.save(&mut article).await?;

        if let Some(provider) = &self.embeddings {
            match provider.generate_embedding(&request.content).await {
                Ok(embedding) => {
                    let mut stored = self.articles.find_by_id(article_id).await?;
                    stored.draft_embedding = embedding;
                    self.articles.save(&mut stored).await?;
                }
                Err(error) => {
                    tracing::warn!(%article_id, %error, "failed to regenerate article embedding");
                }
            }
        }

        self.get_by_id(article_id).await
    }

    pub async fn update_with_context(&self, article_id: Uuid) -> Result<Article, AppError> {
        let mut article = self.articles.find_by_id(article_id).await?;
        let writer = self.context_writer.as_ref().ok_or(AppError::External)?;
        let updated = writer.update_with_context(&article).await?;
        if updated != article.draft_content {
            self.articles.create_draft_snapshot(article_id).await?;
            article.draft_content = updated;
            article.updated_at = Some(Utc::now());
            self.articles.save(&mut article).await?;
        }
        Ok(article)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.articles.delete(id).await
    }

    pub async fn publish(
        &self,
        article_id: Uuid,
        published_at: Option<DateTime<Utc>>,
    ) -> Result<ArticleListItem, AppError> {
        let mut article = self.articles.find_by_id(article_id).await?;
        self.articles.publish(&mut article, published_at).await?;
        self.get_by_id(article_id).await
    }

    pub async fn unpublish(&self, article_id: Uuid) -> Result<ArticleListItem, AppError> {
        let mut article = self.articles.find_by_id(article_id).await?;
        if article.published_at.is_none() {
            return Err(AppError::InvalidInput(
                "article is not published".to_owned(),
            ));
        }
        self.articles.unpublish(&mut article).await?;
        self.get_by_id(article_id).await
    }

    pub async fn list_versions(
        &self,
        article_id: Uuid,
    ) -> Result<ArticleVersionListResponse, AppError> {
        self.articles.find_by_id(article_id).await?;
        let versions: Vec<_> = self
            .articles
            .list_versions(article_id)
            .await?
            .into_iter()
            .map(Into::into)
            .collect();
        Ok(ArticleVersionListResponse {
            total: versions.len(),
            versions,
        })
    }

    pub async fn get_version(&self, version_id: Uuid) -> Result<ArticleVersionResponse, AppError> {
        Ok(self.articles.get_version(version_id).await?.into())
    }

    pub async fn revert_to_version(
        &self,
        article_id: Uuid,
        version_id: Uuid,
    ) -> Result<ArticleListItem, AppError> {
        self.articles.find_by_id(article_id).await?;
        self.articles
            .revert_to_version(article_id, version_id)
            .await?;
        self.get_by_id(article_id).await
    }

    async fn enrich_page(
        &self,
        articles: Vec<Article>,
        total: i64,
        per_page: i64,
        status: &str,
    ) -> Result<ArticleListResponse, AppError> {
        let mut items = Vec::with_capacity(articles.len());
        for article in articles {
            items.push(self.enrich(article).await?);
        }
        Ok(ArticleListResponse {
            articles: items,
            total_pages: if total == 0 {
                0
            } else {
                (total + per_page - 1) / per_page
            },
            include_drafts: status == "all" || status == "drafts",
        })
    }

    async fn enrich(&self, article: Article) -> Result<ArticleListItem, AppError> {
        let author = match self.accounts.find_by_id(AccountId(article.author_id)).await {
            Ok(Some(account)) => AuthorData {
                id: article.author_id,
                name: account.name,
            },
            Ok(None) | Err(_) => AuthorData {
                id: article.author_id,
                name: String::new(),
            },
        };
        let tags = match article.tag_ids.as_deref() {
            Some(ids) if !ids.is_empty() => self
                .tags
                .find_by_ids(ids)
                .await?
                .into_iter()
                .map(|tag| TagData {
                    article_id: article.id,
                    tag_id: tag.id,
                    name: tag.name,
                })
                .collect(),
            _ => Vec::new(),
        };
        Ok(ArticleListItem {
            article,
            author,
            tags,
        })
    }

    async fn external_link_change(
        &self,
        article_id: Uuid,
        article: &Article,
        request: &UpdateArticle,
    ) -> Result<ExternalLinkChange, AppError> {
        let Some(value) = &request.external_url else {
            return Ok(ExternalLinkChange::Unchanged);
        };
        let trimmed = value.as_deref().unwrap_or("").trim();
        if trimmed.is_empty() {
            return Ok(ExternalLinkChange::Clear);
        }
        if article.external_url.as_deref() == Some(trimmed) {
            return Ok(ExternalLinkChange::Unchanged);
        }
        match self
            .resolve_external_link(Some(article_id), trimmed)
            .await?
        {
            Some(page) if article.external_url.as_deref() == Some(page.url.as_str()) => {
                Ok(ExternalLinkChange::Unchanged)
            }
            Some(page) => Ok(ExternalLinkChange::Set(page)),
            None => Ok(ExternalLinkChange::Clear),
        }
    }

    async fn resolve_external_link(
        &self,
        exclude_id: Option<Uuid>,
        raw: &str,
    ) -> Result<Option<ExternalPage>, AppError> {
        let raw = raw.trim();
        if raw.is_empty() {
            return Ok(None);
        }
        let pages = self
            .external_pages
            .as_ref()
            .ok_or_else(|| AppError::InvalidInput("external blogs are unavailable".to_owned()))?;
        let page = pages.load(raw).await?;
        if self
            .articles
            .external_url_exists(&page.url, exclude_id)
            .await?
        {
            return Err(AppError::Conflict(
                "an external blog for this link already exists".to_owned(),
            ));
        }
        Ok(Some(page))
    }

    async fn cache_page_image(
        &self,
        article_id: Uuid,
        page: &ExternalPage,
    ) -> Result<Option<String>, AppError> {
        let Some(image) = &page.image else {
            return Ok(None);
        };
        let Some(images) = &self.image_cache else {
            return Ok(None);
        };
        let key = format!("external-blogs/{article_id}.{}", image.extension);
        Ok(Some(
            images
                .store(&key, &image.content_type, image.bytes.clone())
                .await?,
        ))
    }

    async fn unique_slug(&self, title: &str, exclude_id: Option<Uuid>) -> Result<String, AppError> {
        let base = generate_slug(title);
        if self.articles.slug_exists(&base, exclude_id).await? {
            Ok(format!("{base}-{}", &Uuid::new_v4().to_string()[..8]))
        } else {
            Ok(base)
        }
    }
}

fn new_article(
    slug: String,
    draft_title: String,
    draft_content: String,
    draft_image_url: String,
    author_id: Uuid,
    tag_ids: Vec<i64>,
    now: DateTime<Utc>,
) -> Article {
    Article {
        id: Uuid::new_v4(),
        slug,
        author_id,
        tag_ids: Some(tag_ids),
        draft_title,
        draft_content,
        draft_image_url,
        draft_embedding: Vec::new(),
        published_title: None,
        published_content: None,
        published_image_url: None,
        published_embedding: Vec::new(),
        published_at: None,
        current_draft_version_id: None,
        current_published_version_id: None,
        imagen_request_id: None,
        session_memory: None,
        created_at: Some(now),
        updated_at: Some(now),
        external_url: None,
        draft_upload_file_id: None,
        published_upload_file_id: None,
        draft_image: None,
        published_image: None,
        body_images: Vec::new(),
    }
}

impl ArticleService {
    async fn lookup_upload_id(&self, url: &str) -> Result<Option<Uuid>, AppError> {
        let Some(uploads) = &self.uploads else {
            return Ok(None);
        };
        Ok(uploads
            .find_by_public_url(url)
            .await?
            .map(|file| file.id))
    }
}

enum ExternalLinkChange {
    Unchanged,
    Clear,
    Set(ExternalPage),
}

fn should_replace_external_image(image_url: &str) -> bool {
    let image_url = image_url.trim();
    image_url.is_empty() || image_url.contains("/external-blogs/")
}

fn fill_missing_copy(title: &mut String, content: &mut String, page: &ExternalPage) {
    if title.trim().chars().count() < 3 {
        *title = fit_title(&page.title);
    }
    if content.trim().chars().count() < 10 {
        *content = fit_excerpt(&page.excerpt);
    }
}

fn fit_title(title: &str) -> String {
    let title = title.trim();
    if title.chars().count() <= 200 {
        return title.to_owned();
    }
    let mut shortened: String = title.chars().take(197).collect();
    shortened.push_str("...");
    shortened
}

fn fit_excerpt(excerpt: &str) -> String {
    let excerpt = excerpt.trim();
    if excerpt.chars().count() >= 10 {
        excerpt.to_owned()
    } else {
        format!("{excerpt} Read the original article.")
    }
}

fn timestamp_to_utc(timestamp: i64) -> Result<DateTime<Utc>, AppError> {
    let result = if timestamp > 1_000_000_000_000 {
        Utc.timestamp_millis_opt(timestamp)
    } else {
        Utc.timestamp_opt(timestamp, 0)
    };
    result
        .single()
        .ok_or_else(|| AppError::InvalidInput("published_at is out of range".to_owned()))
}

pub fn generate_slug(title: &str) -> String {
    let mut slug = String::with_capacity(title.len());
    let mut previous_dash = false;
    for character in title.to_lowercase().chars() {
        if character.is_ascii_alphanumeric() {
            slug.push(character);
            previous_dash = false;
        } else if (character == ' ' || character == '-') && !slug.is_empty() && !previous_dash {
            slug.push('-');
            previous_dash = true;
        }
    }
    while slug.ends_with('-') {
        slug.pop();
    }
    if slug.is_empty() {
        "untitled".to_owned()
    } else {
        slug
    }
}

fn validate_create(request: &CreateArticle) -> Result<(), AppError> {
    validate_article_fields(
        &request.title,
        &request.content,
        &request.image_url,
        &request.tags,
    )
}

fn validate_update(request: &UpdateArticle) -> Result<(), AppError> {
    validate_article_fields(
        &request.title,
        &request.content,
        &request.image_url,
        &request.tags,
    )
}

fn validate_article_fields(
    title: &str,
    content: &str,
    image_url: &str,
    tags: &[String],
) -> Result<(), AppError> {
    if !(3..=200).contains(&title.chars().count()) {
        return Err(AppError::InvalidInput(
            "title must contain between 3 and 200 characters".to_owned(),
        ));
    }
    if content.chars().count() < 10 {
        return Err(AppError::InvalidInput(
            "content must contain at least 10 characters".to_owned(),
        ));
    }
    if tags.len() > 10
        || tags
            .iter()
            .any(|tag| !(2..=30).contains(&tag.chars().count()))
    {
        return Err(AppError::InvalidInput(
            "tags must contain 2 to 30 characters and at most 10 entries".to_owned(),
        ));
    }
    if !image_url.is_empty() && Url::parse(image_url).is_err() {
        return Err(AppError::InvalidInput(
            "image_url must be a valid URL".to_owned(),
        ));
    }
    Ok(())
}
