use std::sync::Arc;

use uuid::Uuid;

use crate::{core::tag::TagRepository, error::AppError};

use super::{
    Project, ProjectCreateRequest, ProjectDetail, ProjectListOptions, ProjectListResult,
    ProjectRepository, ProjectUpdateRequest,
};

#[derive(Clone)]
pub struct ProjectService {
    projects: Arc<dyn ProjectRepository>,
    tags: Arc<dyn TagRepository>,
}

impl ProjectService {
    pub fn new(projects: Arc<dyn ProjectRepository>, tags: Arc<dyn TagRepository>) -> Self {
        Self { projects, tags }
    }

    pub async fn get_by_id(&self, id: Uuid) -> Result<Project, AppError> {
        self.projects.find_by_id(id).await
    }

    pub async fn get_detail(&self, id: Uuid) -> Result<ProjectDetail, AppError> {
        let project = self.projects.find_by_id(id).await?;
        let tags = if project.tag_ids.is_empty() {
            Vec::new()
        } else {
            self.tags
                .find_by_ids(&project.tag_ids)
                .await
                .map(|tags| tags.into_iter().map(|tag| tag.name).collect())
                .unwrap_or_default()
        };
        Ok(ProjectDetail { project, tags })
    }

    pub async fn list(&self, page: i64, per_page: i64) -> Result<ProjectListResult, AppError> {
        let page = if page <= 0 { 1 } else { page };
        let per_page = if per_page <= 0 { 20 } else { per_page };
        let (projects, total) = self
            .projects
            .list(ProjectListOptions { page, per_page })
            .await?;
        let total_pages = if total == 0 {
            0
        } else {
            ((total - 1) / per_page) + 1
        };
        Ok(ProjectListResult {
            projects,
            total,
            page,
            per_page,
            total_pages,
        })
    }

    pub async fn create(&self, request: ProjectCreateRequest) -> Result<Project, AppError> {
        if request.title.is_empty() || request.description.is_empty() {
            return Err(AppError::InvalidInput("validation failed".to_owned()));
        }

        let tag_ids = self.tags.ensure_exists(&request.tags).await?;
        let mut project = Project {
            id: Uuid::new_v4(),
            title: request.title,
            description: request.description,
            content: request.content,
            tag_ids,
            image_url: request.image_url,
            url: request.url,
            created_at: None,
            updated_at: None,
        };
        self.projects.save(&mut project).await?;
        Ok(project)
    }

    pub async fn update(
        &self,
        id: Uuid,
        request: ProjectUpdateRequest,
    ) -> Result<Project, AppError> {
        let mut project = self.projects.find_by_id(id).await?;
        if let Some(title) = request.title {
            project.title = title;
        }
        if let Some(description) = request.description {
            project.description = description;
        }
        if let Some(content) = request.content {
            project.content = content;
        }
        if let Some(tags) = request.tags {
            project.tag_ids = self.tags.ensure_exists(&tags).await?;
        }
        if let Some(image_url) = request.image_url {
            project.image_url = image_url;
        }
        if let Some(url) = request.url {
            project.url = url;
        }
        project.updated_at = Some(chrono::Utc::now());
        self.projects.update(&project).await?;
        Ok(project)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.projects.delete(id).await
    }
}
