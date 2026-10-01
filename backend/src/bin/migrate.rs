use diesel::{Connection, PgConnection, connection::SimpleConnection};
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use secrecy::ExposeSecret;

use blog_backend::config::database_url_from_env;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;
    // Coordinate with stamping and other migrators. The connection owns this
    // session lock until process exit, including across migration transactions.
    connection
        .batch_execute("SELECT pg_advisory_lock(hashtext('blog-diesel-migration-stamp'));")?;
    let applied = connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| anyhow::anyhow!("migration failed: {error}"))?;
    for version in &applied {
        println!("applied Diesel migration {version}");
    }
    println!(
        "migration complete: {} pending migrations applied",
        applied.len()
    );
    Ok(())
}
