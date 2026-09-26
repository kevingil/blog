use std::sync::Arc;

use async_trait::async_trait;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use blog_backend::{
    core::{
        conversation::ConversationService,
        speech::{SpeechAudio, SpeechPort, silent_wav},
    },
    error::AppError,
};

struct FixtureSpeech;

#[async_trait]
impl SpeechPort for FixtureSpeech {
    async fn transcribe(&self, audio: &[u8], _mime_type: &str) -> Result<String, AppError> {
        if audio.is_empty() {
            return Err(AppError::InvalidInput("audio is empty".to_owned()));
        }
        Ok("tighten the intro".to_owned())
    }

    async fn synthesize(&self, text: &str) -> Result<SpeechAudio, AppError> {
        if text.trim().is_empty() {
            return Err(AppError::InvalidInput("speech text is empty".to_owned()));
        }
        Ok(SpeechAudio {
            bytes: silent_wav(),
            mime_type: "audio/wav".to_owned(),
        })
    }
}

#[tokio::test]
async fn conversation_turn_transcribes_audio_and_keeps_parallel_text() {
    let service = ConversationService::new(Arc::new(FixtureSpeech));
    let turn = service
        .prepare_turn(
            "https://example.com/notes",
            &STANDARD.encode(silent_wav()),
            "audio/wav",
        )
        .await
        .expect("prepare turn");
    assert_eq!(turn.channel, "voice");
    assert_eq!(turn.transcript, "tighten the intro");
    assert!(turn.message.contains("tighten the intro"));
    assert!(turn.message.contains("https://example.com/notes"));
}

#[tokio::test]
async fn conversation_turn_accepts_text_only_in_voice_channel() {
    let service = ConversationService::new(Arc::new(FixtureSpeech));
    let turn = service
        .prepare_turn("https://example.com/notes", "", "")
        .await
        .expect("prepare text turn");
    assert_eq!(turn.channel, "voice");
    assert_eq!(turn.message, "https://example.com/notes");
    assert!(turn.transcript.is_empty());
}

#[tokio::test]
async fn conversation_turn_requires_audio_or_text() {
    let service = ConversationService::new(Arc::new(FixtureSpeech));
    let error = service
        .prepare_turn("  ", "", "")
        .await
        .expect_err("empty turn");
    assert!(matches!(error, AppError::InvalidInput(_)));
}
