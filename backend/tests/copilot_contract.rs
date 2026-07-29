use blog_backend::core::{
    chat::{ArtifactInfo, MessageMetadata, UserAction},
    copilot::{
        StreamResponse,
        metadata::{
            ARTIFACT_STATUS_PENDING, ARTIFACT_TYPE_CODE_EDIT, MetadataBuilder, message_context,
            validate, with_document_hash,
        },
    },
};
use chrono::Utc;

#[test]
fn stream_response_uses_exact_frontend_field_names_and_omissions() {
    let mut event = StreamResponse::new("request-1", "content_delta");
    event.content = "hello".to_owned();
    let value = serde_json::to_value(event).unwrap_or_default();
    assert_eq!(value["requestId"], "request-1");
    assert_eq!(value["type"], "content_delta");
    assert_eq!(value["content"], "hello");
    assert!(value.get("done").is_none());
    assert!(value.get("tool_input").is_none());

    let terminal = serde_json::to_value(StreamResponse::terminal_error(
        "request-2",
        "Request not found",
    ))
    .unwrap_or_default();
    assert_eq!(terminal["done"], true);
    assert_eq!(terminal["error"], "Request not found");
}

#[test]
fn metadata_builder_hashes_context_and_validates_artifacts_and_actions() {
    let context = with_document_hash(
        message_context("article", "session", "request", "user"),
        "hello",
    );
    assert_eq!(
        context.document_hash,
        "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
    );
    let metadata = MetadataBuilder::new()
        .with_context(context)
        .with_artifact(ArtifactInfo {
            id: "artifact".to_owned(),
            artifact_type: ARTIFACT_TYPE_CODE_EDIT.to_owned(),
            status: ARTIFACT_STATUS_PENDING.to_owned(),
            content: "new".to_owned(),
            diff_preview: "old -> new".to_owned(),
            title: "Edit".to_owned(),
            description: "Reason".to_owned(),
            applied_at: None,
        })
        .with_user_action(UserAction {
            action: "accept".to_owned(),
            timestamp: Utc::now(),
            artifact_id: "artifact".to_owned(),
            feedback: String::new(),
            reason: String::new(),
        })
        .build();
    assert!(validate(Some(&metadata)).is_ok());

    let invalid = MessageMetadata {
        artifact: Some(ArtifactInfo {
            artifact_type: "invented".to_owned(),
            ..ArtifactInfo::default()
        }),
        ..MessageMetadata::default()
    };
    assert!(validate(Some(&invalid)).is_err());
}
