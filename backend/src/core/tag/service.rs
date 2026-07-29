use std::sync::Arc;

use crate::error::AppError;

use super::{Tag, TagRepository};

#[derive(Clone)]
pub struct TagService {
    repository: Arc<dyn TagRepository>,
}

impl TagService {
    pub fn new(repository: Arc<dyn TagRepository>) -> Self {
        Self { repository }
    }

    pub async fn get_by_id(&self, id: i32) -> Result<Tag, AppError> {
        self.repository.find_by_id(id).await
    }

    pub async fn get_by_name(&self, name: &str) -> Result<Tag, AppError> {
        self.repository.find_by_name(name).await
    }

    pub async fn get_by_ids(&self, ids: &[i64]) -> Result<Vec<Tag>, AppError> {
        self.repository.find_by_ids(ids).await
    }

    pub async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError> {
        self.repository.ensure_exists(names).await
    }

    pub async fn list(&self) -> Result<Vec<Tag>, AppError> {
        self.repository.list().await
    }

    pub async fn create(&self, name: String) -> Result<Tag, AppError> {
        let mut tag = Tag {
            id: 0,
            name,
            created_at: None,
        };
        self.repository.save(&mut tag).await?;
        Ok(tag)
    }

    pub async fn delete(&self, id: i32) -> Result<(), AppError> {
        self.repository.delete(id).await
    }

    pub async fn resolve_tag_names(&self, ids: &[i64]) -> Result<Vec<String>, AppError> {
        if ids.is_empty() {
            return Ok(Vec::new());
        }

        Ok(self
            .repository
            .find_by_ids(ids)
            .await?
            .into_iter()
            .map(|tag| tag.name)
            .collect())
    }

    pub async fn is_tag_used(&self, id: i32) -> Result<bool, AppError> {
        self.repository.is_used(id).await
    }
}
