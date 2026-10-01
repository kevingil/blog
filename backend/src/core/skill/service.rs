use std::sync::Arc;

use async_trait::async_trait;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::error::AppError;

use super::types::{
    AgentSkill, CreateAgentSkill, UpdateAgentSkill, format_active_skills,
};

#[async_trait]
pub trait SkillRepository: Send + Sync {
    async fn list(&self) -> Result<Vec<AgentSkill>, AppError>;
    async fn find(&self, id: Uuid) -> Result<AgentSkill, AppError>;
    async fn create(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError>;
    async fn update(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}

#[async_trait]
pub trait SkillContextPort: Send + Sync {
    async fn active_prompt(&self) -> Result<String, AppError>;
}

pub struct SkillService {
    repository: Arc<dyn SkillRepository>,
    cancellation: CancellationToken,
}

impl SkillService {
    pub fn new(
        repository: Arc<dyn SkillRepository>,
        cancellation: CancellationToken,
    ) -> Arc<Self> {
        Arc::new(Self {
            repository,
            cancellation,
        })
    }

    pub async fn list(&self) -> Result<Vec<AgentSkill>, AppError> {
        self.cancelled()?;
        self.repository.list().await
    }

    pub async fn list_enabled(&self) -> Result<Vec<AgentSkill>, AppError> {
        Ok(self
            .list()
            .await?
            .into_iter()
            .filter(|skill| skill.enabled)
            .collect())
    }

    pub async fn create(&self, request: CreateAgentSkill) -> Result<AgentSkill, AppError> {
        self.cancelled()?;
        let name = validate_name(&request.name)?;
        let instructions = validate_instructions(&request.instructions)?;
        self.ensure_unique_name(&name, None).await?;
        let skill = AgentSkill {
            id: Uuid::new_v4(),
            name,
            description: request.description.trim().to_owned(),
            instructions,
            enabled: request.enabled,
            created_at: None,
            updated_at: None,
        };
        self.repository.create(&skill).await
    }

    pub async fn update(&self, id: Uuid, request: UpdateAgentSkill) -> Result<AgentSkill, AppError> {
        self.cancelled()?;
        let mut skill = self.repository.find(id).await?;
        if let Some(name) = request.name {
            skill.name = validate_name(&name)?;
        }
        if let Some(description) = request.description {
            skill.description = description.trim().to_owned();
        }
        if let Some(instructions) = request.instructions {
            skill.instructions = validate_instructions(&instructions)?;
        }
        if let Some(enabled) = request.enabled {
            skill.enabled = enabled;
        }
        self.ensure_unique_name(&skill.name, Some(skill.id)).await?;
        self.repository.update(&skill).await
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.cancelled()?;
        self.repository.delete(id).await
    }

    async fn ensure_unique_name(&self, name: &str, skip: Option<Uuid>) -> Result<(), AppError> {
        let exists = self.repository.list().await?.into_iter().any(|skill| {
            Some(skill.id) != skip && skill.name.eq_ignore_ascii_case(name)
        });
        if exists {
            Err(AppError::InvalidInput(
                "a skill with this name already exists".to_owned(),
            ))
        } else {
            Ok(())
        }
    }

    fn cancelled(&self) -> Result<(), AppError> {
        if self.cancellation.is_cancelled() {
            Err(AppError::Internal)
        } else {
            Ok(())
        }
    }
}

#[async_trait]
impl SkillContextPort for SkillService {
    async fn active_prompt(&self) -> Result<String, AppError> {
        Ok(format_active_skills(&self.list_enabled().await?))
    }
}

fn validate_name(name: &str) -> Result<String, AppError> {
    let name = name.trim();
    if name.is_empty() {
        Err(AppError::InvalidInput("name is required".to_owned()))
    } else {
        Ok(name.to_owned())
    }
}

fn validate_instructions(instructions: &str) -> Result<String, AppError> {
    let instructions = instructions.trim();
    if instructions.is_empty() {
        Err(AppError::InvalidInput("instructions are required".to_owned()))
    } else {
        Ok(instructions.to_owned())
    }
}
