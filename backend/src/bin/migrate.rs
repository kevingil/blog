use diesel::{Connection, PgConnection};
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::ExposeSecret;

use blog_backend::config::database_url_from_env;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;
    connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| anyhow::anyhow!("migration failed: {error}"))?;
    Ok(())
}
