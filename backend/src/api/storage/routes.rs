use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_folder, __path_delete_file, __path_list_files, __path_update_folder,
        __path_upload_file, create_folder, delete_file, list_files, update_folder, upload_file,
    },
    state::StorageState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    StorageState: FromRef<S>,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_files))
        .routes(routes!(upload_file))
        .routes(routes!(create_folder, update_folder))
        .routes(routes!(delete_file))
}
