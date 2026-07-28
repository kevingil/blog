use std::error::Error;

use axum::{Router, http::StatusCode, response::Html, routing::get};
use blog_backend::{
    core::source::FetchExtractPort, error::AppError, integrations::fetch::HttpFetchExtract,
};
use tokio::net::TcpListener;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[tokio::test]
async fn fetch_extract_uses_go_compatible_title_main_content_and_truncation() -> TestResult {
    let long = "content word ".repeat(600);
    let html = format!(
        "<html><head><title>  Fixture Title </title></head><body>\
         <nav><p>navigation should not be selected</p></nav>\
         <article><p>First meaningful paragraph.</p><h2>Section heading</h2>\
         <p>{long}</p></article></body></html>"
    );
    let app = Router::new()
        .route("/article", get(move || async move { Html(html.clone()) }))
        .route(
            "/missing",
            get(|| async { (StatusCode::NOT_FOUND, "missing") }),
        );
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move { axum::serve(listener, app).await });
    let adapter = HttpFetchExtract::new()?;

    let result = adapter
        .fetch_extract(&format!("http://{address}/article"))
        .await?;
    assert_eq!(result.title, "Fixture Title");
    assert!(result.content.starts_with("First meaningful paragraph."));
    assert!(!result.content.contains("navigation should not be selected"));
    assert!(result.content.ends_with("..."));
    assert_eq!(result.content.trim_end_matches("...").chars().count(), 5000);
    assert!(matches!(
        adapter
            .fetch_extract(&format!("http://{address}/missing"))
            .await,
        Err(AppError::External)
    ));

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn fetch_extract_rejects_empty_or_invalid_urls() -> TestResult {
    let adapter = HttpFetchExtract::new()?;
    assert!(matches!(
        adapter.fetch_extract("").await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        adapter.fetch_extract("://").await,
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}
