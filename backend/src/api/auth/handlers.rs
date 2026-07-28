use axum::{
    Json,
    extract::{FromRequest, Multipart, Request, State},
    http::{
        HeaderMap, StatusCode,
        header::{AUTHORIZATION, CONTENT_TYPE},
    },
    response::{IntoResponse, Response},
};
use serde::de::DeserializeOwned;

use crate::{
    api::{
        auth::{
            dto::{
                AuthErrorResponse, DeleteAccountRequest, LoginRequest, LoginResponse,
                MessageResponse, RegisterRequest, UpdateAccountRequest, UpdatePasswordRequest,
                Validate, ValidationIssue,
            },
            state::AuthState,
        },
        response::SuccessResponse,
    },
    core::auth::AccountId,
    error::AppError,
};

const MAX_AUTH_MULTIPART_FIELDS: usize = 3;
const MAX_AUTH_MULTIPART_FIELD_BYTES: usize = 1_024;

pub type AuthResult<T> = Result<T, AuthApiError>;

pub struct AuthApiError {
    status: StatusCode,
    body: AuthErrorResponse,
}

impl AuthApiError {
    fn invalid_body() -> Self {
        Self {
            status: StatusCode::BAD_REQUEST,
            body: AuthErrorResponse {
                error: "Invalid request body".to_owned(),
                code: "INVALID_INPUT",
                details: None,
            },
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
            body: AuthErrorResponse {
                error: first_error,
                code: "VALIDATION_ERROR",
                details: Some(details),
            },
        }
    }

    fn unauthorized(message: &'static str) -> Self {
        Self {
            status: StatusCode::UNAUTHORIZED,
            body: AuthErrorResponse {
                error: message.to_owned(),
                code: "UNAUTHORIZED",
                details: None,
            },
        }
    }
}

impl From<AppError> for AuthApiError {
    fn from(error: AppError) -> Self {
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
            AppError::Database => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "Database error".to_owned(),
                "DATABASE_ERROR",
            ),
            AppError::External => (
                StatusCode::BAD_GATEWAY,
                "external service error".to_owned(),
                "EXTERNAL_SERVICE_ERROR",
            ),
            AppError::Internal => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error".to_owned(),
                "INTERNAL_ERROR",
            ),
        };
        Self {
            status,
            body: AuthErrorResponse {
                error: message,
                code,
                details: None,
            },
        }
    }
}

impl IntoResponse for AuthApiError {
    fn into_response(self) -> Response {
        (self.status, Json(self.body)).into_response()
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
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn login(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<LoginResponse>>> {
    let body: LoginRequest = parse_json_and_validate(request).await?;
    let result = state.service().login(body.into()).await?;
    Ok(Json(SuccessResponse::new(result.into())))
}

#[utoipa::path(
    post,
    path = "/auth/register",
    operation_id = "authRegister",
    tag = "auth",
    request_body = RegisterRequest,
    responses(
        (status = 201, body = SuccessResponse<MessageResponse>),
        (status = 400, body = AuthErrorResponse),
        (status = 409, body = AuthErrorResponse),
        (status = 500, body = AuthErrorResponse)
    )
)]
pub async fn register(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<(StatusCode, Json<SuccessResponse<MessageResponse>>)> {
    let body: RegisterRequest = parse_json_and_validate(request).await?;
    state.service().register(body.into()).await?;
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
    let account_id = authenticated_account_id(request.headers(), &state)?;
    let body: UpdateAccountRequest = parse_and_validate(request, &["name", "email"]).await?;
    state
        .service()
        .update_account(account_id, body.into())
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
    security(("bearerAuth" = [])),
    request_body(
        content(
            (UpdatePasswordRequest = "application/json"),
            (UpdatePasswordRequest = "multipart/form-data")
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
pub async fn update_password(
    State(state): State<AuthState>,
    request: Request,
) -> AuthResult<Json<SuccessResponse<MessageResponse>>> {
    let account_id = authenticated_account_id(request.headers(), &state)?;
    let body: UpdatePasswordRequest = parse_and_validate(
        request,
        &["currentPassword", "newPassword", "confirmPassword"],
    )
    .await?;
    state
        .service()
        .update_password(account_id, body.into())
        .await?;
    Ok(Json(SuccessResponse::new(MessageResponse {
        message: "Password updated successfully",
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
    let account_id = authenticated_account_id(request.headers(), &state)?;
    let body: DeleteAccountRequest = parse_and_validate(request, &["password"]).await?;
    state
        .service()
        .delete_account(account_id, &body.password)
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

fn authenticated_account_id(headers: &HeaderMap, state: &AuthState) -> AuthResult<AccountId> {
    let header = headers
        .get(AUTHORIZATION)
        .ok_or_else(|| AuthApiError::unauthorized("Not authenticated"))?;
    let header = header
        .to_str()
        .map_err(|_| AuthApiError::unauthorized("Invalid token format"))?;
    let token = header
        .strip_prefix("Bearer ")
        .filter(|token| !token.is_empty())
        .ok_or_else(|| AuthApiError::unauthorized("Invalid token format"))?;

    state
        .service()
        .account_id_from_token(token)
        .map_err(|_| AuthApiError::unauthorized("Invalid or expired token"))
}
