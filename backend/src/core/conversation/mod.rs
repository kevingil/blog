use std::sync::Arc;

use base64::{Engine as _, engine::general_purpose::STANDARD};

use crate::error::AppError;

use super::speech::SpeechPort;

pub const CHANNEL_TEXT: &str = "text";
pub const CHANNEL_VOICE: &str = "voice";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ConversationTurn {
    pub message: String,
    pub transcript: String,
    pub channel: String,
}

pub struct ConversationService {
    speech: Arc<dyn SpeechPort>,
}

impl ConversationService {
    pub fn new(speech: Arc<dyn SpeechPort>) -> Arc<Self> {
        Arc::new(Self { speech })
    }

    pub async fn prepare_turn(
        &self,
        message: &str,
        audio_base64: &str,
        mime_type: &str,
    ) -> Result<ConversationTurn, AppError> {
        let mut text = message.trim().to_owned();
        let mut transcript = String::new();
        if !audio_base64.trim().is_empty() {
            let audio = STANDARD
                .decode(audio_base64.trim())
                .map_err(|_| AppError::InvalidInput("audio must be valid base64".to_owned()))?;
            if audio.is_empty() {
                return Err(AppError::InvalidInput("audio is empty".to_owned()));
            }
            let mime = if mime_type.trim().is_empty() {
                "audio/webm"
            } else {
                mime_type
            };
            transcript = self.speech.transcribe(&audio, mime).await?;
            if transcript.trim().is_empty() {
                return Err(AppError::InvalidInput(
                    "could not transcribe audio".to_owned(),
                ));
            }
            if text.is_empty() {
                text = transcript.clone();
            } else {
                text = format!("{transcript}\n\n{text}");
            }
        }
        if text.is_empty() {
            return Err(AppError::InvalidInput(
                "message or audio is required".to_owned(),
            ));
        }
        Ok(ConversationTurn {
            message: text,
            transcript,
            channel: CHANNEL_VOICE.to_owned(),
        })
    }
}
