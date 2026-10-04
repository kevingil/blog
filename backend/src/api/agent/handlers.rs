use std::collections::BTreeMap;

use axum::{
    Json,
    body::Bytes,
    extract::{Path, Query, State},
    response::Redirect,
};
use uuid::Uuid;

use crate::{
    api::{auth::AuthenticatedAccount, request::JsonBody, response::SuccessResponse},
    core::mcp::{
        CompletedOauth, CreateMcpConnector, PRESETS, TRANSPORT_HTTP, UpdateMcpConnector,
        form_encode, preset_by_id, validate_mcp_server,
    },
    error::AppError,
};

use super::{
    dto::{
        AgentToolListResponse, AgentToolResponse, ArtifactFeedbackRequest, ChatRequest,
        ChatRequestResponse, ConnectorListResponse, ConnectorPresetListResponse,
        ConnectorPresetResponse, ConnectorRefreshResponse, ConnectorResponse,
        ConnectorUpdateRequest, ConnectorWriteRequest, ConversationHistoryResponse,
        ConversationQuery, ConversationTurnRequest, ConversationTurnResponse, OauthCallbackQuery,
        OauthConnectResponse, OauthConnectorRequest, PendingArtifactsResponse, SkillListResponse,
        SkillResponse, SkillUpdateRequest, SkillWriteRequest, SuccessFlagResponse,
    },
    state::AgentState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

#[utoipa::path(
    post,
    path = "/agent",
    request_body = ChatRequest,
    responses(
        (status = 200, body = SuccessResponse<ChatRequestResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "submitAgentRequest"
)]
pub async fn submit_agent_request(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    JsonBody(request): JsonBody<ChatRequest>,
) -> ApiResult<ChatRequestResponse> {
    if request.message.is_empty() {
        return Err(AppError::InvalidInput(
            "message is a required field".to_owned(),
        ));
    }
    if request.article_id.is_empty() {
        return Err(AppError::InvalidInput(
            "articleId is a required field".to_owned(),
        ));
    }
    let request_id = state.requests()?.submit(request).await?;
    Ok(Json(SuccessResponse::new(ChatRequestResponse {
        request_id,
        status: "processing".to_owned(),
    })))
}

#[utoipa::path(
    post,
    path = "/agent/conversation/turn",
    request_body = ConversationTurnRequest,
    responses(
        (status = 200, body = SuccessResponse<ConversationTurnResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "submitConversationTurn"
)]
pub async fn submit_conversation_turn(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    JsonBody(request): JsonBody<ConversationTurnRequest>,
) -> ApiResult<ConversationTurnResponse> {
    if request.article_id.is_empty() {
        return Err(AppError::InvalidInput(
            "articleId is a required field".to_owned(),
        ));
    }
    let turn = state
        .conversation()?
        .prepare_turn(&request.message, &request.audio_base64, &request.mime_type)
        .await?;
    let request_id = state
        .requests()?
        .submit(ChatRequest {
            message: turn.message,
            document_content: request.document_content,
            document_markdown: request.document_markdown,
            document_title: request.document_title,
            article_id: request.article_id,
            channel: turn.channel.clone(),
        })
        .await?;
    Ok(Json(SuccessResponse::new(ConversationTurnResponse {
        request_id,
        status: "processing".to_owned(),
        transcript: turn.transcript,
        channel: turn.channel,
    })))
}

#[utoipa::path(
    get,
    path = "/agent/tools",
    responses(
        (status = 200, body = SuccessResponse<AgentToolListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "listAgentTools"
)]
pub async fn list_agent_tools(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
) -> ApiResult<AgentToolListResponse> {
    let tools = state
        .registry()?
        .describe()
        .into_iter()
        .map(AgentToolResponse::from)
        .collect();
    Ok(Json(SuccessResponse::new(AgentToolListResponse { tools })))
}

#[utoipa::path(
    get,
    path = "/agent/connectors",
    responses(
        (status = 200, body = SuccessResponse<ConnectorListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "listMcpConnectors"
)]
pub async fn list_mcp_connectors(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
) -> ApiResult<ConnectorListResponse> {
    let connectors = state
        .connectors()?
        .list()
        .await?
        .into_iter()
        .map(ConnectorResponse::from)
        .collect();
    Ok(Json(SuccessResponse::new(ConnectorListResponse {
        connectors,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/connectors",
    request_body = ConnectorWriteRequest,
    responses(
        (status = 200, body = SuccessResponse<ConnectorResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "createMcpConnector"
)]
pub async fn create_mcp_connector(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    JsonBody(request): JsonBody<ConnectorWriteRequest>,
) -> ApiResult<ConnectorResponse> {
    let connector = state.connectors()?.create(request.into()).await?;
    Ok(Json(SuccessResponse::new(connector.into())))
}

#[utoipa::path(
    patch,
    path = "/agent/connectors/{connectorId}",
    params(("connectorId" = String, Path)),
    request_body = ConnectorUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<ConnectorResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "updateMcpConnector"
)]
pub async fn update_mcp_connector(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(connector_id): Path<String>,
    JsonBody(request): JsonBody<ConnectorUpdateRequest>,
) -> ApiResult<ConnectorResponse> {
    let id = parse_connector_id(&connector_id)?;
    let connector = state.connectors()?.update(id, request.into()).await?;
    Ok(Json(SuccessResponse::new(connector.into())))
}

#[utoipa::path(
    delete,
    path = "/agent/connectors/{connectorId}",
    params(("connectorId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "deleteMcpConnector"
)]
pub async fn delete_mcp_connector(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(connector_id): Path<String>,
) -> ApiResult<SuccessFlagResponse> {
    let id = parse_connector_id(&connector_id)?;
    state.connectors()?.delete(id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/connectors/{connectorId}/refresh",
    params(("connectorId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<ConnectorRefreshResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "refreshMcpConnector"
)]
pub async fn refresh_mcp_connector(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(connector_id): Path<String>,
) -> ApiResult<ConnectorRefreshResponse> {
    let id = parse_connector_id(&connector_id)?;
    let refresh = state.connectors()?.refresh(id).await?;
    Ok(Json(SuccessResponse::new(ConnectorRefreshResponse {
        connector_id: refresh.connector_id.to_string(),
        tool_names: refresh.tool_names,
    })))
}

#[utoipa::path(
    get,
    path = "/agent/connector-presets",
    responses(
        (status = 200, body = SuccessResponse<ConnectorPresetListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "listConnectorPresets"
)]
pub async fn list_connector_presets(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
) -> ApiResult<ConnectorPresetListResponse> {
    let connectors = state.connectors()?.list().await?;
    let presets = PRESETS
        .iter()
        .map(|preset| {
            let url = preset.official_url.to_owned();
            let connected = connectors.iter().find(|connector| {
                connector.env.get("preset_id").map(String::as_str) == Some(preset.id)
                    || connector.url == url
            });
            ConnectorPresetResponse {
                id: preset.id.to_owned(),
                name: preset.name.to_owned(),
                description: preset.description.to_owned(),
                official_url: preset.official_url.to_owned(),
                url,
                connected: connected.is_some(),
                connector_id: connected
                    .map(|connector| connector.id.to_string())
                    .unwrap_or_default(),
                last_error: connected
                    .map(|connector| connector.last_error.clone())
                    .unwrap_or_default(),
            }
        })
        .collect();
    Ok(Json(SuccessResponse::new(ConnectorPresetListResponse {
        presets,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/connector-presets/{presetId}/connect",
    params(("presetId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<OauthConnectResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "connectConnectorPreset"
)]
pub async fn connect_connector_preset(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(preset_id): Path<String>,
) -> ApiResult<OauthConnectResponse> {
    let preset = preset_by_id(&preset_id)?;
    let authorization_url = state
        .mcp_oauth()
        .start(preset.official_url, preset.name, preset.id)
        .await?;
    Ok(Json(SuccessResponse::new(OauthConnectResponse {
        connected: false,
        authorization_url,
        connector: None,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/connectors/oauth",
    request_body = OauthConnectorRequest,
    responses(
        (status = 200, body = SuccessResponse<OauthConnectResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "connectOauthMcp"
)]
pub async fn connect_oauth_mcp(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    JsonBody(request): JsonBody<OauthConnectorRequest>,
) -> ApiResult<OauthConnectResponse> {
    let (name, url) = validate_mcp_server(&request.name, &request.url)?;
    let authorization_url = state.mcp_oauth().start(&url, &name, "").await?;
    Ok(Json(SuccessResponse::new(OauthConnectResponse {
        connected: false,
        authorization_url,
        connector: None,
    })))
}

#[utoipa::path(
    get,
    path = "/agent/connectors/oauth/callback",
    params(OauthCallbackQuery),
    responses(
        (status = 302, description = "Redirects to the connectors page after sign-in"),
        (status = 400, description = "The sign-in query could not be read")
    ),
    tag = "agent",
    operation_id = "completeOauthConnector"
)]
pub async fn complete_oauth_connector(
    State(state): State<AgentState>,
    Query(query): Query<OauthCallbackQuery>,
) -> Redirect {
    let destination = oauth_return_url(&state, query).await;
    Redirect::temporary(&destination)
}

async fn oauth_return_url(state: &AgentState, query: OauthCallbackQuery) -> String {
    let oauth = state.mcp_oauth();
    if let Some(message) = provider_error(&query) {
        return oauth_error_redirect(oauth, &message);
    }
    let code = query.code.unwrap_or_default();
    let oauth_state = query.state.unwrap_or_default();
    if code.is_empty() || oauth_state.is_empty() {
        return oauth_error_redirect(oauth, "Sign-in did not return a code. Click Connect again.");
    }
    match oauth.finish(&code, &oauth_state).await {
        Ok(completed) => match store_oauth_connector(state, &completed).await {
            Ok(()) => format!(
                "{}?connected={}",
                oauth.app_connectors_url(),
                form_encode(&completed.name)
            ),
            Err(error) => oauth_error_redirect(oauth, &user_message(error)),
        },
        Err(error) => oauth_error_redirect(oauth, &user_message(error)),
    }
}

fn provider_error(query: &OauthCallbackQuery) -> Option<String> {
    let error = query.error.as_deref().unwrap_or_default();
    if error.is_empty() {
        return None;
    }
    let description = query.error_description.as_deref().unwrap_or_default();
    if description.is_empty() {
        Some(error.to_owned())
    } else {
        Some(description.to_owned())
    }
}

fn oauth_error_redirect(oauth: &crate::core::mcp::McpOauth, message: &str) -> String {
    let message = message.replace(['\n', '\r'], " ");
    format!(
        "{}?oauth_error={}",
        oauth.app_connectors_url(),
        form_encode(&message)
    )
}

fn user_message(error: AppError) -> String {
    match error {
        AppError::InvalidInput(message) => message,
        _ => "Could not finish sign-in.".to_owned(),
    }
}

async fn store_oauth_connector(
    state: &AgentState,
    completed: &CompletedOauth,
) -> Result<(), AppError> {
    let mut headers = BTreeMap::new();
    headers.insert(
        "Authorization".to_owned(),
        format!("Bearer {}", completed.access_token),
    );
    let mut env = BTreeMap::new();
    if !completed.preset_id.is_empty() {
        env.insert("preset_id".to_owned(), completed.preset_id.clone());
    }
    env.insert("oauth_resource".to_owned(), completed.resource.clone());
    env.insert(
        "oauth_token_endpoint".to_owned(),
        completed.token_endpoint.clone(),
    );
    env.insert("oauth_client_id".to_owned(), completed.client_id.clone());
    if !completed.refresh_token.is_empty() {
        env.insert(
            "oauth_refresh_token".to_owned(),
            completed.refresh_token.clone(),
        );
    }
    let connectors = state.connectors()?.list().await?;
    if let Some(existing) = connectors.into_iter().find(|connector| {
        if completed.preset_id.is_empty() {
            connector.url == completed.server_url && connector.env.get("preset_id").is_none()
        } else {
            connector.env.get("preset_id").map(String::as_str) == Some(completed.preset_id.as_str())
        }
    }) {
        state
            .connectors()?
            .update(
                existing.id,
                UpdateMcpConnector {
                    name: Some(completed.name.clone()),
                    transport: Some(TRANSPORT_HTTP.to_owned()),
                    url: Some(completed.server_url.clone()),
                    headers: Some(headers),
                    env: Some(env),
                    enabled: Some(true),
                    ..UpdateMcpConnector::default()
                },
            )
            .await?;
        return Ok(());
    }
    state
        .connectors()?
        .create(CreateMcpConnector {
            name: completed.name.clone(),
            transport: TRANSPORT_HTTP.to_owned(),
            command: String::new(),
            args: Vec::new(),
            url: completed.server_url.clone(),
            headers,
            env,
            enabled: true,
        })
        .await?;
    Ok(())
}

#[utoipa::path(
    get,
    path = "/agent/skills",
    responses(
        (status = 200, body = SuccessResponse<SkillListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "listAgentSkills"
)]
pub async fn list_agent_skills(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
) -> ApiResult<SkillListResponse> {
    let skills = state
        .skills()?
        .list()
        .await?
        .into_iter()
        .map(SkillResponse::from)
        .collect();
    Ok(Json(SuccessResponse::new(SkillListResponse { skills })))
}

#[utoipa::path(
    post,
    path = "/agent/skills",
    request_body = SkillWriteRequest,
    responses(
        (status = 200, body = SuccessResponse<SkillResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "createAgentSkill"
)]
pub async fn create_agent_skill(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    JsonBody(request): JsonBody<SkillWriteRequest>,
) -> ApiResult<SkillResponse> {
    let skill = state.skills()?.create(request.into()).await?;
    Ok(Json(SuccessResponse::new(skill.into())))
}

#[utoipa::path(
    patch,
    path = "/agent/skills/{skillId}",
    params(("skillId" = String, Path)),
    request_body = SkillUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<SkillResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "updateAgentSkill"
)]
pub async fn update_agent_skill(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(skill_id): Path<String>,
    JsonBody(request): JsonBody<SkillUpdateRequest>,
) -> ApiResult<SkillResponse> {
    let id = parse_skill_id(&skill_id)?;
    let skill = state.skills()?.update(id, request.into()).await?;
    Ok(Json(SuccessResponse::new(skill.into())))
}

#[utoipa::path(
    delete,
    path = "/agent/skills/{skillId}",
    params(("skillId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "deleteAgentSkill"
)]
pub async fn delete_agent_skill(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(skill_id): Path<String>,
) -> ApiResult<SuccessFlagResponse> {
    let id = parse_skill_id(&skill_id)?;
    state.skills()?.delete(id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    get,
    path = "/agent/conversations/{articleId}",
    params(
        ("articleId" = String, Path),
        ConversationQuery
    ),
    responses(
        (status = 200, body = SuccessResponse<ConversationHistoryResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "getConversationHistory"
)]
pub async fn get_conversation_history(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(article_id): Path<String>,
    Query(query): Query<ConversationQuery>,
) -> ApiResult<ConversationHistoryResponse> {
    if article_id.is_empty() {
        return Err(AppError::InvalidInput("Article ID is required".to_owned()));
    }
    let parsed_id = Uuid::parse_str(&article_id)
        .map_err(|_| AppError::InvalidInput("Invalid article ID format".to_owned()))?;
    let limit = query
        .limit
        .as_deref()
        .and_then(|value| value.parse::<i64>().ok())
        .filter(|value| *value > 0)
        .unwrap_or(50);
    let messages = state
        .chat()?
        .conversation_history(parsed_id, limit)
        .await?
        .into_iter()
        .map(Into::into)
        .collect::<Vec<_>>();
    let total = messages.len();
    Ok(Json(SuccessResponse::new(ConversationHistoryResponse {
        messages,
        article_id,
        total,
    })))
}

#[utoipa::path(
    delete,
    path = "/agent/conversations/{articleId}",
    params(("articleId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "clearConversationHistory"
)]
pub async fn clear_conversation_history(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(article_id): Path<String>,
) -> ApiResult<SuccessFlagResponse> {
    if article_id.is_empty() {
        return Err(AppError::InvalidInput("Article ID is required".to_owned()));
    }
    let article_id = Uuid::parse_str(&article_id)
        .map_err(|_| AppError::InvalidInput("Invalid article ID format".to_owned()))?;
    state.chat()?.clear_conversation_history(article_id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    get,
    path = "/agent/artifacts/{articleId}/pending",
    params(("articleId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<PendingArtifactsResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "getPendingArtifacts"
)]
pub async fn get_pending_artifacts(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(article_id): Path<String>,
) -> ApiResult<PendingArtifactsResponse> {
    let article_id = Uuid::parse_str(&article_id)
        .map_err(|_| AppError::InvalidInput("Invalid article ID".to_owned()))?;
    let artifacts = state
        .chat()?
        .pending_artifacts(article_id)
        .await?
        .into_iter()
        .map(Into::into)
        .collect();
    Ok(Json(SuccessResponse::new(PendingArtifactsResponse {
        artifacts,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/artifacts/{messageId}/accept",
    params(("messageId" = String, Path)),
    request_body(content = ArtifactFeedbackRequest, content_type = "application/json"),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "acceptArtifact"
)]
pub async fn accept_artifact(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(message_id): Path<String>,
    body: Bytes,
) -> ApiResult<SuccessFlagResponse> {
    let message_id = parse_message_id(&message_id)?;
    let request = optional_feedback(&body)?;
    state
        .chat()?
        .accept_artifact(message_id, request.feedback)
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    post,
    path = "/agent/artifacts/{messageId}/reject",
    params(("messageId" = String, Path)),
    request_body(content = ArtifactFeedbackRequest, content_type = "application/json"),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "agent",
    operation_id = "rejectArtifact"
)]
pub async fn reject_artifact(
    _authenticated: AuthenticatedAccount,
    State(state): State<AgentState>,
    Path(message_id): Path<String>,
    body: Bytes,
) -> ApiResult<SuccessFlagResponse> {
    let message_id = parse_message_id(&message_id)?;
    let request = optional_feedback(&body)?;
    state
        .chat()?
        .reject_artifact(message_id, request.feedback)
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

fn parse_connector_id(connector_id: &str) -> Result<Uuid, AppError> {
    Uuid::parse_str(connector_id)
        .map_err(|_| AppError::InvalidInput("Invalid connector ID".to_owned()))
}

fn parse_skill_id(skill_id: &str) -> Result<Uuid, AppError> {
    Uuid::parse_str(skill_id).map_err(|_| AppError::InvalidInput("Invalid skill ID".to_owned()))
}

fn parse_message_id(message_id: &str) -> Result<Uuid, AppError> {
    Uuid::parse_str(message_id).map_err(|_| AppError::InvalidInput("Invalid message ID".to_owned()))
}

fn optional_feedback(body: &[u8]) -> Result<ArtifactFeedbackRequest, AppError> {
    if body.is_empty() {
        return Ok(ArtifactFeedbackRequest::default());
    }
    serde_json::from_slice(body)
        .map_err(|_| AppError::InvalidInput("Invalid request body".to_owned()))
}
