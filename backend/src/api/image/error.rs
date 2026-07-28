use axum::{
    Json,
    http::StatusCode,
    response::{IntoResponse, Response},
};

use crate::error::{AppError, ErrorEnvelope};

pub struct ImageApiError {
    error: AppError,
    validation: bool,
}

impl From<AppError> for ImageApiError {
    fn from(error: AppError) -> Self {
        Self {
            error,
            validation: false,
        }
    }
}

impl ImageApiError {
    pub fn validation(error: AppError) -> Self {
        Self {
            error,
            validation: true,
        }
    }
}

impl IntoResponse for ImageApiError {
    fn into_response(self) -> Response {
        let validation = self.validation;
        let (status, code, message) = match self.error {
            AppError::InvalidInput(message) if validation => {
                (StatusCode::BAD_REQUEST, "VALIDATION_ERROR", message)
            }
            AppError::InvalidInput(message) => (StatusCode::BAD_REQUEST, "INVALID_INPUT", message),
            AppError::Unauthorized => (
                StatusCode::UNAUTHORIZED,
                "UNAUTHORIZED",
                "authentication required".to_owned(),
            ),
            AppError::Forbidden => (
                StatusCode::FORBIDDEN,
                "FORBIDDEN",
                "access forbidden".to_owned(),
            ),
            AppError::NotFound => (
                StatusCode::NOT_FOUND,
                "NOT_FOUND",
                "resource not found".to_owned(),
            ),
            AppError::Conflict(message) => (StatusCode::CONFLICT, "ALREADY_EXISTS", message),
            AppError::Database => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "DATABASE_ERROR",
                "Database error".to_owned(),
            ),
            AppError::External => (
                StatusCode::BAD_GATEWAY,
                "EXTERNAL_SERVICE_ERROR",
                "external service operation failed".to_owned(),
            ),
            AppError::Internal => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "INTERNAL_ERROR",
                "Internal server error".to_owned(),
            ),
        };
        (
            status,
            Json(ErrorEnvelope {
                error: message,
                code,
                details: None,
            }),
        )
            .into_response()
    }
}
