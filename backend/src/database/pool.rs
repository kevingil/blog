use std::sync::Arc;

use diesel::ConnectionError;
use diesel_async::{
    AsyncPgConnection,
    pooled_connection::{AsyncDieselConnectionManager, ManagerConfig, deadpool::Pool},
};
use futures_util::FutureExt;
use secrecy::{ExposeSecret, SecretString};
use tokio_postgres_rustls::MakeRustlsConnect;

pub type PgPool = Pool<AsyncPgConnection>;

pub fn create_pool(database_url: &SecretString) -> anyhow::Result<PgPool> {
    let mut roots = rustls::RootCertStore::empty();
    for certificate in rustls_native_certs::load_native_certs().certs {
        roots.add(certificate)?;
    }
    anyhow::ensure!(
        !roots.is_empty(),
        "no trusted PostgreSQL TLS certificates found"
    );
    let tls_config = rustls::ClientConfig::builder_with_provider(Arc::new(
        rustls::crypto::aws_lc_rs::default_provider(),
    ))
    .with_safe_default_protocol_versions()?
    .with_root_certificates(roots)
    .with_no_client_auth();
    let tls = MakeRustlsConnect::new(tls_config);
    let mut config = ManagerConfig::default();
    config.custom_setup = Box::new(move |url| {
        let tls = tls.clone();
        async move {
            let (client, connection) = tokio_postgres::connect(url, tls)
                .await
                .map_err(|error| ConnectionError::BadConnection(error.to_string()))?;
            // Diesel owns the connection driver, observes errors, and shuts it
            // down when the pooled connection is dropped.
            AsyncPgConnection::try_from_client_and_connection(client, connection).await
        }
        .boxed()
    });
    let manager = AsyncDieselConnectionManager::<AsyncPgConnection>::new_with_config(
        database_url.expose_secret(),
        config,
    );
    Pool::builder(manager)
        .max_size(10)
        .build()
        .map_err(|error| anyhow::anyhow!("failed to create PostgreSQL pool: {error}"))
}
