use std::{collections::BTreeMap, env, error::Error, io, time::Duration as StdDuration};

use blog_backend::{
    core::article::{
        Article, ArticleListOptions, ArticleRepository, ArticleSearchOptions, ArticleVersion,
    },
    database::{
        pool::{PgPool, create_pool},
        repository::article::DieselArticleRepository,
    },
    error::AppError,
    schema::{account, article},
};
use chrono::{Duration, Utc};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::{AsyncConnection, RunQueryDsl};
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use serde_json::json;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the article_repository target; start the Docker PostgreSQL 17.4+pgvector service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("article test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

fn article_fixture(
    id: Uuid,
    author_id: Uuid,
    slug: String,
    tag_ids: Vec<i64>,
    embedding_value: f32,
) -> Article {
    let mut memory = BTreeMap::new();
    memory.insert("phase".to_owned(), json!("article-repository-test"));
    Article {
        id,
        slug,
        author_id,
        tag_ids: Some(tag_ids),
        draft_title: "Draft Alpha".to_owned(),
        draft_content: "Rust repository body".to_owned(),
        draft_image_url: "https://example.com/draft.png".to_owned(),
        draft_embedding: vec![embedding_value; 1536],
        published_title: None,
        published_content: None,
        published_image_url: None,
        published_embedding: Vec::new(),
        published_at: None,
        current_draft_version_id: None,
        current_published_version_id: None,
        imagen_request_id: None,
        session_memory: Some(memory),
        created_at: None,
        updated_at: None,
    }
}

fn embedding_fixture(run_id: Uuid) -> Vec<f32> {
    let bytes = run_id.as_bytes();
    (0..1536)
        .map(|index| f32::from(bytes[index % bytes.len()]) / 255.0)
        .collect()
}

async fn cleanup(pool: &PgPool, article_ids: &[Uuid], author_id: Uuid) -> TestResult {
    let mut connection = pool.get().await?;
    diesel::delete(article::table.filter(article::id.eq_any(article_ids)))
        .execute(&mut connection)
        .await?;
    diesel::delete(account::table.find(author_id))
        .execute(&mut connection)
        .await?;
    Ok(())
}

async fn exercise_repository(
    pool: &PgPool,
    repository: &DieselArticleRepository,
    author_id: Uuid,
    article_ids: &mut [Uuid; 4],
    slug_prefix: &str,
    run_id: Uuid,
) -> TestResult {
    let now = Utc::now();
    let popular_tag = i64::from(i32::MAX) - 101;
    let less_popular_tag = i64::from(i32::MAX) - 102;

    let mut first = article_fixture(
        article_ids[0],
        author_id,
        format!("{slug_prefix}-first"),
        vec![popular_tag, popular_tag, less_popular_tag],
        0.0,
    );
    first.published_title = Some(format!("Published Needle {slug_prefix}"));
    first.published_content = Some("Public first content".to_owned());
    first.published_image_url = Some("https://example.com/first.png".to_owned());
    first.draft_embedding = embedding_fixture(run_id);
    first.published_embedding = first.draft_embedding.clone();
    first.published_at = Some(now - Duration::days(2));
    first.created_at = Some(now - Duration::days(4));
    first.draft_title = format!("Draft Alpha {slug_prefix}");
    repository.save(&mut first).await?;

    let mut second = article_fixture(
        article_ids[1],
        author_id,
        format!("{slug_prefix}-second"),
        vec![popular_tag],
        1.0,
    );
    second.draft_title = "Second Draft".to_owned();
    second.draft_content = "Second searchable haystack".to_owned();
    second.published_title = Some("Second Published".to_owned());
    second.published_content = Some("Second public content".to_owned());
    second.published_image_url = Some("https://example.com/second.png".to_owned());
    second.published_embedding = second.draft_embedding.clone();
    second.published_at = Some(now - Duration::days(1));
    second.created_at = Some(now - Duration::days(3));
    repository.save(&mut second).await?;

    let mut third = article_fixture(
        article_ids[2],
        author_id,
        format!("{slug_prefix}-third"),
        Vec::new(),
        2.0,
    );
    third.draft_title = "Unpublished Draft".to_owned();
    third.draft_embedding.clear();
    repository.save(&mut third).await?;

    let mut generated_id = article_fixture(
        Uuid::nil(),
        author_id,
        format!("{slug_prefix}-generated"),
        Vec::new(),
        3.0,
    );
    generated_id.draft_embedding.clear();
    generated_id.tag_ids = None;
    repository.save(&mut generated_id).await?;
    assert!(!generated_id.id.is_nil());
    article_ids[3] = generated_id.id;
    assert!(
        repository
            .find_by_id(generated_id.id)
            .await?
            .tag_ids
            .is_none()
    );
    assert_eq!(
        repository.find_by_id(third.id).await?.tag_ids,
        Some(Vec::new())
    );

    let loaded = repository.find_by_id(first.id).await?;
    assert_eq!(loaded.slug, first.slug);
    assert_eq!(loaded.tag_ids, first.tag_ids);
    assert_eq!(loaded.session_memory, first.session_memory);
    assert_eq!(loaded.draft_embedding.len(), 1536);
    assert!(loaded.is_published());
    assert_eq!(
        loaded.title(false),
        format!("Published Needle {slug_prefix}")
    );
    assert_eq!(loaded.content(true), "Rust repository body");
    assert_eq!(
        repository.find_by_slug(&first.slug).await?.id,
        article_ids[0]
    );
    assert!(matches!(
        repository.find_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));

    assert!(repository.slug_exists(&first.slug, None).await?);
    assert!(!repository.slug_exists(&first.slug, Some(first.id)).await?);

    let mut connection = pool.get().await?;
    diesel::update(article::table.find(first.id))
        .set(article::session_memory.eq::<Option<serde_json::Value>>(None))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert!(
        repository
            .find_by_id(first.id)
            .await?
            .session_memory
            .is_none()
    );
    let mut connection = pool.get().await?;
    diesel::update(article::table.find(first.id))
        .set(article::session_memory.eq(Some(json!({}))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert_eq!(
        repository.find_by_id(first.id).await?.session_memory,
        Some(BTreeMap::new())
    );
    let mut connection = pool.get().await?;
    diesel::update(article::table.find(first.id))
        .set(article::session_memory.eq(Some(json!([]))))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert!(
        repository
            .find_by_id(first.id)
            .await?
            .session_memory
            .is_none()
    );

    let (default_order, default_total) = repository
        .list(ArticleListOptions {
            page: 1,
            per_page: 10,
            author_id: Some(author_id),
            ..ArticleListOptions::default()
        })
        .await?;
    assert_eq!(default_total, 4);
    assert_eq!(default_order[0].id, second.id);
    assert_eq!(default_order[1].id, first.id);
    assert!(
        default_order[2..]
            .iter()
            .all(|value| value.published_at.is_none())
    );
    let (fallback_order, _) = repository
        .list(ArticleListOptions {
            page: 1,
            per_page: 10,
            author_id: Some(author_id),
            sort_by: "title; DROP TABLE article".to_owned(),
            sort_order: "sideways".to_owned(),
            ..ArticleListOptions::default()
        })
        .await?;
    assert_eq!(
        fallback_order
            .iter()
            .map(|value| value.id)
            .collect::<Vec<_>>(),
        default_order
            .iter()
            .map(|value| value.id)
            .collect::<Vec<_>>()
    );

    let (published, total) = repository
        .list(ArticleListOptions {
            page: 1,
            per_page: 10,
            published_only: true,
            author_id: Some(author_id),
            tag_id: Some(popular_tag),
            sort_by: String::new(),
            sort_order: String::new(),
        })
        .await?;
    assert_eq!(total, 2);
    assert_eq!(
        published.iter().map(|value| value.id).collect::<Vec<_>>(),
        vec![second.id, first.id]
    );

    let (created, total) = repository
        .list(ArticleListOptions {
            page: 2,
            per_page: 1,
            author_id: Some(author_id),
            sort_by: "created_at".to_owned(),
            sort_order: "asc".to_owned(),
            ..ArticleListOptions::default()
        })
        .await?;
    assert_eq!(total, 4);
    assert_eq!(created.len(), 1);
    assert_eq!(created[0].id, second.id);

    for page in [0, -1] {
        let (gorm_first_page, _) = repository
            .list(ArticleListOptions {
                page,
                per_page: 1,
                author_id: Some(author_id),
                sort_by: "created_at".to_owned(),
                sort_order: "asc".to_owned(),
                ..ArticleListOptions::default()
            })
            .await?;
        assert_eq!(gorm_first_page[0].id, first.id);
    }
    let (zero_limit, zero_limit_total) = repository
        .list(ArticleListOptions {
            page: 1,
            per_page: 0,
            author_id: Some(author_id),
            ..ArticleListOptions::default()
        })
        .await?;
    assert!(zero_limit.is_empty());
    assert_eq!(zero_limit_total, 4);
    let (cancelled_limit, cancelled_limit_total) = repository
        .list(ArticleListOptions {
            page: 1,
            per_page: -1,
            author_id: Some(author_id),
            ..ArticleListOptions::default()
        })
        .await?;
    assert_eq!(cancelled_limit.len(), 4);
    assert_eq!(cancelled_limit_total, 4);

    let (search_results, search_total) = repository
        .search(ArticleSearchOptions {
            query: format!("needle {slug_prefix}"),
            page: 1,
            per_page: 10,
            published_only: true,
        })
        .await?;
    assert_eq!(search_total, 1);
    assert_eq!(search_results[0].id, first.id);

    let case_query = format!("dRaFt aLpHa {slug_prefix}");
    let (case_results, case_total) = repository
        .search(ArticleSearchOptions {
            query: case_query,
            page: 1,
            per_page: 10,
            published_only: false,
        })
        .await?;
    assert_eq!(case_total, 1);
    assert_eq!(case_results[0].id, first.id);
    let mut wildcard_query = format!("draft alpha {slug_prefix}");
    wildcard_query.pop();
    wildcard_query.push('_');
    let (wildcard_results, wildcard_total) = repository
        .search(ArticleSearchOptions {
            query: wildcard_query,
            page: 1,
            per_page: 10,
            published_only: false,
        })
        .await?;
    assert_eq!(wildcard_total, 1);
    assert_eq!(wildcard_results[0].id, first.id);

    let nearest = repository
        .search_by_embedding(&embedding_fixture(run_id), 2)
        .await?;
    assert_eq!(
        nearest.as_slice().first().map(|value| value.id),
        Some(first.id)
    );
    assert!(
        repository
            .search_by_embedding(&vec![0.0; 1536], 0)
            .await?
            .is_empty()
    );
    assert!(matches!(
        repository.search_by_embedding(&vec![0.0; 1536], -1).await,
        Err(AppError::Database)
    ));

    let popular_tags = repository.get_popular_tags(10_000).await?;
    assert!(repository.get_popular_tags(0).await?.is_empty());
    assert!(matches!(
        repository.get_popular_tags(-1).await,
        Err(AppError::Database)
    ));
    let popular_position = popular_tags
        .iter()
        .position(|tag_id| *tag_id == popular_tag)
        .ok_or_else(|| io::Error::other("popular fixture tag missing"))?;
    let less_popular_position = popular_tags
        .iter()
        .position(|tag_id| *tag_id == less_popular_tag)
        .ok_or_else(|| io::Error::other("less-popular fixture tag missing"))?;
    assert!(popular_position < less_popular_position);

    let original_embedding = first.draft_embedding.clone();
    first.draft_title = "Updated through Save".to_owned();
    first.draft_embedding.clear();
    first.published_embedding.clear();
    repository.save(&mut first).await?;
    let saved = repository.find_by_id(first.id).await?;
    assert_eq!(saved.draft_title, "Updated through Save");
    assert_eq!(saved.draft_embedding, original_embedding);
    assert_eq!(saved.published_embedding, original_embedding);

    let mut overflow = article_fixture(
        Uuid::new_v4(),
        author_id,
        format!("{slug_prefix}-overflow"),
        vec![i64::from(i32::MAX) + 1],
        0.0,
    );
    overflow.draft_embedding.clear();
    assert!(matches!(
        repository.save(&mut overflow).await,
        Err(AppError::InvalidInput(message)) if message.contains("does not fit PostgreSQL INTEGER")
    ));
    assert!(matches!(
        repository
            .list(ArticleListOptions {
                page: 1,
                per_page: 10,
                tag_id: Some(i64::from(i32::MIN) - 1),
                ..ArticleListOptions::default()
            })
            .await,
        Err(AppError::InvalidInput(message)) if message.contains("does not fit PostgreSQL INTEGER")
    ));

    let before_failed_mutation = repository.find_by_id(first.id).await?;
    let mut invalid_draft = before_failed_mutation.clone();
    invalid_draft.draft_title = "x".repeat(501);
    assert!(matches!(
        repository.save_draft(&mut invalid_draft).await,
        Err(AppError::Database)
    ));
    repository.drain_background_tasks().await?;
    let after_failed_mutation = repository.find_by_id(first.id).await?;
    assert_eq!(
        (
            after_failed_mutation.draft_title,
            after_failed_mutation.draft_content,
            after_failed_mutation.updated_at,
        ),
        (
            before_failed_mutation.draft_title,
            before_failed_mutation.draft_content,
            before_failed_mutation.updated_at,
        )
    );

    first.draft_title = "Draft version one".to_owned();
    first.draft_content = "draft one body".to_owned();
    first.draft_image_url = "draft-one.png".to_owned();
    first.draft_embedding = original_embedding.clone();
    repository.save_draft(&mut first).await?;
    assert_eq!(
        repository.find_by_id(first.id).await?.draft_content,
        "draft one body"
    );
    repository.drain_background_tasks().await?;
    let versions_after_draft = repository.list_versions(first.id).await?;
    assert_eq!(versions_after_draft.len(), 1);
    assert_eq!(versions_after_draft[0].status, "draft");
    assert_eq!(versions_after_draft[0].edited_by, Some(author_id));
    assert_eq!(
        repository
            .find_by_id(first.id)
            .await?
            .current_draft_version_id,
        Some(versions_after_draft[0].id)
    );

    let explicit_publish_time = now - Duration::hours(6);
    repository
        .publish(&mut first, Some(explicit_publish_time))
        .await?;
    assert_eq!(first.published_at, Some(explicit_publish_time));
    repository.drain_background_tasks().await?;
    let versions_after_publish = repository.list_versions(first.id).await?;
    assert_eq!(
        versions_after_publish
            .iter()
            .map(|version| version.version_number)
            .collect::<Vec<_>>(),
        vec![2, 1]
    );
    assert_eq!(versions_after_publish[0].status, "published");
    assert_eq!(
        repository
            .find_by_id(first.id)
            .await?
            .current_published_version_id,
        Some(versions_after_publish[0].id)
    );
    let published_version = repository.get_version(versions_after_publish[0].id).await?;
    assert_eq!(published_version.title, "Draft version one");

    repository.unpublish(&mut first).await?;
    let unpublished = repository.find_by_id(first.id).await?;
    assert!(!unpublished.is_published());
    assert!(unpublished.current_published_version_id.is_none());

    first.draft_title = "Snapshot title".to_owned();
    first.draft_content = "Snapshot content".to_owned();
    first.draft_image_url = "snapshot.png".to_owned();
    repository.save_draft(&mut first).await?;
    repository.drain_background_tasks().await?;
    let snapshot_id = repository.create_draft_snapshot(first.id).await?;
    assert_eq!(
        repository.get_version(snapshot_id).await?.edited_by,
        Some(author_id)
    );

    let version_count = repository.list_versions(first.id).await?.len();
    repository
        .update_draft_content(first.id, "agent turn content")
        .await?;
    assert_eq!(
        repository.list_versions(first.id).await?.len(),
        version_count
    );
    assert_eq!(
        repository.find_by_id(first.id).await?.draft_content,
        "agent turn content"
    );

    repository
        .revert_to_version(first.id, published_version.id)
        .await?;
    repository.drain_background_tasks().await?;
    let reverted = repository.find_by_id(first.id).await?;
    assert_eq!(reverted.draft_title, published_version.title);
    assert_eq!(reverted.draft_content, published_version.content);
    assert_eq!(reverted.draft_embedding, published_version.embedding);

    assert!(matches!(
        repository
            .revert_to_version(second.id, published_version.id)
            .await,
        Err(AppError::InvalidInput(message)) if message == "Version does not belong to this article"
    ));
    assert!(matches!(
        repository.get_version(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));

    let empty_embedding_snapshot = repository.create_draft_snapshot(third.id).await?;
    assert!(
        repository
            .get_version(empty_embedding_snapshot)
            .await?
            .embedding
            .is_empty()
    );
    let mut third_with_embedding = repository.find_by_id(third.id).await?;
    third_with_embedding.draft_embedding = vec![4.0; 1536];
    repository.save(&mut third_with_embedding).await?;
    repository
        .revert_to_version(third.id, empty_embedding_snapshot)
        .await?;
    repository.drain_background_tasks().await?;
    assert!(
        repository
            .find_by_id(third.id)
            .await?
            .draft_embedding
            .is_empty()
    );

    repository.delete(third.id).await?;
    assert!(matches!(
        repository.delete(third.id).await,
        Err(AppError::NotFound)
    ));

    let nonexistent = article_fixture(
        Uuid::new_v4(),
        author_id,
        format!("{slug_prefix}-nonexistent"),
        Vec::new(),
        0.0,
    );
    let mut nonexistent = nonexistent;
    nonexistent.draft_embedding.clear();
    repository.save_draft(&mut nonexistent).await?;
    let mut second_after_failure = repository.find_by_id(second.id).await?;
    second_after_failure.draft_title = "Version after delayed failure".to_owned();
    repository.save_draft(&mut second_after_failure).await?;
    assert!(matches!(
        repository.drain_background_tasks().await,
        Err(AppError::NotFound)
    ));
    repository.drain_background_tasks().await?;
    assert_eq!(repository.list_versions(second.id).await?.len(), 1);

    let base_version_count = repository.list_versions(second.id).await?.len();
    let mut concurrent_one = repository.find_by_id(second.id).await?;
    concurrent_one.draft_title = "Concurrent one".to_owned();
    let mut concurrent_two = concurrent_one.clone();
    concurrent_two.draft_title = "Concurrent two".to_owned();
    let mut concurrent_three = concurrent_one.clone();
    concurrent_three.draft_title = "Concurrent three".to_owned();
    let (one, two, three) = tokio::join!(
        repository.save_draft(&mut concurrent_one),
        repository.save_draft(&mut concurrent_two),
        repository.save_draft(&mut concurrent_three),
    );
    one?;
    two?;
    three?;
    repository.drain_background_tasks().await?;
    let concurrent_versions = repository.list_versions(second.id).await?;
    assert_eq!(concurrent_versions.len(), base_version_count + 3);
    assert_eq!(
        concurrent_versions
            .iter()
            .map(|version| version.version_number)
            .collect::<Vec<_>>(),
        (1..=i32::try_from(concurrent_versions.len())?)
            .rev()
            .collect::<Vec<_>>()
    );
    assert_eq!(
        repository
            .find_by_id(second.id)
            .await?
            .current_draft_version_id,
        concurrent_versions
            .as_slice()
            .first()
            .map(|version| version.id)
    );

    let closing_repository = DieselArticleRepository::new(pool.clone());
    let mut lock_connection = pool.get().await?;
    let (locked_sender, locked_receiver) = tokio::sync::oneshot::channel();
    let (release_sender, release_receiver) = tokio::sync::oneshot::channel();
    let locked_article_id = first.id;
    let advisory_lock_task = tokio::spawn(async move {
        lock_connection
            .transaction::<(), diesel::result::Error, _>(async |connection| {
                diesel::sql_query("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")
                    .bind::<diesel::sql_types::Text, _>(locked_article_id.to_string())
                    .execute(connection)
                    .await?;
                locked_sender
                    .send(())
                    .map_err(|()| diesel::result::Error::RollbackTransaction)?;
                release_receiver
                    .await
                    .map_err(|_| diesel::result::Error::RollbackTransaction)?;
                Ok(())
            })
            .await
    });
    locked_receiver.await?;

    let mut closing_article = repository.find_by_id(first.id).await?;
    closing_article.draft_title = "Shutdown in flight".to_owned();
    closing_article.draft_content = "Accepted before shutdown".to_owned();
    closing_repository.save_draft(&mut closing_article).await?;
    assert!(matches!(
        closing_repository
            .shutdown_background_tasks(StdDuration::from_millis(100))
            .await,
        Err(AppError::Internal)
    ));
    release_sender
        .send(())
        .map_err(|()| io::Error::other("advisory lock task ended before release"))?;
    advisory_lock_task.await??;

    let persisted_before_rejection = repository.find_by_id(first.id).await?;
    closing_article.draft_title = "Rejected after shutdown".to_owned();
    closing_article.draft_content = "Must not be persisted".to_owned();
    assert!(matches!(
        closing_repository.save_draft(&mut closing_article).await,
        Err(AppError::Internal)
    ));
    let persisted_after_rejection = repository.find_by_id(first.id).await?;
    assert_eq!(
        (
            persisted_after_rejection.draft_title,
            persisted_after_rejection.draft_content,
            persisted_after_rejection.updated_at,
        ),
        (
            persisted_before_rejection.draft_title,
            persisted_before_rejection.draft_content,
            persisted_before_rejection.updated_at,
        )
    );
    assert!(matches!(
        closing_repository.drain_background_tasks().await,
        Err(AppError::Internal)
    ));
    assert!(matches!(
        closing_repository.drain_background_tasks().await,
        Err(AppError::Internal)
    ));
    closing_repository.drain_background_tasks().await?;

    Ok(())
}

#[tokio::test]
async fn article_repository_preserves_go_database_contracts() -> TestResult {
    let pool = test_pool()?;
    let repository = DieselArticleRepository::new(pool.clone());
    let run_id = Uuid::new_v4();
    let author_id = Uuid::new_v4();
    let article_ids = [
        Uuid::new_v4(),
        Uuid::new_v4(),
        Uuid::new_v4(),
        Uuid::new_v4(),
    ];
    let slug_prefix = format!("article-repository-{run_id}");

    let mut connection = pool.get().await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(author_id),
            account::name.eq("Article Repository Test"),
            account::email.eq(format!("{run_id}@article-repository.test")),
            account::password_hash.eq("test-only-not-a-login"),
            account::role.eq("user"),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let mut cleanup_ids = article_ids;
    let result = exercise_repository(
        &pool,
        &repository,
        author_id,
        &mut cleanup_ids,
        &slug_prefix,
        run_id,
    )
    .await;

    let cleanup_result = cleanup(&pool, &cleanup_ids, author_id).await;
    result?;
    cleanup_result
}

#[allow(dead_code)]
fn _assert_version_is_send_sync(_: &ArticleVersion) {
    fn assert_send_sync<T: Send + Sync>() {}
    assert_send_sync::<ArticleVersion>();
}
