use async_trait::async_trait;
use futures_util::{SinkExt, StreamExt};
use serde_json::Value;
use tokio::net::TcpStream;
use tokio_tungstenite::{
    MaybeTlsStream, WebSocketStream, connect_async,
    tungstenite::{
        Message,
        client::IntoClientRequest,
        http::header::{AUTHORIZATION, HeaderValue},
    },
};

use crate::{
    core::live::{LiveConnection, LiveUpstream, live_websocket_url},
    error::AppError,
};

use super::OpenAiClient;

struct TungsteniteLiveConnection {
    socket: WebSocketStream<MaybeTlsStream<TcpStream>>,
}

#[async_trait]
impl LiveConnection for TungsteniteLiveConnection {
    async fn send_event(&mut self, event: Value) -> Result<(), AppError> {
        let text = serde_json::to_string(&event).map_err(|_| AppError::Internal)?;
        self.socket
            .send(Message::Text(text.into()))
            .await
            .map_err(|error| {
                tracing::warn!(%error, "gpt-live send failed");
                AppError::External
            })
    }

    async fn recv_event(&mut self) -> Result<Option<Value>, AppError> {
        loop {
            match self.socket.next().await {
                None => return Ok(None),
                Some(Err(error)) => {
                    tracing::warn!(%error, "gpt-live receive failed");
                    return Err(AppError::External);
                }
                Some(Ok(Message::Text(text))) => {
                    let value =
                        serde_json::from_str(text.as_str()).map_err(|_| AppError::External)?;
                    return Ok(Some(value));
                }
                Some(Ok(Message::Ping(payload))) => {
                    self.socket
                        .send(Message::Pong(payload))
                        .await
                        .map_err(|_| AppError::External)?;
                }
                Some(Ok(Message::Close(_))) => return Ok(None),
                Some(Ok(_)) => {}
            }
        }
    }
}

#[async_trait]
impl LiveUpstream for OpenAiClient {
    async fn connect(&self) -> Result<Box<dyn LiveConnection>, AppError> {
        let (base_url, api_key) = self.live_credentials();
        let url = live_websocket_url(base_url);
        let mut request = url.into_client_request().map_err(|error| {
            tracing::warn!(%error, "gpt-live url was rejected");
            AppError::External
        })?;
        if !api_key.is_empty() {
            let value = HeaderValue::from_str(&format!("Bearer {api_key}")).map_err(|_| {
                AppError::InvalidInput("OpenAI API key cannot be sent as a header".to_owned())
            })?;
            request.headers_mut().insert(AUTHORIZATION, value);
        }
        let (socket, _response) = connect_async(request).await.map_err(|error| {
            tracing::warn!(%error, "gpt-live websocket connection failed");
            AppError::External
        })?;
        Ok(Box::new(TungsteniteLiveConnection { socket }))
    }
}
