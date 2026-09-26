use async_trait::async_trait;

use crate::error::AppError;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpeechAudio {
    pub bytes: Vec<u8>,
    pub mime_type: String,
}

#[async_trait]
pub trait SpeechPort: Send + Sync {
    async fn transcribe(&self, audio: &[u8], mime_type: &str) -> Result<String, AppError>;
    async fn synthesize(&self, text: &str) -> Result<SpeechAudio, AppError>;
}

/// Minimal PCM WAV used by fixtures and conversation tests.
pub fn silent_wav() -> Vec<u8> {
    let data_size: u32 = 32;
    let mut bytes = Vec::with_capacity(44 + data_size as usize);
    bytes.extend_from_slice(b"RIFF");
    bytes.extend_from_slice(&(36 + data_size).to_le_bytes());
    bytes.extend_from_slice(b"WAVE");
    bytes.extend_from_slice(b"fmt ");
    bytes.extend_from_slice(&16u32.to_le_bytes());
    bytes.extend_from_slice(&1u16.to_le_bytes());
    bytes.extend_from_slice(&1u16.to_le_bytes());
    bytes.extend_from_slice(&8_000u32.to_le_bytes());
    bytes.extend_from_slice(&16_000u32.to_le_bytes());
    bytes.extend_from_slice(&2u16.to_le_bytes());
    bytes.extend_from_slice(&16u16.to_le_bytes());
    bytes.extend_from_slice(b"data");
    bytes.extend_from_slice(&data_size.to_le_bytes());
    bytes.extend(std::iter::repeat_n(0u8, data_size as usize));
    bytes
}
