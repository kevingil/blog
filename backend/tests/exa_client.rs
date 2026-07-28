use std::{error::Error, sync::Arc};

use axum::{Json, Router, extract::State, http::HeaderMap, routing::post};
use blog_backend::{
    core::datasource::{RecommendationSearchPort, SearchOptions, SimilarOptions},
    core::ml::llm::ResearchPort,
    error::AppError,
    integrations::exa::ExaClient,
};
use serde_json::{Value, json};
use tokio::{net::TcpListener, sync::Mutex};

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone, Default)]
struct Received {
    requests: Arc<Mutex<Vec<(String, HeaderMap, Value)>>>,
}

async fn capture(
    State(received): State<Received>,
    headers: HeaderMap,
    uri: axum::http::Uri,
    Json(body): Json<Value>,
) -> Json<Value> {
    let path = uri.path().to_owned();
    received
        .requests
        .lock()
        .await
        .push((path.clone(), headers, body));
    if path == "/answer" {
        return Json(json!({
            "answer": "Axum is a Rust web framework.",
            "citations": [{
                "url": "https://example.com/axum",
                "title": "Axum",
                "author": "Example Author",
                "publishedDate": "2026-07-27",
                "favicon": "https://example.com/favicon.ico",
                "text": "Citation text"
            }],
            "costDollars": {"total": 0.001}
        }));
    }
    Json(json!({
        "requestId": "exa-request",
        "resolvedSearchType": "neural",
        "costDollars": {"total": 0.002},
        "results": [{
            "title": "Rust result",
            "url": "https://example.com/rust",
            "id": "exa-result",
            "publishedDate": "2026-07-27T00:00:00Z",
            "author": "Example Author",
            "score": 0.75,
            "text": "Full text",
            "highlights": ["highlight"],
            "summary": "Summary",
            "image": "https://example.com/image.png",
            "favicon": "https://example.com/favicon.ico"
        }]
    }))
}

#[tokio::test]
async fn exa_adapter_preserves_request_defaults_headers_paths_and_results() -> TestResult {
    let received = Received::default();
    let app = Router::new()
        .route("/search", post(capture))
        .route("/findSimilar", post(capture))
        .route("/answer", post(capture))
        .with_state(received.clone());
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move { axum::serve(listener, app).await });
    let client = ExaClient::with_base_url("test-key", format!("http://{address}"))?;

    let search = RecommendationSearchPort::search(
        &client,
        "Rust",
        SearchOptions {
            include_text: true,
            include_highlights: true,
            include_summary: true,
            use_autoprompt: true,
            include_domains: vec!["example.com".to_owned()],
            start_date: "2026-01-01".to_owned(),
            end_date: "2026-07-27".to_owned(),
            ..SearchOptions::default()
        },
    )
    .await?;
    let similar = client
        .find_similar(
            "https://example.com",
            SimilarOptions {
                exclude_source_domain: true,
                ..SimilarOptions::default()
            },
        )
        .await?;
    let research = ResearchPort::search(&client, "Axum research").await?;
    let answer = client.answer("What is Axum?").await?;
    assert_eq!(search.results.len(), 1);
    assert_eq!(search.results[0].title, "Rust result");
    assert_eq!(search.results[0].id, "exa-result");
    assert_eq!(search.results[0].author, "Example Author");
    assert_eq!(search.results[0].published_date, "2026-07-27T00:00:00Z");
    assert_eq!(similar.results[0].score, 0.75);
    assert_eq!(research.request_id, "exa-request");
    assert_eq!(research.resolved_search_type, "neural");
    assert_eq!(research.results[0].title, "Rust result");
    assert_eq!(answer.answer, "Axum is a Rust web framework.");
    assert_eq!(answer.citations[0].author, "Example Author");

    let requests = received.requests.lock().await;
    assert_eq!(requests.len(), 4);
    assert_eq!(requests[0].0, "/search");
    assert_eq!(requests[0].1["x-api-key"], "test-key");
    assert_eq!(requests[0].2["query"], "Rust");
    assert_eq!(requests[0].2["type"], "auto");
    assert_eq!(requests[0].2["numResults"], 10);
    assert_eq!(requests[0].2["useAutoprompt"], true);
    assert_eq!(requests[0].2["includeDomains"], json!(["example.com"]));
    assert_eq!(requests[0].2["startCrawlDate"], "2026-01-01");
    assert_eq!(requests[0].2["startPublishedDate"], "2026-01-01");
    assert_eq!(requests[0].2["endCrawlDate"], "2026-07-27");
    assert_eq!(requests[0].2["endPublishedDate"], "2026-07-27");
    assert_eq!(requests[1].0, "/findSimilar");
    assert_eq!(requests[1].2["url"], "https://example.com");
    assert_eq!(requests[1].2["excludeSourceDomain"], true);
    assert_eq!(requests[2].0, "/search");
    assert_eq!(requests[2].2["query"], "Axum research");
    assert_eq!(requests[2].2["text"], true);
    assert_eq!(requests[2].2["highlights"], true);
    assert_eq!(requests[2].2["summary"], true);
    assert_eq!(requests[3].0, "/answer");
    assert_eq!(
        requests[3].2,
        json!({"query": "What is Axum?", "text": true})
    );
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn exa_adapter_rejects_invalid_input_and_unconfigured_calls() -> TestResult {
    let unconfigured = ExaClient::new("")?;
    assert!(!unconfigured.is_configured());
    assert!(matches!(
        RecommendationSearchPort::search(&unconfigured, "query", SearchOptions::default()).await,
        Err(AppError::External)
    ));
    let configured = ExaClient::new("key")?;
    assert!(matches!(
        RecommendationSearchPort::search(&configured, "", SearchOptions::default()).await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        configured.find_similar("", SimilarOptions::default()).await,
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}
