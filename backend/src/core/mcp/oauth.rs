use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
    time::{Duration, Instant},
};

use base64::{Engine as _, engine::general_purpose::URL_SAFE_NO_PAD};
use serde::Deserialize;
use serde_json::json;
use sha2::{Digest, Sha256};
use uuid::Uuid;

use crate::error::AppError;

const SESSION_TTL: Duration = Duration::from_secs(15 * 60);

#[derive(Clone)]
pub struct McpOauth {
    http: reqwest::Client,
    sessions: OauthSessions,
    public_api_url: String,
    public_app_url: String,
}

#[derive(Clone, Default)]
struct OauthSessions {
    pending: Arc<Mutex<HashMap<String, PendingOauth>>>,
    clients: Arc<Mutex<HashMap<String, RegisteredClient>>>,
}

struct PendingOauth {
    created: Instant,
    code_verifier: String,
    client_id: String,
    client_secret: String,
    token_endpoint: String,
    resource: String,
    redirect_uri: String,
    name: String,
    preset_id: String,
    server_url: String,
}

#[derive(Clone)]
struct RegisteredClient {
    client_id: String,
    client_secret: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CompletedOauth {
    pub name: String,
    pub preset_id: String,
    pub server_url: String,
    pub access_token: String,
    pub refresh_token: String,
    pub token_endpoint: String,
    pub client_id: String,
    pub resource: String,
}

#[derive(Debug, Deserialize)]
struct ProtectedResource {
    resource: String,
    #[serde(default)]
    authorization_servers: Vec<String>,
    #[serde(default)]
    scopes_supported: Vec<String>,
}

#[derive(Debug, Deserialize)]
struct AuthorizationServer {
    authorization_endpoint: String,
    token_endpoint: String,
    #[serde(default)]
    registration_endpoint: String,
}

#[derive(Debug, Deserialize)]
struct ClientRegistration {
    client_id: String,
    #[serde(default)]
    client_secret: String,
}

#[derive(Debug, Deserialize)]
struct TokenResponse {
    access_token: String,
    #[serde(default)]
    refresh_token: String,
}

impl McpOauth {
    pub fn new(public_api_url: impl Into<String>, public_app_url: impl Into<String>) -> Self {
        let http = reqwest::Client::builder()
            .timeout(Duration::from_secs(20))
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());
        Self {
            http,
            sessions: OauthSessions::default(),
            public_api_url: trim_slash(&public_api_url.into()),
            public_app_url: trim_slash(&public_app_url.into()),
        }
    }

    pub fn redirect_uri(&self) -> String {
        format!("{}/agent/connectors/oauth/callback", self.public_api_url)
    }

    pub fn app_connectors_url(&self) -> String {
        format!("{}/dashboard/connectors", self.public_app_url)
    }

    pub async fn start(
        &self,
        server_url: &str,
        name: &str,
        preset_id: &str,
    ) -> Result<String, AppError> {
        let metadata = discover(&self.http, server_url).await?;
        let redirect_uri = self.redirect_uri();
        let client = self
            .client_for(&metadata.registration_endpoint, &redirect_uri)
            .await?;
        let code_verifier = format!("{}{}", Uuid::new_v4().simple(), Uuid::new_v4().simple());
        let challenge = code_challenge(&code_verifier);
        let state = Uuid::new_v4().simple().to_string();
        let resource = if metadata.resource.is_empty() {
            server_url.to_owned()
        } else {
            metadata.resource.clone()
        };
        self.sessions.insert(state.clone(), PendingOauth {
            created: Instant::now(),
            code_verifier,
            client_id: client.client_id.clone(),
            client_secret: client.client_secret,
            token_endpoint: metadata.token_endpoint.clone(),
            resource: resource.clone(),
            redirect_uri: redirect_uri.clone(),
            name: name.to_owned(),
            preset_id: preset_id.to_owned(),
            server_url: server_url.to_owned(),
        });
        Ok(authorize_url(
            &metadata.authorization_endpoint,
            &client.client_id,
            &redirect_uri,
            &challenge,
            &state,
            &metadata.scopes,
            &resource,
        ))
    }

    pub async fn finish(&self, code: &str, state: &str) -> Result<CompletedOauth, AppError> {
        let pending = self.sessions.take(state).ok_or_else(|| {
            AppError::InvalidInput("Sign-in session expired. Click Connect again.".to_owned())
        })?;
        if pending.created.elapsed() > SESSION_TTL {
            return Err(AppError::InvalidInput(
                "Sign-in session expired. Click Connect again.".to_owned(),
            ));
        }
        let token = exchange_code(&self.http, &pending, code).await?;
        if token.access_token.is_empty() {
            return Err(AppError::InvalidInput(
                "The provider did not return an access token.".to_owned(),
            ));
        }
        Ok(CompletedOauth {
            name: pending.name,
            preset_id: pending.preset_id,
            server_url: pending.server_url,
            access_token: token.access_token,
            refresh_token: token.refresh_token,
            token_endpoint: pending.token_endpoint,
            client_id: pending.client_id,
            resource: pending.resource,
        })
    }

    async fn client_for(
        &self,
        registration_endpoint: &str,
        redirect_uri: &str,
    ) -> Result<RegisteredClient, AppError> {
        let cache_key = format!("{registration_endpoint}|{redirect_uri}");
        if let Some(client) = self.sessions.cached_client(&cache_key) {
            return Ok(client);
        }
        let client = register_client(&self.http, registration_endpoint, redirect_uri).await?;
        self.sessions.remember_client(cache_key, client.clone());
        Ok(client)
    }
}

impl OauthSessions {
    fn insert(&self, state: String, pending: PendingOauth) {
        let mut pending_map = self
            .pending
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        pending_map.insert(state, pending);
    }

    fn take(&self, state: &str) -> Option<PendingOauth> {
        let mut pending_map = self
            .pending
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        pending_map.remove(state)
    }

    fn cached_client(&self, key: &str) -> Option<RegisteredClient> {
        let clients = self
            .clients
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        clients.get(key).cloned()
    }

    fn remember_client(&self, key: String, client: RegisteredClient) {
        let mut clients = self
            .clients
            .lock()
            .unwrap_or_else(|error| error.into_inner());
        clients.insert(key, client);
    }
}

pub fn code_challenge(verifier: &str) -> String {
    let digest = Sha256::digest(verifier.as_bytes());
    URL_SAFE_NO_PAD.encode(digest)
}

pub fn resource_metadata_url(header: &str) -> Option<String> {
    quoted_param(header, "resource_metadata")
}

pub fn authorize_url(
    endpoint: &str,
    client_id: &str,
    redirect_uri: &str,
    challenge: &str,
    state: &str,
    scopes: &[String],
    resource: &str,
) -> String {
    let mut query = vec![
        ("response_type", "code".to_owned()),
        ("client_id", client_id.to_owned()),
        ("redirect_uri", redirect_uri.to_owned()),
        ("code_challenge", challenge.to_owned()),
        ("code_challenge_method", "S256".to_owned()),
        ("state", state.to_owned()),
        ("resource", resource.to_owned()),
    ];
    if !scopes.is_empty() {
        query.push(("scope", scopes.join(" ")));
    }
    let encoded = query
        .into_iter()
        .map(|(key, value)| format!("{key}={}", form_encode(&value)))
        .collect::<Vec<_>>()
        .join("&");
    format!("{endpoint}?{encoded}")
}

async fn discover(http: &reqwest::Client, server_url: &str) -> Result<DiscoveredOauth, AppError> {
    let metadata_url = metadata_url_from_probe(http, server_url)
        .await?
        .or_else(|| fallback_metadata_url(server_url));
    let Some(metadata_url) = metadata_url else {
        return Err(AppError::InvalidInput(
            "This MCP server did not advertise an OAuth sign-in.".to_owned(),
        ));
    };
    let protected: ProtectedResource = get_json(http, &metadata_url).await?;
    let issuer = protected
        .authorization_servers
        .into_iter()
        .find(|server| !server.trim().is_empty())
        .ok_or_else(|| {
            AppError::InvalidInput(
                "This MCP server did not advertise an authorization server.".to_owned(),
            )
        })?;
    let server: AuthorizationServer =
        get_json(http, &authorization_server_metadata_url(&issuer)).await?;
    if server.authorization_endpoint.is_empty() || server.token_endpoint.is_empty() {
        return Err(AppError::InvalidInput(
            "The authorization server is missing sign-in endpoints.".to_owned(),
        ));
    }
    if server.registration_endpoint.is_empty() {
        return Err(AppError::InvalidInput(
            "The authorization server does not support dynamic client registration.".to_owned(),
        ));
    }
    Ok(DiscoveredOauth {
        authorization_endpoint: server.authorization_endpoint,
        token_endpoint: server.token_endpoint,
        registration_endpoint: server.registration_endpoint,
        resource: if protected.resource.is_empty() {
            server_url.to_owned()
        } else {
            protected.resource
        },
        scopes: protected.scopes_supported,
    })
}

struct DiscoveredOauth {
    authorization_endpoint: String,
    token_endpoint: String,
    registration_endpoint: String,
    resource: String,
    scopes: Vec<String>,
}

async fn metadata_url_from_probe(
    http: &reqwest::Client,
    server_url: &str,
) -> Result<Option<String>, AppError> {
    let response = http
        .post(server_url)
        .header("accept", "application/json, text/event-stream")
        .header("content-type", "application/json")
        .body(
            json!({
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {}
            })
            .to_string(),
        )
        .send()
        .await
        .map_err(|_| {
            AppError::InvalidInput("Could not reach the MCP server to start sign-in.".to_owned())
        })?;
    let header = response
        .headers()
        .get(reqwest::header::WWW_AUTHENTICATE)
        .and_then(|value| value.to_str().ok())
        .unwrap_or_default()
        .to_owned();
    Ok(resource_metadata_url(&header))
}

async fn register_client(
    http: &reqwest::Client,
    registration_endpoint: &str,
    redirect_uri: &str,
) -> Result<RegisteredClient, AppError> {
    let response = http
        .post(registration_endpoint)
        .json(&json!({
            "client_name": "Blog Copilot",
            "redirect_uris": [redirect_uri],
            "grant_types": ["authorization_code", "refresh_token"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none"
        }))
        .send()
        .await
        .map_err(|_| {
            AppError::InvalidInput("Could not register this app with the provider.".to_owned())
        })?;
    if !response.status().is_success() {
        return Err(AppError::InvalidInput(
            "The provider rejected client registration.".to_owned(),
        ));
    }
    let registration: ClientRegistration = response.json().await.map_err(|_| {
        AppError::InvalidInput(
            "The provider returned an unreadable client registration.".to_owned(),
        )
    })?;
    if registration.client_id.is_empty() {
        return Err(AppError::InvalidInput(
            "The provider did not return a client id.".to_owned(),
        ));
    }
    Ok(RegisteredClient {
        client_id: registration.client_id,
        client_secret: registration.client_secret,
    })
}

async fn exchange_code(
    http: &reqwest::Client,
    pending: &PendingOauth,
    code: &str,
) -> Result<TokenResponse, AppError> {
    let mut form = vec![
        ("grant_type".to_owned(), "authorization_code".to_owned()),
        ("code".to_owned(), code.to_owned()),
        ("redirect_uri".to_owned(), pending.redirect_uri.clone()),
        ("client_id".to_owned(), pending.client_id.clone()),
        ("code_verifier".to_owned(), pending.code_verifier.clone()),
        ("resource".to_owned(), pending.resource.clone()),
    ];
    if !pending.client_secret.is_empty() {
        form.push(("client_secret".to_owned(), pending.client_secret.clone()));
    }
    let body = form
        .into_iter()
        .map(|(key, value)| format!("{}={}", form_encode(&key), form_encode(&value)))
        .collect::<Vec<_>>()
        .join("&");
    let response = http
        .post(&pending.token_endpoint)
        .header("content-type", "application/x-www-form-urlencoded")
        .body(body)
        .send()
        .await
        .map_err(|_| AppError::InvalidInput("Could not exchange the sign-in code.".to_owned()))?;
    if !response.status().is_success() {
        return Err(AppError::InvalidInput(
            "The provider rejected the sign-in code.".to_owned(),
        ));
    }
    response.json().await.map_err(|_| {
        AppError::InvalidInput("The provider returned an unreadable token response.".to_owned())
    })
}

async fn get_json<T: for<'de> Deserialize<'de>>(
    http: &reqwest::Client,
    url: &str,
) -> Result<T, AppError> {
    let response = http.get(url).send().await.map_err(|_| {
        AppError::InvalidInput("Could not read the provider's OAuth metadata.".to_owned())
    })?;
    if !response.status().is_success() {
        return Err(AppError::InvalidInput(
            "The provider's OAuth metadata was unavailable.".to_owned(),
        ));
    }
    response.json().await.map_err(|_| {
        AppError::InvalidInput("The provider's OAuth metadata was unreadable.".to_owned())
    })
}

fn fallback_metadata_url(server_url: &str) -> Option<String> {
    let (origin, path) = split_origin_path(server_url)?;
    if path.is_empty() {
        Some(format!("{origin}/.well-known/oauth-protected-resource"))
    } else {
        Some(format!(
            "{origin}/.well-known/oauth-protected-resource{path}"
        ))
    }
}

pub fn authorization_server_metadata_url(issuer: &str) -> String {
    format!(
        "{}/.well-known/oauth-authorization-server",
        trim_slash(issuer)
    )
}

pub fn split_origin_path(url: &str) -> Option<(String, String)> {
    let rest = url
        .strip_prefix("https://")
        .or_else(|| url.strip_prefix("http://"))?;
    let scheme = if url.starts_with("https://") {
        "https://"
    } else {
        "http://"
    };
    let (host, path) = match rest.find('/') {
        Some(index) => (&rest[..index], &rest[index..]),
        None => (rest, ""),
    };
    if host.is_empty() {
        return None;
    }
    let path = path.split('?').next().unwrap_or(path);
    let path = path.trim_end_matches('/');
    Some((format!("{scheme}{host}"), path.to_owned()))
}

fn quoted_param(header: &str, name: &str) -> Option<String> {
    let key = format!("{name}=\"");
    let start = header.find(&key)? + key.len();
    let rest = header.get(start..)?;
    let end = rest.find('"')?;
    Some(rest[..end].to_owned())
}

pub fn form_encode(value: &str) -> String {
    let mut encoded = String::new();
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                encoded.push(byte as char);
            }
            _ => encoded.push_str(&format!("%{byte:02X}")),
        }
    }
    encoded
}

fn trim_slash(value: &str) -> String {
    value.trim().trim_end_matches('/').to_owned()
}
