use async_trait::async_trait;
use axum::extract::ws::{Message, WebSocket};
use futures_util::{
    SinkExt, StreamExt,
    stream::{SplitSink, SplitStream},
};

use super::connection::OutboundFrame;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum InboundFrame {
    Text(String),
    Ping,
    Pong,
    Close,
    Binary,
}

#[derive(Debug, thiserror::Error)]
#[error("{message}")]
pub struct SocketError {
    message: String,
}

impl SocketError {
    pub fn new(message: impl Into<String>) -> Self {
        Self {
            message: message.into(),
        }
    }
}

#[async_trait]
pub trait SocketReader: Send {
    async fn receive(&mut self) -> Result<Option<InboundFrame>, SocketError>;
}

#[async_trait]
pub trait SocketWriter: Send {
    async fn send(&mut self, frame: OutboundFrame) -> Result<(), SocketError>;
}

pub trait WebSocketTransport: Send + 'static {
    fn split(self: Box<Self>) -> (Box<dyn SocketReader>, Box<dyn SocketWriter>);
}

pub struct AxumWebSocketTransport {
    socket: WebSocket,
}

impl AxumWebSocketTransport {
    pub fn new(socket: WebSocket) -> Self {
        Self { socket }
    }
}

impl WebSocketTransport for AxumWebSocketTransport {
    fn split(self: Box<Self>) -> (Box<dyn SocketReader>, Box<dyn SocketWriter>) {
        let (writer, reader) = self.socket.split();
        (
            Box::new(AxumSocketReader { reader }),
            Box::new(AxumSocketWriter { writer }),
        )
    }
}

struct AxumSocketReader {
    reader: SplitStream<WebSocket>,
}

#[async_trait]
impl SocketReader for AxumSocketReader {
    async fn receive(&mut self) -> Result<Option<InboundFrame>, SocketError> {
        match self.reader.next().await {
            Some(Ok(Message::Text(text))) => Ok(Some(InboundFrame::Text(text.to_string()))),
            Some(Ok(Message::Ping(_))) => Ok(Some(InboundFrame::Ping)),
            Some(Ok(Message::Pong(_))) => Ok(Some(InboundFrame::Pong)),
            Some(Ok(Message::Close(_))) => Ok(Some(InboundFrame::Close)),
            Some(Ok(Message::Binary(_))) => Ok(Some(InboundFrame::Binary)),
            Some(Err(error)) => Err(SocketError::new(error.to_string())),
            None => Ok(None),
        }
    }
}

struct AxumSocketWriter {
    writer: SplitSink<WebSocket, Message>,
}

#[async_trait]
impl SocketWriter for AxumSocketWriter {
    async fn send(&mut self, frame: OutboundFrame) -> Result<(), SocketError> {
        let message = match frame {
            OutboundFrame::Text(text) => Message::Text(text.into()),
            OutboundFrame::Ping => Message::Ping(Default::default()),
            OutboundFrame::Pong => Message::Pong(Default::default()),
            OutboundFrame::Close => Message::Close(None),
        };
        self.writer
            .send(message)
            .await
            .map_err(|error| SocketError::new(error.to_string()))
    }
}
