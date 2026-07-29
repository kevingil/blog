use std::time::Duration;

use axum::Router;
use hyper::server::conn::http1::Builder;
use hyper_util::{rt::TokioIo, service::TowerToHyperService};
use tokio::{
    net::{TcpListener, TcpStream},
    task::{JoinError, JoinSet},
    time::{Instant, timeout_at},
};
use tokio_util::sync::CancellationToken;

/// Serve an Axum router while retaining ownership of every accepted connection.
///
/// Once `cancellation` is triggered, the listener stops accepting, each active
/// connection receives Hyper's graceful-shutdown signal, and all connection
/// tasks are joined. Connections that are still active at `shutdown_timeout`
/// are aborted and then fully reaped before this function returns.
///
/// HTTP upgrades are enabled so WebSocket handshakes work. A WebSocket handler
/// must still arrange ownership of any application tasks it explicitly spawns;
/// those tasks are outside the HTTP connection future.
pub async fn serve(
    listener: TcpListener,
    router: Router,
    cancellation: CancellationToken,
    shutdown_timeout: Duration,
) -> anyhow::Result<()> {
    let connections_shutdown = cancellation.child_token();
    let mut connections = JoinSet::new();
    let mut errors = Vec::new();

    loop {
        tokio::select! {
            biased;
            () = cancellation.cancelled() => break,
            completed = connections.join_next(), if !connections.is_empty() => {
                if record_connection_result(completed, &mut errors) {
                    break;
                }
            }
            accepted = listener.accept() => {
                match accepted {
                    Ok((stream, peer_address)) => {
                        let router = router.clone();
                        let connection_shutdown = connections_shutdown.clone();
                        connections.spawn(async move {
                            let result =
                                serve_connection(stream, router, connection_shutdown).await;
                            if let Err(error) = &result {
                                tracing::debug!(
                                    peer_address = %peer_address,
                                    %error,
                                    "HTTP connection closed with an error"
                                );
                            }
                            result
                        });
                    }
                    Err(error) => {
                        errors.push(format!("failed to accept HTTP connection: {error}"));
                        break;
                    }
                }
            }
        }
    }

    drop(listener);
    connections_shutdown.cancel();
    drain_connections(&mut connections, shutdown_timeout, &mut errors).await;

    if errors.is_empty() {
        Ok(())
    } else {
        Err(anyhow::anyhow!(errors.join("; ")))
    }
}

async fn serve_connection(
    stream: TcpStream,
    router: Router,
    cancellation: CancellationToken,
) -> anyhow::Result<()> {
    let io = TokioIo::new(stream);
    let service = TowerToHyperService::new(router);
    let builder = Builder::new();
    let connection = builder.serve_connection(io, service).with_upgrades();
    tokio::pin!(connection);

    tokio::select! {
        biased;
        () = cancellation.cancelled() => {
            connection.as_mut().graceful_shutdown();
            connection
                .as_mut()
                .await
                .map_err(|error| anyhow::anyhow!("HTTP connection failed: {error}"))
        }
        result = connection.as_mut() => {
            result.map_err(|error| anyhow::anyhow!("HTTP connection failed: {error}"))
        }
    }
}

async fn drain_connections(
    connections: &mut JoinSet<anyhow::Result<()>>,
    shutdown_timeout: Duration,
    errors: &mut Vec<String>,
) {
    let deadline = Instant::now() + shutdown_timeout;
    let mut exceeded_deadline = false;

    while !connections.is_empty() {
        match timeout_at(deadline, connections.join_next()).await {
            Ok(completed) => {
                record_connection_result(completed, errors);
            }
            Err(_) => {
                exceeded_deadline = true;
                break;
            }
        }
    }

    if exceeded_deadline {
        errors.push(format!(
            "HTTP connections exceeded the {shutdown_timeout:?} shutdown deadline",
        ));
        connections.abort_all();

        // JoinSet::abort_all only requests cancellation. Reaping every handle
        // here is what guarantees that no connection task survives `serve`.
        while let Some(completed) = connections.join_next().await {
            match completed {
                Ok(Ok(())) => {}
                Ok(Err(error)) => {
                    tracing::debug!(%error, "HTTP connection failed during forced shutdown");
                }
                Err(error) if error.is_cancelled() => {}
                Err(error) => {
                    errors.push(format!("connection task failed during shutdown: {error}"));
                }
            }
        }
    }
}

fn record_connection_result(
    completed: Option<Result<anyhow::Result<()>, JoinError>>,
    errors: &mut Vec<String>,
) -> bool {
    match completed {
        Some(Ok(Ok(()))) | None => false,
        Some(Ok(Err(error))) => {
            tracing::debug!(%error, "HTTP connection closed with an error");
            false
        }
        Some(Err(error)) => {
            errors.push(format!("connection task failed: {error}"));
            true
        }
    }
}

#[cfg(test)]
mod tests {
    use std::sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    };

    use axum::{Router, extract::State, routing::get};
    use tokio::{sync::Notify, time::timeout};

    use super::*;

    struct SlowHandlerState {
        started: Notify,
        finished: AtomicBool,
        delay: Duration,
    }

    async fn slow_handler(State(state): State<Arc<SlowHandlerState>>) -> &'static str {
        state.started.notify_one();
        tokio::time::sleep(state.delay).await;
        state.finished.store(true, Ordering::SeqCst);
        "done"
    }

    #[tokio::test]
    async fn forced_shutdown_aborts_and_reaps_active_connection() -> anyhow::Result<()> {
        let state = Arc::new(SlowHandlerState {
            started: Notify::new(),
            finished: AtomicBool::new(false),
            delay: Duration::from_millis(250),
        });
        let router = Router::new()
            .route("/", get(slow_handler))
            .with_state(state.clone());
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let address = listener.local_addr()?;
        let cancellation = CancellationToken::new();
        let server_cancellation = cancellation.clone();
        let server = tokio::spawn(async move {
            serve(
                listener,
                router,
                server_cancellation,
                Duration::from_millis(20),
            )
            .await
        });
        let mut client =
            tokio::spawn(async move { reqwest::get(format!("http://{address}/")).await });

        timeout(Duration::from_secs(1), state.started.notified()).await?;
        cancellation.cancel();

        let server_error = server
            .await?
            .err()
            .ok_or_else(|| anyhow::anyhow!("forced shutdown unexpectedly succeeded"))?;
        anyhow::ensure!(
            server_error.to_string().contains("shutdown deadline"),
            "unexpected server error: {server_error}"
        );
        match timeout(Duration::from_secs(1), &mut client).await {
            Ok(joined) => {
                let _ = joined?;
            }
            Err(_) => {
                client.abort();
                let _ = client.await;
                return Err(anyhow::anyhow!(
                    "HTTP client did not stop after forced connection shutdown"
                ));
            }
        }

        tokio::time::sleep(state.delay + Duration::from_millis(20)).await;
        anyhow::ensure!(
            !state.finished.load(Ordering::SeqCst),
            "aborted handler continued after the server returned"
        );
        Ok(())
    }

    #[tokio::test]
    async fn graceful_shutdown_waits_for_active_connection() -> anyhow::Result<()> {
        let state = Arc::new(SlowHandlerState {
            started: Notify::new(),
            finished: AtomicBool::new(false),
            delay: Duration::from_millis(20),
        });
        let router = Router::new()
            .route("/", get(slow_handler))
            .with_state(state.clone());
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let address = listener.local_addr()?;
        let cancellation = CancellationToken::new();
        let server_cancellation = cancellation.clone();
        let server = tokio::spawn(async move {
            serve(
                listener,
                router,
                server_cancellation,
                Duration::from_secs(1),
            )
            .await
        });
        let client = tokio::spawn(async move { reqwest::get(format!("http://{address}/")).await });

        timeout(Duration::from_secs(1), state.started.notified()).await?;
        cancellation.cancel();

        server.await??;
        let response = client.await??;
        anyhow::ensure!(response.status().is_success());
        anyhow::ensure!(
            state.finished.load(Ordering::SeqCst),
            "graceful shutdown returned before the active handler"
        );
        Ok(())
    }
}
