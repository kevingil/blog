use diesel_async::{
    AsyncPgConnection,
    pooled_connection::{AsyncDieselConnectionManager, deadpool::Pool},
};
use secrecy::{ExposeSecret, SecretString};

pub type PgPool = Pool<AsyncPgConnection>;

pub fn create_pool(database_url: &SecretString) -> anyhow::Result<PgPool> {
    let manager =
        AsyncDieselConnectionManager::<AsyncPgConnection>::new(database_url.expose_secret());
    Pool::builder(manager)
        .max_size(10)
        .build()
        .map_err(|error| anyhow::anyhow!("failed to create PostgreSQL pool: {error}"))
}
