use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AgentSkill {
    pub id: Uuid,
    pub name: String,
    pub description: String,
    pub instructions: String,
    pub enabled: bool,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct CreateAgentSkill {
    pub name: String,
    #[serde(default)]
    pub description: String,
    pub instructions: String,
    #[serde(default = "default_enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct UpdateAgentSkill {
    pub name: Option<String>,
    pub description: Option<String>,
    pub instructions: Option<String>,
    pub enabled: Option<bool>,
}

fn default_enabled() -> bool {
    true
}

pub fn format_active_skills(skills: &[AgentSkill]) -> String {
    if skills.is_empty() {
        return String::new();
    }
    let mut output = String::from(
        "## Active skills\nFollow these activated skills when they apply. Skills that are turned off are omitted.\n",
    );
    for skill in skills {
        output.push_str("\n### ");
        output.push_str(&skill.name);
        output.push('\n');
        if !skill.description.trim().is_empty() {
            output.push_str("When relevant: ");
            output.push_str(skill.description.trim());
            output.push('\n');
        }
        output.push_str(skill.instructions.trim());
        output.push('\n');
    }
    output
}
