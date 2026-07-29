use std::sync::Arc;

use uuid::Uuid;

use crate::error::AppError;

use super::{
    Page, PageCreateRequest, PageListOptions, PageListResult, PageRepository, PageUpdateRequest,
};

#[derive(Clone)]
pub struct PageService {
    repository: Arc<dyn PageRepository>,
}

impl PageService {
    pub fn new(repository: Arc<dyn PageRepository>) -> Self {
        Self { repository }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<Page, AppError> {
        self.repository.find_by_id(id).await
    }

    pub async fn get_by_slug(&self, slug: &str) -> Result<Page, AppError> {
        self.repository.find_by_slug(slug).await
    }

    pub async fn list(
        &self,
        page: i64,
        per_page: i64,
        is_published: Option<bool>,
    ) -> Result<PageListResult, AppError> {
        let page = if page <= 0 { 1 } else { page };
        let per_page = if per_page <= 0 { 20 } else { per_page };
        let (pages, total) = self
            .repository
            .list(PageListOptions {
                page,
                per_page,
                is_published,
            })
            .await?;
        let total_pages = if total == 0 {
            0
        } else {
            ((total - 1) / per_page) + 1
        };
        Ok(PageListResult {
            pages,
            total,
            page,
            per_page,
            total_pages,
        })
    }

    pub async fn create(&self, request: PageCreateRequest) -> Result<Page, AppError> {
        match self.repository.find_by_slug(&request.slug).await {
            Ok(_) => {
                return Err(AppError::Conflict("resource already exists".to_owned()));
            }
            Err(AppError::NotFound) => {}
            Err(error) => return Err(error),
        }

        let mut page = Page {
            id: Uuid::new_v4(),
            slug: request.slug,
            title: request.title,
            content: request.content,
            description: request.description,
            image_url: request.image_url,
            meta_data: request.meta_data,
            is_published: request.is_published,
            created_at: None,
            updated_at: None,
        };
        self.repository.save(&mut page).await?;
        Ok(page)
    }

    pub async fn update(&self, id: Uuid, request: PageUpdateRequest) -> Result<Page, AppError> {
        let mut page = self.repository.find_by_id(id).await?;
        if let Some(title) = request.title {
            page.title = title;
        }
        if let Some(content) = request.content {
            page.content = content;
        }
        if let Some(description) = request.description {
            page.description = description;
        }
        if let Some(image_url) = request.image_url {
            page.image_url = image_url;
        }
        if let Some(meta_data) = request.meta_data {
            page.meta_data = Some(meta_data);
        }
        if let Some(is_published) = request.is_published {
            page.is_published = is_published;
        }
        self.repository.save(&mut page).await?;
        Ok(page)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.repository.delete(id).await
    }
}
