use async_trait::async_trait;
use chrono::Utc;
use diesel::{ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper};
use diesel_async::RunQueryDsl;
use uuid::Uuid;

use crate::{
    core::skill::{AgentSkill, SkillRepository},
    database::{
        models::agent_skill::{AgentSkillRow, NewAgentSkillRow},
        pool::PgPool,
    },
    error::AppError,
    schema::agent_skill,
};

#[derive(Clone)]
pub struct DieselSkillRepository {
    pool: PgPool,
}

impl DieselSkillRepository {
    pub const fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }
}

#[async_trait]
impl SkillRepository for DieselSkillRepository {
    async fn list(&self) -> Result<Vec<AgentSkill>, AppError> {
        let mut connection = self.connection().await?;
        let rows = agent_skill::table
            .order(agent_skill::created_at.asc())
            .select(AgentSkillRow::as_select())
            .load::<AgentSkillRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        Ok(rows.into_iter().map(AgentSkill::from).collect())
    }

    async fn find(&self, id: Uuid) -> Result<AgentSkill, AppError> {
        let mut connection = self.connection().await?;
        let row = agent_skill::table
            .find(id)
            .select(AgentSkillRow::as_select())
            .first::<AgentSkillRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)?;
        Ok(row.into())
    }

    async fn create(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError> {
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(agent_skill::table)
            .values(NewAgentSkillRow::from(skill))
            .returning(AgentSkillRow::as_returning())
            .get_result::<AgentSkillRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        Ok(inserted.into())
    }

    async fn update(&self, skill: &AgentSkill) -> Result<AgentSkill, AppError> {
        let mut connection = self.connection().await?;
        let updated = diesel::update(agent_skill::table.find(skill.id))
            .set((
                agent_skill::name.eq(&skill.name),
                agent_skill::description.eq(optional_text(&skill.description)),
                agent_skill::instructions.eq(&skill.instructions),
                agent_skill::enabled.eq(skill.enabled),
                agent_skill::updated_at.eq(Some(Utc::now())),
            ))
            .returning(AgentSkillRow::as_returning())
            .get_result::<AgentSkillRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)?;
        Ok(updated.into())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let deleted = diesel::delete(agent_skill::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        if deleted == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}

impl From<&AgentSkill> for NewAgentSkillRow {
    fn from(skill: &AgentSkill) -> Self {
        Self {
            id: skill.id,
            name: skill.name.clone(),
            description: optional_text(&skill.description),
            instructions: skill.instructions.clone(),
            enabled: skill.enabled,
        }
    }
}

impl From<AgentSkillRow> for AgentSkill {
    fn from(row: AgentSkillRow) -> Self {
        Self {
            id: row.id,
            name: row.name,
            description: row.description.unwrap_or_default(),
            instructions: row.instructions,
            enabled: row.enabled,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

fn optional_text(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_owned())
    }
}
