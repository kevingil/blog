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
        ArtifactFeedbackRequest, ChatRequest, ChatRequestResponse, ConversationHistoryResponse,
        ConversationQuery, PendingArtifactsResponse, SuccessFlagResponse,
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
