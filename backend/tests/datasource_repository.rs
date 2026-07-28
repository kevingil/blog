use std::{env, error::Error, io};

use blog_backend::{
    core::{
        datasource::{CrawledContent, CrawledContentRepository, DataSource, DataSourceRepository},
        image::{IMAGE_STATUS_PENDING, ImageGeneration, ImageRepository},
        insight::{
            ContentTopicMatch, ContentTopicMatchRepository, Insight, InsightRepository,
            InsightTopic, InsightTopicRepository, UserInsightStatusRepository,
        },
        source::{Source, SourceListOptions, SourceRepository},
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            content_topic_match::DieselContentTopicMatchRepository,
            crawled_content::DieselCrawledContentRepository,
            data_source::DieselDataSourceRepository, image::DieselImageRepository,
            insight::DieselInsightRepository, insight_topic::DieselInsightTopicRepository,
            source::DieselSourceRepository, user_insight_status::DieselUserInsightStatusRepository,
        },
    },
    schema::{account, article, data_source, imagen_request, insight, insight_topic, organization},
};
use chrono::{Duration, Utc};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for datasource_repository; start the Docker PostgreSQL 17.4+pgvector service",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("data repository migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

async fn seed_owner(
    pool: &PgPool,
    user_id: Uuid,
    organization_id: Uuid,
    article_id: Uuid,
    suffix: &str,
) -> TestResult {
    let mut connection = pool.get().await?;
    diesel::insert_into(organization::table)
        .values((
            organization::id.eq(organization_id),
            organization::name.eq(format!("Data Repository {suffix}")),
            organization::slug.eq(format!("data-repository-{suffix}")),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(user_id),
            account::name.eq("Data Repository User"),
            account::email.eq(format!("{suffix}@example.com")),
            account::password_hash.eq("not-used"),
            account::role.eq("user"),
            account::organization_id.eq(Some(organization_id)),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(article::table)
        .values((
            article::id.eq(article_id),
            article::slug.eq(format!("data-repository-{suffix}")),
            article::author_id.eq(user_id),
            article::draft_title.eq(Some("Repository Article")),
            article::draft_content.eq(Some("Body")),
            article::draft_image_url.eq(Some("")),
        ))
        .execute(&mut connection)
        .await?;
    Ok(())
}

async fn cleanup(
    pool: &PgPool,
    user_id: Uuid,
    organization_id: Uuid,
    article_id: Uuid,
    image_id: Uuid,
) -> TestResult {
    let mut connection = pool.get().await?;
    diesel::delete(article::table.find(article_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(data_source::table.filter(data_source::organization_id.eq(organization_id)))
        .execute(&mut connection)
        .await?;
    diesel::delete(insight::table.filter(insight::organization_id.eq(organization_id)))
        .execute(&mut connection)
        .await?;
    diesel::delete(insight_topic::table.filter(insight_topic::organization_id.eq(organization_id)))
        .execute(&mut connection)
        .await?;
    diesel::delete(imagen_request::table.find(image_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(account::table.find(user_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(organization::table.find(organization_id))
        .execute(&mut connection)
        .await?;
    Ok(())
}

#[tokio::test]
async fn postgres_data_content_image_insight_and_source_repositories_preserve_contracts()
-> TestResult {
    let pool = test_pool()?;
    let run_id = Uuid::new_v4();
    let suffix = run_id.simple().to_string();
    let user_id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let article_id = Uuid::new_v4();
    seed_owner(&pool, user_id, organization_id, article_id, &suffix).await?;

    let data_sources = DieselDataSourceRepository::new(pool.clone());
    let contents = DieselCrawledContentRepository::new(pool.clone());
    let topics = DieselInsightTopicRepository::new(pool.clone());
    let insights = DieselInsightRepository::new(pool.clone());
    let matches = DieselContentTopicMatchRepository::new(pool.clone());
    let statuses = DieselUserInsightStatusRepository::new(pool.clone());
    let images = DieselImageRepository::new(pool.clone());
    let sources = DieselSourceRepository::new(pool.clone());
    let embedding = vec![0.125; 1536];

    let data_source_id = Uuid::new_v4();
    let mut data_source_value = DataSource {
        id: data_source_id,
        organization_id: Some(organization_id),
        user_id: Some(user_id),
        name: "Repository Source".to_owned(),
        url: format!("https://{suffix}.example.com"),
        feed_url: None,
        source_type: "blog".to_owned(),
        crawl_frequency: "daily".to_owned(),
        is_enabled: true,
        is_discovered: false,
        discovered_from_id: None,
        last_crawled_at: None,
        next_crawl_at: Some(Utc::now() - Duration::minutes(1)),
        crawl_status: "pending".to_owned(),
        error_message: None,
        content_count: 0,
        subscriber_count: 1,
        meta_data: None,
        created_at: None,
        updated_at: None,
    };
    data_sources.save(&mut data_source_value).await?;
    assert_eq!(
        data_sources
            .find_by_url(&data_source_value.url)
            .await?
            .map(|value| value.id),
        Some(data_source_id)
    );
    assert_eq!(
        data_sources
            .find_by_organization_id(organization_id)
            .await?
            .len(),
        1
    );
    assert_eq!(data_sources.find_by_user_id(user_id).await?.len(), 1);
    assert_eq!(data_sources.find_due_to_crawl(10).await?.len(), 1);
    data_sources
        .update_crawl_status(data_source_id, "success", None)
        .await?;
    assert!(
        data_sources
            .find_by_id(data_source_id)
            .await?
            .last_crawled_at
            .is_some()
    );
    data_sources
        .increment_content_count(data_source_id, 2)
        .await?;
    assert_eq!(
        data_sources.find_by_id(data_source_id).await?.content_count,
        2
    );

    let content_id = Uuid::new_v4();
    let mut crawled = CrawledContent {
        id: content_id,
        data_source_id,
        url: format!("https://{suffix}.example.com/post"),
        title: None,
        content: "Crawled body".to_owned(),
        summary: Some("Summary".to_owned()),
        author: None,
        published_at: None,
        embedding: Some(embedding.clone()),
        meta_data: None,
        created_at: None,
    };
    contents.save(&mut crawled).await?;
    assert!(contents.find_by_id(content_id).await?.title.is_none());
    let conflicting_caller_id = Uuid::new_v4();
    let mut duplicate_url = CrawledContent {
        id: conflicting_caller_id,
        summary: Some("Updated summary".to_owned()),
        ..crawled.clone()
    };
    contents.save(&mut duplicate_url).await?;
    assert_eq!(
        duplicate_url.id, content_id,
        "a conflict update must return the persisted row id to the caller"
    );
    assert_ne!(duplicate_url.id, conflicting_caller_id);
    assert_eq!(
        contents.find_by_id(content_id).await?.summary.as_deref(),
        Some("Updated summary")
    );
    assert_eq!(
        contents
            .find_by_data_source_id(data_source_id, 0, 20)
            .await?
            .1,
        1
    );
    assert_eq!(
        CrawledContentRepository::find_by_ids(&contents, &[content_id])
            .await?
            .len(),
        1
    );
    assert_eq!(
        CrawledContentRepository::search_similar(&contents, &embedding, 5)
            .await?
            .len(),
        1
    );
    assert_eq!(
        CrawledContentRepository::search_similar_by_org(&contents, organization_id, &embedding, 5,)
            .await?
            .len(),
        1
    );
    assert_eq!(
        CrawledContentRepository::find_recent_by_org(&contents, organization_id, 5)
            .await?
            .len(),
        1
    );
    assert_eq!(contents.count_by_data_source_id(data_source_id).await?, 1);

    let topic_id = Uuid::new_v4();
    let mut topic_value = InsightTopic {
        id: topic_id,
        organization_id: Some(organization_id),
        name: "Repository Topic".to_owned(),
        description: None,
        keywords: Some(Vec::new()),
        embedding: Some(embedding.clone()),
        is_auto_generated: false,
        content_count: 0,
        last_insight_at: None,
        color: None,
        icon: None,
        created_at: None,
        updated_at: None,
    };
    topics.save(&mut topic_value).await?;
    assert_eq!(topics.find_all().await?.len(), 1);
    let (similar_topics, scores) = topics.search_similar(&embedding, 5, 0.9).await?;
    assert_eq!(similar_topics.len(), 1);
    assert_eq!(scores.len(), 1);

    let mut content_matches = vec![ContentTopicMatch {
        id: Uuid::nil(),
        content_id,
        topic_id,
        similarity_score: 0.99,
        is_primary: true,
        created_at: None,
    }];
    matches.save_batch(&mut content_matches).await?;
    assert!(!content_matches[0].id.is_nil());
    assert_eq!(matches.count_by_topic_id(topic_id).await?, 1);
    assert_eq!(
        matches
            .find_primary_by_topic_id(topic_id, 0, 20)
            .await?
            .0
            .len(),
        1
    );

    let insight_id = Uuid::new_v4();
    let mut insight_value = Insight {
        id: insight_id,
        organization_id: Some(organization_id),
        topic_id: Some(topic_id),
        title: "Repository Insight".to_owned(),
        summary: "Summary".to_owned(),
        content: None,
        key_points: None,
        source_content_ids: vec![content_id],
        embedding: Some(embedding.clone()),
        generated_at: None,
        period_start: None,
        period_end: None,
        is_read: false,
        is_pinned: false,
        is_used_in_article: false,
        meta_data: None,
    };
    insights.save(&mut insight_value).await?;
    let loaded_insight = insights.find_by_id(insight_id).await?;
    assert!(loaded_insight.key_points == Some(Vec::new()) || loaded_insight.key_points.is_none());
    assert_eq!(loaded_insight.source_content_ids, vec![content_id]);
    assert_eq!(insights.search_similar(&embedding, 5).await?.len(), 1);
    assert_eq!(
        insights
            .search_similar_by_org(organization_id, &embedding, 5)
            .await?
            .len(),
        1
    );
    insights.mark_as_read(insight_id).await?;
    insights.toggle_pinned(insight_id).await?;
    insights.mark_as_used_in_article(insight_id).await?;
    let loaded_insight = insights.find_by_id(insight_id).await?;
    assert!(loaded_insight.is_read);
    assert!(loaded_insight.is_pinned);
    assert!(loaded_insight.is_used_in_article);

    statuses.mark_as_read(user_id, insight_id).await?;
    assert!(
        statuses
            .find_by_user_and_insight(user_id, insight_id)
            .await?
            .is_some_and(|status| status.is_read)
    );
    assert!(statuses.toggle_pinned(user_id, insight_id).await?);
    statuses
        .mark_as_used_in_article(user_id, insight_id)
        .await?;
    let status = statuses
        .find_by_user_and_insight(user_id, insight_id)
        .await?
        .ok_or_else(|| io::Error::other("status must exist after writes"))?;
    assert!(status.is_read);
    assert!(status.is_pinned);
    assert!(status.is_used_in_article);

    statuses.mark_as_read(user_id, insight_id).await?;
    let status = statuses
        .find_by_user_and_insight(user_id, insight_id)
        .await?
        .ok_or_else(|| io::Error::other("status must exist after mark as read"))?;
    assert!(
        status.is_pinned && status.is_used_in_article,
        "marking read must preserve independently-owned flags"
    );

    let first_toggle = statuses.clone();
    let second_toggle = statuses.clone();
    let (first_result, second_result) = tokio::join!(
        first_toggle.toggle_pinned(user_id, insight_id),
        second_toggle.toggle_pinned(user_id, insight_id)
    );
    let mut returned_states = vec![first_result?, second_result?];
    returned_states.sort_unstable();
    assert_eq!(returned_states, vec![false, true]);
    let status = statuses
        .find_by_user_and_insight(user_id, insight_id)
        .await?
        .ok_or_else(|| io::Error::other("status must exist after toggles"))?;
    assert!(
        status.is_pinned,
        "two atomic toggles must cancel each other"
    );
    assert!(status.is_read);
    assert!(status.is_used_in_article);
    assert_eq!(
        statuses
            .get_status_map_for_insights(user_id, &[insight_id])
            .await?
            .len(),
        1
    );

    let image_id = Uuid::new_v4();
    let mut image = ImageGeneration {
        id: image_id,
        prompt: "Image prompt".to_owned(),
        provider: "provider".to_owned(),
        model_name: "model".to_owned(),
        request_id: format!("request-{suffix}"),
        status: IMAGE_STATUS_PENDING.to_owned(),
        output_url: String::new(),
        file_index_id: None,
        error_message: String::new(),
        meta_data: None,
        created_at: None,
        completed_at: None,
    };
    images.save(&mut image).await?;
    assert_eq!(
        images.find_by_request_id(&image.request_id).await?.id,
        image_id
    );
    image.status = "completed".to_owned();
    image.output_url = "https://example.com/image.png".to_owned();
    images.update(&image).await?;
    assert_eq!(images.find_by_id(image_id).await?.status, "completed");

    let source_id = Uuid::new_v4();
    let mut source_value = Source {
        id: source_id,
        article_id,
        title: String::new(),
        content: "Source body".to_owned(),
        url: String::new(),
        source_type: "manual".to_owned(),
        embedding: Some(embedding.clone()),
        meta_data: None,
        created_at: None,
    };
    sources.save(&mut source_value).await?;
    let loaded_source = sources.find_by_id(source_id).await?;
    assert_eq!(loaded_source.title, "");
    assert_eq!(loaded_source.url, "");
    assert_eq!(sources.find_by_article_id(article_id).await?.len(), 1);
    assert_eq!(
        sources
            .list(SourceListOptions {
                page: 0,
                per_page: 0,
            })
            .await?
            .0[0]
            .article_title,
        "Repository Article"
    );
    assert_eq!(
        sources
            .search_similar(article_id, &embedding, 5)
            .await?
            .len(),
        1
    );

    cleanup(&pool, user_id, organization_id, article_id, image_id).await?;
    Ok(())
}
