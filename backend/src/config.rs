use std::{net::IpAddr, str::FromStr};

use secrecy::SecretString;
use thiserror::Error;

use crate::constants::{DEFAULT_CORS_ORIGINS, DEFAULT_HOST, DEFAULT_PORT};

#[derive(Clone)]
pub struct Config {
    pub host: IpAddr,
    pub port: u16,
    pub database_url: SecretString,
    pub jwt_secret: SecretString,
    pub cors_origins: Vec<String>,
}

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("DATABASE_URL is required")]
    MissingDatabaseUrl,
    #[error("AUTH_SECRET is required")]
    MissingJwtSecret,
    #[error("HOST must be a valid IP address")]
    InvalidHost,
    #[error("PORT must be a valid TCP port")]
    InvalidPort,
}

impl Config {
    pub fn from_env() -> Result<Self, ConfigError> {
        let host = std::env::var("HOST").unwrap_or_else(|_| DEFAULT_HOST.to_owned());
        let port = std::env::var("PORT").unwrap_or_else(|_| DEFAULT_PORT.to_string());
        let database_url =
            std::env::var("DATABASE_URL").map_err(|_| ConfigError::MissingDatabaseUrl)?;
        let jwt_secret = std::env::var("AUTH_SECRET").map_err(|_| ConfigError::MissingJwtSecret)?;
        if jwt_secret.is_empty() {
            return Err(ConfigError::MissingJwtSecret);
        }
        let cors_origins = std::env::var("ALLOWED_ORIGINS")
            .unwrap_or_else(|_| DEFAULT_CORS_ORIGINS.to_owned())
            .split(',')
            .map(str::trim)
            .filter(|origin| !origin.is_empty())
            .map(ToOwned::to_owned)
            .collect();

        Ok(Self {
            host: IpAddr::from_str(&host).map_err(|_| ConfigError::InvalidHost)?,
            port: port.parse().map_err(|_| ConfigError::InvalidPort)?,
            database_url: SecretString::from(database_url),
            jwt_secret: SecretString::from(jwt_secret),
            cors_origins,
        })
    }
}

pub fn database_url_from_env() -> Result<SecretString, ConfigError> {
    std::env::var("DATABASE_URL")
        .map(SecretString::from)
        .map_err(|_| ConfigError::MissingDatabaseUrl)
}
