use crate::error::AppError;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct McpPreset {
    pub id: &'static str,
    pub name: &'static str,
    pub description: &'static str,
    pub official_url: &'static str,
}

pub const PRESETS: &[McpPreset] = &[
    McpPreset {
        id: "notion",
        name: "Notion",
        description: "Pages and databases from the connected Notion workspace.",
        official_url: "https://mcp.notion.com/mcp",
    },
    McpPreset {
        id: "granola",
        name: "Granola",
        description: "Meeting notes captured in Granola.",
        official_url: "https://mcp.granola.ai/mcp",
    },
    McpPreset {
        id: "fireflies",
        name: "Fireflies",
        description: "Meeting transcripts from Fireflies.",
        official_url: "https://api.fireflies.ai/mcp",
    },
];

pub fn preset_by_id(id: &str) -> Result<McpPreset, AppError> {
    PRESETS
        .iter()
        .find(|preset| preset.id == id)
        .copied()
        .ok_or_else(|| AppError::InvalidInput(format!("unknown connector preset: {id}")))
}

pub fn validate_mcp_server(name: &str, server_url: &str) -> Result<(String, String), AppError> {
    let name = name.trim();
    if name.is_empty() {
        return Err(AppError::InvalidInput("name is required".to_owned()));
    }
    let server_url = server_url.trim();
    if !(server_url.starts_with("https://") || server_url.starts_with("http://")) {
        return Err(AppError::InvalidInput(
            "url must be an http(s) MCP server".to_owned(),
        ));
    }
    Ok((name.to_owned(), server_url.to_owned()))
}
