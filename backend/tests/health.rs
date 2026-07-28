use axum::{body::Body, http::Request};
use blog_backend::api;
use http::StatusCode;
use tower::ServiceExt;

#[tokio::test]
async fn health_is_dependency_light() -> Result<(), Box<dyn std::error::Error>> {
    let (router, _) = api::health::router::<()>().split_for_parts();
    let response = router
        .with_state(())
        .oneshot(Request::builder().uri("/health").body(Body::empty())?)
        .await?;

    assert_eq!(response.status(), StatusCode::OK);
    Ok(())
}
