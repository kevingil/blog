use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use crate::constants::WEBSOCKET_BUFFER_CAPACITY;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum OutboundFrame {
    Text(String),
    Ping,
    Pong,
    Close,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum QueueResult {
    Queued,
    DroppedNewest,
    Closed,
}

/// The cloneable, non-blocking producer side of one WebSocket writer.
///
/// The receiver is deliberately not cloneable: exactly one writer task owns it.
#[derive(Clone)]
pub struct Connection {
    sender: mpsc::Sender<OutboundFrame>,
    cancellation: CancellationToken,
}

impl Connection {
    pub fn new(parent: &CancellationToken) -> (Self, mpsc::Receiver<OutboundFrame>) {
        let (sender, receiver) = mpsc::channel(WEBSOCKET_BUFFER_CAPACITY);
        (
            Self {
                sender,
                cancellation: parent.child_token(),
            },
            receiver,
        )
    }

    /// Queue without waiting. A full 256-frame queue drops this newest frame,
    /// matching the Go connection contract.
    pub fn send(&self, frame: OutboundFrame) -> QueueResult {
        if self.cancellation.is_cancelled() {
            return QueueResult::Closed;
        }

        match self.sender.try_send(frame) {
            Ok(()) => QueueResult::Queued,
            Err(mpsc::error::TrySendError::Full(_)) => QueueResult::DroppedNewest,
            Err(mpsc::error::TrySendError::Closed(_)) => QueueResult::Closed,
        }
    }

    pub fn close(&self) {
        self.cancellation.cancel();
    }

    pub fn cancellation_token(&self) -> CancellationToken {
        self.cancellation.clone()
    }
}
