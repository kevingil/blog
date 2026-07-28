use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct ModelId(pub String);

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct ModelProvider(pub String);

impl ModelProvider {
    pub const OPENAI: &'static str = "openai";
    pub const MOCK: &'static str = "__mock";
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Model {
    pub id: ModelId,
    pub name: String,
    pub provider: ModelProvider,
    pub api_model: String,
    pub cost_per_1m_in: f64,
    pub cost_per_1m_out: f64,
    pub cost_per_1m_in_cached: f64,
    pub cost_per_1m_out_cached: f64,
    pub context_window: i64,
    pub default_max_tokens: i64,
    pub can_reason: bool,
    pub supports_attachments: bool,
}

impl Model {
    pub fn openai(
        id: impl Into<String>,
        api_model: impl Into<String>,
        max_tokens: i64,
        can_reason: bool,
    ) -> Self {
        let id = id.into();
        Self {
            name: id.clone(),
            id: ModelId(id),
            provider: ModelProvider(ModelProvider::OPENAI.to_owned()),
            api_model: api_model.into(),
            cost_per_1m_in: 0.0,
            cost_per_1m_out: 0.0,
            cost_per_1m_in_cached: 0.0,
            cost_per_1m_out_cached: 0.0,
            context_window: 400_000,
            default_max_tokens: max_tokens,
            can_reason,
            supports_attachments: true,
        }
    }
}
