use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        project::{
            Project, ProjectCreateRequest, ProjectListOptions, ProjectRepository, ProjectService,
            ProjectUpdateRequest,
        },
        tag::{Tag, TagRepository},
    },
    error::AppError,
};
use uuid::Uuid;

#[derive(Default)]
struct MemoryProjects {
    values: Mutex<HashMap<Uuid, Project>>,
    last_list: Mutex<Option<ProjectListOptions>>,
}

#[async_trait]
impl ProjectRepository for MemoryProjects {
    async fn find_by_id(&self, id: Uuid) -> Result<Project, AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .get(&id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list(&self, options: ProjectListOptions) -> Result<(Vec<Project>, i64), AppError> {
        *self.last_list.lock().map_err(|_| AppError::Internal)? = Some(options);
        let values: Vec<_> = self
            .values
            .lock()
            .map_err(|_| AppError::Internal)?
            .values()
            .cloned()
            .collect();
        Ok((values.clone(), values.len() as i64))
    }

    async fn save(&self, project: &mut Project) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .insert(project.id, project.clone());
        Ok(())
    }

    async fn update(&self, project: &Project) -> Result<(), AppError> {
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        if !values.contains_key(&project.id) {
            return Err(AppError::NotFound);
        }
        values.insert(project.id, project.clone());
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

#[derive(Default)]
struct MemoryTags {
    values: Mutex<HashMap<i32, Tag>>,
    fail_find: Mutex<bool>,
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
        if *self.fail_find.lock().map_err(|_| AppError::Internal)? {
            return Err(AppError::Database);
        }
        let values = self.values.lock().map_err(|_| AppError::Internal)?;
        Ok(ids
            .iter()
            .filter_map(|id| i32::try_from(*id).ok())
            .filter_map(|id| values.get(&id).cloned())
            .collect())
    }

    async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError> {
        let mut values = self.values.lock().map_err(|_| AppError::Internal)?;
        let mut result = Vec::with_capacity(names.len());
        for name in names {
            if let Some(tag) = values
                .values()
                .find(|tag| tag.name.eq_ignore_ascii_case(name))
            {
                result.push(i64::from(tag.id));
                continue;
            }
            let id = i32::try_from(values.len() + 1).map_err(|_| AppError::Internal)?;
            values.insert(
                id,
                Tag {
                    id,
                    name: name.clone(),
                    created_at: None,
                },
            );
            result.push(i64::from(id));
        }
        Ok(result)
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

    async fn is_used(&self, _id: i32) -> Result<bool, AppError> {
        Ok(false)
    }
}

fn project(title: &str, tag_ids: Vec<i64>) -> Project {
    Project {
        id: Uuid::new_v4(),
        title: title.to_owned(),
        description: "Original description".to_owned(),
        content: "Original content".to_owned(),
        tag_ids,
        image_url: "https://example.com/image.png".to_owned(),
        url: "https://example.com".to_owned(),
        created_at: None,
        updated_at: None,
    }
}

fn service(projects: Arc<MemoryProjects>, tags: Arc<MemoryTags>) -> ProjectService {
    ProjectService::new(projects, tags)
}

#[tokio::test]
async fn project_get_and_detail_preserve_tag_resolution_and_errors() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    let value = project("Test Project", vec![1, 2]);
    projects
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    {
        let mut values = tags
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        values.insert(
            1,
            Tag {
                id: 1,
                name: "golang".to_owned(),
                created_at: None,
            },
        );
        values.insert(
            2,
            Tag {
                id: 2,
                name: "testing".to_owned(),
                created_at: None,
            },
        );
    }
    let service = service(projects, tags.clone());
    assert!(
        matches!(service.get_by_id(value.id).await, Ok(ref project) if project.title == "Test Project")
    );
    assert!(matches!(
        service.get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    let detail = service.get_detail(value.id).await;
    assert!(matches!(detail, Ok(ref detail) if detail.tags == ["golang", "testing"]));
    *tags
        .fail_find
        .lock()
        .unwrap_or_else(|error| error.into_inner()) = true;
    let swallowed = service.get_detail(value.id).await;
    assert!(matches!(swallowed, Ok(ref detail) if detail.tags.is_empty()));
    assert!(matches!(
        service.get_detail(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn project_detail_skips_tag_repository_for_empty_ids() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    let value = project("No Tags", Vec::new());
    projects
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    *tags
        .fail_find
        .lock()
        .unwrap_or_else(|error| error.into_inner()) = true;
    let result = service(projects, tags).get_detail(value.id).await;
    assert!(matches!(result, Ok(ref detail) if detail.tags.is_empty()));
}

#[tokio::test]
async fn project_list_preserves_defaults_and_total_page_calculation() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    for index in 0..25 {
        let value = project(&format!("Project {index}"), Vec::new());
        projects
            .values
            .lock()
            .unwrap_or_else(|error| error.into_inner())
            .insert(value.id, value);
    }
    let service = service(projects.clone(), tags);
    let result = service.list(1, 10).await;
    assert!(matches!(result, Ok(ref result) if result.total == 25 && result.total_pages == 3));
    let defaults = service.list(0, 0).await;
    assert!(matches!(defaults, Ok(ref result) if result.page == 1 && result.per_page == 20));
    assert_eq!(
        *projects
            .last_list
            .lock()
            .unwrap_or_else(|error| error.into_inner()),
        Some(ProjectListOptions {
            page: 1,
            per_page: 20
        })
    );
}

#[tokio::test]
async fn project_create_preserves_tags_empty_tags_and_validation_cases() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    let service = service(projects, tags);
    let created = service
        .create(ProjectCreateRequest {
            title: "New Project".to_owned(),
            description: "This is the description of the new project".to_owned(),
            content: "Project content here".to_owned(),
            tags: vec!["golang".to_owned(), "testing".to_owned()],
            image_url: "https://example.com/image.png".to_owned(),
            url: "https://example.com/project".to_owned(),
        })
        .await;
    assert!(
        matches!(created, Ok(ref project) if project.tag_ids == [1, 2] && project.content == "Project content here")
    );
    let no_tags = service
        .create(ProjectCreateRequest {
            title: "No Tags".to_owned(),
            description: "This is the description".to_owned(),
            content: "Content".to_owned(),
            tags: Vec::new(),
            image_url: String::new(),
            url: String::new(),
        })
        .await;
    assert!(matches!(no_tags, Ok(ref project) if project.tag_ids.is_empty()));
    for (title, description) in [("", "Valid description"), ("Valid title", "")] {
        let invalid = service
            .create(ProjectCreateRequest {
                title: title.to_owned(),
                description: description.to_owned(),
                content: String::new(),
                tags: Vec::new(),
                image_url: String::new(),
                url: String::new(),
            })
            .await;
        assert!(matches!(invalid, Err(AppError::InvalidInput(_))));
    }
}

#[tokio::test]
async fn project_update_changes_only_provided_fields_and_preserves_not_found() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    let value = project("Original Title", vec![1]);
    projects
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = service(projects, tags);
    let updated = service
        .update(
            value.id,
            ProjectUpdateRequest {
                title: Some("Updated Title".to_owned()),
                description: Some("Updated description".to_owned()),
                content: Some("Updated content".to_owned()),
                tags: Some(vec!["newtag".to_owned()]),
                image_url: Some("https://example.com/new-image.png".to_owned()),
                url: Some("https://example.com/new-url".to_owned()),
            },
        )
        .await;
    assert!(
        matches!(updated, Ok(ref project) if project.title == "Updated Title" && project.tag_ids == [1] && project.url.ends_with("new-url"))
    );
    let partial = service
        .update(
            value.id,
            ProjectUpdateRequest {
                title: Some("Updated Title Only".to_owned()),
                ..ProjectUpdateRequest::default()
            },
        )
        .await;
    assert!(
        matches!(partial, Ok(ref project) if project.title == "Updated Title Only" && project.description == "Updated description")
    );
    assert!(matches!(
        service
            .update(Uuid::new_v4(), ProjectUpdateRequest::default())
            .await,
        Err(AppError::NotFound)
    ));
}

#[tokio::test]
async fn project_delete_preserves_success_and_not_found() {
    let projects = Arc::new(MemoryProjects::default());
    let tags = Arc::new(MemoryTags::default());
    let value = project("Delete", Vec::new());
    projects
        .values
        .lock()
        .unwrap_or_else(|error| error.into_inner())
        .insert(value.id, value.clone());
    let service = service(projects, tags);
    assert!(service.delete(value.id).await.is_ok());
    assert!(matches!(
        service.delete(value.id).await,
        Err(AppError::NotFound)
    ));
}
