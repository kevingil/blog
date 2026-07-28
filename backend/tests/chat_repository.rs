use std::{env, error::Error, io, sync::Arc};

use blog_backend::{
    core::chat::{ArtifactInfo, ChatMessageRepository, ChatMessageService, MessageMetadata},
    database::{
        pool::{PgPool, create_pool},
        repository::chat_message::DieselChatMessageRepository,
    },
    schema::{account, article, chat_message},
};
use chrono::{Duration, Utc};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use serde_json::{Value, json};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the chat_repository target; start the Docker PostgreSQL service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("chat test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

fn metadata(status: &str) -> MessageMetadata {
    MessageMetadata {
        artifact: Some(ArtifactInfo {
            id: "artifact".to_owned(),
            artifact_type: "rewrite".to_owned(),
            status: status.to_owned(),
            content: "content".to_owned(),
            diff_preview: "diff".to_owned(),
            title: "title".to_owned(),
            description: "description".to_owned(),
            applied_at: None,
        }),
        ..MessageMetadata::default()
    }
}

#[tokio::test]
async fn postgres_chat_repository_preserves_order_json_nulls_and_row_counts() -> TestResult {
    let pool = test_pool()?;
    let repository = Arc::new(DieselChatMessageRepository::new(pool.clone()));
    let service = ChatMessageService::new(repository.clone(), CancellationToken::new());
    let author_id = Uuid::new_v4();
    let article_id = Uuid::new_v4();
    let slug = format!("chat-repository-{article_id}");
    let mut connection = pool.get().await?;
    diesel::insert_into(account::table)
        .values((
            account::id.eq(author_id),
            account::name.eq("Chat Repository Author"),
            account::email.eq(format!("{author_id}@example.test")),
            account::password_hash.eq("not-used"),
            account::role.eq("user"),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(article::table)
        .values((
            article::id.eq(article_id),
            article::slug.eq(&slug),
            article::author_id.eq(author_id),
        ))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let first = service
        .save_message(article_id, "assistant", "first", Some(metadata("pending")))
        .await?;
    let second = service
        .save_message(
            article_id,
            "assistant",
            "second",
            Some(metadata("accepted")),
        )
        .await?;
    let third = service
        .save_message(article_id, "user", "third", None)
        .await?;
    assert!(!first.id.is_nil());
    assert!(first.created_at.is_some());

    let now = Utc::now();
    let mut connection = pool.get().await?;
    diesel::update(chat_message::table.find(first.id))
        .set(chat_message::created_at.eq(Some(now - Duration::seconds(3))))
        .execute(&mut connection)
        .await?;
    diesel::update(chat_message::table.find(second.id))
        .set(chat_message::created_at.eq(Some(now - Duration::seconds(2))))
        .execute(&mut connection)
        .await?;
    diesel::update(chat_message::table.find(third.id))
        .set(chat_message::created_at.eq(Some(now - Duration::seconds(1))))
        .execute(&mut connection)
        .await?;
    drop(connection);

    let descending = repository.list_by_article(article_id, 2).await?;
    assert_eq!(
        descending
            .iter()
            .map(|message| message.content.as_str())
            .collect::<Vec<_>>(),
        vec!["third", "second"]
    );
    let chronological = service.conversation_history(article_id, 2).await?;
    assert_eq!(
        chronological
            .iter()
            .map(|message| message.content.as_str())
            .collect::<Vec<_>>(),
        vec!["second", "third"]
    );
    let pending = repository.list_pending_artifacts(article_id).await?;
    assert_eq!(pending.len(), 1);
    assert_eq!(pending[0].id, first.id);

    let mut connection = pool.get().await?;
    diesel::update(chat_message::table.find(third.id))
        .set(chat_message::meta_data.eq(None::<Value>))
        .execute(&mut connection)
        .await?;
    drop(connection);
    assert!(repository.find_by_id(third.id).await?.meta_data.is_none());
    assert_eq!(repository.update_metadata(third.id, Value::Null).await?, 1);
    assert_eq!(
        repository.find_by_id(third.id).await?.meta_data,
        Some(Value::Null)
    );
    assert_eq!(
        repository
            .update_metadata(Uuid::new_v4(), json!({}))
            .await?,
        0
    );
    assert_eq!(repository.delete_by_article(article_id).await?, 3);
    assert!(repository.list_by_article(article_id, 50).await?.is_empty());

    let mut connection = pool.get().await?;
    diesel::delete(article::table.find(article_id))
        .execute(&mut connection)
        .await?;
    diesel::delete(account::table.find(author_id))
        .execute(&mut connection)
        .await?;
    Ok(())
}
