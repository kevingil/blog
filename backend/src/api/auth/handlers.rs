use std::time::Duration;

use axum::{
    Json,
    extract::{FromRequest, Multipart, Path, Request, State},
    http::{
        HeaderMap, HeaderValue, StatusCode,
        header::{AUTHORIZATION, CONTENT_TYPE, RETRY_AFTER},
    },
    response::{IntoResponse, Response},
};
use serde::de::DeserializeOwned;
use uuid::Uuid;

use crate::{
    api::{
        auth::{
            dto::{
                AuthErrorResponse, CreateMcpTokenRequest, DeleteAccountRequest, LoginRequest,
                LoginResponse, McpTokenCreatedResponse, McpTokenResponse, MessageResponse,
                PasswordUpdateResponse, RegisterRequest, UpdateAccountRequest,
                UpdatePasswordRequest, Validate, ValidationIssue,
            },
            state::{AuthState, bearer_token},
        },
        response::SuccessResponse,
    },
    core::auth::Account,
    error::AppError,
    setup::SUPERUSER_ROLE,
};

const MAX_AUTH_MULTIPART_FIELDS: usize = 3;
const MAX_AUTH_MULTIPART_FIELD_BYTES: usize = 1_024;

pub type AuthResult<T> = Result<T, AuthApiError>;

pub struct AuthApiError {
    status: StatusCode,
    body: AuthErrorResponse,
    cause: String,
    retry_after: Option<Duration>,
}

impl AuthApiError {
    fn invalid_body() -> Self {
        Self {
            status: StatusCode::BAD_REQUEST,
            cause: "Invalid request body".to_owned(),
            body: AuthErrorResponse {
                error: "Invalid request body".to_owned(),
                code: "INVALID_INPUT",
                details: None,
            },
            retry_after: None,
        }
    }

    fn too_many_attempts(retry_after: Duration) -> Self {
        let message = "Too many failed sign-in attempts. Try again later.";
        Self {
            status: StatusCode::TOO_MANY_REQUESTS,
            cause: message.to_owned(),
            body: AuthErrorResponse {
                error: message.to_owned(),
                code: "TOO_MANY_REQUESTS",
                details: None,
            },
            retry_after: Some(retry_after),
        }
    }

    fn validation(issues: Vec<ValidationIssue>) -> Self {
        let first_error = match issues.first() {
            Some(issue) => format!("{}: {}", issue.field, issue.message),
            None => "validation failed".to_owned(),
        };
        let details = issues
            .into_iter()
            .map(|issue| (issue.field.to_owned(), issue.message))
            .collect();
        Self {
            status: StatusCode::BAD_REQUEST,
            cause: first_error.clone(),
            body: AuthErrorResponse {
                error: first_error,
                code: "VALIDATION_ERROR",
                details: Some(details),
            },
            retry_after: None,
        }
    }

    fn unauthorized(message: &'static str) -> Self {
        Self {
            status: StatusCode::UNAUTHORIZED,
            cause: message.to_owned(),
            body: AuthErrorResponse {
                error: message.to_owned(),
                code: "UNAUTHORIZED",
                details: None,
            },
            retry_after: None,
        }
    }
}

impl From<AppError> for AuthApiError {
    fn from(error: AppError) -> Self {
        let cause = error.to_string();
        let (status, message, code) = match error {
            AppError::InvalidInput(message) => (StatusCode::BAD_REQUEST, message, "INVALID_INPUT"),
            AppError::Unauthorized => (
                StatusCode::UNAUTHORIZED,
                "unauthorized".to_owned(),
                "UNAUTHORIZED",
            ),
            AppError::Forbidden => (
                StatusCode::FORBIDDEN,
                "access forbidden".to_owned(),
                "FORBIDDEN",
            ),
            AppError::NotFound => (
                StatusCode::NOT_FOUND,
                "resource not found".to_owned(),
                "NOT_FOUND",
            ),
            AppError::Conflict(_) => (
                StatusCode::CONFLICT,
                "resource already exists".to_owned(),
                "ALREADY_EXISTS",
            ),
            AppError::Database(_) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "Database error".to_owned(),
                "DATABASE_ERROR",
            ),
            AppError::External(_) => (
                StatusCode::BAD_GATEWAY,
                "external service error".to_owned(),
                "EXTERNAL_SERVICE_ERROR",
            ),
            AppError::Internal(_) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error".to_owned(),
                "INTERNAL_ERROR",
            ),
        };
        Self {
            status,
            cause,
            body: AuthErrorResponse {
                error: message,
                code,
                details: None,
            },
            retry_after: None,
        }
    }
}

impl IntoResponse for AuthApiError {
    fn into_response(self) -> Response {
        crate::error::log_error_response(self.status, self.body.code, &self.cause);
        let mut response = (self.status, Json(self.body)).into_response();
        if let Some(retry_after) = self.retry_after {
            let seconds = retry_after.as_secs().saturating_add(1);
            response
                .headers_mut()
                .insert(RETRY_AFTER, HeaderValue::from(seconds));
        }
        response
    }
}

#[utoipa::path(
    post,
    path = "/auth/login",
    operation_id = "authLogin",
    tag = "auth",
    request_body = LoginRequest,
    responses(
        (status = 200, body = SuccessResponse<LoginResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 429, body = AuthErrorResponse,
            description = "Too many failed attempts for this email; see Retry-After"),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn login(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<LoginResponse>>> {
    let body: LoginRequest = parse_json_and_validate(request).await?;
    let throttle = state.login_throttle();
    if let Some(retry_after) = throttle.retry_after(&body.email) {
        return Err(AuthApiError::too_many_attempts(retry_after));
    }
    let email = body.email.clone();
    match state.service().login(body.into()).await {
        Ok(result) => {
            throttle.record_success(&email);
            Ok(Json(SuccessResponse::new(result.into())))
        }
        Err(AppError::Unauthorized) => {
            throttle.record_failure(&email);
            Err(AppError::Unauthorized.into())
        }
        Err(error) => Err(error.into()),
    }
}

#[utoipa::path(
    post,
    path = "/auth/refresh",
    operation_id = "authRefresh",
    tag = "auth",
    security(("bearerAuth" = [])),
    responses(
        (status = 200, body = SuccessResponse<LoginResponse>,
            description = "A new token and the current account details"),
        (status = 401, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn refresh(
    State(state): State<AuthState>,
    headers: HeaderMap,
) -> AuthResult<Json<SuccessResponse<LoginResponse>>> {
    let account = authenticated_account(&headers, &state).await?;
    let result = state.service().issue_session(account)?;
    Ok(Json(SuccessResponse::new(result.into())))
}

#[utoipa::path(
    post,
    path = "/auth/register",
    operation_id = "authRegister",
    tag = "auth",
    security((), ("bearerAuth" = [])),
    request_body = RegisterRequest,
    responses(
        (status = 201, body = SuccessResponse<MessageResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 403, body = AuthErrorResponse,
            description = "Accounts already exist and the caller is not an admin"),
        (status = 409, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn register(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<(StatusCode, Json<SuccessResponse<MessageResponse>>)> {
    let requested_by = state.optional_account(request.headers()).await?;
    let body: RegisterRequest = parse_json_and_validate(request).await?;
    state
        .service()
        .register(body.into(), requested_by.as_ref())
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(MessageResponse {
            message: "User registered successfully",
        })),
    ))
}

#[utoipa::path(
    post,
    path = "/auth/logout",
    operation_id = "authLogout",
    tag = "auth",
    responses((status = 200, body = SuccessResponse<MessageResponse>))
)]
pub async fn logout() -> Json<SuccessResponse<MessageResponse>> {
    Json(SuccessResponse::new(MessageResponse {
        message: "Logged out successfully",
    }))
}

#[utoipa::path(
    put,
    path = "/auth/account",
    operation_id = "authUpdateAccount",
    tag = "auth",
    security(("bearerAuth" = [])),
    request_body(
        content(
            (UpdateAccountRequest = "application/json"),
            (UpdateAccountRequest = "multipart/form-data")
        )
    ),
    responses(
        (status = 200, body = SuccessResponse<MessageResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 404, body = AuthErrorResponse),
        (status = 409, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn update_account(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<MessageResponse>>> {
    let account = authenticated_account(request.headers(), &state).await?;
    let body: UpdateAccountRequest = parse_and_validate(request, &["name", "email"]).await?;
    state
        .service()
        .update_account(account.id, body.into())
        .await?;
    Ok(Json(SuccessResponse::new(MessageResponse {
        message: "Account updated successfully",
    })))
}

#[utoipa::path(
    put,
    path = "/auth/password",
    operation_id = "authUpdatePassword",
    tag = "auth",
    description = "Requires a JWT session. An MCP access token cannot change the password.",
    security(("bearerAuth" = [])),
    request_body(
        content(
            (UpdatePasswordRequest = "application/json"),
            (UpdatePasswordRequest = "multipart/form-data")
        )
    ),
    responses(
        (status = 200, body = SuccessResponse<PasswordUpdateResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 404, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn update_password(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<PasswordUpdateResponse>>> {
    let account = session_account(request.headers(), &state).await?;
    let body: UpdatePasswordRequest = parse_and_validate(request, &[
        "currentPassword",
        "newPassword",
        "confirmPassword",
    ])
    .await?;
    let token = state
        .service()
        .update_password(account.id, body.into())
        .await?;
    Ok(Json(SuccessResponse::new(PasswordUpdateResponse {
        message: "Password updated successfully",
        token,
    })))
}

#[utoipa::path(
    delete,
    path = "/auth/account",
    operation_id = "authDeleteAccount",
    tag = "auth",
    security(("bearerAuth" = [])),
    request_body(
        content(
            (DeleteAccountRequest = "application/json"),
            (DeleteAccountRequest = "multipart/form-data")
        )
    ),
    responses(
        (status = 200, body = SuccessResponse<MessageResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 404, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn delete_account(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<MessageResponse>>> {
    let account = authenticated_account(request.headers(), &state).await?;
    let body: DeleteAccountRequest = parse_and_validate(request, &["password"]).await?;
    state
        .service()
        .delete_account(account.id, &body.password)
        .await?;
    Ok(Json(SuccessResponse::new(MessageResponse {
        message: "Account deleted successfully",
    })))
}

async fn parse_json_and_validate<T>(request: Request) -> AuthResult<T>
where
    T: DeserializeOwned + Validate,
{
    parse_and_validate(request, &[]).await
}

async fn parse_and_validate<T>(request: Request, multipart_fields: &[&str]) -> AuthResult<T>
where
    T: DeserializeOwned + Validate,
{
    let body: T = parse_body(request, multipart_fields).await?;
    let issues = body.validate();
    if issues.is_empty() {
        Ok(body)
    } else {
        Err(AuthApiError::validation(issues))
    }
}

async fn parse_body<T: DeserializeOwned>(
    request: Request,
    multipart_fields: &[&str],
) -> AuthResult<T> {
    let content_type = request
        .headers()
        .get(CONTENT_TYPE)
        .and_then(|value| value.to_str().ok())
        .unwrap_or_default();

    if content_type
        .split(';')
        .next()
        .is_some_and(|value| value.trim().eq_ignore_ascii_case("application/json"))
    {
        let Json(body) = Json::<T>::from_request(request, &())
            .await
            .map_err(|_| AuthApiError::invalid_body())?;
        return Ok(body);
    }

    if content_type
        .split(';')
        .next()
        .is_some_and(|value| value.trim().eq_ignore_ascii_case("multipart/form-data"))
    {
        if multipart_fields.is_empty() {
            return Err(AuthApiError::invalid_body());
        }
        let mut multipart = Multipart::from_request(request, &())
            .await
            .map_err(|_| AuthApiError::invalid_body())?;
        let mut object = serde_json::Map::new();
        let mut field_count = 0_usize;
        while let Some(field) = multipart
            .next_field()
            .await
            .map_err(|_| AuthApiError::invalid_body())?
        {
            field_count += 1;
            if field_count > MAX_AUTH_MULTIPART_FIELDS || field.file_name().is_some() {
                return Err(AuthApiError::invalid_body());
            }
            let name = field
                .name()
                .map(ToOwned::to_owned)
                .ok_or_else(AuthApiError::invalid_body)?;
            if !multipart_fields.contains(&name.as_str()) || object.contains_key(&name) {
                return Err(AuthApiError::invalid_body());
            }
            let value = field
                .text()
                .await
                .map_err(|_| AuthApiError::invalid_body())?;
            if value.len() > MAX_AUTH_MULTIPART_FIELD_BYTES {
                return Err(AuthApiError::invalid_body());
            }
            object.insert(name, serde_json::Value::String(value));
        }
        return serde_json::from_value(serde_json::Value::Object(object))
            .map_err(|_| AuthApiError::invalid_body());
    }

    Err(AuthApiError::invalid_body())
}

#[utoipa::path(
    post,
    path = "/auth/mcp-tokens",
    operation_id = "authCreateMcpToken",
    tag = "auth",
    description = "Creates an MCP access token for the authenticated admin. The plaintext token is returned once. Requires a JWT session; an MCP token cannot call this route.",
    security(("bearerAuth" = [])),
    request_body = CreateMcpTokenRequest,
    responses(
        (status = 201, body = SuccessResponse<McpTokenCreatedResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 403, body = AuthErrorResponse,
            description = "The JWT session is not an admin"),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn create_mcp_token(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<(StatusCode, Json<SuccessResponse<McpTokenCreatedResponse>>)> {
    let account = admin_session(request.headers(), &state).await?;
    let body: CreateMcpTokenRequest = parse_json_and_validate(request).await?;
    let issued = state
        .service()
        .create_mcp_token(account.id, &body.name)
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(issued.into())),
    ))
}

#[utoipa::path(
    get,
    path = "/auth/mcp-tokens",
    operation_id = "authListMcpTokens",
    tag = "auth",
    description = "Lists the authenticated admin's MCP access tokens. The plaintext and the hash are omitted. Requires a JWT session.",
    security(("bearerAuth" = [])),
    responses(
        (status = 200, body = SuccessResponse<Vec<McpTokenResponse>>),
        (status = 401, body = AuthErrorResponse),
        (status = 403, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn list_mcp_tokens(
    State(state): State<AuthState>,
    headers: HeaderMap,
) -> AuthResult<Json<SuccessResponse<Vec<McpTokenResponse>>>> {
    let account = admin_session(&headers, &state).await?;
    let tokens = state.service().list_mcp_tokens(account.id).await?;
    Ok(Json(SuccessResponse::new(
        tokens.into_iter().map(McpTokenResponse::from).collect(),
    )))
}

#[utoipa::path(
    delete,
    path = "/auth/mcp-tokens/{id}",
    operation_id = "authRevokeMcpToken",
    tag = "auth",
    description = "Revokes one of the authenticated admin's MCP access tokens. Requires a JWT session.",
    params(("id" = Uuid, Path)),
    security(("bearerAuth" = [])),
    responses(
        (status = 200, body = SuccessResponse<McpTokenResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 401, body = AuthErrorResponse),
        (status = 403, body = AuthErrorResponse),
        (status = 404, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn revoke_mcp_token(
    State(state): State<AuthState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> AuthResult<Json<SuccessResponse<McpTokenResponse>>> {
    let account = admin_session(&headers, &state).await?;
    let id = Uuid::parse_str(&id)
        .map_err(|_| AuthApiError::from(AppError::InvalidInput("Invalid token id".to_owned())))?;
    let token = state.service().revoke_mcp_token(account.id, id).await?;
    Ok(Json(SuccessResponse::new(token.into())))
}

enum CredentialKind {
    /// JWT session or MCP access token.
    Bearer,
    /// JWT session only. MCP access tokens are rejected.
    Session,
}

async fn authenticated_account(headers: &HeaderMap, state: &AuthState) -> AuthResult<Account> {
    account_from_headers(headers, state, CredentialKind::Bearer).await
}

async fn session_account(headers: &HeaderMap, state: &AuthState) -> AuthResult<Account> {
    account_from_headers(headers, state, CredentialKind::Session).await
}

async fn admin_session(headers: &HeaderMap, state: &AuthState) -> AuthResult<Account> {
    let account = session_account(headers, state).await?;
    if account.role != SUPERUSER_ROLE {
        return Err(AppError::Forbidden.into());
    }
    Ok(account)
}

async fn account_from_headers(
    headers: &HeaderMap,
    state: &AuthState,
    kind: CredentialKind,
) -> AuthResult<Account> {
    if !headers.contains_key(AUTHORIZATION) {
        return Err(AuthApiError::unauthorized("Not authenticated"));
    }
    let token =
        bearer_token(headers).ok_or_else(|| AuthApiError::unauthorized("Invalid token format"))?;
    let result = match kind {
        CredentialKind::Bearer => state.service().authenticate_bearer(token).await,
        CredentialKind::Session => state.service().authenticate(token).await,
    };
    match result {
        Ok(account) => Ok(account),
        Err(AppError::Unauthorized) => Err(AuthApiError::unauthorized("Invalid or expired token")),
        Err(error) => Err(error.into()),
    }
}
