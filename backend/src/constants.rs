use std::time::Duration;

pub const DEFAULT_PORT: u16 = 8080;
pub const DEFAULT_HOST: &str = "0.0.0.0";
pub const DEFAULT_REQUEST_TIMEOUT: Duration = Duration::from_secs(30);
pub const BACKGROUND_SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(10);
pub const DEFAULT_CORS_ORIGINS: &str = "http://localhost:3000,http://localhost:5173,http://localhost:8080,http://127.0.0.1:3000,http://127.0.0.1:5173,http://127.0.0.1:8080";
pub const MAX_REQUEST_BODY_BYTES: usize = 50 * 1024 * 1024;
pub const WEBSOCKET_BUFFER_CAPACITY: usize = 256;
