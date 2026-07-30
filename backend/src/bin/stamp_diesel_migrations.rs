use diesel::{
    Connection, PgConnection, QueryableByName, RunQueryDsl,
    connection::SimpleConnection,
    sql_types::{Bool, Text},
};
use secrecy::ExposeSecret;

use blog_backend::{config::database_url_from_env, database::fingerprint::schema_fingerprint};

const MIGRATION_SCHEMA_SHA256: &str =
    "81fa7f13268ae949c1c627f62ea860d4fe7dfb72698a4f40c5b4706cadd07b29";
const MIGRATION_VERSIONS: [&str; 7] = [
    "20250723064003",
    "20250813062742",
    "20260119004828",
    "20260125045327",
    "20260129000000",
    "20260130000000",
    "20260315000000",
];

#[derive(QueryableByName)]
struct BooleanValue {
    #[diesel(sql_type = Bool)]
    value: bool,
}

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;

    connection.transaction::<_, anyhow::Error, _>(|connection| {
        connection.batch_execute(
            "SELECT pg_advisory_xact_lock(hashtext('blog-diesel-migration-stamp'));",
        )?;

        let diesel_ledger_exists = diesel::sql_query(
            "SELECT to_regclass('public.__diesel_schema_migrations') IS NOT NULL AS value",
        )
        .get_result::<BooleanValue>(connection)?
        .value;
        if diesel_ledger_exists {
            anyhow::bail!(
                "Diesel migration ledger already exists; refusing to overwrite migration history"
            );
        }

        let actual_fingerprint = schema_fingerprint(connection)?;
        if actual_fingerprint != MIGRATION_SCHEMA_SHA256 {
            anyhow::bail!(
                "database schema does not match the schema declared by the existing Diesel \
                 migrations: expected fingerprint {MIGRATION_SCHEMA_SHA256}, found \
                 {actual_fingerprint}; no migration history was written"
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
        for version in MIGRATION_VERSIONS {
            diesel::sql_query("INSERT INTO __diesel_schema_migrations (version) VALUES ($1)")
                .bind::<Text, _>(version)
                .execute(connection)?;
        }
        Ok(())
    })?;

    println!(
        "verified the existing schema and stamped {} Diesel migration versions",
        MIGRATION_VERSIONS.len()
    );
    Ok(())
}
