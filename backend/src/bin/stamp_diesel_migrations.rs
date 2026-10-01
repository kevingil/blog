use clap::Parser;
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

#[derive(Parser)]
struct Args {
    /// Skip stamping when all seven baseline versions are already recorded.
    #[arg(long)]
    if_needed: bool,
}

#[derive(QueryableByName)]
struct MigrationVersion {
    #[diesel(sql_type = Text)]
    version: String,
}

#[derive(QueryableByName)]
struct BooleanValue {
    #[diesel(sql_type = Bool)]
    value: bool,
}

fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    dotenvy::dotenv().ok();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;

    let stamped = connection.transaction::<_, anyhow::Error, _>(|connection| {
        connection.batch_execute(
            "SELECT pg_advisory_xact_lock(hashtext('blog-diesel-migration-stamp'));",
        )?;

        let diesel_ledger_exists = diesel::sql_query(
            "SELECT to_regclass('public.__diesel_schema_migrations') IS NOT NULL AS value",
        )
        .get_result::<BooleanValue>(connection)?
        .value;
        if diesel_ledger_exists {
            if args.if_needed {
                let applied = diesel::sql_query(
                    "SELECT version::text FROM public.__diesel_schema_migrations",
                )
                .load::<MigrationVersion>(connection)?;
                let missing: Vec<_> = MIGRATION_VERSIONS
                    .iter()
                    .filter(|version| !applied.iter().any(|row| row.version == **version))
                    .collect();
                if !missing.is_empty() {
                    anyhow::bail!(
                        "Diesel ledger is missing baseline versions {missing:?}; \
                         refusing to stamp or replay historical migrations"
                    );
                }
                return Ok(false);
            }
            anyhow::bail!(
                "Diesel migration ledger already exists; refusing to overwrite migration history"
            );
        }

        let actual_fingerprint = schema_fingerprint(connection)?;
        if actual_fingerprint != MIGRATION_SCHEMA_SHA256 {
            anyhow::bail!(
                "database schema does not match the seven-migration Goose baseline \
                 through 20260315000000: expected fingerprint {MIGRATION_SCHEMA_SHA256}, found \
                 {actual_fingerprint}; no migration history was written"
            );
        }

        connection.batch_execute(
            r#"
            CREATE TABLE public.__diesel_schema_migrations (
                version VARCHAR(50) PRIMARY KEY NOT NULL,
                run_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
            );
            "#,
        )?;
        for version in MIGRATION_VERSIONS {
            diesel::sql_query(
                "INSERT INTO public.__diesel_schema_migrations (version) VALUES ($1)",
            )
            .bind::<Text, _>(version)
            .execute(connection)?;
        }
        Ok(true)
    })?;

    if stamped {
        println!(
            "verified the existing schema and stamped {} Diesel migration versions",
            MIGRATION_VERSIONS.len()
        );
    } else {
        println!("all seven baseline versions already recorded; no stamping needed");
    }
    Ok(())
}
