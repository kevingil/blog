use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
    time::Duration,
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        article::{
            Article, ArticleListOptions, ArticleRepository, ArticleSearchOptions, ArticleService,
            ArticleVersion, CreateArticle, UpdateArticle, generate_slug,
        },
        auth::{Account, AccountId, AccountRepository},
        tag::{Tag, TagRepository},
    },
    error::AppError,
};
use chrono::Utc;
use uuid::Uuid;

#[derive(Default)]
struct MemoryArticles {
    articles: Mutex<HashMap<Uuid, Article>>,
    versions: Mutex<HashMap<Uuid, ArticleVersion>>,
}

#[async_trait]
impl ArticleRepository for MemoryArticles {
    async fn find_by_id(&self, id: Uuid) -> Result<Article, AppError> {
        self.articles
            .lock()
            .expect("articles lock")
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Article, AppError> {
        self.articles
            .lock()
            .expect("articles lock")
            .values()
            .find(|article| article.slug == slug)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list(&self, options: ArticleListOptions) -> Result<(Vec<Article>, i64), AppError> {
        let mut articles: Vec<_> = self
            .articles
            .lock()
            .expect("articles lock")
            .values()
            .filter(|article| !options.published_only || article.published_at.is_some())
            .filter(|article| {
                options.tag_id.is_none_or(|tag| {
                    article
                        .tag_ids
                        .as_ref()
                        .is_some_and(|ids| ids.contains(&tag))
                })
            })
            .cloned()
            .collect();
        articles.sort_by_key(|article| article.created_at);
        let total = i64::try_from(articles.len()).expect("test article count fits i64");
        Ok((articles, total))
    }

    async fn search(&self, options: ArticleSearchOptions) -> Result<(Vec<Article>, i64), AppError> {
        let query = options.query.to_lowercase();
        let articles: Vec<_> = self
            .articles
            .lock()
            .expect("articles lock")
            .values()
            .filter(|article| !options.published_only || article.published_at.is_some())
            .filter(|article| {
                article.draft_title.to_lowercase().contains(&query)
                    || article.draft_content.to_lowercase().contains(&query)
            })
            .cloned()
            .collect();
        let total = i64::try_from(articles.len()).expect("test article count fits i64");
        Ok((articles, total))
    }

    async fn search_by_embedding(
        &self,
        _embedding: &[f32],
        _limit: i64,
    ) -> Result<Vec<Article>, AppError> {
        Ok(Vec::new())
    }

    async fn save(&self, article: &mut Article) -> Result<(), AppError> {
        self.articles
            .lock()
            .expect("articles lock")
            .insert(article.id, article.clone());
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.articles
            .lock()
            .expect("articles lock")
            .remove(&id)
            .map(|_| ())
            .ok_or(AppError::NotFound)
    }

    async fn get_popular_tags(&self, limit: i64) -> Result<Vec<i64>, AppError> {
        let mut counts = HashMap::<i64, usize>::new();
        for tag in self
            .articles
            .lock()
            .expect("articles lock")
            .values()
            .flat_map(|article| article.tag_ids.clone().unwrap_or_default())
        {
            *counts.entry(tag).or_default() += 1;
        }
        let mut counts: Vec<_> = counts.into_iter().collect();
        counts.sort_by(|left, right| right.1.cmp(&left.1).then(left.0.cmp(&right.0)));
        Ok(counts
            .into_iter()
            .take(usize::try_from(limit).unwrap_or_default())
            .map(|(id, _)| id)
            .collect())
    }

    async fn slug_exists(&self, slug: &str, exclude_id: Option<Uuid>) -> Result<bool, AppError> {
        Ok(self
            .articles
            .lock()
            .expect("articles lock")
            .values()
            .any(|article| article.slug == slug && Some(article.id) != exclude_id))
    }

    async fn save_draft(&self, article: &mut Article) -> Result<(), AppError> {
        self.save(article).await
    }

    async fn publish(
        &self,
        article: &mut Article,
        published_at: Option<chrono::DateTime<Utc>>,
    ) -> Result<(), AppError> {
        article.published_title = Some(article.draft_title.clone());
        article.published_content = Some(article.draft_content.clone());
        article.published_image_url = Some(article.draft_image_url.clone());
        article.published_at = Some(published_at.unwrap_or_else(Utc::now));
        self.save(article).await
    }

    async fn unpublish(&self, article: &mut Article) -> Result<(), AppError> {
        article.published_at = None;
        self.save(article).await
    }

    async fn list_versions(&self, article_id: Uuid) -> Result<Vec<ArticleVersion>, AppError> {
        Ok(self
            .versions
            .lock()
            .expect("versions lock")
            .values()
            .filter(|version| version.article_id == article_id)
            .cloned()
            .collect())
    }

    async fn get_version(&self, version_id: Uuid) -> Result<ArticleVersion, AppError> {
        self.versions
            .lock()
            .expect("versions lock")
            .get(&version_id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn revert_to_version(&self, article_id: Uuid, version_id: Uuid) -> Result<(), AppError> {
        let version = self.get_version(version_id).await?;
        if version.article_id != article_id {
            return Err(AppError::NotFound);
        }
        let mut articles = self.articles.lock().expect("articles lock");
        let article = articles.get_mut(&article_id).ok_or(AppError::NotFound)?;
        article.draft_title = version.title;
        article.draft_content = version.content;
        article.draft_image_url = version.image_url;
        Ok(())
    }

    async fn create_draft_snapshot(&self, _article_id: Uuid) -> Result<Uuid, AppError> {
        Ok(Uuid::new_v4())
    }

    async fn update_draft_content(
        &self,
        article_id: Uuid,
        html_content: &str,
    ) -> Result<(), AppError> {
        let mut articles = self.articles.lock().expect("articles lock");
        let article = articles.get_mut(&article_id).ok_or(AppError::NotFound)?;
        article.draft_content = html_content.to_owned();
        Ok(())
    }

    async fn drain_background_tasks(&self) -> Result<(), AppError> {
        Ok(())
    }

    async fn shutdown_background_tasks(&self, _timeout: Duration) -> Result<(), AppError> {
        Ok(())
    }
}

#[derive(Default)]
struct MemoryAccounts {
    accounts: Mutex<HashMap<AccountId, Account>>,
}

#[async_trait]
impl AccountRepository for MemoryAccounts {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(self
            .accounts
            .lock()
            .expect("accounts lock")
            .get(&id)
            .cloned())
    }

    async fn find_by_email(&self, email: &str) -> Result<Option<Account>, AppError> {
        Ok(self
            .accounts
            .lock()
            .expect("accounts lock")
            .values()
            .find(|account| account.email == email)
            .cloned())
    }

    async fn create(&self, account: &Account) -> Result<(), AppError> {
        self.accounts
            .lock()
            .expect("accounts lock")
            .insert(account.id, account.clone());
        Ok(())
    }

    async fn update_identity(
        &self,
        _id: AccountId,
        _name: &str,
        _email: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn update_password_if_current(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
        _new_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn delete_if_password_hash(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }
}

#[derive(Default)]
struct MemoryTags {
    tags: Mutex<HashMap<i32, Tag>>,
}

#[async_trait]
impl TagRepository for MemoryTags {
    async fn find_by_id(&self, id: i32) -> Result<Tag, AppError> {
        self.tags
            .lock()
            .expect("tags lock")
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_name(&self, name: &str) -> Result<Tag, AppError> {
        self.tags
            .lock()
            .expect("tags lock")
            .values()
            .find(|tag| tag.name.eq_ignore_ascii_case(name))
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_ids(&self, ids: &[i64]) -> Result<Vec<Tag>, AppError> {
        let tags = self.tags.lock().expect("tags lock");
        Ok(ids
            .iter()
            .filter_map(|id| {
                i32::try_from(*id)
                    .ok()
                    .and_then(|id| tags.get(&id).cloned())
            })
            .collect())
    }

    async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError> {
        let mut tags = self.tags.lock().expect("tags lock");
        let mut ids = Vec::new();
        for name in names
            .iter()
            .map(|name| name.trim())
            .filter(|name| !name.is_empty())
        {
            let existing = tags
                .values()
                .find(|tag| tag.name.eq_ignore_ascii_case(name))
                .map(|tag| tag.id);
            let id = existing.unwrap_or_else(|| {
                let id = i32::try_from(tags.len() + 1).expect("test tag count fits i32");
                tags.insert(
                    id,
                    Tag {
                        id,
                        name: name.to_owned(),
                        created_at: Some(Utc::now()),
                    },
                );
                id
            });
            ids.push(i64::from(id));
        }
        Ok(ids)
    }

    async fn list(&self) -> Result<Vec<Tag>, AppError> {
        Ok(self
            .tags
            .lock()
            .expect("tags lock")
            .values()
            .cloned()
            .collect())
    }

    async fn save(&self, tag: &mut Tag) -> Result<(), AppError> {
        self.tags
            .lock()
            .expect("tags lock")
            .insert(tag.id, tag.clone());
        Ok(())
    }

    async fn delete(&self, id: i32) -> Result<(), AppError> {
        self.tags
            .lock()
            .expect("tags lock")
            .remove(&id)
            .map(|_| ())
            .ok_or(AppError::NotFound)
    }

    async fn is_used(&self, _id: i32) -> Result<bool, AppError> {
        Ok(false)
    }
}

fn account(id: Uuid, name: &str) -> Account {
    Account {
        id: AccountId(id),
        name: name.to_owned(),
        email: format!("{}@example.com", name.to_lowercase()),
        password_hash: String::new(),
        role: "admin".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: None,
    }
}

fn article(id: Uuid, author_id: Uuid, title: &str, tags: Vec<i64>) -> Article {
    Article {
        id,
        slug: generate_slug(title),
        author_id,
        tag_ids: Some(tags),
        draft_title: title.to_owned(),
        draft_content: "Complete article content".to_owned(),
        draft_image_url: String::new(),
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
        created_at: Some(Utc::now()),
        updated_at: Some(Utc::now()),
    }
}

fn fixture() -> (
    Arc<MemoryArticles>,
    Arc<MemoryAccounts>,
    Arc<MemoryTags>,
    ArticleService,
) {
    let articles = Arc::new(MemoryArticles::default());
    let accounts = Arc::new(MemoryAccounts::default());
    let tags = Arc::new(MemoryTags::default());
    let service = ArticleService::new(articles.clone(), accounts.clone(), tags.clone());
    (articles, accounts, tags, service)
}

#[tokio::test]
async fn get_by_id_and_slug_enrich_articles_with_author_and_tags() {
    let (articles, accounts, tags, service) = fixture();
    let article_id = Uuid::new_v4();
    let author_id = Uuid::new_v4();
    accounts
        .accounts
        .lock()
        .expect("accounts lock")
        .insert(AccountId(author_id), account(author_id, "Test Author"));
    tags.tags.lock().expect("tags lock").extend([
        (
            1,
            Tag {
                id: 1,
                name: "rust".to_owned(),
                created_at: None,
            },
        ),
        (
            2,
            Tag {
                id: 2,
                name: "testing".to_owned(),
                created_at: None,
            },
        ),
    ]);
    articles.articles.lock().expect("articles lock").insert(
        article_id,
        article(article_id, author_id, "Test Article", vec![1, 2]),
    );

    let by_id = service.get_by_id(article_id).await.expect("article by id");
    let by_slug = service
        .get_by_slug("test-article")
        .await
        .expect("article by slug");
    assert_eq!(by_id.author.name, "Test Author");
    assert_eq!(by_id.tags.len(), 2);
    assert_eq!(by_slug, by_id);
    assert!(matches!(
        service.get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn create_validates_fields_creates_tags_and_uses_a_unique_slug() {
    let (articles, accounts, tags, service) = fixture();
    let author_id = Uuid::new_v4();
    accounts
        .accounts
        .lock()
        .expect("accounts lock")
        .insert(AccountId(author_id), account(author_id, "Author"));
    let existing_id = Uuid::new_v4();
    articles.articles.lock().expect("articles lock").insert(
        existing_id,
        article(existing_id, author_id, "Duplicate Title", Vec::new()),
    );

    let created = service
        .create(CreateArticle {
            title: "Duplicate Title".to_owned(),
            content: "This is the content of the new article".to_owned(),
            image_url: String::new(),
            tags: vec!["rust".to_owned(), "testing".to_owned()],
            publish: false,
            author_id,
        })
        .await
        .expect("create article");
    assert!(created.article.slug.starts_with("duplicate-title-"));
    assert_ne!(created.article.slug, "duplicate-title");
    assert_eq!(created.tags.len(), 2);
    assert_eq!(tags.tags.lock().expect("tags lock").len(), 2);

    let error = service
        .create(CreateArticle {
            title: "x".to_owned(),
            content: "short".to_owned(),
            image_url: String::new(),
            tags: Vec::new(),
            publish: false,
            author_id,
        })
        .await
        .expect_err("invalid fields");
    assert!(matches!(error, AppError::InvalidInput(_)));
}

#[tokio::test]
async fn generated_content_updates_the_existing_draft_shell() {
    let (articles, _accounts, _tags, service) = fixture();
    let author_id = Uuid::new_v4();
    let shell = service
        .create_draft_shell("Generated Article", author_id)
        .await
        .expect("create draft shell");

    service
        .update_generated_draft(shell.id, "<p>Generated body</p>")
        .await
        .expect("persist generated content");

    assert_eq!(
        articles
            .find_by_id(shell.id)
            .await
            .expect("stored draft")
            .draft_content,
        "<p>Generated body</p>"
    );
    assert!(matches!(
        service.update_generated_draft(shell.id, "  ").await,
        Err(AppError::InvalidInput(_))
    ));
}

#[tokio::test]
async fn generated_image_is_attached_to_the_article_draft() {
    let (articles, _accounts, _tags, service) = fixture();
    let article_id = Uuid::new_v4();
    let author_id = Uuid::new_v4();
    let image_id = Uuid::new_v4();
    articles.articles.lock().expect("articles lock").insert(
        article_id,
        article(article_id, author_id, "Image Article", Vec::new()),
    );

    service
        .apply_generated_image(article_id, image_id, "https://example.com/generated.png")
        .await
        .expect("attach generated image");

    let stored = articles
        .find_by_id(article_id)
        .await
        .expect("stored article");
    assert_eq!(stored.imagen_request_id, Some(image_id));
    assert_eq!(stored.draft_image_url, "https://example.com/generated.png");
    assert!(matches!(
        service
            .apply_generated_image(article_id, image_id, " ")
            .await,
        Err(AppError::InvalidInput(_))
    ));
}

#[tokio::test]
async fn list_search_popular_and_recommended_preserve_go_contracts() {
    let (articles, accounts, tags, service) = fixture();
    let author_id = Uuid::new_v4();
    accounts
        .accounts
        .lock()
        .expect("accounts lock")
        .insert(AccountId(author_id), account(author_id, "Author"));
    tags.tags.lock().expect("tags lock").insert(
        1,
        Tag {
            id: 1,
            name: "rust".to_owned(),
            created_at: None,
        },
    );
    let current_id = Uuid::new_v4();
    let mut current = article(current_id, author_id, "Current Article", vec![1]);
    current.published_at = Some(Utc::now());
    let recommended_id = Uuid::new_v4();
    let mut recommended = article(recommended_id, author_id, "Recommended Rust", vec![1]);
    recommended.published_at = Some(Utc::now());
    articles
        .articles
        .lock()
        .expect("articles lock")
        .extend([(current_id, current), (recommended_id, recommended)]);

    let listed = service
        .list(1, "rust", "published", 6, "", "")
        .await
        .expect("list");
    assert_eq!(listed.articles.len(), 2);
    assert_eq!(listed.total_pages, 1);
    assert!(!listed.include_drafts);
    assert_eq!(
        service
            .search("rust", 1, "published")
            .await
            .expect("search")
            .articles
            .len(),
        1
    );
    assert_eq!(
        service.get_popular_tags().await.expect("popular tags"),
        vec!["rust"]
    );
    let results = service
        .get_recommended(current_id)
        .await
        .expect("recommended");
    assert_eq!(results.len(), 1);
    assert_eq!(results[0].id, recommended_id);
    assert_eq!(results[0].author.as_deref(), Some("Author"));
}

#[tokio::test]
async fn update_publish_unpublish_and_delete_preserve_lifecycle() {
    let (articles, accounts, _tags, service) = fixture();
    let author_id = Uuid::new_v4();
    accounts
        .accounts
        .lock()
        .expect("accounts lock")
        .insert(AccountId(author_id), account(author_id, "Author"));
    let id = Uuid::new_v4();
    articles
        .articles
        .lock()
        .expect("articles lock")
        .insert(id, article(id, author_id, "Initial Title", Vec::new()));

    let updated = service
        .update(
            id,
            UpdateArticle {
                title: "Updated Title".to_owned(),
                content: "Updated article content".to_owned(),
                image_url: "https://example.com/image.jpg".to_owned(),
                tags: vec!["rust".to_owned()],
                published_at: Some(1_700_000_000),
            },
        )
        .await
        .expect("update");
    assert_eq!(updated.article.slug, "updated-title");
    assert!(updated.article.published_at.is_some());

    let published_at = Utc::now();
    let published = service
        .publish(id, Some(published_at))
        .await
        .expect("publish");
    assert_eq!(published.article.published_at, Some(published_at));
    service.unpublish(id).await.expect("unpublish");
    assert!(matches!(
        service.unpublish(id).await,
        Err(AppError::InvalidInput(_))
    ));
    service.delete(id).await.expect("delete");
    assert!(matches!(service.delete(id).await, Err(AppError::NotFound)));
}

#[tokio::test]
async fn version_listing_lookup_and_revert_are_scoped_to_the_article() {
    let (articles, accounts, _tags, service) = fixture();
    let author_id = Uuid::new_v4();
    accounts
        .accounts
        .lock()
        .expect("accounts lock")
        .insert(AccountId(author_id), account(author_id, "Author"));
    let article_id = Uuid::new_v4();
    articles.articles.lock().expect("articles lock").insert(
        article_id,
        article(article_id, author_id, "Current Title", Vec::new()),
    );
    let version_id = Uuid::new_v4();
    articles.versions.lock().expect("versions lock").insert(
        version_id,
        ArticleVersion {
            id: version_id,
            article_id,
            version_number: 1,
            status: "draft".to_owned(),
            title: "Old Title".to_owned(),
            content: "Old version content".to_owned(),
            image_url: String::new(),
            embedding: Vec::new(),
            edited_by: None,
            created_at: Some(Utc::now()),
        },
    );

    let versions = service
        .list_versions(article_id)
        .await
        .expect("list versions");
    assert_eq!(versions.total, 1);
    assert_eq!(
        service
            .get_version(version_id)
            .await
            .expect("get version")
            .version_number,
        1
    );
    let reverted = service
        .revert_to_version(article_id, version_id)
        .await
        .expect("revert");
    assert_eq!(reverted.article.draft_title, "Old Title");
}

#[test]
fn slug_generation_matches_go_normalization() {
    assert_eq!(generate_slug("Hello,   WORLD!"), "hello-world");
    assert_eq!(generate_slug("---"), "untitled");
    assert_eq!(generate_slug("A--B"), "a-b");
    assert_eq!(generate_slug("Café"), "caf");
}

#[test]
fn article_openapi_exposes_all_fifteen_stable_operations() {
    let document = serde_json::to_value(blog_backend::openapi::document()).expect("OpenAPI JSON");
    let operations = [
        ("get", "/blog/articles/search", "searchArticles"),
        ("get", "/blog/articles/{slug}", "getArticleData"),
        (
            "get",
            "/blog/articles/{id}/recommended",
            "getRecommendedArticles",
        ),
        ("get", "/blog/articles", "getArticles"),
        ("get", "/blog/tags/popular", "getPopularTags"),
        ("post", "/blog/generate", "generateArticle"),
        ("put", "/blog/{id}/update", "updateArticleWithContext"),
        ("post", "/blog/articles/{slug}/update", "updateArticle"),
        ("post", "/blog/articles", "createArticle"),
        ("delete", "/blog/articles/{slug}", "deleteArticle"),
        ("post", "/blog/articles/{slug}/publish", "publishArticle"),
        (
            "post",
            "/blog/articles/{slug}/unpublish",
            "unpublishArticle",
        ),
        (
            "get",
            "/blog/articles/{slug}/versions",
            "listArticleVersions",
        ),
        (
            "get",
            "/blog/articles/versions/{versionId}",
            "getArticleVersion",
        ),
        (
            "post",
            "/blog/articles/{slug}/revert/{versionId}",
            "revertArticleToVersion",
        ),
    ];

    for (method, path, operation_id) in operations {
        assert_eq!(
            document["paths"][path][method]["operationId"], operation_id,
            "{method} {path}"
        );
    }
    assert_eq!(
        document["paths"]["/blog/generate"]["post"]["security"][0]["bearerAuth"],
        serde_json::json!([])
    );
}
