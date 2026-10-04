use axum::{
    Json,
    extract::{
        Multipart, Path, Query, State,
        multipart::{Field, MultipartRejection},
    },
    http::StatusCode,
};

use crate::{
    api::{auth::AuthenticatedAccount, request::JsonBody, response::SuccessResponse},
    error::AppError,
};

use super::{
    dto::{
        CreateFolderRequest, GenerateBlurhashRequest, GenerateBlurhashResponse, ListFilesQuery,
        ListFilesResponse, SuccessFlagResponse, UpdateFolderRequest, UploadFileRequest,
        UploadFileResponse,
    },
    state::StorageState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

async fn file_bytes(field: Field<'_>) -> Result<Vec<u8>, AppError> {
    match field.bytes().await {
        Ok(bytes) => Ok(bytes.to_vec()),
        Err(error) if error.status() == StatusCode::PAYLOAD_TOO_LARGE => {
            Err(AppError::InvalidInput("File is too large".to_owned()))
        }
        Err(error) if error.status().is_client_error() => {
            Err(AppError::InvalidInput("Invalid request body".to_owned()))
        }
        Err(_) => Err(AppError::Internal),
    }
}

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
    let mut content_type = None;
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
                content_type = field.content_type().map(str::to_owned);
                file = Some(file_bytes(field).await?);
            }
            _ => {}
        }
    }
    let key = key
        .filter(|value| !value.is_empty())
        .ok_or_else(|| AppError::InvalidInput("File key is required".to_owned()))?;
    let file = file.ok_or_else(|| AppError::InvalidInput("File is required".to_owned()))?;
    let service = state.service()?;
    let recorded = service
        .put_recorded(
            &key,
            content_type
                .as_deref()
                .unwrap_or("application/octet-stream"),
            file,
            Some(_authenticated.0.into_inner()),
        )
        .await?;
    Ok(Json(SuccessResponse::new(UploadFileResponse {
        success: true,
        url: recorded.url,
        key: recorded.key,
        id: recorded.id,
        content_type: recorded.content_type,
        byte_size: recorded.byte_size,
        width: recorded.width,
        height: recorded.height,
        blurhash: recorded.blurhash,
    })))
}

#[utoipa::path(
    post,
    path = "/storage/blurhash",
    request_body = GenerateBlurhashRequest,
    responses(
        (status = 200, body = SuccessResponse<GenerateBlurhashResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "storage",
    operation_id = "generateStorageBlurhash"
)]
pub async fn generate_blurhash(
    _authenticated: AuthenticatedAccount,
    State(state): State<StorageState>,
    JsonBody(request): JsonBody<GenerateBlurhashRequest>,
) -> ApiResult<GenerateBlurhashResponse> {
    let service = state.service()?;
    let recorded = if let Some(id) = request.id {
        service.generate_blurhash_for_id(id).await?
    } else {
        service.generate_blurhash(&request.key).await?
    };
    let blurhash = recorded
        .blurhash
        .ok_or_else(|| AppError::InvalidInput("Could not read this image".to_owned()))?;
    Ok(Json(SuccessResponse::new(GenerateBlurhashResponse {
        id: recorded.id,
        key: recorded.key,
        url: recorded.url,
        blurhash,
        width: recorded.width,
        height: recorded.height,
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
