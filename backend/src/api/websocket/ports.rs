use tokio::sync::mpsc;

use super::types::{AgentStreamEvent, WorkerStatusSnapshot, WorkerStatusUpdate};

/// Consumer-owned port for the copilot request registry.
pub trait AgentStreamProvider: Send + Sync {
    /// The live Go manager exposes a single receiver for a request. Returning
    /// `None` produces the exact WS-003 terminal error frame.
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>>;
}

/// Consumer-owned port for worker snapshots and lossy, bounded status updates.
pub trait WorkerStatusProvider: Send + Sync {
    fn snapshot(&self) -> Vec<WorkerStatusSnapshot>;

    /// Implementations retain Go's capacity-100/drop-newest behavior before
    /// handing this single-consumer receiver to a WebSocket subscription.
    fn subscribe(&self) -> mpsc::Receiver<WorkerStatusUpdate>;
}

/// Production-safe composition used until the copilot domain is ported.
/// Every non-empty request subscription becomes the exact WS-003 error.
#[derive(Debug, Default)]
pub struct UnavailableAgentStreamProvider;

impl AgentStreamProvider for UnavailableAgentStreamProvider {
    fn take_response_stream(&self, _request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        None
    }
}

/// Production-safe composition used until worker status ownership is ported.
/// A subscription is acknowledged, has no initial statuses, and its update
/// stream is already closed.
#[derive(Debug, Default)]
pub struct EmptyWorkerStatusProvider;

impl WorkerStatusProvider for EmptyWorkerStatusProvider {
    fn snapshot(&self) -> Vec<WorkerStatusSnapshot> {
        Vec::new()
    }

    fn subscribe(&self) -> mpsc::Receiver<WorkerStatusUpdate> {
        let (_sender, receiver) = mpsc::channel(1);
        receiver
    }
}
