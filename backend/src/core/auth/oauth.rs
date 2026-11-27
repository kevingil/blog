use chrono::{Duration, Utc};
use sha2::{Digest, Sha256};

use crate::{error::AppError, setup::SUPERUSER_ROLE};

use super::{
    Account, AccountId, AuthService, McpAccessToken, NewAuthorizationCode, NewMcpAccessToken,
    NewOauthClient, OauthClient, OauthTokenRotation, mcp_token_hash, service::random_bearer,
};

const ACCESS_TOKEN_TTL_SECONDS: i64 = 60 * 60;
const REFRESH_TOKEN_TTL_SECONDS: i64 = 30 * 24 * 60 * 60;
const CODE_TTL_SECONDS: i64 = 5 * 60;
const REFRESH_TOKEN_PREFIX: &str = "mcr_";
const MAX_REDIRECT_URIS: usize = 8;
const MAX_CLIENT_NAME_CHARS: usize = 200;
const TOKEN_NAME_CHARS: usize = 100;
const S256_CHALLENGE_LEN: usize = 43;
const MAX_STATE_LEN: usize = 1024;
const MAX_SCOPE_LEN: usize = 512;

#[derive(Debug)]
pub enum OauthError {
    InvalidRequest(&'static str),
    InvalidClient,
    InvalidGrant,
    InvalidRedirect(&'static str),
    InvalidClientMetadata(&'static str),
    UnsupportedGrantType,
    AccessDenied,
    Internal(AppError),
}

impl From<AppError> for OauthError {
    fn from(error: AppError) -> Self {
        Self::Internal(error)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OauthClientRegistration {
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub grant_types: Vec<String>,
    pub response_types: Vec<String>,
    pub token_endpoint_auth_method: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RegisteredOauthClient {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub grant_types: Vec<String>,
    pub response_types: Vec<String>,
    pub token_endpoint_auth_method: &'static str,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AuthorizationRequest {
    pub response_type: String,
    pub client_id: String,
    pub redirect_uri: String,
    pub code_challenge: String,
    pub code_challenge_method: String,
    pub state: Option<String>,
    pub scope: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AuthorizationPreview {
    Consent { client_name: String },
    Redirect(String),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IssuedOauthToken {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_in: u64,
    pub record: McpAccessToken,
}

pub fn trim_trailing_slash(value: &str) -> String {
    let mut trimmed = value.to_owned();
    while trimmed.ends_with('/') {
        trimmed.pop();
    }
    trimmed
}

pub fn resource_url(public_base_url: &str) -> String {
    format!("{}/mcp", trim_trailing_slash(public_base_url))
}

pub fn resource_metadata_header(public_base_url: &str) -> String {
    format!(
        "Bearer resource_metadata=\"{}/.well-known/oauth-protected-resource\"",
        trim_trailing_slash(public_base_url)
    )
}

/// `true` when `uri` is an absolute https URL, or http on localhost / 127.0.0.1 / ::1.
pub fn redirect_uri_allowed(uri: &str) -> bool {
    if uri.is_empty()
        || uri.len() > 2048
        || uri.contains('#')
        || uri
            .bytes()
            .any(|byte| byte < 0x20 || byte == 0x7f || byte == b'\\')
    {
        return false;
    }
    let Some((scheme, rest)) = uri.split_once("://") else {
        return false;
    };
    if rest.is_empty() || rest.starts_with('/') {
        return false;
    }
    let without_query = rest.split_once('?').map_or(rest, |(path, _)| path);
    let (authority, _) = without_query.split_once('/').unwrap_or((without_query, ""));
    if authority.is_empty() || authority.contains('@') {
        return false;
    }
    let Some(host) = authority_host(authority) else {
        return false;
    };
    if host.is_empty() {
        return false;
    }
    match scheme.to_ascii_lowercase().as_str() {
        "https" => true,
        "http" => matches!(
            host.to_ascii_lowercase().as_str(),
            "localhost" | "127.0.0.1" | "::1"
        ),
        _ => false,
    }
}

fn authority_host(authority: &str) -> Option<&str> {
    if let Some(rest) = authority.strip_prefix('[') {
        let (host, after) = rest.split_once(']')?;
        if let Some(port) = after.strip_prefix(':') {
            if !port_allowed(port) {
                return None;
            }
        } else if !after.is_empty() {
            return None;
        }
        return Some(host);
    }
    match authority.split_once(':') {
        Some((host, port)) if port_allowed(port) => Some(host),
        Some(_) => None,
        None => Some(authority),
    }
}

fn port_allowed(port: &str) -> bool {
    !port.is_empty() && port.bytes().all(|byte| byte.is_ascii_digit())
}

/// BASE64URL(SHA256(verifier)) for a PKCE verifier. `None` when the verifier is not 43–128 unreserved characters.
pub fn pkce_s256(verifier: &str) -> Option<String> {
    if !(43..=128).contains(&verifier.len()) {
        return None;
    }
    if !verifier.bytes().all(
        |byte| matches!(byte, b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'.' | b'_' | b'~'),
    ) {
        return None;
    }
    use base64::Engine;
    Some(
        base64::engine::general_purpose::URL_SAFE_NO_PAD
            .encode(Sha256::digest(verifier.as_bytes())),
    )
}

fn constant_time_eq(left: &str, right: &str) -> bool {
    let left = left.as_bytes();
    let right = right.as_bytes();
    if left.len() != right.len() {
        return false;
    }
    let mut diff = 0u8;
    for (left, right) in left.iter().zip(right) {
        diff |= left ^ right;
    }
    diff == 0
}

fn encode_component(value: &str) -> String {
    let mut encoded = String::with_capacity(value.len());
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'.' | b'_' | b'~' => {
                encoded.push(char::from(byte));
            }
            _ => encoded.push_str(&format!("%{byte:02X}")),
        }
    }
    encoded
}

fn append_query(uri: &str, pairs: &[(&str, &str)]) -> String {
    let mut out = String::from(uri);
    let mut first = !uri.contains('?');
    for (key, value) in pairs {
        out.push(if first { '?' } else { '&' });
        first = false;
        out.push_str(&encode_component(key));
        out.push('=');
        out.push_str(&encode_component(value));
    }
    out
}

pub fn authorization_error_redirect(
    redirect_uri: &str,
    error: &str,
    description: &str,
    state: Option<&str>,
) -> String {
    error_redirect(redirect_uri, error, description, state)
}

fn error_redirect(
    redirect_uri: &str,
    error: &str,
    description: &str,
    state: Option<&str>,
) -> String {
    let mut pairs = vec![("error", error), ("error_description", description)];
    if let Some(state) = state.filter(|state| !state.is_empty()) {
        pairs.push(("state", state));
    }
    append_query(redirect_uri, &pairs)
}

fn success_redirect(redirect_uri: &str, code: &str, state: Option<&str>) -> String {
    let mut pairs = vec![("code", code)];
    if let Some(state) = state.filter(|state| !state.is_empty()) {
        pairs.push(("state", state));
    }
    append_query(redirect_uri, &pairs)
}

fn authorization_protocol_error(request: &AuthorizationRequest) -> Option<&'static str> {
    if request.response_type != "code" {
        return Some("response_type must be code");
    }
    if request.code_challenge_method != "S256" {
        return Some("code_challenge_method must be S256");
    }
    if request.code_challenge.len() != S256_CHALLENGE_LEN
        || !request
            .code_challenge
            .bytes()
            .all(|byte| matches!(byte, b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_'))
    {
        return Some("code_challenge must be the S256 challenge");
    }
    if request
        .state
        .as_ref()
        .is_some_and(|state| state.len() > MAX_STATE_LEN)
    {
        return Some("state is too long");
    }
    if request
        .scope
        .as_ref()
        .is_some_and(|scope| scope.len() > MAX_SCOPE_LEN)
    {
        return Some("scope is too long");
    }
    None
}

fn deduped_redirects(uris: &[String]) -> Vec<String> {
    let mut unique = Vec::new();
    for uri in uris {
        if !unique.iter().any(|existing: &String| existing == uri) {
            unique.push(uri.clone());
        }
    }
    unique
}

impl AuthService {
    pub async fn register_oauth_client(
        &self,
        input: OauthClientRegistration,
    ) -> Result<RegisteredOauthClient, OauthError> {
        let client_name = input.client_name.trim();
        if client_name.is_empty() || client_name.chars().count() > MAX_CLIENT_NAME_CHARS {
            return Err(OauthError::InvalidClientMetadata(
                "client_name must be 1 to 200 characters",
            ));
        }
        if input.redirect_uris.is_empty() || input.redirect_uris.len() > MAX_REDIRECT_URIS {
            return Err(OauthError::InvalidRedirect(
                "redirect_uris must contain 1 to 8 URIs",
            ));
        }
        if input
            .redirect_uris
            .iter()
            .any(|uri| !redirect_uri_allowed(uri))
        {
            return Err(OauthError::InvalidRedirect(
                "redirect URI must be https, or http on localhost, 127.0.0.1, or [::1]",
            ));
        }
        let grant_types = if input.grant_types.is_empty() {
            vec!["authorization_code".to_owned(), "refresh_token".to_owned()]
        } else {
            input.grant_types
        };
        if !grant_types
            .iter()
            .any(|grant| grant == "authorization_code")
            || grant_types
                .iter()
                .any(|grant| grant != "authorization_code" && grant != "refresh_token")
        {
            return Err(OauthError::InvalidClientMetadata(
                "grant_types must be authorization_code and refresh_token",
            ));
        }
        let response_types = if input.response_types.is_empty() {
            vec!["code".to_owned()]
        } else {
            input.response_types
        };
        if response_types != ["code"] {
            return Err(OauthError::InvalidClientMetadata(
                "response_types must be code",
            ));
        }
        if input
            .token_endpoint_auth_method
            .as_deref()
            .is_some_and(|method| method != "none")
        {
            return Err(OauthError::InvalidClientMetadata(
                "token_endpoint_auth_method must be none",
            ));
        }

        let client_id = uuid::Uuid::new_v4().to_string();
        let redirect_uris = deduped_redirects(&input.redirect_uris);
        self.accounts
            .insert_oauth_client(&NewOauthClient {
                client_id: client_id.clone(),
                client_name: client_name.to_owned(),
                redirect_uris: redirect_uris.clone(),
                created_at: Utc::now(),
            })
            .await?;
        Ok(RegisteredOauthClient {
            client_id,
            client_name: client_name.to_owned(),
            redirect_uris,
            grant_types: vec!["authorization_code".to_owned(), "refresh_token".to_owned()],
            response_types,
            token_endpoint_auth_method: "none",
        })
    }

    pub async fn preview_authorization(
        &self,
        request: &AuthorizationRequest,
    ) -> Result<AuthorizationPreview, OauthError> {
        let client = self.registered_redirect(request).await?;
        if let Some(reason) = authorization_protocol_error(request) {
            return Ok(AuthorizationPreview::Redirect(error_redirect(
                &request.redirect_uri,
                "invalid_request",
                reason,
                request.state.as_deref(),
            )));
        }
        Ok(AuthorizationPreview::Consent {
            client_name: client.client_name,
        })
    }

    pub async fn grant_authorization_code(
        &self,
        account: &Account,
        request: &AuthorizationRequest,
    ) -> Result<String, OauthError> {
        match self.preview_authorization(request).await? {
            AuthorizationPreview::Redirect(location) => Ok(location),
            AuthorizationPreview::Consent { .. } => {
                if account.role != SUPERUSER_ROLE {
                    return Err(OauthError::AccessDenied);
                }
                let plaintext = random_bearer("");
                let expires_at = Utc::now() + Duration::seconds(CODE_TTL_SECONDS);
                self.accounts
                    .insert_authorization_code(&NewAuthorizationCode {
                        code_hash: mcp_token_hash(&plaintext),
                        client_id: request.client_id.clone(),
                        account_id: account.id,
                        redirect_uri: request.redirect_uri.clone(),
                        code_challenge: request.code_challenge.clone(),
                        expires_at,
                    })
                    .await?;
                Ok(success_redirect(
                    &request.redirect_uri,
                    &plaintext,
                    request.state.as_deref(),
                ))
            }
        }
    }

    pub async fn exchange_authorization_code(
        &self,
        client_id: &str,
        code: &str,
        redirect_uri: &str,
        code_verifier: &str,
    ) -> Result<IssuedOauthToken, OauthError> {
        if client_id.is_empty()
            || code.is_empty()
            || redirect_uri.is_empty()
            || code_verifier.is_empty()
        {
            return Err(OauthError::InvalidRequest(
                "client_id, code, redirect_uri, and code_verifier are required",
            ));
        }
        let client = self.require_client(client_id).await?;
        let Some(pending) = self
            .accounts
            .consume_authorization_code(&mcp_token_hash(code))
            .await?
        else {
            return Err(OauthError::InvalidGrant);
        };
        if pending.client_id != client_id || pending.redirect_uri != redirect_uri {
            return Err(OauthError::InvalidGrant);
        }
        let Some(expected) = pkce_s256(code_verifier) else {
            return Err(OauthError::InvalidGrant);
        };
        if !constant_time_eq(&expected, &pending.code_challenge) {
            return Err(OauthError::InvalidGrant);
        }
        self.issue_oauth_token(&client, pending.account_id).await
    }

    pub async fn refresh_oauth_token(
        &self,
        client_id: &str,
        refresh_token: &str,
    ) -> Result<IssuedOauthToken, OauthError> {
        if client_id.is_empty() || refresh_token.is_empty() {
            return Err(OauthError::InvalidRequest(
                "client_id and refresh_token are required",
            ));
        }
        let client = self.require_client(client_id).await?;
        let presented = mcp_token_hash(refresh_token);
        let (access_token, refresh_token, rotation) = new_token_pair();
        let Some(record) = self
            .accounts
            .rotate_oauth_refresh_token(&presented, client_id, &rotation)
            .await?
        else {
            return Err(OauthError::InvalidGrant);
        };
        Ok(IssuedOauthToken {
            access_token,
            refresh_token,
            expires_in: u64::try_from(ACCESS_TOKEN_TTL_SECONDS).map_err(AppError::internal)?,
            record: McpAccessToken {
                client_name: Some(client.client_name),
                ..record
            },
        })
    }

    async fn registered_redirect(
        &self,
        request: &AuthorizationRequest,
    ) -> Result<OauthClient, OauthError> {
        if request.client_id.is_empty() || request.redirect_uri.is_empty() {
            return Err(OauthError::InvalidRequest(
                "client_id and redirect_uri are required",
            ));
        }
        let Some(client) = self.accounts.find_oauth_client(&request.client_id).await? else {
            return Err(OauthError::InvalidClient);
        };
        if !client
            .redirect_uris
            .iter()
            .any(|uri| uri == &request.redirect_uri)
        {
            return Err(OauthError::InvalidClient);
        }
        Ok(client)
    }

    async fn require_client(&self, client_id: &str) -> Result<OauthClient, OauthError> {
        self.accounts
            .find_oauth_client(client_id)
            .await?
            .ok_or(OauthError::InvalidClient)
    }

    async fn issue_oauth_token(
        &self,
        client: &OauthClient,
        account_id: AccountId,
    ) -> Result<IssuedOauthToken, OauthError> {
        let (access_token, refresh_token, rotation) = new_token_pair();
        let created_at = Utc::now();
        let name: String = client.client_name.chars().take(TOKEN_NAME_CHARS).collect();
        let record = McpAccessToken {
            id: uuid::Uuid::new_v4(),
            account_id,
            name,
            token_prefix: rotation.token_prefix.clone(),
            created_at,
            last_used_at: None,
            revoked_at: None,
            client_name: Some(client.client_name.clone()),
            expires_at: Some(rotation.expires_at),
        };
        self.accounts
            .insert_mcp_token(&NewMcpAccessToken {
                id: record.id,
                account_id,
                name: record.name.clone(),
                token_hash: rotation.token_hash,
                token_prefix: rotation.token_prefix,
                created_at,
                oauth_client_id: Some(client.client_id.clone()),
                client_name: record.client_name.clone(),
                expires_at: record.expires_at,
                refresh_token_hash: Some(rotation.refresh_token_hash),
                refresh_expires_at: Some(rotation.refresh_expires_at),
            })
            .await?;
        Ok(IssuedOauthToken {
            access_token,
            refresh_token,
            expires_in: u64::try_from(ACCESS_TOKEN_TTL_SECONDS).map_err(AppError::internal)?,
            record,
        })
    }
}

fn new_token_pair() -> (String, String, OauthTokenRotation) {
    let access_token = random_bearer(super::service::MCP_ACCESS_TOKEN_PREFIX);
    let refresh_token = random_bearer(REFRESH_TOKEN_PREFIX);
    let now = Utc::now();
    let rotation = OauthTokenRotation {
        token_hash: mcp_token_hash(&access_token),
        token_prefix: access_token
            .chars()
            .take(super::service::MCP_ACCESS_TOKEN_PREFIX_CHARS)
            .collect(),
        expires_at: now + Duration::seconds(ACCESS_TOKEN_TTL_SECONDS),
        refresh_token_hash: mcp_token_hash(&refresh_token),
        refresh_expires_at: now + Duration::seconds(REFRESH_TOKEN_TTL_SECONDS),
    };
    (access_token, refresh_token, rotation)
}

#[cfg(test)]
mod tests {
    use super::{pkce_s256, redirect_uri_allowed};

    #[test]
    fn redirect_uris_allow_https_and_loopback_only() {
        assert!(redirect_uri_allowed(
            "https://claude.ai/api/mcp/auth_callback"
        ));
        assert!(redirect_uri_allowed("http://localhost:3000/callback"));
        assert!(redirect_uri_allowed("http://127.0.0.1:9/cb"));
        assert!(redirect_uri_allowed("http://[::1]:8080/cb"));
        assert!(!redirect_uri_allowed("http://example.com/cb"));
        assert!(!redirect_uri_allowed("http://127.0.0.1.evil.com/cb"));
        assert!(!redirect_uri_allowed("javascript:alert(1)"));
        assert!(!redirect_uri_allowed("https://example.com/cb#frag"));
        assert!(!redirect_uri_allowed("https://user:pass@example.com/cb"));
    }

    #[test]
    fn pkce_s256_matches_the_rfc_7636_vector() {
        let challenge = pkce_s256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk");
        assert_eq!(
            challenge.as_deref(),
            Some("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")
        );
        assert!(pkce_s256("short").is_none());
        assert!(pkce_s256(&"a".repeat(43).replace('a', " ")).is_none());
    }
}
