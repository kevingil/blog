use utoipa::openapi::{
    Info, OpenApi,
    security::{HttpAuthScheme, HttpBuilder, SecurityScheme},
};
use utoipa_axum::router::OpenApiRouter;

use crate::{api, app::AppState};

pub fn router() -> OpenApiRouter<AppState> {
    OpenApiRouter::new()
        .merge(api::health::router())
        .merge(api::auth::router())
        .merge(api::websocket::router())
}

pub fn document() -> OpenApi {
    let (_, document) = split_for_parts();
    document
}

pub fn split_for_parts() -> (axum::Router<AppState>, OpenApi) {
    let (router, document) = router().split_for_parts();
    (router, decorate(document))
}

fn decorate(mut document: OpenApi) -> OpenApi {
    document.info = Info::new("Blog API", env!("CARGO_PKG_VERSION"));
    document
        .components
        .get_or_insert_default()
        .add_security_scheme(
            "bearerAuth",
            SecurityScheme::Http(
                HttpBuilder::new()
                    .scheme(HttpAuthScheme::Bearer)
                    .bearer_format("JWT")
                    .build(),
            ),
        );
    document
}

pub fn canonical_json() -> Result<String, serde_json::Error> {
    serde_json::to_string_pretty(&document()).map(|json| format!("{json}\n"))
}
