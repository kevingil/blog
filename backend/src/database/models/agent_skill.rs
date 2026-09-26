use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::schema::agent_skill;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = agent_skill)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct AgentSkillRow {
    pub id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub instructions: String,
    pub enabled: bool,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = agent_skill)]
pub struct NewAgentSkillRow {
    pub id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub instructions: String,
    pub enabled: bool,
}
