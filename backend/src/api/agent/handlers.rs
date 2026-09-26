use axum::{
    Json,
    body::Bytes,
    extract::{Path, Query, State},
};
use uuid::Uuid;

use crate::{
    api::{auth::AuthenticatedAccount, request::JsonBody, response::SuccessResponse},
    error::AppError,
};

use super::{
    dto::{
        AgentToolListResponse, AgentToolResponse, ArtifactFeedbackRequest, ChatRequest,
        ChatRequestResponse, ConnectorListResponse, ConnectorRefreshResponse, ConnectorResponse,
        ConnectorUpdateRequest, ConnectorWriteRequest, ConversationHistoryResponse,
        ConversationQuery, ConversationTurnRequest, ConversationTurnResponse,
        PendingArtifactsResponse, SuccessFlagResponse,
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
