use axum::{
    Json,
    extract::{FromRef, FromRequestParts},
    http::{StatusCode, header::AUTHORIZATION, request::Parts},
    response::{IntoResponse, Response},
};

use crate::{
    api::auth::AuthState,
    core::auth::AccountId,
    error::{AppError, ErrorEnvelope},
};

pub struct ProfileApiError {
    status: StatusCode,
    body: ErrorEnvelope,
}

impl ProfileApiError {
    pub fn invalid_body() -> Self {
        Self::new(
            StatusCode::BAD_REQUEST,
            "Invalid request body",
            "INVALID_INPUT",
        )
    }

    pub fn forbidden(message: &'static str) -> Self {
        Self::new(StatusCode::FORBIDDEN, message, "FORBIDDEN")
    }

    fn unauthorized(message: &'static str) -> Self {
        Self::new(StatusCode::UNAUTHORIZED, message, "UNAUTHORIZED")
    }

    fn new(status: StatusCode, error: impl Into<String>, code: &'static str) -> Self {
        Self {
            status,
            body: ErrorEnvelope {
                error: error.into(),
                code,
                details: None,
            },
        }
    }
}

impl From<AppError> for ProfileApiError {
    fn from(error: AppError) -> Self {
        match error {
            AppError::InvalidInput(message) => {
                Self::new(StatusCode::BAD_REQUEST, message, "INVALID_INPUT")
            }
            AppError::Unauthorized => Self::unauthorized("Not authenticated"),
            AppError::Forbidden => {
                Self::new(StatusCode::FORBIDDEN, "access forbidden", "FORBIDDEN")
            }
            AppError::NotFound => {
                Self::new(StatusCode::NOT_FOUND, "resource not found", "NOT_FOUND")
            }
            AppError::Conflict(message) => {
                Self::new(StatusCode::CONFLICT, message, "ALREADY_EXISTS")
            }
            AppError::Database => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Database error",
                "DATABASE_ERROR",
            ),
            AppError::External => Self::new(
                StatusCode::BAD_GATEWAY,
                "external service operation failed",
                "EXTERNAL_SERVICE_ERROR",
            ),
            AppError::Internal => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error",
                "INTERNAL_ERROR",
            ),
        }
    }
}

impl IntoResponse for ProfileApiError {
    fn into_response(self) -> Response {
        (self.status, Json(self.body)).into_response()
    }
}

pub struct ProfileAuthenticated(pub AccountId);

impl<S> FromRequestParts<S> for ProfileAuthenticated
where
    S: Send + Sync,
    AuthState: FromRef<S>,
{
    type Rejection = ProfileApiError;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let header = parts
            .headers
            .get(AUTHORIZATION)
            .ok_or_else(|| ProfileApiError::unauthorized("Not authenticated"))?
            .to_str()
            .map_err(|_| ProfileApiError::unauthorized("Invalid token format"))?;
        if !header.starts_with("Bearer ") {
            return Err(ProfileApiError::unauthorized("Invalid token format"));
        }
        let account_id = AuthState::from_ref(state)
            .authenticate(&parts.headers)
            .map_err(|_| ProfileApiError::unauthorized("Invalid or expired token"))?;
        Ok(Self(account_id))
    }
}
