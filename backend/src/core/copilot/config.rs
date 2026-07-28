use std::{env, num::NonZeroUsize, time::Duration};

use thiserror::Error;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CopilotConfig {
    pub max_concurrent_requests: NonZeroUsize,
    pub request_timeout: Duration,
    pub channel_buffer: NonZeroUsize,
    pub cleanup_delay: Duration,
}

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum CopilotConfigError {
    #[error("{0} must be greater than zero")]
    Zero(&'static str),
    #[error("{0} is too large")]
    TooLarge(&'static str),
}

impl Default for CopilotConfig {
    fn default() -> Self {
        Self {
            max_concurrent_requests: NonZeroUsize::new(10).unwrap_or(NonZeroUsize::MIN),
            request_timeout: Duration::from_secs(10 * 60),
            channel_buffer: NonZeroUsize::new(100).unwrap_or(NonZeroUsize::MIN),
            cleanup_delay: Duration::from_secs(15 * 60),
        }
    }
}

impl CopilotConfig {
    pub fn from_env() -> Result<Self, CopilotConfigError> {
        let defaults = Self::default();
        Self::new(
            env_usize(
                "AGENT_MAX_CONCURRENT",
                defaults.max_concurrent_requests.get(),
            ),
            env_minutes(
                "AGENT_REQUEST_TIMEOUT",
                defaults.request_timeout.as_secs() / 60,
            ),
            env_usize("AGENT_CHANNEL_BUFFER", defaults.channel_buffer.get()),
            env_minutes("AGENT_CLEANUP_DELAY", defaults.cleanup_delay.as_secs() / 60),
        )
    }

    pub fn new(
        max_concurrent_requests: usize,
        request_timeout_minutes: u64,
        channel_buffer: usize,
        cleanup_delay_minutes: u64,
    ) -> Result<Self, CopilotConfigError> {
        let max_concurrent_requests = NonZeroUsize::new(max_concurrent_requests)
            .ok_or(CopilotConfigError::Zero("AGENT_MAX_CONCURRENT"))?;
        let channel_buffer = NonZeroUsize::new(channel_buffer)
            .ok_or(CopilotConfigError::Zero("AGENT_CHANNEL_BUFFER"))?;
        let request_timeout = minutes(request_timeout_minutes, "AGENT_REQUEST_TIMEOUT", false)?;
        let cleanup_delay = minutes(cleanup_delay_minutes, "AGENT_CLEANUP_DELAY", true)?;
        Ok(Self {
            max_concurrent_requests,
            request_timeout,
            channel_buffer,
            cleanup_delay,
        })
    }
}

fn env_usize(name: &str, default: usize) -> usize {
    env::var(name)
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(default)
}

fn env_minutes(name: &str, default: u64) -> u64 {
    env::var(name)
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(default)
}

fn minutes(
    value: u64,
    name: &'static str,
    allow_zero: bool,
) -> Result<Duration, CopilotConfigError> {
    if value == 0 && !allow_zero {
        return Err(CopilotConfigError::Zero(name));
    }
    value
        .checked_mul(60)
        .map(Duration::from_secs)
        .ok_or(CopilotConfigError::TooLarge(name))
}
