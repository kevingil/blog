use axum::{
    Json,
    extract::{Multipart, Path, Query, State, multipart::MultipartRejection},
};

use crate::{
    api::{auth::AuthenticatedAccount, request::JsonBody, response::SuccessResponse},
    error::AppError,
};

use super::{
    dto::{
        CreateFolderRequest, ListFilesQuery, ListFilesResponse, SuccessFlagResponse,
        UpdateFolderRequest, UploadFileRequest, UploadFileResponse,
    },
    state::StorageState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

#[utoipa::path(
    get,
    path = "/storage/files",
    params(ListFilesQuery),
    responses(
        (status = 200, body = SuccessResponse<ListFilesResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "listStorageFiles"
)]
pub async fn list_files(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    Query(query): Query<ListFilesQuery>,
) -> ApiResult<ListFilesResponse> {
    let result = state
        .service()?
        .list_files(query.prefix.as_deref().unwrap_or_default())
        .await?;
    Ok(Json(SuccessResponse::new(ListFilesResponse {
        files: result.files.into_iter().map(Into::into).collect(),
        folders: result.folders.into_iter().map(Into::into).collect(),
    })))
}

#[utoipa::path(
    post,
    path = "/storage/upload",
    request_body(content = UploadFileRequest, content_type = "multipart/form-data"),
    responses(
        (status = 200, body = SuccessResponse<UploadFileResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "uploadStorageFile"
)]
pub async fn upload_file(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    multipart: Result<Multipart, MultipartRejection>,
) -> ApiResult<UploadFileResponse> {
    let mut multipart =
        multipart.map_err(|_| AppError::InvalidInput("Invalid request body".to_owned()))?;
    let mut key = None;
    let mut file = None;
    while let Some(field) = multipart
        .next_field()
        .await
        .map_err(|_| AppError::InvalidInput("Invalid request body".to_owned()))?
    {
        match field.name() {
            Some("key") if key.is_none() => {
                key = Some(
                    field
                        .text()
                        .await
                        .map_err(|_| AppError::InvalidInput("Invalid request body".to_owned()))?,
                );
            }
            Some("file") if file.is_none() => {
                file = Some(
                    field
                        .bytes()
                        .await
                        .map_err(|_| AppError::Internal)?
                        .to_vec(),
                );
            }
            _ => {}
        }
    }
    let key = key
        .filter(|value| !value.is_empty())
        .ok_or_else(|| AppError::InvalidInput("File key is required".to_owned()))?;
    let file = file.ok_or_else(|| AppError::InvalidInput("File is required".to_owned()))?;
    let service = state.service()?;
    service.upload_file(&key, file).await?;
    Ok(Json(SuccessResponse::new(UploadFileResponse {
        success: true,
        url: format!("{}/{}", service.url_prefix(), key),
        key,
    })))
}

#[utoipa::path(
    delete,
    path = "/storage/{key}",
    params(("key" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "deleteStorageFile"
)]
pub async fn delete_file(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    Path(key): Path<String>,
) -> ApiResult<SuccessFlagResponse> {
    if key.is_empty() {
        return Err(AppError::InvalidInput("File key is required".to_owned()));
    }
    state.service()?.delete_file(&key).await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    post,
    path = "/storage/folders",
    request_body = CreateFolderRequest,
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "createStorageFolder"
)]
pub async fn create_folder(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    JsonBody(request): JsonBody<CreateFolderRequest>,
) -> ApiResult<SuccessFlagResponse> {
    if request.path.is_empty() {
        return Err(AppError::InvalidInput(
            "path is a required field".to_owned(),
        ));
    }
    state.service()?.create_folder(&request.path).await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}

#[utoipa::path(
    put,
    path = "/storage/folders",
    request_body = UpdateFolderRequest,
    responses(
        (status = 200, body = SuccessResponse<SuccessFlagResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "updateStorageFolder"
)]
pub async fn update_folder(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    JsonBody(request): JsonBody<UpdateFolderRequest>,
) -> ApiResult<SuccessFlagResponse> {
    if request.old_path.is_empty() {
        return Err(AppError::InvalidInput(
            "oldPath is a required field".to_owned(),
        ));
    }
    if request.new_path.is_empty() {
        return Err(AppError::InvalidInput(
            "newPath is a required field".to_owned(),
        ));
    }
    state
        .service()?
        .update_folder(&request.old_path, &request.new_path)
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlagResponse {
        success: true,
    })))
}
