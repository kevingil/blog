use std::sync::Arc;

use axum::{body::Body, http::Request};
use blog_backend::{
    api::{
        auth::AuthState,
        websocket::{
            EmptyWorkerStatusProvider, UnavailableAgentStreamProvider, WebSocketConfig,
            WebSocketSupervisor,
        },
    },
    app::{self, AppState},
    core::auth::AuthService,
    database::{pool::create_pool, repository::account::DieselAccountRepository},
};
use http::StatusCode;
use secrecy::SecretString;
use tower::ServiceExt;

fn state() -> Result<AppState, Box<dyn std::error::Error>> {
    let pool = create_pool(&SecretString::from(
        "postgres://blog:blog@127.0.0.1:5432/blog".to_owned(),
    ))?;
    let auth = AuthState::new(Arc::new(AuthService::new(
        Arc::new(DieselAccountRepository::new(pool.clone())),
        "test-secret",
    )?));
    let (websocket, _supervisor) = WebSocketSupervisor::new(
        WebSocketConfig::default(),
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    Ok(AppState::new(pool, auth, websocket))
}

#[tokio::test]
async fn health_is_dependency_light() -> Result<(), Box<dyn std::error::Error>> {
    let response = app::router(state()?, &[])?
        .oneshot(Request::builder().uri("/health").body(Body::empty())?)
        .await?;

    assert_eq!(response.status(), StatusCode::OK);
    Ok(())
}
