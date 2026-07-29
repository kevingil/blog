use axum::{
    extract::{FromRef, State, WebSocketUpgrade},
    response::Response,
};
use utoipa_axum::{router::OpenApiRouter, routes};

use super::supervisor::{WebSocketSupervisorHandle, hand_off_upgrade};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    WebSocketSupervisorHandle: FromRef<S>,
{
    OpenApiRouter::new().routes(routes!(websocket_handler))
}

#[utoipa::path(
    get,
    path = "/websocket",
    operation_id = "connectWebSocket",
    tag = "agent",
    responses(
        (status = 101, description = "WebSocket switching protocols"),
        (status = 400, description = "Invalid WebSocket upgrade headers"),
        (status = 426, description = "Connection cannot be upgraded")
    )
)]
async fn websocket_handler(
    State(supervisor): State<WebSocketSupervisorHandle>,
    upgrade: WebSocketUpgrade,
) -> Response {
    upgrade.on_upgrade(move |socket| hand_off_upgrade(socket, supervisor))
}
