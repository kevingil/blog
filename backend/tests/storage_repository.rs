use std::{env, error::Error, io};

use blog_backend::{
    database::{
        models::files::FileIndexRow,
        pool::{PgPool, create_pool},
    },
    schema::file_index,
};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl, SelectableHelper};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::SecretString;
use serde_json::{Value, json};
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the storage_repository target; start the Docker PostgreSQL service and provide its URL",
        )
    })?;
    let mut migration_connection = PgConnection::establish(&database_url)?;
    migration_connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("storage test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

#[tokio::test]
async fn file_index_row_preserves_sql_null_empty_and_json_null_distinctions() -> TestResult {
    let pool = test_pool()?;
    let null_id = Uuid::new_v4();
    let empty_id = Uuid::new_v4();
    let json_null_id = Uuid::new_v4();
    let mut connection = pool.get().await?;
    diesel::insert_into(file_index::table)
        .values((
            file_index::id.eq(null_id),
            file_index::s3_key.eq(format!("storage-test/{null_id}")),
            file_index::filename.eq("null.txt"),
            file_index::directory_path.eq(None::<String>),
            file_index::file_type.eq(None::<String>),
            file_index::file_size.eq(None::<i64>),
            file_index::content_type.eq(None::<String>),
            file_index::meta_data.eq(None::<Value>),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(file_index::table)
        .values((
            file_index::id.eq(empty_id),
            file_index::s3_key.eq(format!("storage-test/{empty_id}")),
            file_index::filename.eq("empty.txt"),
            file_index::directory_path.eq(Some(String::new())),
            file_index::file_type.eq(Some(String::new())),
            file_index::file_size.eq(Some(0_i64)),
            file_index::content_type.eq(Some(String::new())),
            file_index::meta_data.eq(Some(json!({}))),
        ))
        .execute(&mut connection)
        .await?;
    diesel::insert_into(file_index::table)
        .values((
            file_index::id.eq(json_null_id),
            file_index::s3_key.eq(format!("storage-test/{json_null_id}")),
            file_index::filename.eq("json-null.txt"),
            file_index::meta_data.eq(Some(Value::Null)),
        ))
        .execute(&mut connection)
        .await?;

    let null_row = file_index::table
        .find(null_id)
        .select(FileIndexRow::as_select())
        .first::<FileIndexRow>(&mut connection)
        .await?;
    assert_eq!(null_row.directory_path, None);
    assert_eq!(null_row.file_type, None);
    assert_eq!(null_row.file_size, None);
    assert_eq!(null_row.content_type, None);
    assert_eq!(null_row.meta_data, None);

    let empty_row = file_index::table
        .find(empty_id)
        .select(FileIndexRow::as_select())
        .first::<FileIndexRow>(&mut connection)
        .await?;
    assert_eq!(empty_row.directory_path.as_deref(), Some(""));
    assert_eq!(empty_row.file_type.as_deref(), Some(""));
    assert_eq!(empty_row.file_size, Some(0));
    assert_eq!(empty_row.content_type.as_deref(), Some(""));
    assert_eq!(empty_row.meta_data, Some(json!({})));

    let json_null_row = file_index::table
        .find(json_null_id)
        .select(FileIndexRow::as_select())
        .first::<FileIndexRow>(&mut connection)
        .await?;
    assert_eq!(json_null_row.meta_data, Some(Value::Null));

    diesel::delete(file_index::table.filter(file_index::id.eq_any([
        null_id,
        empty_id,
        json_null_id,
    ])))
    .execute(&mut connection)
    .await?;
    Ok(())
}
