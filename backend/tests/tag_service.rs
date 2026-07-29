use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::tag::{Tag, TagRepository, TagService},
    error::AppError,
};

#[derive(Default)]
struct MemoryTags {
    values: Mutex<HashMap<i32, Tag>>,
    used: Mutex<HashMap<i32, bool>>,
}

#[async_trait]
impl TagRepository for MemoryTags {
    async fn find_by_id(&self, id: i32) -> Result<Tag, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_name(&self, name: &str) -> Result<Tag, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .find(|tag| tag.name.eq_ignore_ascii_case(name))
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_ids(&self, ids: &[i64]) -> Result<Vec<Tag>, AppError> {
        let values = self.values.lock().map_err(|_| AppError::Internal)?;
        Ok(ids
            .iter()
            .filter_map(|id| i32::try_from(*id).ok())
            .filter_map(|id| values.get(&id).cloned())
            .collect())
    }

    async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError> {
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        let mut ids = Vec::new();
        for name in names {
            if let Some(tag) = values
                .values()
                .find(|tag| tag.name.eq_ignore_ascii_case(name))
            {
                ids.push(i64::from(tag.id));
            } else {
                let id = i32::try_from(values.len() + 1).map_err(|_| AppError::Internal)?;
                values.insert(
                    id,
                    Tag {
                        id,
                        name: name.clone(),
                        created_at: None,
                    },
                );
                ids.push(i64::from(id));
            }
        }
        Ok(ids)
    }

    async fn list(&self) -> Result<Vec<Tag>, AppError> {
        Ok(self
            .values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .cloned()
            .collect())
    }

    async fn save(&self, tag: &mut Tag) -> Result<(), AppError> {
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        if tag.id == 0 {
            tag.id = i32::try_from(values.len() + 1).map_err(|_| AppError::Internal)?;
        }
        values.insert(tag.id, tag.clone());
        Ok(())
    }

    async fn delete(&self, id: i32) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .remove(&id)
            .map(|_| ())
            .ok_or(AppError::NotFound)
    }

    async fn is_used(&self, id: i32) -> Result<bool, AppError> {
        Ok(self
            .used
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .copied()
            .unwrap_or(false))
    }
}

#[tokio::test]
async fn tag_service_preserves_all_go_delegation_and_resolution_cases() {
    let repository = Arc::new(MemoryTags::default());
    let service = TagService::new(repository.clone());
    let created = service.create("golang".to_owned()).await;
    assert!(matches!(created, Ok(ref tag) if tag.id == 1 && tag.name == "golang"));
    assert!(matches!(service.get_by_id(1).await, Ok(ref tag) if tag.name == "golang"));
    assert!(matches!(service.get_by_name("GOLANG").await, Ok(ref tag) if tag.id == 1));
    assert!(matches!(
        service.get_by_id(999).await,
        Err(AppError::NotFound)
    ));

    let names = vec!["backend".to_owned(), "api".to_owned()];
    let ids = service.ensure_exists(&names).await;
    assert!(matches!(ids, Ok(ref ids) if ids == &[2, 3]));
    let fetched = service.get_by_ids(&[1, 2, 3]).await;
    assert!(matches!(fetched, Ok(ref tags) if tags.len() == 3));
    let listed = service.list().await;
    assert!(matches!(listed, Ok(ref tags) if tags.len() == 3));
    let resolved = service.resolve_tag_names(&[1, 2]).await;
    assert!(matches!(resolved, Ok(ref names) if names == &["golang", "backend"]));
    assert_eq!(
        service
            .resolve_tag_names(&[])
            .await
            .unwrap_or_else(|_| Vec::new()),
        Vec::<String>::new()
    );
    assert!(!service.is_tag_used(1).await.unwrap_or(true));
    repository
        .used
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(1, true);
    assert!(service.is_tag_used(1).await.unwrap_or(false));
    assert!(service.delete(1).await.is_ok());
}
