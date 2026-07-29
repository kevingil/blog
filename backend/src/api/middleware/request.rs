use std::time::Duration;

use axum::{body::Body, http::Request, response::Response};

pub const fn timeout() -> Duration {
    crate::constants::DEFAULT_REQUEST_TIMEOUT
}

pub type HttpRequest = Request<Body>;
pub type HttpResponse = Response;
