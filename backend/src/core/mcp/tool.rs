use std::{collections::BTreeMap, sync::Arc};

use async_trait::async_trait;
use serde_json::{Map, Value, json};

use crate::{
    core::ml::llm::{Tool, ToolCallRequest, ToolContext, ToolInfo, ToolResponse},
    error::AppError,
};

use super::{client::McpClient, types::McpToolDescriptor};

pub struct McpTool {
    pub qualified_name: String,
    remote_name: String,
    description: String,
    parameters: BTreeMap<String, Value>,
    required: Vec<String>,
    client: Arc<McpClient>,
}

impl McpTool {
    pub fn new(prefix: &str, descriptor: McpToolDescriptor, client: Arc<McpClient>) -> Self {
        let (parameters, required) = schema_fields(&descriptor.input_schema);
        Self {
            qualified_name: format!("{prefix}__{}", descriptor.name),
            remote_name: descriptor.name,
            description: if descriptor.description.is_empty() {
                "MCP connector tool".to_owned()
            } else {
                descriptor.description
            },
            parameters,
            required,
            client,
        }
    }
}

#[async_trait]
impl Tool for McpTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: self.qualified_name.clone(),
            description: self.description.clone(),
            parameters: self.parameters.clone(),
            required: self.required.clone(),
            parallel_safe: true,
        }
    }

    async fn run(
        &self,
        _context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let mut arguments: Value = serde_json::from_str(&call.input).unwrap_or_else(|_| json!({}));
        omit_null_fields(&mut arguments);
        let result = self
            .client
            .call_tool(&self.remote_name, arguments)
            .await
            .map_err(|_| AppError::External)?;
        let content = mcp_result_text(&result);
        let mut object = Map::new();
        object.insert(
            "tool_name".to_owned(),
            Value::String(self.qualified_name.clone()),
        );
        object.insert("mcp_result".to_owned(), result);
        object.insert("content".to_owned(), Value::String(content.clone()));
        Ok(ToolResponse::structured(content, object, None))
    }
}

fn omit_null_fields(value: &mut Value) {
    match value {
        Value::Object(map) => {
            map.retain(|_, child| !child.is_null());
            for child in map.values_mut() {
                omit_null_fields(child);
            }
        }
        Value::Array(items) => {
            for item in items {
                omit_null_fields(item);
            }
        }
        _ => {}
    }
}

fn schema_fields(schema: &Value) -> (BTreeMap<String, Value>, Vec<String>) {
    let properties = schema
        .get("properties")
        .and_then(Value::as_object)
        .cloned()
        .unwrap_or_default();
    let required = schema
        .get("required")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
        .filter_map(Value::as_str)
        .map(ToOwned::to_owned)
        .collect();
    (properties.into_iter().collect(), required)
}

fn mcp_result_text(result: &Value) -> String {
    if let Some(content) = result.get("content").and_then(Value::as_array) {
        let text = content
            .iter()
            .filter_map(|part| {
                if part.get("type").and_then(Value::as_str) == Some("text") {
                    part.get("text").and_then(Value::as_str)
                } else {
                    None
                }
            })
            .collect::<Vec<_>>()
            .join("\n");
        if !text.is_empty() {
            return text;
        }
    }
    serde_json::to_string(result).unwrap_or_else(|_| result.to_string())
}
