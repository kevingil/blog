use std::collections::BTreeMap;

use async_trait::async_trait;
use chrono::Utc;
use diesel::{ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper};
use diesel_async::RunQueryDsl;
use serde_json::{Value, json};
use uuid::Uuid;

use crate::{
    core::mcp::{McpConnector, McpConnectorRepository},
    database::{
        models::mcp_connector::{McpConnectorRow, NewMcpConnectorRow},
        pool::PgPool,
    },
    error::AppError,
    schema::mcp_connector,
};

#[derive(Clone)]
pub struct DieselMcpConnectorRepository {
    pool: PgPool,
}

impl DieselMcpConnectorRepository {
    pub const fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }
}

#[async_trait]
impl McpConnectorRepository for DieselMcpConnectorRepository {
    async fn list(&self) -> Result<Vec<McpConnector>, AppError> {
        let mut connection = self.connection().await?;
        let rows = mcp_connector::table
            .order(mcp_connector::created_at.asc())
            .select(McpConnectorRow::as_select())
            .load::<McpConnectorRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        rows.into_iter().map(TryFrom::try_from).collect()
    }

    async fn find(&self, id: Uuid) -> Result<McpConnector, AppError> {
        let mut connection = self.connection().await?;
        mcp_connector::table
            .find(id)
            .select(McpConnectorRow::as_select())
            .first::<McpConnectorRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn create(&self, connector: &McpConnector) -> Result<McpConnector, AppError> {
        let mut connection = self.connection().await?;
        let inserted = diesel::insert_into(mcp_connector::table)
            .values(NewMcpConnectorRow::from(connector))
            .returning(McpConnectorRow::as_returning())
            .get_result::<McpConnectorRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        inserted.try_into()
    }

    async fn update(&self, connector: &McpConnector) -> Result<McpConnector, AppError> {
        let mut connection = self.connection().await?;
        let updated = diesel::update(mcp_connector::table.find(connector.id))
            .set((
                mcp_connector::name.eq(&connector.name),
                mcp_connector::transport.eq(&connector.transport),
                mcp_connector::command.eq(optional_text(&connector.command)),
                mcp_connector::args.eq(string_array(&connector.args)),
                mcp_connector::url.eq(optional_text(&connector.url)),
                mcp_connector::headers.eq(string_map(&connector.headers)),
                mcp_connector::env.eq(string_map(&connector.env)),
                mcp_connector::enabled.eq(connector.enabled),
                mcp_connector::last_error.eq(optional_text(&connector.last_error)),
                mcp_connector::updated_at.eq(Some(Utc::now())),
            ))
            .returning(McpConnectorRow::as_returning())
            .get_result::<McpConnectorRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)?;
        updated.try_into()
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let deleted = diesel::delete(mcp_connector::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        if deleted == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}

impl From<&McpConnector> for NewMcpConnectorRow {
    fn from(connector: &McpConnector) -> Self {
        Self {
            id: connector.id,
            name: connector.name.clone(),
            transport: connector.transport.clone(),
            command: optional_text(&connector.command),
            args: string_array(&connector.args),
            url: optional_text(&connector.url),
            headers: string_map(&connector.headers),
            env: string_map(&connector.env),
            enabled: connector.enabled,
            last_error: optional_text(&connector.last_error),
        }
    }
}

impl TryFrom<McpConnectorRow> for McpConnector {
    type Error = AppError;

    fn try_from(row: McpConnectorRow) -> Result<Self, Self::Error> {
        Ok(Self {
            id: row.id,
            name: row.name,
            transport: row.transport,
            command: row.command.unwrap_or_default(),
            args: value_strings(&row.args),
            url: row.url.unwrap_or_default(),
            headers: value_map(&row.headers),
            env: value_map(&row.env),
            enabled: row.enabled,
            last_error: row.last_error.unwrap_or_default(),
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}

fn optional_text(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_owned())
    }
}

fn string_array(values: &[String]) -> Value {
    json!(values)
}

fn string_map(values: &BTreeMap<String, String>) -> Value {
    json!(values)
}

fn value_strings(value: &Value) -> Vec<String> {
    value
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(Value::as_str)
        .map(ToOwned::to_owned)
        .collect()
}

fn value_map(value: &Value) -> BTreeMap<String, String> {
    value
        .as_object()
        .into_iter()
        .flatten()
        .filter_map(|(key, value)| Some((key.clone(), value.as_str()?.to_owned())))
        .collect()
}
