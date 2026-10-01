use std::sync::Arc;

use blog_backend::{
    core::{
        mcp::{
            CreateMcpConnector, InMemoryMcpConnectorRepository, McpClient, McpConnectorService,
            McpTool, ScriptedMcpTransport, TRANSPORT_HTTP,
        },
        ml::llm::{
            Tool, ToolCallRequest, ToolContext, ToolRegistry,
        },
    },
    error::AppError,
};
use serde_json::{Value, json};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

#[tokio::test]
async fn scripted_mcp_client_lists_and_calls_tools() {
    let client = McpClient::new(Box::new(ScriptedMcpTransport::new(|method, params| {
        match method {
            "initialize" => Ok(json!({"protocolVersion": "2024-11-05"})),
            "tools/list" => Ok(json!({
                "tools": [{
                    "name": "lookup_docs",
                    "description": "Look up documentation",
                    "inputSchema": {
                        "type": "object",
                        "properties": {"query": {"type": "string"}},
                        "required": ["query"]
                    }
                }]
            })),
            "tools/call" => {
                let query = params
                    .get("arguments")
                    .and_then(|value| value.get("query"))
                    .and_then(Value::as_str)
                    .unwrap_or_default();
                Ok(json!({
                    "content": [{"type": "text", "text": format!("docs for {query}")}]
                }))
            }
            other => Err(AppError::InvalidInput(other.to_owned())),
        }
    })));
    client
        .initialize()
        .await
        .expect("initialize");
    let tools = client.list_tools().await.expect("list tools");
    assert_eq!(tools.len(), 1);
    assert_eq!(tools[0].name, "lookup_docs");
    let result = client
        .call_tool("lookup_docs", json!({"query": "axum"}))
        .await
        .expect("call tool");
    assert!(
        result
            .get("content")
            .and_then(Value::as_array)
            .is_some_and(|content| content
                .iter()
                .any(|part| part.get("text").and_then(Value::as_str) == Some("docs for axum")))
    );
}

#[tokio::test]
async fn mcp_tool_wraps_remote_result_for_the_agent_harness() {
    let client = Arc::new(McpClient::new(Box::new(ScriptedMcpTransport::new(
        |method, _params| match method {
            "tools/call" => Ok(json!({
                "content": [{"type": "text", "text": "connector answer"}]
            })),
            _ => Ok(json!({})),
        },
    ))));
    let tool = McpTool::new(
        "exa_docs",
        blog_backend::core::mcp::McpToolDescriptor {
            name: "lookup_docs".to_owned(),
            description: "Look up documentation".to_owned(),
            input_schema: json!({"type": "object"}),
        },
        client,
    );
    assert_eq!(tool.info().name, "exa_docs__lookup_docs");
    let response = tool
        .run(
            ToolContext::new(
                "session",
                "message",
                "request",
                None,
                "",
                "",
                CancellationToken::new(),
            ),
            ToolCallRequest {
                id: "call_1".to_owned(),
                name: tool.info().name,
                input: json!({"query": "axum"}).to_string(),
            },
        )
        .await
        .expect("run mcp tool");
    assert!(!response.is_error);
    assert!(response.content.contains("connector answer"));
    assert_eq!(
        response.result.get("tool_name").and_then(Value::as_str),
        Some("exa_docs__lookup_docs")
    );
}

#[tokio::test]
async fn connector_service_validates_and_lists_without_connecting_disabled_servers() {
    let registry = ToolRegistry::from_builtin(Vec::new());
    let service = McpConnectorService::new(
        Arc::new(InMemoryMcpConnectorRepository::default()),
        registry.clone(),
        CancellationToken::new(),
    );
    let created = service
        .create(CreateMcpConnector {
            name: "Docs".to_owned(),
            transport: TRANSPORT_HTTP.to_owned(),
            url: "http://127.0.0.1:9/mcp".to_owned(),
            enabled: false,
            ..CreateMcpConnector::default()
        })
        .await
        .expect("create connector");
    assert_eq!(created.name, "Docs");
    assert!(!created.enabled);
    let listed = service.list().await.expect("list");
    assert_eq!(listed.len(), 1);
    assert!(registry.names().is_empty());
    let missing = service
        .create(CreateMcpConnector {
            name: "Broken".to_owned(),
            transport: TRANSPORT_HTTP.to_owned(),
            enabled: true,
            ..CreateMcpConnector::default()
        })
        .await;
    assert!(matches!(missing, Err(AppError::InvalidInput(_))));
    let _ = created.id != Uuid::nil();
}
