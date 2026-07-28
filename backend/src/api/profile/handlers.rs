use axum::{
    Json,
    extract::{FromRequest, Request, State, rejection::JsonRejection},
};

use crate::api::response::SuccessResponse;

use super::{
    dto::{
        ProfileUpdateRequest, PublicProfileResponse, SiteSettingsResponse,
        SiteSettingsUpdateRequest, UserProfileResponse,
    },
    error::{ProfileApiError, ProfileAuthenticated},
    state::ProfileState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, ProfileApiError>;

#[utoipa::path(
    get,
    path = "/profile/public",
    responses(
        (status = 200, body = SuccessResponse<PublicProfileResponse>),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    tag = "profile",
    operation_id = "getPublicProfile"
)]
pub async fn get_public_profile(
    State(state): State<ProfileState>,
) -> ApiResult<PublicProfileResponse> {
    Ok(Json(SuccessResponse::new(
        state.service().get_public_profile().await?.into(),
    )))
}

#[utoipa::path(
    get,
    path = "/profile",
    responses(
        (status = 200, body = SuccessResponse<UserProfileResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "profile",
    operation_id = "getMyProfile"
)]
pub async fn get_my_profile(
    ProfileAuthenticated(account_id): ProfileAuthenticated,
    State(state): State<ProfileState>,
) -> ApiResult<UserProfileResponse> {
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .get_user_profile(account_id.into_inner())
            .await?
            .into(),
    )))
}

#[utoipa::path(
    put,
    path = "/profile",
    request_body = ProfileUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<UserProfileResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "profile",
    operation_id = "updateProfile"
)]
pub async fn update_profile(
    ProfileAuthenticated(account_id): ProfileAuthenticated,
    State(state): State<ProfileState>,
    body: Result<Json<ProfileUpdateRequest>, JsonRejection>,
) -> ApiResult<UserProfileResponse> {
    let Json(request) = body.map_err(|_| ProfileApiError::invalid_body())?;
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .update_user_profile(account_id.into_inner(), request.into())
            .await?
            .into(),
    )))
}

#[utoipa::path(
    get,
    path = "/profile/settings",
    responses(
        (status = 200, body = SuccessResponse<SiteSettingsResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "profile",
    operation_id = "getSiteSettings"
)]
pub async fn get_site_settings(
    _authenticated: ProfileAuthenticated,
    State(state): State<ProfileState>,
) -> ApiResult<SiteSettingsResponse> {
    Ok(Json(SuccessResponse::new(
        state.service().get_site_settings().await?.into(),
    )))
}

#[utoipa::path(
    put,
    path = "/profile/settings",
    request_body = SiteSettingsUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<SiteSettingsResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 403, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "profile",
    operation_id = "updateSiteSettings"
)]
pub async fn update_site_settings(
    ProfileAuthenticated(account_id): ProfileAuthenticated,
    State(state): State<ProfileState>,
    request: Request,
) -> ApiResult<SiteSettingsResponse> {
    if !state
        .service()
        .is_user_admin(account_id.into_inner())
        .await?
    {
        return Err(ProfileApiError::forbidden(
            "Only admins can update site settings",
        ));
    }
    let Json(body) = Json::<SiteSettingsUpdateRequest>::from_request(request, &())
        .await
        .map_err(|_| ProfileApiError::invalid_body())?;
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .update_site_settings(body.into())
            .await?
            .into(),
    )))
}
