use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::page::{
        Page, PageCreateRequest, PageListOptions, PageRepository, PageService, PageUpdateRequest,
    },
    error::AppError,
};
use uuid::Uuid;

#[derive(Default)]
struct MemoryPages {
    values: Mutex<HashMap<Uuid, Page>>,
    last_list: Mutex<Option<PageListOptions>>,
}

#[async_trait]
impl PageRepository for MemoryPages {
    async fn find_by_id(&self, id: Uuid) -> Result<Page, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Page, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .find(|page| page.slug == slug)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list(&self, options: PageListOptions) -> Result<(Vec<Page>, i64), AppError> {
        *self.last_list.lock().map_err(|_| AppError::Internal)? = Some(options);
        let values = self.values.lock().map_err(|_| AppError::Internal)?;
        let pages: Vec<_> = values
            .values()
            .filter(|page| {
                options
                    .is_published
                    .is_none_or(|published| page.is_published == published)
            })
            .cloned()
            .collect();
        Ok((pages.clone(), pages.len() as i64))
    }

    async fn save(&self, page: &mut Page) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .insert(page.id, page.clone());
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .remove(&id)
            .map(|_| ())
            .ok_or(AppError::NotFound)
    }
}

fn page(slug: &str, title: &str, published: bool) -> Page {
    Page {
        id: Uuid::new_v4(),
        slug: slug.to_owned(),
        title: title.to_owned(),
        content: "Original content".to_owned(),
        description: "Original description".to_owned(),
        image_url: String::new(),
        meta_data: None,
        is_published: published,
        created_at: None,
        updated_at: None,
    }
}

#[tokio::test]
async fn page_get_by_id_and_slug_preserve_found_and_not_found_cases() {
    let repository = Arc::new(MemoryPages::default());
    let value = page("about-us", "About Us", true);
    repository
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = PageService::new(repository);

    assert!(matches!(service.get_by_id(value.id).await, Ok(ref page) if page.title == "About Us"));
    assert!(
        matches!(service.get_by_slug("about-us").await, Ok(ref page) if page.slug == "about-us")
    );
    assert!(matches!(
        service.get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    assert!(matches!(
        service.get_by_slug("nonexistent").await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn page_list_preserves_pagination_defaults_filter_and_total_pages() {
    let repository = Arc::new(MemoryPages::default());
    let published = page("about", "About", true);
    let draft = page("contact", "Contact", false);
    {
        let mut values = repository
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        values.insert(published.id, published);
        values.insert(draft.id, draft);
    }
    let service = PageService::new(repository.clone());

    let result = service.list(0, 0, None).await;
    assert!(
        matches!(result, Ok(ref value) if value.pages.len() == 2 && value.page == 1 && value.per_page == 20 && value.total_pages == 1)
    );
    assert_eq!(
        *repository
            .last_list
            .lock()
            .unwrap_or_else(|error| error.into_inner()),
        Some(PageListOptions {
            page: 1,
            per_page: 20,
            is_published: None,
        })
    );
    let filtered = service.list(1, 20, Some(true)).await;
    assert!(
        matches!(filtered, Ok(ref value) if value.pages.len() == 1 && value.pages[0].is_published)
    );
}

#[tokio::test]
async fn page_create_preserves_success_and_duplicate_slug_cases() {
    let repository = Arc::new(MemoryPages::default());
    let service = PageService::new(repository);
    let request = PageCreateRequest {
        slug: "new-page".to_owned(),
        title: "New Page".to_owned(),
        content: "This is the content of the new page.".to_owned(),
        description: "A brand new page".to_owned(),
        image_url: String::new(),
        meta_data: None,
        is_published: true,
    };
    let created = service.create(request.clone()).await;
    assert!(matches!(created, Ok(ref page) if page.title == "New Page" && page.is_published));
    assert!(matches!(
        service.create(request).await,
        Err(AppError::Conflict(_))
    ));
}

#[tokio::test]
async fn page_update_changes_only_provided_fields_and_preserves_not_found() {
    let repository = Arc::new(MemoryPages::default());
    let value = page("contact", "Contact", true);
    repository
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = PageService::new(repository);

    let updated = service
        .update(
            value.id,
            PageUpdateRequest {
                title: Some("Updated Contact".to_owned()),
                content: Some("Updated content here".to_owned()),
                is_published: Some(false),
                ..PageUpdateRequest::default()
            },
        )
        .await;
    assert!(
        matches!(updated, Ok(ref page) if page.title == "Updated Contact" && page.content == "Updated content here" && !page.is_published && page.description == "Original description")
    );

    let partial = service
        .update(
            value.id,
            PageUpdateRequest {
                description: Some("New contact description".to_owned()),
                ..PageUpdateRequest::default()
            },
        )
        .await;
    assert!(
        matches!(partial, Ok(ref page) if page.title == "Updated Contact" && page.description == "New contact description")
    );
    assert!(matches!(
        service
            .update(Uuid::new_v4(), PageUpdateRequest::default())
            .await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn page_delete_preserves_success_and_not_found() {
    let repository = Arc::new(MemoryPages::default());
    let value = page("delete-me", "Delete Me", false);
    repository
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = PageService::new(repository);
    assert!(service.delete(value.id).await.is_ok());
    assert!(matches!(
        service.delete(value.id).await,
        Err(AppError::NotFound)
    ));
}
