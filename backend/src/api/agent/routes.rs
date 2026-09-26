use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_accept_artifact, __path_clear_conversation_history, __path_create_mcp_connector,
        __path_delete_mcp_connector, __path_get_conversation_history, __path_get_pending_artifacts,
        __path_list_agent_tools, __path_list_mcp_connectors, __path_refresh_mcp_connector,
        __path_reject_artifact, __path_submit_agent_request, __path_submit_conversation_turn,
        __path_update_mcp_connector, accept_artifact, clear_conversation_history,
        create_mcp_connector, delete_mcp_connector, get_conversation_history, get_pending_artifacts,
        list_agent_tools, list_mcp_connectors, refresh_mcp_connector, reject_artifact,
        submit_agent_request, submit_conversation_turn, update_mcp_connector,
    },
    state::AgentState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AgentState: FromRef<S>,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(submit_agent_request))
        .routes(routes!(submit_conversation_turn))
        .routes(routes!(list_agent_tools))
        .routes(routes!(list_mcp_connectors, create_mcp_connector))
        .routes(routes!(update_mcp_connector, delete_mcp_connector))
        .routes(routes!(refresh_mcp_connector))
        .routes(routes!(
            get_conversation_history,
            clear_conversation_history
        ))
        .routes(routes!(get_pending_artifacts))
        .routes(routes!(accept_artifact))
        .routes(routes!(reject_artifact))
}
