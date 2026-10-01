use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use chrono::Utc;
use uuid::Uuid;

use crate::error::AppError;

use super::{service::SkillRepository, types::AgentSkill};

#[derive(Clone, Default)]
pub struct InMemorySkillRepository {
    skills: Arc<Mutex<Vec<AgentSkill>>>,
}

#[async_trait]
impl SkillRepository for InMemorySkillRepository {
    async fn list(&self) -> Result<Vec<AgentSkill>, AppError> {
        self.skills
            .lock()
            .map(|skills| skills.clone())
            .map_err(|_| AppError::Internal)
    }

    async fn find(&self, id: Uuid) -> Result<AgentSkill, AppError> {
        self.skills
            .lock()
            .map_err(|_| AppError::Internal)?
            .iter()
            .find(|skill| skill.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn create(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError> {
        let now = Utc::now();
        let mut saved = skill.clone();
        saved.created_at = Some(now);
        saved.updated_at = Some(now);
        self.skills
            .lock()
            .map_err(|_| AppError::Internal)?
            .push(saved.clone());
        Ok(saved)
    }

    async fn update(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError> {
        let mut skills = self.skills.lock().map_err(|_| AppError::Internal)?;
        let Some(existing) = skills.iter_mut().find(|item| item.id == skill.id) else {
            return Err(AppError::NotFound);
        };
        let mut saved = skill.clone();
        saved.created_at = existing.created_at;
        saved.updated_at = Some(Utc::now());
        *existing = saved.clone();
        Ok(saved)
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut skills = self.skills.lock().map_err(|_| AppError::Internal)?;
        let before = skills.len();
        skills.retain(|skill| skill.id != id);
        if skills.len() == before {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}
