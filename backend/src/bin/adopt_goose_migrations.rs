use clap::Parser;
use diesel::{
    Connection, PgConnection, QueryableByName, RunQueryDsl,
    connection::SimpleConnection,
    sql_types::{BigInt, Bool, Nullable, Text},
};
use secrecy::ExposeSecret;

use blog_backend::{config::database_url_from_env, database::fingerprint::schema_fingerprint};

const EXPECTED_VERSIONS: [i64; 7] = [
    20250723064003,
    20250813062742,
    20260119004828,
    20260125045327,
    20260129000000,
    20260130000000,
    20260315000000,
];

#[derive(Debug, Parser)]
struct Args {
    #[arg(long)]
    expected_schema_sha256: String,
}

#[derive(QueryableByName)]
struct OptionalText {
    #[diesel(sql_type = Nullable<Text>)]
    value: Option<String>,
}

#[derive(QueryableByName)]
struct BooleanValue {
    #[diesel(sql_type = Bool)]
    value: bool,
}

#[derive(QueryableByName)]
struct GooseState {
    #[diesel(sql_type = BigInt)]
    version_id: i64,
    #[diesel(sql_type = Bool)]
    is_applied: bool,
}

#[derive(QueryableByName)]
struct ExtensionRow {
    #[diesel(sql_type = Text)]
    extname: String,
    #[diesel(sql_type = Text)]
    extversion: String,
}

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let args = Args::parse();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;

    connection.transaction::<_, anyhow::Error, _>(|connection| {
        connection.batch_execute(
            "SELECT pg_advisory_xact_lock(hashtext('blog-goose-diesel-adoption'));",
        )?;

        validate_goose_shape(connection)?;
        validate_goose_state(connection)?;
        validate_extensions(connection)?;

        let actual_fingerprint = schema_fingerprint(connection)?;
        if actual_fingerprint != args.expected_schema_sha256 {
            anyhow::bail!(
                "schema fingerprint mismatch: expected {}, found {}",
                args.expected_schema_sha256,
                actual_fingerprint
            );
        }

        let diesel_exists = diesel::sql_query(
            "SELECT to_regclass('public.__diesel_schema_migrations') IS NOT NULL AS value",
        )
        .get_result::<BooleanValue>(connection)?
        .value;
        if diesel_exists {
            anyhow::bail!(
                "Diesel migration ledger already exists; adoption refuses to overwrite it"
            );
        }

        connection.batch_execute(
            r#"
            CREATE TABLE __diesel_schema_migrations (
                version VARCHAR(50) PRIMARY KEY NOT NULL,
                run_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
            );
            "#,
        )?;
        for version in EXPECTED_VERSIONS {
            diesel::sql_query("INSERT INTO __diesel_schema_migrations (version) VALUES ($1)")
                .bind::<Text, _>(version.to_string())
                .execute(connection)?;
        }
        Ok(())
    })?;

    println!(
        "adopted {} Goose migrations into Diesel",
        EXPECTED_VERSIONS.len()
    );
    Ok(())
}

fn validate_goose_shape(connection: &mut PgConnection) -> anyhow::Result<()> {
    let table = diesel::sql_query("SELECT to_regclass('public.goose_db_version')::text AS value")
        .get_result::<OptionalText>(connection)?
        .value;
    if table.is_none() {
        anyhow::bail!("goose_db_version is absent; explicit baseline approval is required");
    }

    let shape = diesel::sql_query(
        r#"
        SELECT string_agg(column_name || ':' || data_type, ',' ORDER BY ordinal_position) AS value
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'goose_db_version'
        "#,
    )
    .get_result::<OptionalText>(connection)?
    .value
    .ok_or_else(|| anyhow::anyhow!("Goose ledger has no columns"))?;

    let expected =
        "id:integer,version_id:bigint,is_applied:boolean,tstamp:timestamp without time zone";
    if shape != expected {
        anyhow::bail!("unsupported Goose ledger shape: {shape}");
    }
    Ok(())
}

fn validate_goose_state(connection: &mut PgConnection) -> anyhow::Result<()> {
    let rows = diesel::sql_query(
        r#"
        WITH latest AS (
            SELECT DISTINCT ON (version_id) version_id, is_applied, id
            FROM goose_db_version
            WHERE version_id <> 0
            ORDER BY version_id, id DESC
        )
        SELECT version_id, is_applied
        FROM latest
        ORDER BY version_id
        "#,
    )
    .load::<GooseState>(connection)?;

    let actual: Vec<i64> = rows.iter().map(|row| row.version_id).collect();
    if actual != EXPECTED_VERSIONS {
        anyhow::bail!("Goose versions mismatch: expected {EXPECTED_VERSIONS:?}, found {actual:?}");
    }
    if let Some(row) = rows.iter().find(|row| !row.is_applied) {
        anyhow::bail!("Goose version {} is not currently applied", row.version_id);
    }
    Ok(())
}

fn validate_extensions(connection: &mut PgConnection) -> anyhow::Result<()> {
    let extensions = diesel::sql_query(
        r#"
        SELECT extname::text, extversion::text
        FROM pg_extension
        WHERE extname IN ('uuid-ossp', 'vector')
        ORDER BY extname
        "#,
    )
    .load::<ExtensionRow>(connection)?;

    let names: Vec<&str> = extensions.iter().map(|row| row.extname.as_str()).collect();
    if names != ["uuid-ossp", "vector"] {
        anyhow::bail!("required extension mismatch: found {names:?}");
    }
    for extension in extensions {
        if extension.extversion.is_empty() {
            anyhow::bail!("extension {} has no version", extension.extname);
        }
    }
    Ok(())
}
