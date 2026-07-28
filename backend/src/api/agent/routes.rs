use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_accept_artifact, __path_clear_conversation_history, __path_get_conversation_history,
        __path_get_pending_artifacts, __path_reject_artifact, __path_submit_agent_request,
        accept_artifact, clear_conversation_history, get_conversation_history,
        get_pending_artifacts, reject_artifact, submit_agent_request,
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
        .routes(routes!(
            get_conversation_history,
            clear_conversation_history
        ))
        .routes(routes!(get_pending_artifacts))
        .routes(routes!(accept_artifact))
        .routes(routes!(reject_artifact))
}
