use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use blog_backend::{
    core::ml::llm::{
        ApplyPatchTool, ArticleSourceView, DraftSaver, ReadDocumentTool, ReplaceLinesTool,
        SetTitleTool, SourceResource, SourceResourcePort, SourceSelection, Tool, ToolCallRequest,
        ToolContext, UpdateSourcesTool, WebSearchResult,
    },
    error::AppError,
};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

#[derive(Default)]
struct CapturingDraftSaver {
    values: Mutex<Vec<String>>,
    fail: bool,
}

#[async_trait]
impl DraftSaver for CapturingDraftSaver {
    async fn update_draft_content(
        &self,
        _article_id: Uuid,
        markdown_content: &str,
    ) -> Result<(), AppError> {
        self.values
            .lock()
            .map_err(AppError::internal)?
            .push(markdown_content.to_owned());
        if self.fail {
            Err(AppError::database("no underlying error was recorded"))
        } else {
            Ok(())
        }
    }
}

fn context(markdown: &str) -> ToolContext {
    ToolContext::new(
        "session",
        "message",
        "request",
        Some(Uuid::new_v4()),
        "",
        markdown,
        CancellationToken::new(),
    )
}

#[tokio::test]
async fn read_document_returns_numbered_content_and_sections() {
    let response = ReadDocumentTool
        .run(context("Intro\n\n## Details\nBody"), ToolCallRequest {
            id: "read-1".to_owned(),
            name: "read_document".to_owned(),
            input: "{}".to_owned(),
        })
        .await;
    assert!(response.is_ok());
    let Ok(response) = response else {
        return;
    };
    let value: serde_json::Value =
        serde_json::from_str(&response.content).unwrap_or(serde_json::Value::Null);
    assert_eq!(value["total_lines"], 4);
    assert_eq!(value["sections"][0]["heading"], "## Details");
    assert!(
        value["content"]
            .as_str()
            .unwrap_or_default()
            .contains("   1| Intro")
    );
}

#[tokio::test]
async fn replace_lines_updates_shared_turn_state_and_persists_best_effort() {
    let saver = Arc::new(CapturingDraftSaver {
        values: Mutex::new(Vec::new()),
        fail: true,
    });
    let tool = ReplaceLinesTool::new(Some(saver.clone()));
    let context = context("one\ntwo\nthree");
    let response = tool
        .run(context.clone(), ToolCallRequest {
            id: "edit-1".to_owned(),
            name: "replace_lines".to_owned(),
            input: serde_json::json!({
                "start_line": 2,
                "end_line": 2,
                "new_content": "TWO\n2.5",
                "reason": "clarify"
            })
            .to_string(),
        })
        .await;
    assert!(response.is_ok());
    let Ok(response) = response else {
        return;
    };
    assert!(!response.is_error);
    assert_eq!(
        context.document_markdown().unwrap_or_default(),
        "one\nTWO\n2.5\nthree"
    );
    assert_eq!(
        saver
            .values
            .lock()
            .map(|values| values.clone())
            .unwrap_or_default(),
        vec!["one\nTWO\n2.5\nthree"]
    );
    assert_eq!(
        response
            .artifact
            .as_ref()
            .map(|artifact| artifact.artifact_type.as_str()),
        Some("diff")
    );
}

#[tokio::test]
async fn replace_lines_handles_empty_documents_and_invalid_ranges() {
    let tool = ReplaceLinesTool::new(None);
    let empty = context("");
    let invalid = tool
        .run(empty.clone(), ToolCallRequest {
            id: "edit-1".to_owned(),
            name: "replace_lines".to_owned(),
            input: r#"{"start_line":2,"end_line":2,"new_content":"x","reason":"draft"}"#.to_owned(),
        })
        .await;
    assert!(invalid.is_ok());
    let Ok(invalid) = invalid else {
        return;
    };
    assert!(invalid.is_error);

    let created = tool
        .run(empty.clone(), ToolCallRequest {
            id: "edit-2".to_owned(),
            name: "replace_lines".to_owned(),
            input: r#"{"start_line":1,"end_line":1,"new_content":"first draft","reason":"draft"}"#
                .to_owned(),
        })
        .await;
    assert!(created.is_ok());
    let Ok(created) = created else {
        return;
    };
    assert!(!created.is_error);
    assert_eq!(empty.document_markdown().unwrap_or_default(), "first draft");
}

#[tokio::test]
async fn apply_patch_writes_an_empty_article_and_replaces_unique_text() {
    let saver = Arc::new(CapturingDraftSaver::default());
    let tool = ApplyPatchTool::new(Some(saver.clone()));
    let empty = context("");
    let created = tool
        .run(empty.clone(), ToolCallRequest {
            id: "patch-1".to_owned(),
            name: "apply_patch".to_owned(),
            input: r#"{"patch":"","old_str":"","new_str":"Hello draft","reason":"write"}"#
                .to_owned(),
        })
        .await;
    assert!(created.is_ok());
    let Ok(created) = created else {
        return;
    };
    assert!(!created.is_error);
    assert_eq!(empty.document_markdown().unwrap_or_default(), "Hello draft");
    assert_eq!(
        saver
            .values
            .lock()
            .map(|values| values.clone())
            .unwrap_or_default(),
        vec!["Hello draft"]
    );

    let edited = tool
        .run(
            empty.clone(),
            ToolCallRequest {
                id: "patch-2".to_owned(),
                name: "apply_patch".to_owned(),
                input: r#"{"patch":"*** Begin Patch\n*** Update File: article.md\n@@\n-Hello draft\n+Hello article\n*** End Patch","old_str":"","new_str":"","reason":"rename"}"#
                    .to_owned(),
            },
        )
        .await;
    assert!(edited.is_ok());
    let Ok(edited) = edited else {
        return;
    };
    assert!(!edited.is_error, "{}", edited.content);
    assert_eq!(
        empty.document_markdown().unwrap_or_default(),
        "Hello article"
    );

    let ambiguous = context("one one");
    let rejected = tool
        .run(ambiguous, ToolCallRequest {
            id: "patch-3".to_owned(),
            name: "apply_patch".to_owned(),
            input: r#"{"patch":"","old_str":"one","new_str":"two","reason":"ambiguous"}"#
                .to_owned(),
        })
        .await;
    assert!(rejected.is_ok());
    let Ok(rejected) = rejected else {
        return;
    };
    assert!(rejected.is_error);
}

#[derive(Default)]
struct RecordingSaver {
    content: Mutex<Vec<String>>,
    titles: Mutex<Vec<String>>,
}

#[async_trait]
impl DraftSaver for RecordingSaver {
    async fn update_draft_content(
        &self,
        _article_id: Uuid,
        markdown_content: &str,
    ) -> Result<(), AppError> {
        self.content
            .lock()
            .map_err(AppError::internal)?
            .push(markdown_content.to_owned());
        Ok(())
    }

    async fn update_draft_title(&self, _article_id: Uuid, title: &str) -> Result<(), AppError> {
        self.titles
            .lock()
            .map_err(AppError::internal)?
            .push(title.to_owned());
        Ok(())
    }
}

#[derive(Default)]
struct MemorySources {
    values: Mutex<Vec<SourceResource>>,
}

#[async_trait]
impl SourceResourcePort for MemorySources {
    async fn create_web_source(
        &self,
        _article_id: Uuid,
        _query: &str,
        _result: &WebSearchResult,
        _request_id: &str,
    ) -> Result<SourceResource, AppError> {
        Err(AppError::InvalidInput("unused".to_owned()))
    }

    async fn list(&self, article_id: Uuid) -> Result<Vec<SourceResource>, AppError> {
        Ok(self
            .values
            .lock()
            .map_err(AppError::internal)?
            .iter()
            .filter(|source| source.article_id == article_id)
            .cloned()
            .collect())
    }

    async fn search_similar(
        &self,
        article_id: Uuid,
        _query: &str,
        _limit: i64,
    ) -> Result<Vec<SourceResource>, AppError> {
        self.list(article_id).await
    }

    async fn select(
        &self,
        _article_id: Uuid,
        _selection: SourceSelection,
        _request_id: &str,
    ) -> Result<SourceResource, AppError> {
        Err(AppError::InvalidInput("unused".to_owned()))
    }

    async fn apply_edits(
        &self,
        article_id: Uuid,
        upserts: Vec<blog_backend::core::ml::llm::SourceEdit>,
        remove_ids: Vec<Uuid>,
    ) -> Result<Vec<SourceResource>, AppError> {
        let mut values = self.values.lock().map_err(AppError::internal)?;
        for id in &remove_ids {
            if !values
                .iter()
                .any(|source| source.id == *id && source.article_id == article_id)
            {
                return Err(AppError::NotFound);
            }
        }
        values.retain(|source| source.article_id != article_id || !remove_ids.contains(&source.id));
        for edit in upserts {
            if let Some(id) = edit.id {
                let Some(source) = values
                    .iter_mut()
                    .find(|source| source.id == id && source.article_id == article_id)
                else {
                    return Err(AppError::NotFound);
                };
                if !edit.title.trim().is_empty() {
                    source.title = edit.title;
                }
                source.url = edit.url;
                if edit.replace_content && !edit.content.trim().is_empty() {
                    source.content = edit.content;
                }
                continue;
            }
            values.push(SourceResource {
                id: Uuid::new_v4(),
                article_id,
                title: if edit.title.trim().is_empty() {
                    edit.url.clone()
                } else {
                    edit.title.clone()
                },
                content: if edit.content.trim().is_empty() {
                    edit.title
                } else {
                    edit.content
                },
                url: edit.url,
                source_type: "web".to_owned(),
                meta_data: Default::default(),
                created_at: None,
            });
        }
        Ok(values
            .iter()
            .filter(|source| source.article_id == article_id)
            .cloned()
            .collect())
    }
}

#[tokio::test]
async fn replace_lines_lifts_a_leading_title_out_of_the_body() {
    let saver = Arc::new(RecordingSaver::default());
    let tool = ReplaceLinesTool::new(Some(saver.clone()));
    let document = context("");
    document.set_document_title("Old title").expect("title");
    let response = tool
        .run(document.clone(), ToolCallRequest {
            id: "edit-title".to_owned(),
            name: "replace_lines".to_owned(),
            input: serde_json::json!({
                "start_line": 1,
                "end_line": 1,
                "new_content": "# RSI Is Also a Systems Problem\n\nThe machinery matters.",
                "reason": "draft"
            })
            .to_string(),
        })
        .await
        .expect("tool result");
    assert!(!response.is_error, "{}", response.content);
    assert_eq!(
        document.document_markdown().unwrap_or_default(),
        "The machinery matters."
    );
    assert_eq!(
        document.document_title().unwrap_or_default(),
        "RSI Is Also a Systems Problem"
    );
    assert_eq!(
        saver
            .titles
            .lock()
            .map(|titles| titles.clone())
            .unwrap_or_default(),
        vec!["RSI Is Also a Systems Problem"]
    );
    let value: serde_json::Value =
        serde_json::from_str(&response.content).unwrap_or(serde_json::Value::Null);
    assert_eq!(value["title_updated"], true);
    assert_eq!(value["new_markdown"], "The machinery matters.");
}

#[tokio::test]
async fn apply_patch_moves_a_sources_section_into_source_objects() {
    let sources = Arc::new(MemorySources::default());
    let tool = ApplyPatchTool::new(None).with_sources(sources.clone());
    let document = context("The machinery matters.");
    let response = tool
        .run(
            document.clone(),
            ToolCallRequest {
                id: "patch-sources".to_owned(),
                name: "apply_patch".to_owned(),
                input: serde_json::json!({
                    "patch": "",
                    "old_str": "The machinery matters.",
                    "new_str": "The machinery matters.\n\n## Sources\n- [GraalVM](https://www.graalvm.org)\n- Field notes",
                    "reason": "cite"
                })
                .to_string(),
            },
        )
        .await
        .expect("tool result");
    assert!(!response.is_error, "{}", response.content);
    assert_eq!(
        document.document_markdown().unwrap_or_default(),
        "The machinery matters."
    );
    let saved = sources.list(document.article_id.expect("article")).await;
    let saved = saved.expect("sources");
    assert_eq!(saved.len(), 2);
    assert_eq!(saved[0].title, "GraalVM");
    assert_eq!(saved[0].url, "https://www.graalvm.org");
    assert_eq!(saved[1].title, "Field notes");
    let value: serde_json::Value =
        serde_json::from_str(&response.content).unwrap_or(serde_json::Value::Null);
    assert_eq!(value["sources_updated"], true);
    assert_eq!(document.document_sources().expect("views").len(), 2);
}

#[tokio::test]
async fn set_title_updates_the_title_field_without_touching_the_body() {
    let saver = Arc::new(RecordingSaver::default());
    let tool = SetTitleTool::new(Some(saver.clone()));
    let document = context("Body stays.");
    let response = tool
        .run(document.clone(), ToolCallRequest {
            id: "title-1".to_owned(),
            name: "set_title".to_owned(),
            input: r#"{"title":"A separate title","reason":"rename"}"#.to_owned(),
        })
        .await
        .expect("tool result");
    assert!(!response.is_error, "{}", response.content);
    assert_eq!(
        document.document_markdown().unwrap_or_default(),
        "Body stays."
    );
    assert_eq!(
        document.document_title().unwrap_or_default(),
        "A separate title"
    );
    assert_eq!(
        saver
            .titles
            .lock()
            .map(|titles| titles.clone())
            .unwrap_or_default(),
        vec!["A separate title"]
    );
}

#[tokio::test]
async fn update_sources_edits_the_source_list() {
    let sources = Arc::new(MemorySources::default());
    let article_id = Uuid::new_v4();
    let document = ToolContext::new(
        "session",
        "message",
        "request",
        Some(article_id),
        "",
        "Body",
        CancellationToken::new(),
    );
    let tool = UpdateSourcesTool::new(sources.clone());
    let created = tool
        .run(
            document.clone(),
            ToolCallRequest {
                id: "sources-1".to_owned(),
                name: "update_sources".to_owned(),
                input: r#"{"sources":[{"id":null,"title":"GraalVM","url":"https://www.graalvm.org","note":"runtime"}],"remove_ids":[]}"#.to_owned(),
            },
        )
        .await
        .expect("create");
    assert!(!created.is_error, "{}", created.content);
    let listed = sources.list(article_id).await.expect("list");
    assert_eq!(listed.len(), 1);
    assert_eq!(listed[0].content, "runtime");
    let removed = tool
        .run(document.clone(), ToolCallRequest {
            id: "sources-2".to_owned(),
            name: "update_sources".to_owned(),
            input: format!(r#"{{"sources":[],"remove_ids":["{}"]}}"#, listed[0].id),
        })
        .await
        .expect("remove");
    assert!(!removed.is_error, "{}", removed.content);
    assert!(sources.list(article_id).await.expect("empty").is_empty());
    assert!(document.document_sources().expect("views").is_empty());
}

#[tokio::test]
async fn read_document_reports_title_and_sources_separately() {
    let document = context("Body line");
    document.set_document_title("Kept apart").expect("title");
    document
        .set_document_sources(vec![ArticleSourceView {
            id: Uuid::new_v4(),
            title: "GraalVM".to_owned(),
            url: "https://www.graalvm.org".to_owned(),
            content: "runtime".to_owned(),
        }])
        .expect("sources");
    let response = ReadDocumentTool
        .run(document, ToolCallRequest {
            id: "read-2".to_owned(),
            name: "read_document".to_owned(),
            input: "{}".to_owned(),
        })
        .await
        .expect("read");
    let value: serde_json::Value =
        serde_json::from_str(&response.content).unwrap_or(serde_json::Value::Null);
    assert_eq!(value["title"], "Kept apart");
    assert_eq!(value["sources"][0]["title"], "GraalVM");
    assert!(
        !value["content"]
            .as_str()
            .unwrap_or_default()
            .contains("Kept apart")
    );
}
