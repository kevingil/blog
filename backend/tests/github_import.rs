use std::{error::Error, sync::Arc};

use axum::{
    Json, Router,
    extract::{Path, Query, State},
    http::StatusCode,
    routing::get,
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use blog_backend::{
    core::project::GithubImportPort, error::AppError, integrations::github::GithubClient,
};
use serde::Deserialize;
use serde_json::{Value, json};
use tokio::net::TcpListener;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone)]
struct Fixture {
    body: Value,
    readme: Option<String>,
    seen_ref: Arc<tokio::sync::Mutex<Option<String>>>,
}

#[derive(Deserialize)]
struct RefQuery {
    #[serde(rename = "ref")]
    reference: Option<String>,
}

#[tokio::test]
async fn imports_open_strike_readme_image_and_metadata() -> TestResult {
    let readme = "\
# Open Strike

An open-source tactical FPS inspired by Counter-Strike.

[Quick start](#quick-start) · [Screenshots](#screenshots)

## Screenshots

![ci](https://img.shields.io/badge/build-passing-brightgreen)

![Open Strike home menu with a character overlooking Dust 2](docs/screenshots/menu.png)

![Open Strike inventory](docs/screenshots/inventory.png)
";
    let (base, seen_ref, server) = spawn(open_strike_body(), Some(readme)).await?;
    let client = GithubClient::with_api_base(&base)?;

    let draft = client
        .import_repository("https://github.com/kevingil/open-strike")
        .await?;

    assert_eq!(draft.title, "Open Strike");
    assert_eq!(draft.description, "FPS game inspired by Counter Strike");
    assert_eq!(
        draft.image_url,
        "https://raw.githubusercontent.com/kevingil/open-strike/main/docs/screenshots/menu.png"
    );
    assert_eq!(draft.tags, vec!["Rust".to_owned()]);
    assert_eq!(draft.url, "https://github.com/kevingil/open-strike");
    assert!(draft.content.contains("docs/screenshots/menu.png"));
    assert!(draft.content.contains("img.shields.io"));
    assert_eq!(seen_ref.lock().await.as_deref(), Some("main"));

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn uses_tree_ref_html_images_and_skips_fenced_badges() -> TestResult {
    let readme = "\
```md
![ignored](docs/secret.png)
```

<img alt=\"Menu\" src=\"docs/screenshots/menu.png\">
![blob](https://github.com/kevingil/open-strike/blob/dev/docs/screenshots/inventory.png)
";
    let (base, seen_ref, server) = spawn(open_strike_body(), Some(readme)).await?;
    let client = GithubClient::with_api_base(&base)?;

    let draft = client
        .import_repository("https://github.com/kevingil/open-strike/tree/dev")
        .await?;

    assert_eq!(
        draft.image_url,
        "https://raw.githubusercontent.com/kevingil/open-strike/dev/docs/screenshots/menu.png"
    );
    assert_eq!(seen_ref.lock().await.as_deref(), Some("dev"));
    assert!(!draft.image_url.contains("secret.png"));

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn falls_back_to_repo_name_and_readme_summary_without_an_image() -> TestResult {
    let readme = "Just a short note about the repository.\n";
    let mut body = open_strike_body();
    body["description"] = Value::Null;
    body["language"] = json!("Rust");
    body["topics"] = json!(["rust", "fps"]);
    let (base, _, server) = spawn(body, Some(readme)).await?;
    let client = GithubClient::with_api_base(&base)?;

    let draft = client
        .import_repository("git@github.com:kevingil/open-strike.git")
        .await?;

    assert_eq!(draft.title, "Open Strike");
    assert_eq!(draft.description, "Just a short note about the repository.");
    assert_eq!(draft.image_url, "");
    assert_eq!(draft.tags, vec!["rust".to_owned(), "fps".to_owned()]);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn keeps_remote_images_and_truncates_long_descriptions() -> TestResult {
    let readme = "\
![shot](https://github.com/kevingil/open-strike/blob/main/docs/screenshots/menu.png)
![remote](https://user-images.githubusercontent.com/2253237/menu.png)
";
    let mut body = open_strike_body();
    body["description"] = json!("d".repeat(620));
    let (base, _, server) = spawn(body, Some(readme)).await?;
    let client = GithubClient::with_api_base(&base)?;

    let draft = client
        .import_repository("github.com/kevingil/open-strike.git")
        .await?;

    assert_eq!(draft.description.chars().count(), 500);
    assert_eq!(
        draft.image_url,
        "https://raw.githubusercontent.com/kevingil/open-strike/main/docs/screenshots/menu.png"
    );

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn rejects_missing_repositories_and_non_github_urls() -> TestResult {
    let (base, _, server) = spawn(open_strike_body(), None).await?;
    let client = GithubClient::with_api_base(&base)?;

    assert!(matches!(
        client
            .import_repository("https://github.com/kevingil/missing")
            .await,
        Err(AppError::NotFound)
    ));
    assert!(matches!(
        client
            .import_repository("https://gitlab.com/kevingil/open-strike")
            .await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        client
            .import_repository("https://github.com/kevingil")
            .await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        client.import_repository("").await,
        Err(AppError::InvalidInput(_))
    ));

    server.abort();
    let _ = server.await;
    Ok(())
}

fn open_strike_body() -> Value {
    json!({
        "name": "open-strike",
        "description": "FPS game inspired by Counter Strike ",
        "html_url": "https://github.com/kevingil/open-strike",
        "language": "Rust",
        "topics": [],
        "default_branch": "main"
    })
}

async fn spawn(
    body: Value,
    readme: Option<&str>,
) -> TestResult<(
    String,
    Arc<tokio::sync::Mutex<Option<String>>>,
    tokio::task::JoinHandle<()>,
)> {
    let seen_ref = Arc::new(tokio::sync::Mutex::new(None));
    let fixture = Fixture {
        body,
        readme: readme.map(str::to_owned),
        seen_ref: seen_ref.clone(),
    };
    let app = Router::new()
        .route("/repos/{owner}/{repo}", get(repo_handler))
        .route("/repos/{owner}/{repo}/readme", get(readme_handler))
        .with_state(fixture);
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move {
        let _ = axum::serve(listener, app).await;
    });
    Ok((format!("http://{address}"), seen_ref, server))
}

fn wrapped_base64(value: &str) -> String {
    let encoded = STANDARD.encode(value);
    encoded
        .as_bytes()
        .chunks(60)
        .map(|chunk| String::from_utf8_lossy(chunk).into_owned())
        .collect::<Vec<_>>()
        .join("\n")
}

async fn repo_handler(
    Path((owner, repo)): Path<(String, String)>,
    State(fixture): State<Fixture>,
) -> Result<Json<Value>, StatusCode> {
    if owner == "kevingil" && repo == "open-strike" {
        Ok(Json(fixture.body))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

async fn readme_handler(
    Query(query): Query<RefQuery>,
    State(fixture): State<Fixture>,
) -> Result<Json<Value>, StatusCode> {
    *fixture.seen_ref.lock().await = query.reference;
    let Some(readme) = fixture.readme else {
        return Err(StatusCode::NOT_FOUND);
    };
    Ok(Json(json!({
        "content": wrapped_base64(&readme),
        "encoding": "base64"
    })))
}
