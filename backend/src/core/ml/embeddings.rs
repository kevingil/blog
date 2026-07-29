use std::sync::Arc;

use async_trait::async_trait;

use crate::error::AppError;

pub const EMBEDDING_MODEL: &str = "text-embedding-3-small";
pub const EMBEDDING_DIMENSIONS: usize = 1536;
pub const MAX_EMBEDDING_TEXT_LENGTH: usize = 8_000;

#[async_trait]
pub trait EmbeddingGenerator: Send + Sync {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError>;
}

#[derive(Clone)]
pub struct EmbeddingService {
    provider: Arc<dyn EmbeddingGenerator>,
}

impl EmbeddingService {
    pub fn new(provider: Arc<dyn EmbeddingGenerator>) -> Self {
        Self { provider }
    }

    pub async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        if text.is_empty() {
            return Err(AppError::InvalidInput("text cannot be empty".to_owned()));
        }

        let truncated = truncate_utf8(text, MAX_EMBEDDING_TEXT_LENGTH);
        let embedding = self.provider.generate_embedding(truncated).await?;
        if embedding.len() != EMBEDDING_DIMENSIONS {
            return Err(AppError::External);
        }
        Ok(embedding)
    }
}

fn truncate_utf8(value: &str, max_bytes: usize) -> &str {
    if value.len() <= max_bytes {
        return value;
    }
    let mut boundary = max_bytes;
    while !value.is_char_boundary(boundary) {
        boundary -= 1;
    }
    &value[..boundary]
}
