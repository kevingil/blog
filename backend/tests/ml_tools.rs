use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use blog_backend::{
    core::ml::llm::{
        ApplyPatchTool, DraftSaver, ReadDocumentTool, ReplaceLinesTool, Tool, ToolCallRequest,
        ToolContext,
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
            .map_err(|_| AppError::Internal)?
            .push(markdown_content.to_owned());
        if self.fail {
            Err(AppError::Database)
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
        .run(
            context("Intro\n\n## Details\nBody"),
            ToolCallRequest {
                id: "read-1".to_owned(),
                name: "read_document".to_owned(),
                input: "{}".to_owned(),
            },
        )
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
        .run(
            context.clone(),
            ToolCallRequest {
                id: "edit-1".to_owned(),
                name: "replace_lines".to_owned(),
                input: serde_json::json!({
                    "start_line": 2,
                    "end_line": 2,
                    "new_content": "TWO\n2.5",
                    "reason": "clarify"
                })
                .to_string(),
            },
        )
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
        .run(
            empty.clone(),
            ToolCallRequest {
                id: "edit-1".to_owned(),
                name: "replace_lines".to_owned(),
                input: r#"{"start_line":2,"end_line":2,"new_content":"x","reason":"draft"}"#
                    .to_owned(),
            },
        )
        .await;
    assert!(invalid.is_ok());
    let Ok(invalid) = invalid else {
        return;
    };
    assert!(invalid.is_error);

    let created = tool
        .run(
            empty.clone(),
            ToolCallRequest {
                id: "edit-2".to_owned(),
                name: "replace_lines".to_owned(),
                input:
                    r#"{"start_line":1,"end_line":1,"new_content":"first draft","reason":"draft"}"#
                        .to_owned(),
            },
        )
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
        .run(
            empty.clone(),
            ToolCallRequest {
                id: "patch-1".to_owned(),
                name: "apply_patch".to_owned(),
                input: r#"{"patch":"","old_str":"","new_str":"Hello draft","reason":"write"}"#
                    .to_owned(),
            },
        )
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
        .run(
            ambiguous,
            ToolCallRequest {
                id: "patch-3".to_owned(),
                name: "apply_patch".to_owned(),
                input: r#"{"patch":"","old_str":"one","new_str":"two","reason":"ambiguous"}"#
                    .to_owned(),
            },
        )
        .await;
    assert!(rejected.is_ok());
    let Ok(rejected) = rejected else {
        return;
    };
    assert!(rejected.is_error);
}
