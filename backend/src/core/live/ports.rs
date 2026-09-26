use std::sync::Arc;

use async_trait::async_trait;

use crate::error::AppError;

use super::{LiveConnection, LiveHarness, LiveTurn, LiveTurnHandle, LiveUpstream};

#[derive(Clone)]
pub struct LivePorts {
    harness: Arc<dyn LiveHarness>,
    upstream: Arc<dyn LiveUpstream>,
}

impl LivePorts {
    pub fn new(harness: Arc<dyn LiveHarness>, upstream: Arc<dyn LiveUpstream>) -> Self {
        Self { harness, upstream }
    }

    pub fn disconnected() -> Self {
        Self::new(
            Arc::new(DisconnectedHarness),
            Arc::new(DisconnectedUpstream),
        )
    }

    pub fn harness(&self) -> Arc<dyn LiveHarness> {
        self.harness.clone()
    }

    pub fn upstream(&self) -> Arc<dyn LiveUpstream> {
        self.upstream.clone()
    }
}

struct DisconnectedHarness;

#[async_trait]
impl LiveHarness for DisconnectedHarness {
    async fn run_turn(&self, _turn: LiveTurn) -> Result<LiveTurnHandle, AppError> {
        Err(AppError::External)
    }
}

struct DisconnectedUpstream;

#[async_trait]
impl LiveUpstream for DisconnectedUpstream {
    async fn connect(&self) -> Result<Box<dyn LiveConnection>, AppError> {
        Err(AppError::External)
    }
}
