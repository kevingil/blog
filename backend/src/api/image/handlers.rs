use axum::{
    Json,
    extract::{Path, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::{
    api::{auth::AuthenticatedAccount, response::SuccessResponse},
    error::AppError,
};

use super::{
    dto::{
        GenerateImageRequest, GenerateImageResponse, ImageGenerationResponse, ImageGenerationStatus,
    },
    error::ImageApiError,
    state::{ImageGenerationJob, ImageState},
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, ImageApiError>;

#[utoipa::path(
    post,
    path = "/images/generate",
    request_body = GenerateImageRequest,
    responses(
        (status = 202, body = SuccessResponse<GenerateImageResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "images",
    operation_id = "generateImage"
)]
pub async fn generate_image(
    _authenticated: AuthenticatedAccount,
    State(state): State<ImageState>,
    body: Result<Json<GenerateImageRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<GenerateImageResponse>>), ImageApiError> {
    let Json(request) = body.map_err(|_| {
        ImageApiError::from(AppError::InvalidInput("Invalid request body".to_owned()))
    })?;
    let request = request.validate().map_err(ImageApiError::validation)?;
    let request_id = Uuid::new_v4().to_string();
    let image = state
        .service()
        .create(request.persistence_request(
            state.queue().provider(),
            state.queue().model_name(),
            request_id.clone(),
        ))
        .await?;
    let job = ImageGenerationJob {
        image_id: image.id,
        request_id: request_id.clone(),
        article_id: request.article_id,
        prompt: request.prompt,
        generate_prompt: request.generate_prompt,
    };
    if state.queue().enqueue(job).await.is_err() {
        state
            .service()
            .mark_failed(image.id, "failed to enqueue image generation".to_owned())
            .await
            .map_err(|_| ImageApiError::from(AppError::Internal))?;
        return Err(ImageApiError::from(AppError::External));
    }
    Ok((
        StatusCode::ACCEPTED,
        Json(SuccessResponse::new(GenerateImageResponse { request_id })),
    ))
}

#[utoipa::path(
    get,
    path = "/images/{requestId}",
    params(("requestId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<ImageGenerationResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "images",
    operation_id = "getImageGeneration"
)]
pub async fn get_image_generation(
    _authenticated: AuthenticatedAccount,
    State(state): State<ImageState>,
    Path(request_id): Path<String>,
) -> ApiResult<ImageGenerationResponse> {
    if request_id.trim().is_empty() {
        return Err(AppError::InvalidInput("Request ID is required".to_owned()).into());
    }
    Ok(Json(SuccessResponse::new(
        state.service().get_by_request_id(&request_id).await?.into(),
    )))
}

#[utoipa::path(
    get,
    path = "/images/{requestId}/status",
    params(("requestId" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<ImageGenerationStatus>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "images",
    operation_id = "getImageGenerationStatus"
)]
pub async fn get_image_generation_status(
    _authenticated: AuthenticatedAccount,
    State(state): State<ImageState>,
    Path(request_id): Path<String>,
) -> ApiResult<ImageGenerationStatus> {
    if request_id.trim().is_empty() {
        return Err(AppError::InvalidInput("Request ID is required".to_owned()).into());
    }
    Ok(Json(SuccessResponse::new(
        state.service().get_by_request_id(&request_id).await?.into(),
    )))
}
