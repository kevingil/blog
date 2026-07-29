use std::sync::Arc;

use async_trait::async_trait;

use crate::error::AppError;

pub const IMAGE_PROMPT_SYSTEM: &str = "You are an image prompt generator. Given the content of an article, craft a vivid, concise prompt that an image generation model can use to create a representative illustration. Focus on key subjects, environment, style, mood, and colors. Respond with the prompt only.";

#[async_trait]
pub trait TextGenerationPort: Send + Sync {
    async fn generate_text(&self, instructions: &str, input: &str) -> Result<String, AppError>;
}

#[derive(Clone)]
pub struct TextGenerationService {
    provider: Arc<dyn TextGenerationPort>,
}

impl TextGenerationService {
    pub fn new(provider: Arc<dyn TextGenerationPort>) -> Self {
        Self { provider }
    }

    pub async fn generate_image_prompt(&self, article_text: &str) -> Result<String, AppError> {
        if article_text.is_empty() {
            return Err(AppError::InvalidInput(
                "article text cannot be empty for prompt generation".to_owned(),
            ));
        }
        let prompt = self
            .provider
            .generate_text(IMAGE_PROMPT_SYSTEM, article_text)
            .await?;
        if prompt.is_empty() {
            return Err(AppError::External);
        }
        Ok(prompt)
    }
}
