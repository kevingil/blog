use std::{
    collections::BTreeMap,
    process::Stdio,
    sync::atomic::{AtomicU64, Ordering},
    time::Duration,
};

use async_trait::async_trait;
use serde_json::{Value, json};
use tokio::{
    io::{AsyncBufReadExt, AsyncReadExt, AsyncWriteExt, BufReader},
    process::Command,
    time::timeout,
};

use crate::error::AppError;

use super::types::{McpConnector, McpToolDescriptor, TRANSPORT_HTTP, TRANSPORT_SSE, TRANSPORT_STDIO};

const PROTOCOL_VERSION: &str = "2024-11-05";
const REQUEST_TIMEOUT: Duration = Duration::from_secs(20);

#[async_trait]
pub trait McpTransport: Send + Sync {
    async fn request(&self, method: &str, params: Value) -> Result<Value, AppError>;
    async fn notify(&self, method: &str, params: Value) -> Result<(), AppError>;
}

pub struct McpClient {
    transport: Box<dyn McpTransport>,
}

impl McpClient {
    pub fn new(transport: Box<dyn McpTransport>) -> Self {
        Self { transport }
    }

    pub fn from_connector(connector: &McpConnector) -> Result<Self, AppError> {
        let transport: Box<dyn McpTransport> = match connector.transport.as_str() {
            TRANSPORT_STDIO => Box::new(StdioMcpTransport::spawn(connector)?),
            TRANSPORT_HTTP | TRANSPORT_SSE => Box::new(HttpMcpTransport::new(connector)?),
            other => {
                return Err(AppError::InvalidInput(format!(
                    "unsupported MCP transport: {other}"
                )));
            }
        };
        Ok(Self { transport })
    }

    pub async fn initialize(&self) -> Result<Value, AppError> {
        let result = self
            .transport
            .request(
                "initialize",
                json!({
                    "protocolVersion": PROTOCOL_VERSION,
                    "capabilities": {},
                    "clientInfo": {
                        "name": "blog-copilot",
                        "version": env!("CARGO_PKG_VERSION"),
                    }
                }),
            )
            .await?;
        self.transport
            .notify("notifications/initialized", json!({}))
            .await?;
        Ok(result)
    }

    pub async fn list_tools(&self) -> Result<Vec<McpToolDescriptor>, AppError> {
        let result = self.transport.request("tools/list", json!({})).await?;
        let tools = result
            .get("tools")
            .and_then(Value::as_array)
            .cloned()
            .unwrap_or_default();
        Ok(tools
            .into_iter()
            .filter_map(|tool| {
                let name = tool.get("name")?.as_str()?.to_owned();
                if name.is_empty() {
                    return None;
                }
                Some(McpToolDescriptor {
                    name,
                    description: tool
                        .get("description")
                        .and_then(Value::as_str)
                        .unwrap_or_default()
                        .to_owned(),
                    input_schema: tool
                        .get("inputSchema")
                        .cloned()
                        .unwrap_or_else(|| json!({"type": "object"})),
                })
            })
            .collect())
    }

    pub async fn call_tool(&self, name: &str, arguments: Value) -> Result<Value, AppError> {
        self.transport
            .request(
                "tools/call",
                json!({
                    "name": name,
                    "arguments": arguments,
                }),
            )
            .await
    }
}

pub struct ScriptedMcpTransport {
    next_id: AtomicU64,
    handler: Box<dyn Fn(&str, Value) -> Result<Value, AppError> + Send + Sync>,
}

impl ScriptedMcpTransport {
    pub fn new(
        handler: impl Fn(&str, Value) -> Result<Value, AppError> + Send + Sync + 'static,
    ) -> Self {
        Self {
            next_id: AtomicU64::new(1),
            handler: Box::new(handler),
        }
    }
}

#[async_trait]
impl McpTransport for ScriptedMcpTransport {
    async fn request(&self, method: &str, params: Value) -> Result<Value, AppError> {
        let _ = self.next_id.fetch_add(1, Ordering::Relaxed);
        (self.handler)(method, params)
    }

    async fn notify(&self, _method: &str, _params: Value) -> Result<(), AppError> {
        Ok(())
    }
}

pub struct HttpMcpTransport {
    client: reqwest::Client,
    url: String,
    headers: BTreeMap<String, String>,
    next_id: AtomicU64,
}

impl HttpMcpTransport {
    pub fn new(connector: &McpConnector) -> Result<Self, AppError> {
        if connector.url.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "MCP HTTP/SSE connector requires a url".to_owned(),
            ));
        }
        Ok(Self {
            client: reqwest::Client::builder()
                .timeout(REQUEST_TIMEOUT)
                .build()
                .map_err(|_| AppError::Internal)?,
            url: connector.url.clone(),
            headers: connector.headers.clone(),
            next_id: AtomicU64::new(1),
        })
    }

    fn apply_headers(&self, mut builder: reqwest::RequestBuilder) -> reqwest::RequestBuilder {
        for (name, value) in &self.headers {
            builder = builder.header(name, value);
        }
        builder
    }
}

#[async_trait]
impl McpTransport for HttpMcpTransport {
    async fn request(&self, method: &str, params: Value) -> Result<Value, AppError> {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        let payload = json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params,
        });
        let builder = self.apply_headers(
            self.client
                .post(&self.url)
                .header("content-type", "application/json")
                .header("accept", "application/json, text/event-stream")
                .json(&payload),
        );
        let response = builder.send().await.map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let content_type = response
            .headers()
            .get(reqwest::header::CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .unwrap_or_default()
            .to_owned();
        if content_type.contains("text/event-stream") {
            let text = response.text().await.map_err(|_| AppError::External)?;
            return parse_json_rpc_result(&first_sse_data(&text));
        }
        let body: Value = response.json().await.map_err(|_| AppError::External)?;
        parse_json_rpc_result(&body)
    }

    async fn notify(&self, method: &str, params: Value) -> Result<(), AppError> {
        let payload = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
        });
        let builder = self.apply_headers(
            self.client
                .post(&self.url)
                .header("content-type", "application/json")
                .json(&payload),
        );
        let response = builder.send().await.map_err(|_| AppError::External)?;
        if response.status().is_success() || response.status().as_u16() == 202 {
            Ok(())
        } else {
            Err(AppError::External)
        }
    }
}

pub struct StdioMcpTransport {
    child: tokio::sync::Mutex<tokio::process::Child>,
    next_id: AtomicU64,
}

impl StdioMcpTransport {
    pub fn spawn(connector: &McpConnector) -> Result<Self, AppError> {
        if connector.command.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "MCP stdio connector requires a command".to_owned(),
            ));
        }
        let mut command = Command::new(&connector.command);
        command
            .args(&connector.args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .kill_on_drop(true);
        for (key, value) in &connector.env {
            command.env(key, value);
        }
        let child = command.spawn().map_err(|_| AppError::External)?;
        Ok(Self {
            child: tokio::sync::Mutex::new(child),
            next_id: AtomicU64::new(1),
        })
    }
}

#[async_trait]
impl McpTransport for StdioMcpTransport {
    async fn request(&self, method: &str, params: Value) -> Result<Value, AppError> {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        let payload = json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params,
        });
        let encoded = serde_json::to_vec(&payload).map_err(|_| AppError::Internal)?;
        let mut child = self.child.lock().await;
        let stdin = child.stdin.as_mut().ok_or(AppError::External)?;
        let header = format!("Content-Length: {}\r\n\r\n", encoded.len());
        stdin
            .write_all(header.as_bytes())
            .await
            .map_err(|_| AppError::External)?;
        stdin
            .write_all(&encoded)
            .await
            .map_err(|_| AppError::External)?;
        stdin.flush().await.map_err(|_| AppError::External)?;
        let stdout = child.stdout.as_mut().ok_or(AppError::External)?;
        let body = timeout(REQUEST_TIMEOUT, read_stdio_message(stdout))
            .await
            .map_err(|_| AppError::External)?
            .map_err(|_| AppError::External)?;
        parse_json_rpc_result(&body)
    }

    async fn notify(&self, method: &str, params: Value) -> Result<(), AppError> {
        let payload = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
        });
        let encoded = serde_json::to_vec(&payload).map_err(|_| AppError::Internal)?;
        let mut child = self.child.lock().await;
        let stdin = child.stdin.as_mut().ok_or(AppError::External)?;
        let header = format!("Content-Length: {}\r\n\r\n", encoded.len());
        stdin
            .write_all(header.as_bytes())
            .await
            .map_err(|_| AppError::External)?;
        stdin
            .write_all(&encoded)
            .await
            .map_err(|_| AppError::External)?;
        stdin.flush().await.map_err(|_| AppError::External)?;
        Ok(())
    }
}

fn parse_json_rpc_result(body: &Value) -> Result<Value, AppError> {
    if body.get("error").is_some() {
        return Err(AppError::External);
    }
    Ok(body.get("result").cloned().unwrap_or(Value::Null))
}

fn first_sse_data(stream: &str) -> Value {
    stream
        .lines()
        .find_map(|line| {
            let data = line.strip_prefix("data:")?.trim();
            if data.is_empty() || data == "[DONE]" {
                return None;
            }
            serde_json::from_str(data).ok()
        })
        .unwrap_or(Value::Null)
}

async fn read_stdio_message<R: AsyncReadExt + Unpin>(
    stdout: &mut R,
) -> Result<Value, std::io::Error> {
    let mut reader = BufReader::new(stdout);
    let mut content_length = None;
    loop {
        let mut line = String::new();
        let read = reader.read_line(&mut line).await?;
        if read == 0 {
            return Err(std::io::Error::new(
                std::io::ErrorKind::UnexpectedEof,
                "mcp stdio closed",
            ));
        }
        let trimmed = line.trim();
        if trimmed.is_empty() {
            break;
        }
        if let Some(value) = trimmed
            .strip_prefix("Content-Length:")
            .or_else(|| trimmed.strip_prefix("content-length:"))
        {
            content_length = value.trim().parse::<usize>().ok();
        }
    }
    if let Some(length) = content_length {
        let mut buffer = vec![0u8; length];
        reader.read_exact(&mut buffer).await?;
        return serde_json::from_slice(&buffer).map_err(|error| {
            std::io::Error::new(std::io::ErrorKind::InvalidData, error.to_string())
        });
    }
    let mut line = String::new();
    reader.read_line(&mut line).await?;
    serde_json::from_str(line.trim())
        .map_err(|error| std::io::Error::new(std::io::ErrorKind::InvalidData, error.to_string()))
}
