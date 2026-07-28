use std::{net::SocketAddr, panic::AssertUnwindSafe, sync::Arc};

use diesel_async::RunQueryDsl;
use futures_util::FutureExt;
use secrecy::ExposeSecret;
use tokio::{
    net::TcpListener,
    task::JoinSet,
    time::{Duration, timeout},
};
use tokio_util::sync::CancellationToken;

use crate::{
    api::{
        auth::AuthState,
        websocket::{
            EmptyWorkerStatusProvider, UnavailableAgentStreamProvider, WebSocketConfig,
            WebSocketSupervisor,
        },
    },
    app::{self, AppState},
    config::Config,
    constants::BACKGROUND_SHUTDOWN_TIMEOUT,
    core::{article::ArticleRepository, auth::AuthService},
    database::pool::create_pool,
    database::repository::{account::DieselAccountRepository, article::DieselArticleRepository},
    server,
};

pub struct Application {
    address: SocketAddr,
    router: axum::Router,
    cancellation: CancellationToken,
    tasks: JoinSet<anyhow::Result<()>>,
    articles: Arc<DieselArticleRepository>,
    websocket: WebSocketSupervisor,
}

pub async fn build(config: Config) -> anyhow::Result<Application> {
    let pool = create_pool(&config.database_url)?;
    let mut connection = pool
        .get()
        .await
        .map_err(|error| anyhow::anyhow!("failed to connect to PostgreSQL: {error}"))?;
    diesel::sql_query("SELECT 1")
        .execute(&mut connection)
        .await
        .map_err(|error| anyhow::anyhow!("PostgreSQL startup check failed: {error}"))?;
    drop(connection);
    let accounts = Arc::new(DieselAccountRepository::new(pool.clone()));
    let auth = AuthState::new(Arc::new(AuthService::new(
        accounts,
        config.jwt_secret.expose_secret(),
    )?));
    let articles = Arc::new(DieselArticleRepository::new(pool.clone()));
    let websocket_config = WebSocketConfig {
        shutdown_wait: BACKGROUND_SHUTDOWN_TIMEOUT.saturating_sub(Duration::from_secs(1)),
        ..WebSocketConfig::default()
    };
    let (websocket_handle, websocket) = WebSocketSupervisor::new(
        websocket_config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )
    .map_err(|error| anyhow::anyhow!("invalid WebSocket configuration: {error}"))?;
    let state = AppState::new(pool, auth, websocket_handle);

    Ok(Application {
        address: SocketAddr::new(config.host, config.port),
        router: app::router(state, &config.cors_origins)?,
        cancellation: CancellationToken::new(),
        tasks: JoinSet::new(),
        articles,
        websocket,
    })
}

impl Application {
    pub async fn serve(mut self) -> anyhow::Result<()> {
        let listener = TcpListener::bind(self.address).await?;
        let cancellation = self.cancellation.clone();
        tracing::info!(address = %self.address, "server listening");

        let signal_cancellation = cancellation.clone();
        self.tasks.spawn(async move {
            shutdown_signal(signal_cancellation).await;
            Ok(())
        });
        let websocket_cancellation = cancellation.child_token();
        let websocket_failure_cancellation = cancellation.clone();
        let websocket = self.websocket;
        self.tasks.spawn(async move {
            match AssertUnwindSafe(websocket.run(websocket_cancellation))
                .catch_unwind()
                .await
            {
                Ok(Ok(())) if websocket_failure_cancellation.is_cancelled() => Ok(()),
                Ok(Ok(())) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor stopped unexpectedly"))
                }
                Ok(Err(error)) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor failed: {error}"))
                }
                Err(_) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor panicked"))
                }
            }
        });
        let server_result = server::serve(
            listener,
            self.router,
            cancellation.clone(),
            BACKGROUND_SHUTDOWN_TIMEOUT,
        )
        .await;

        cancellation.cancel();
        let task_errors =
            drain_application_tasks(&mut self.tasks, BACKGROUND_SHUTDOWN_TIMEOUT).await;
        let article_result = self
            .articles
            .shutdown_background_tasks(BACKGROUND_SHUTDOWN_TIMEOUT)
            .await;

        let mut errors = Vec::new();
        if let Err(error) = server_result {
            errors.push(error.to_string());
        }
        if let Err(error) = article_result {
            errors.push(format!("article repository shutdown failed: {error}"));
        }
        errors.extend(task_errors);
        if errors.is_empty() {
            Ok(())
        } else {
            Err(anyhow::anyhow!(errors.join("; ")))
        }
    }
}

async fn drain_application_tasks(
    tasks: &mut JoinSet<anyhow::Result<()>>,
    shutdown_timeout: Duration,
) -> Vec<String> {
    let drain = async {
        let mut errors = Vec::new();
        while let Some(result) = tasks.join_next().await {
            match result {
                Ok(Ok(())) => {}
                Ok(Err(error)) => errors.push(format!("application task failed: {error}")),
                Err(error) => errors.push(format!("application task join failed: {error}")),
            }
        }
        errors
    };

    match timeout(shutdown_timeout, drain).await {
        Ok(errors) => errors,
        Err(_) => {
            tasks.abort_all();
            let abort_wait = async { while tasks.join_next().await.is_some() {} };
            let mut errors = vec![format!(
                "application tasks exceeded the {} second shutdown deadline",
                shutdown_timeout.as_secs()
            )];
            if timeout(Duration::from_secs(1), abort_wait).await.is_err() {
                errors.push("application tasks did not stop after cancellation".to_owned());
            }
            errors
        }
    }
}

async fn shutdown_signal(cancellation: CancellationToken) {
    #[cfg(unix)]
    {
        use tokio::signal::unix::{SignalKind, signal};

        let mut terminate = match signal(SignalKind::terminate()) {
            Ok(signal) => signal,
            Err(error) => {
                tracing::error!(%error, "failed to register SIGTERM handler");
                cancellation.cancel();
                return;
            }
        };
        tokio::select! {
            result = tokio::signal::ctrl_c() => {
                if let Err(error) = result {
                    tracing::error!(%error, "failed to receive ctrl-c");
                }
            }
            _ = terminate.recv() => {}
            _ = cancellation.cancelled() => {}
        }
    }

    #[cfg(not(unix))]
    {
        let _ = tokio::signal::ctrl_c().await;
    }

    cancellation.cancel();
}
