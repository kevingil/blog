use std::sync::Arc;

use async_trait::async_trait;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{core::ml::llm::ToolRegistry, error::AppError};

use super::{
    client::McpClient,
    tool::McpTool,
    types::{
        ConnectorRefresh, CreateMcpConnector, McpConnector, UpdateMcpConnector, TRANSPORT_HTTP,
        TRANSPORT_SSE, TRANSPORT_STDIO, sanitize_tool_prefix,
    },
};

#[async_trait]
pub trait McpConnectorRepository: Send + Sync {
    async fn list(&self) -> Result<Vec<McpConnector>, AppError>;
    async fn find(&self, id: Uuid) -> Result<McpConnector, AppError>;
    async fn create(&self, connector: &McpConnector) -> Result<McpConnector, AppError>;
    async fn update(&self, connector: &McpConnector) -> Result<McpConnector, AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}

pub struct McpConnectorService {
    repository: Arc<dyn McpConnectorRepository>,
    registry: Arc<ToolRegistry>,
    cancellation: CancellationToken,
}

impl McpConnectorService {
    pub fn new(
        repository: Arc<dyn McpConnectorRepository>,
        registry: Arc<ToolRegistry>,
        cancellation: CancellationToken,
    ) -> Arc<Self> {
        Arc::new(Self {
            repository,
            registry,
            cancellation,
        })
    }

    pub fn registry(&self) -> Arc<ToolRegistry> {
        self.registry.clone()
    }

    pub async fn list(&self) -> Result<Vec<McpConnector>, AppError> {
        self.cancelled()?;
        self.repository.list().await
    }

    pub async fn get(&self, id: Uuid) -> Result<McpConnector, AppError> {
        self.cancelled()?;
        self.repository.find(id).await
    }

    pub async fn create(&self, request: CreateMcpConnector) -> Result<McpConnector, AppError> {
        self.cancelled()?;
        validate_create(&request)?;
        let connector = McpConnector {
            id: Uuid::new_v4(),
            name: request.name.trim().to_owned(),
            transport: request.transport,
            command: request.command,
            args: request.args,
            url: request.url,
            headers: request.headers,
            env: request.env,
            enabled: request.enabled,
            last_error: String::new(),
            created_at: None,
            updated_at: None,
        };
        let saved = self.repository.create(&connector).await?;
        self.refresh_enabled().await?;
        Ok(saved)
    }

    pub async fn update(
        &self,
        id: Uuid,
        request: UpdateMcpConnector,
    ) -> Result<McpConnector, AppError> {
        self.cancelled()?;
        let mut connector = self.repository.find(id).await?;
        if let Some(name) = request.name {
            if name.trim().is_empty() {
                return Err(AppError::InvalidInput("name is required".to_owned()));
            }
            connector.name = name.trim().to_owned();
        }
        if let Some(transport) = request.transport {
            validate_transport(&transport)?;
            connector.transport = transport;
        }
        if let Some(command) = request.command {
            connector.command = command;
        }
        if let Some(args) = request.args {
            connector.args = args;
        }
        if let Some(url) = request.url {
            connector.url = url;
        }
        if let Some(headers) = request.headers {
            connector.headers = headers;
        }
        if let Some(env) = request.env {
            connector.env = env;
        }
        if let Some(enabled) = request.enabled {
            connector.enabled = enabled;
        }
        validate_connector(&connector)?;
        let saved = self.repository.update(&connector).await?;
        self.refresh_enabled().await?;
        Ok(saved)
    }

    pub async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.cancelled()?;
        self.repository.delete(id).await?;
        self.refresh_enabled().await?;
        Ok(())
    }

    pub async fn refresh(&self, id: Uuid) -> Result<ConnectorRefresh, AppError> {
        self.cancelled()?;
        let connector = self.repository.find(id).await?;
        match attach_connector(&connector).await {
            Ok(tools) => {
                let names = tools
                    .iter()
                    .map(|tool| tool.qualified_name.clone())
                    .collect();
                self.clear_error(connector).await?;
                self.refresh_enabled().await?;
                Ok(ConnectorRefresh {
                    connector_id: id,
                    tool_names: names,
                })
            }
            Err(error) => {
                self.record_error(connector, &error).await?;
                self.refresh_enabled().await?;
                Err(error)
            }
        }
    }

    pub async fn refresh_enabled(&self) -> Result<Vec<String>, AppError> {
        self.cancelled()?;
        let connectors = self.repository.list().await?;
        let mut tools = Vec::new();
        let mut names = Vec::new();
        for connector in connectors.into_iter().filter(|connector| connector.enabled) {
            match attach_connector(&connector).await {
                Ok(attached) => {
                    names.extend(attached.iter().map(|tool| tool.qualified_name.clone()));
                    tools.extend(attached.into_iter().map(|tool| {
                        Arc::new(tool) as Arc<dyn crate::core::ml::llm::Tool>
                    }));
                    self.clear_error(connector).await?;
                }
                Err(error) => {
                    self.record_error(connector, &error).await?;
                }
            }
        }
        self.registry.replace_extra(tools);
        Ok(names)
    }

    async fn record_error(&self, mut connector: McpConnector, error: &AppError) -> Result<(), AppError> {
        connector.last_error = error.to_string();
        self.repository.update(&connector).await?;
        Ok(())
    }

    async fn clear_error(&self, mut connector: McpConnector) -> Result<(), AppError> {
        if !connector.last_error.is_empty() {
            connector.last_error.clear();
            self.repository.update(&connector).await?;
        }
        Ok(())
    }

    fn cancelled(&self) -> Result<(), AppError> {
        if self.cancellation.is_cancelled() {
            Err(AppError::Internal)
        } else {
            Ok(())
        }
    }
}

async fn attach_connector(connector: &McpConnector) -> Result<Vec<McpTool>, AppError> {
    let client = Arc::new(McpClient::from_connector(connector)?);
    client.initialize().await?;
    let descriptors = client.list_tools().await?;
    let prefix = sanitize_tool_prefix(&connector.name);
    Ok(descriptors
        .into_iter()
        .map(|descriptor| McpTool::new(&prefix, descriptor, client.clone()))
        .collect())
}

fn validate_create(request: &CreateMcpConnector) -> Result<(), AppError> {
    if request.name.trim().is_empty() {
        return Err(AppError::InvalidInput("name is required".to_owned()));
    }
    validate_transport(&request.transport)?;
    let connector = McpConnector {
        id: Uuid::nil(),
        name: request.name.clone(),
        transport: request.transport.clone(),
        command: request.command.clone(),
        args: request.args.clone(),
        url: request.url.clone(),
        headers: request.headers.clone(),
        env: request.env.clone(),
        enabled: request.enabled,
        last_error: String::new(),
        created_at: None,
        updated_at: None,
    };
    validate_connector(&connector)
}

fn validate_transport(transport: &str) -> Result<(), AppError> {
    if matches!(transport, TRANSPORT_STDIO | TRANSPORT_SSE | TRANSPORT_HTTP) {
        Ok(())
    } else {
        Err(AppError::InvalidInput(format!(
            "transport must be {TRANSPORT_STDIO}, {TRANSPORT_SSE}, or {TRANSPORT_HTTP}"
        )))
    }
}

fn validate_connector(connector: &McpConnector) -> Result<(), AppError> {
    match connector.transport.as_str() {
        TRANSPORT_STDIO if connector.command.trim().is_empty() => Err(AppError::InvalidInput(
            "command is required for stdio connectors".to_owned(),
        )),
        TRANSPORT_HTTP | TRANSPORT_SSE if connector.url.trim().is_empty() => {
            Err(AppError::InvalidInput(
                "url is required for http/sse connectors".to_owned(),
            ))
        }
        _ => Ok(()),
    }
}
