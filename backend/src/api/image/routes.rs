use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_generate_image, __path_get_image_generation, __path_get_image_generation_status,
        generate_image, get_image_generation, get_image_generation_status,
    },
    state::ImageState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    ImageState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(generate_image))
        .routes(routes!(get_image_generation_status))
        .routes(routes!(get_image_generation))
}
