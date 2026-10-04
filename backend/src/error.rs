use axum::{
    Json,
    http::StatusCode,
    response::{IntoResponse, Response},
};
use serde::Serialize;
use std::collections::BTreeMap;
use thiserror::Error;
use utoipa::ToSchema;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("invalid input: {0}")]
    InvalidInput(String),
    #[error("authentication required")]
    Unauthorized,
    #[error("forbidden")]
    Forbidden,
    #[error("resource not found")]
    NotFound,
    #[error("conflict: {0}")]
    Conflict(String),
    #[error("database operation failed: {0}")]
    Database(String),
    #[error("external service operation failed: {0}")]
    External(String),
    #[error("internal server error: {0}")]
    Internal(String),
}

impl AppError {
    pub fn database(error: impl std::fmt::Display) -> Self {
        Self::Database(error.to_string())
    }

    pub fn external(error: impl std::fmt::Display) -> Self {
        Self::External(error.to_string())
    }

    pub fn internal(error: impl std::fmt::Display) -> Self {
        Self::Internal(error.to_string())
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ErrorEnvelope {
    pub error: String,
    pub code: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<BTreeMap<String, String>>,
}

pub(crate) fn log_error_response(status: StatusCode, code: &str, error: impl std::fmt::Display) {
    if status.is_server_error() {
        tracing::error!(%status, code, error = %error, "request failed");
    } else {
        tracing::warn!(%status, code, error = %error, "request failed");
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, code) = match &self {
            Self::InvalidInput(_) => (StatusCode::BAD_REQUEST, "INVALID_INPUT"),
            Self::Unauthorized => (StatusCode::UNAUTHORIZED, "UNAUTHORIZED"),
            Self::Forbidden => (StatusCode::FORBIDDEN, "FORBIDDEN"),
            Self::NotFound => (StatusCode::NOT_FOUND, "NOT_FOUND"),
            Self::Conflict(_) => (StatusCode::CONFLICT, "CONFLICT"),
            Self::Database(_) | Self::External(_) | Self::Internal(_) => {
                (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL_ERROR")
            }
        };
        log_error_response(status, code, &self);

        let message = if status.is_server_error() {
            "internal server error".to_owned()
        } else {
            self.to_string()
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

#[cfg(test)]
mod tests {
    use super::*;
    use http_body_util::BodyExt;
    use serde_json::Value;

    async fn envelope(error: AppError) -> Result<(StatusCode, Value), Box<dyn std::error::Error>> {
        let response = error.into_response();
        let status = response.status();
        let bytes = response.into_body().collect().await?.to_bytes();
        let body = serde_json::from_slice(&bytes)?;
        Ok((status, body))
    }

    #[tokio::test]
    async fn server_errors_keep_the_cause_out_of_the_response()
    -> Result<(), Box<dyn std::error::Error>> {
        let (status, body) =
            envelope(AppError::database("relation \"uploads\" does not exist")).await?;
        assert_eq!(status, StatusCode::INTERNAL_SERVER_ERROR);
        assert_eq!(body["error"], "internal server error");
        assert_eq!(body["code"], "INTERNAL_ERROR");
        assert!(body["details"].is_null());
        assert!(!body.to_string().contains("uploads"));

        let (status, body) = envelope(AppError::internal("length limit exceeded")).await?;
        assert_eq!(status, StatusCode::INTERNAL_SERVER_ERROR);
        assert_eq!(body["error"], "internal server error");
        assert!(!body.to_string().contains("length limit"));
        Ok(())
    }
}
