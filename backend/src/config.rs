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
    pub openai_api_key: SecretString,
    pub openai_base_url: String,
    pub groq_api_key: SecretString,
    pub groq_base_url: String,
    pub exa_api_key: SecretString,
    pub exa_base_url: String,
    pub s3_endpoint: String,
    pub s3_access_key_id: SecretString,
    pub s3_secret_access_key: SecretString,
    pub s3_bucket: String,
    pub s3_url_prefix: String,
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
    #[error("{0} is required")]
    MissingStorageSetting(&'static str),
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
        let s3_endpoint = required_env("S3_ENDPOINT")?;
        let s3_access_key_id = required_env("S3_ACCESS_KEY_ID")?;
        let s3_secret_access_key = required_env("S3_ACCESS_KEY_SECRET")?;
        let s3_bucket = required_env("S3_BUCKET")?;
        let s3_url_prefix = required_env("S3_URL_PREFIX")?;

        Ok(Self {
            host: IpAddr::from_str(&host).map_err(|_| ConfigError::InvalidHost)?,
            port: port.parse().map_err(|_| ConfigError::InvalidPort)?,
            database_url: SecretString::from(database_url),
            jwt_secret: SecretString::from(jwt_secret),
            cors_origins,
            openai_api_key: SecretString::from(std::env::var("OPENAI_API_KEY").unwrap_or_default()),
            openai_base_url: std::env::var("OPENAI_BASE_URL")
                .unwrap_or_else(|_| "https://api.openai.com/v1".to_owned()),
            groq_api_key: SecretString::from(std::env::var("GROQ_API_KEY").unwrap_or_default()),
            groq_base_url: std::env::var("GROQ_BASE_URL")
                .unwrap_or_else(|_| "https://api.groq.com/openai/v1".to_owned()),
            exa_api_key: SecretString::from(std::env::var("EXA_API_KEY").unwrap_or_default()),
            exa_base_url: std::env::var("EXA_BASE_URL")
                .unwrap_or_else(|_| "https://api.exa.ai".to_owned()),
            s3_endpoint,
            s3_access_key_id: SecretString::from(s3_access_key_id),
            s3_secret_access_key: SecretString::from(s3_secret_access_key),
            s3_bucket,
            s3_url_prefix,
        })
    }
}

fn required_env(name: &'static str) -> Result<String, ConfigError> {
    std::env::var(name)
        .ok()
        .filter(|value| !value.is_empty())
        .ok_or(ConfigError::MissingStorageSetting(name))
}

pub fn database_url_from_env() -> Result<SecretString, ConfigError> {
    std::env::var("DATABASE_URL")
        .map(SecretString::from)
        .map_err(|_| ConfigError::MissingDatabaseUrl)
}
