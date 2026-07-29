use diesel::{Connection, PgConnection};
use secrecy::ExposeSecret;

use blog_backend::{config::database_url_from_env, database::fingerprint::schema_fingerprint};

fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    let database_url = database_url_from_env()?;
    let mut connection = PgConnection::establish(database_url.expose_secret())?;
    println!("{}", schema_fingerprint(&mut connection)?);
    Ok(())
}
