use std::sync::Arc;

use uuid::Uuid;

use crate::{
    core::ml::{EmbeddingService, TextGenerationPort},
    error::AppError,
};

use super::prompts::{
    EDITOR_CONTEXT_PROMPT, EDITOR_SYSTEM_PROMPT, WRITING_CONTEXT, writer_system_prompt,
    writer_user_prompt,
};

#[derive(Debug, Clone, PartialEq)]
pub struct GeneratedArticle {
    pub draft_title: String,
    pub draft_content: String,
    pub author_id: Uuid,
    pub draft_embedding: Vec<f32>,
}

#[derive(Clone)]
pub struct WriterAgent {
    text: Arc<dyn TextGenerationPort>,
    embeddings: Arc<EmbeddingService>,
}

impl WriterAgent {
    pub fn new(text: Arc<dyn TextGenerationPort>, embeddings: Arc<EmbeddingService>) -> Self {
        Self { text, embeddings }
    }

    pub async fn generate_article(
        &self,
        prompt: &str,
        title: &str,
        author_id: Uuid,
    ) -> Result<GeneratedArticle, AppError> {
        let draft = self
            .text
            .generate_text(
                &writer_system_prompt(WRITING_CONTEXT),
                &writer_user_prompt(title, prompt),
            )
            .await
            .map_err(|_| AppError::External)?;
        let content = self
            .text
            .generate_text(EDITOR_SYSTEM_PROMPT, &draft)
            .await
            .map_err(|_| AppError::External)?;
        let embedding = self.embeddings.generate_embedding(&content).await?;
        Ok(GeneratedArticle {
            draft_title: title.to_owned(),
            draft_content: content,
            author_id,
            draft_embedding: embedding,
        })
    }

    pub async fn update_with_context(
        &self,
        title: &str,
        content: &str,
    ) -> Result<String, AppError> {
        if title.is_empty() && content.is_empty() {
            return Err(AppError::NotFound);
        }
        self.text
            .generate_text(
                EDITOR_CONTEXT_PROMPT,
                &format!("Title: {title:?}\nPrompt: {content}"),
            )
            .await
    }
}
