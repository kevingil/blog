use crate::error::AppError;

pub const LOCAL_OAUTH_AUTHORIZATION: &str = "Bearer fixture";

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

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ConnectPlan {
    pub name: String,
    pub url: String,
    pub authorization_header: Option<String>,
    pub needs_authorization: bool,
    pub authorization_url: String,
}

pub fn preset_server_url(preset: &McpPreset, oauth_base: &str) -> String {
    let base = normalize_base(oauth_base);
    if base.is_empty() {
        preset.official_url.to_owned()
    } else {
        format!("{base}/mcp/{}", preset.id)
    }
}

pub fn connect_preset(id: &str, oauth_base: &str) -> Result<ConnectPlan, AppError> {
    let preset = PRESETS
        .iter()
        .find(|preset| preset.id == id)
        .ok_or_else(|| AppError::InvalidInput(format!("unknown connector preset: {id}")))?;
    let url = preset_server_url(preset, oauth_base);
    Ok(plan_for(preset.name, &url, oauth_base))
}

pub fn connect_custom(
    name: &str,
    server_url: &str,
    oauth_base: &str,
) -> Result<ConnectPlan, AppError> {
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
    Ok(plan_for(name, server_url, oauth_base))
}

fn plan_for(name: &str, server_url: &str, oauth_base: &str) -> ConnectPlan {
    let base = normalize_base(oauth_base);
    let local = !base.is_empty() && server_url.starts_with(&base);
    if local {
        ConnectPlan {
            name: name.to_owned(),
            url: server_url.to_owned(),
            authorization_header: Some(LOCAL_OAUTH_AUTHORIZATION.to_owned()),
            needs_authorization: false,
            authorization_url: String::new(),
        }
    } else {
        ConnectPlan {
            name: name.to_owned(),
            url: server_url.to_owned(),
            authorization_header: None,
            needs_authorization: true,
            authorization_url: server_url.to_owned(),
        }
    }
}

fn normalize_base(oauth_base: &str) -> String {
    oauth_base.trim().trim_end_matches('/').to_owned()
}
