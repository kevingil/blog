use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use blog_backend::{
    core::ml::{
        EMBEDDING_DIMENSIONS, EmbeddingGenerator, EmbeddingService, TextGenerationPort,
        TextGenerationService,
        llm::{DORMANT_CAPABILITIES, copilot_prompt},
    },
    error::AppError,
};

#[derive(Default)]
struct CapturingProvider {
    embedding_input: Mutex<String>,
    text_calls: Mutex<Vec<(String, String)>>,
}

#[async_trait]
impl EmbeddingGenerator for CapturingProvider {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        *self
            .embedding_input
            .lock()
            .map_err(|_| AppError::Internal)? = text.to_owned();
        Ok(vec![0.25; EMBEDDING_DIMENSIONS])
    }
}

#[async_trait]
impl TextGenerationPort for CapturingProvider {
    async fn generate_text(&self, instructions: &str, input: &str) -> Result<String, AppError> {
        self.text_calls
            .lock()
            .map_err(|_| AppError::Internal)?
            .push((instructions.to_owned(), input.to_owned()));
        Ok("cinematic illustration".to_owned())
    }
}

#[tokio::test]
async fn embedding_service_validates_and_truncates_on_utf8_boundary() {
    let provider = Arc::new(CapturingProvider::default());
    let service = EmbeddingService::new(provider.clone());
    assert!(matches!(
        service.generate_embedding("").await,
        Err(AppError::InvalidInput(_))
    ));

    let input = format!("{}é", "a".repeat(7_999));
    let embedding = service.generate_embedding(&input).await;
    assert!(embedding.is_ok());
    let Ok(embedding) = embedding else {
        return;
    };
    assert_eq!(embedding.len(), EMBEDDING_DIMENSIONS);
    let captured = provider
        .embedding_input
        .lock()
        .map(|input| input.clone())
        .unwrap_or_default();
    assert_eq!(captured.len(), 7_999);
    assert!(captured.is_char_boundary(captured.len()));
}

#[tokio::test]
async fn image_prompt_service_preserves_the_active_prompt_contract() {
    let provider = Arc::new(CapturingProvider::default());
    let service = TextGenerationService::new(provider.clone());
    assert!(matches!(
        service.generate_image_prompt("").await,
        Err(AppError::InvalidInput(_))
    ));
    assert_eq!(
        service
            .generate_image_prompt("article body")
            .await
            .unwrap_or_default(),
        "cinematic illustration"
    );
    let calls = provider
        .text_calls
        .lock()
        .map(|calls| calls.clone())
        .unwrap_or_default();
    assert_eq!(calls.len(), 1);
    assert!(calls[0].0.contains("image prompt generator"));
    assert_eq!(calls[0].1, "article body");
}

#[test]
fn copilot_prompt_only_advertises_registered_tools() {
    let prompt = copilot_prompt(&["read_document".to_owned(), "replace_lines".to_owned()]);
    assert!(prompt.contains("**read_document**"));
    assert!(prompt.contains("**replace_lines**"));
    assert!(!prompt.contains("| **ask_question** |"));
    assert!(prompt.contains("Present a plan of proposed changes before editing"));
}

#[test]
fn only_unregistered_mcp_and_nested_agent_paths_remain_inventory_only() {
    let names = DORMANT_CAPABILITIES
        .iter()
        .map(|capability| capability.name)
        .collect::<Vec<_>>();
    assert!(!names.contains(&"anthropic"));
    assert!(!names.contains(&"gemini"));
    assert!(!names.contains(&"groq"));
    assert!(!names.contains(&"vertex_ai"));
    assert!(names.contains(&"mcp_stdio"));
    assert!(names.contains(&"mcp_sse"));
    assert!(names.contains(&"nested_agent"));
    assert!(
        DORMANT_CAPABILITIES
            .iter()
            .all(|capability| capability.disposition.starts_with("inventory-only"))
    );
}
