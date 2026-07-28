use std::{sync::Arc, time::Duration};

use blog_backend::{
    api::{
        auth::AuthState,
        websocket::{
            EmptyWorkerStatusProvider, UnavailableAgentStreamProvider, WebSocketConfig,
            WebSocketSupervisor,
        },
    },
    app::{self, AppState},
    core::auth::AuthService,
    database::{pool::create_pool, repository::account::DieselAccountRepository},
    server,
};
use futures_util::{SinkExt, StreamExt};
use secrecy::SecretString;
use serde_json::{Value, json};
use tokio::{net::TcpListener, time::timeout};
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tokio_util::sync::CancellationToken;

#[tokio::test]
async fn upgraded_session_is_owned_and_reaped_before_shutdown_returns() -> anyhow::Result<()> {
    let pool = create_pool(&SecretString::from(
        "postgres://blog:blog@127.0.0.1:5432/blog".to_owned(),
    ))?;
    let auth = AuthState::new(Arc::new(AuthService::new(
        Arc::new(DieselAccountRepository::new(pool.clone())),
        "websocket-network-secret",
    )?));
    let websocket_config = WebSocketConfig {
        shutdown_wait: Duration::from_millis(250),
        ..WebSocketConfig::default()
    };
    let (websocket_handle, websocket_supervisor) = WebSocketSupervisor::new(
        websocket_config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let state = AppState::new(pool, auth, websocket_handle.clone());
    let router = app::router(state, &[])?;

    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let cancellation = CancellationToken::new();
    let server_cancellation = cancellation.clone();
    let server_task = tokio::spawn(async move {
        server::serve(
            listener,
            router,
            server_cancellation,
            Duration::from_millis(500),
        )
        .await
    });
    let supervisor_cancellation = cancellation.child_token();
    let supervisor_task =
        tokio::spawn(async move { websocket_supervisor.run(supervisor_cancellation).await });

    let (mut socket, response) = connect_async(format!("ws://{address}/websocket")).await?;
    anyhow::ensure!(response.status().as_u16() == 101);

    socket
        .send(Message::Text(
            json!({"action": "subscribe", "requestId": "missing"})
                .to_string()
                .into(),
        ))
        .await?;
    let missing = next_text(&mut socket).await?;
    anyhow::ensure!(
        missing
            == json!({
                "requestId": "missing",
                "type": "error",
                "error": "Request not found",
                "done": true,
            })
    );

    socket
        .send(Message::Text(
            json!({"action": "subscribe", "channel": "worker-status"})
                .to_string()
                .into(),
        ))
        .await?;
    let worker_ack = next_text(&mut socket).await?;
    anyhow::ensure!(worker_ack == json!({"type": "subscribed", "channel": "worker-status"}));
    anyhow::ensure!(websocket_handle.active_connections() == 1);

    cancellation.cancel();
    timeout(Duration::from_secs(1), server_task).await???;
    timeout(Duration::from_secs(1), supervisor_task).await???;

    anyhow::ensure!(!websocket_handle.is_accepting());
    anyhow::ensure!(websocket_handle.active_connections() == 0);
    match timeout(Duration::from_secs(1), socket.next()).await? {
        Some(Ok(Message::Close(_))) | None => {}
        Some(Err(_)) => {}
        Some(Ok(other)) => {
            return Err(anyhow::anyhow!(
                "unexpected frame after root cancellation: {other:?}"
            ));
        }
    }
    Ok(())
}

async fn next_text(
    socket: &mut tokio_tungstenite::WebSocketStream<
        tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
    >,
) -> anyhow::Result<Value> {
    loop {
        let frame = timeout(Duration::from_secs(1), socket.next())
            .await?
            .ok_or_else(|| anyhow::anyhow!("WebSocket closed before the expected text frame"))??;
        match frame {
            Message::Text(text) => return Ok(serde_json::from_str(&text)?),
            Message::Ping(payload) => socket.send(Message::Pong(payload)).await?,
            Message::Close(_) => {
                return Err(anyhow::anyhow!(
                    "WebSocket closed before the expected text frame"
                ));
            }
            Message::Binary(_) | Message::Pong(_) | Message::Frame(_) => {}
        }
    }
}
