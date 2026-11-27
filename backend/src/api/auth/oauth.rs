use axum::{
    Json,
    extract::{Form, Query, State},
    http::{
        HeaderValue, StatusCode,
        header::{CACHE_CONTROL, LOCATION, RETRY_AFTER},
    },
    response::{Html, IntoResponse, Response},
};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

use crate::core::auth::oauth::{
    AuthorizationPreview, AuthorizationRequest, OauthClientRegistration, OauthError, resource_url,
    trim_trailing_slash,
};

use super::state::AuthState;

#[derive(Debug, Serialize, ToSchema)]
pub struct ProtectedResourceMetadata {
    pub resource: String,
    pub authorization_servers: Vec<String>,
    pub bearer_methods_supported: Vec<String>,
    pub scopes_supported: Vec<String>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AuthorizationServerMetadata {
    pub issuer: String,
    pub authorization_endpoint: String,
    pub token_endpoint: String,
    pub registration_endpoint: String,
    pub response_types_supported: Vec<String>,
    pub grant_types_supported: Vec<String>,
    pub code_challenge_methods_supported: Vec<String>,
    pub token_endpoint_auth_methods_supported: Vec<String>,
    pub scopes_supported: Vec<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct OauthRegisterRequest {
    #[serde(default)]
    client_name: String,
    #[serde(default)]
    redirect_uris: Vec<String>,
    #[serde(default)]
    grant_types: Vec<String>,
    #[serde(default)]
    response_types: Vec<String>,
    token_endpoint_auth_method: Option<String>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct OauthRegisterResponse {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub grant_types: Vec<String>,
    pub response_types: Vec<String>,
    pub token_endpoint_auth_method: String,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct OauthTokenRequest {
    #[serde(default)]
    grant_type: String,
    #[serde(default)]
    code: String,
    #[serde(default)]
    redirect_uri: String,
    #[serde(default)]
    client_id: String,
    #[serde(default)]
    code_verifier: String,
    #[serde(default)]
    refresh_token: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct OauthTokenResponse {
    pub access_token: String,
    pub token_type: &'static str,
    pub expires_in: u64,
    pub refresh_token: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct OauthErrorBody {
    pub error: &'static str,
    pub error_description: String,
}

#[derive(Debug, Deserialize, Default)]
pub struct AuthorizeParams {
    #[serde(default)]
    response_type: String,
    #[serde(default)]
    client_id: String,
    #[serde(default)]
    redirect_uri: String,
    #[serde(default)]
    code_challenge: String,
    #[serde(default)]
    code_challenge_method: String,
    #[serde(default)]
    state: String,
    #[serde(default)]
    scope: String,
    #[serde(default)]
    email: String,
    #[serde(default)]
    password: String,
    #[serde(default)]
    decision: String,
}

fn protected_resource(base: &str) -> ProtectedResourceMetadata {
    let base = trim_trailing_slash(base);
    ProtectedResourceMetadata {
        resource: resource_url(&base),
        authorization_servers: vec![base],
        bearer_methods_supported: vec!["header".to_owned()],
        scopes_supported: vec!["mcp".to_owned()],
    }
}

fn authorization_server(base: &str) -> AuthorizationServerMetadata {
    let base = trim_trailing_slash(base);
    AuthorizationServerMetadata {
        issuer: base.clone(),
        authorization_endpoint: format!("{base}/oauth/authorize"),
        token_endpoint: format!("{base}/oauth/token"),
        registration_endpoint: format!("{base}/oauth/register"),
        response_types_supported: vec!["code".to_owned()],
        grant_types_supported: vec!["authorization_code".to_owned(), "refresh_token".to_owned()],
        code_challenge_methods_supported: vec!["S256".to_owned()],
        token_endpoint_auth_methods_supported: vec!["none".to_owned()],
        scopes_supported: vec!["mcp".to_owned()],
    }
}

#[utoipa::path(
    get,
    path = "/.well-known/oauth-protected-resource",
    operation_id = "oauthProtectedResource",
    tag = "oauth",
    description = "RFC 9728 metadata for the future /mcp resource.",
    responses((status = 200, body = ProtectedResourceMetadata))
)]
pub async fn oauth_protected_resource(
    State(state): State<AuthState>,
) -> Json<ProtectedResourceMetadata> {
    Json(protected_resource(state.public_base_url()))
}

#[utoipa::path(
    get,
    path = "/.well-known/oauth-protected-resource/mcp",
    operation_id = "oauthProtectedResourceMcp",
    tag = "oauth",
    description = "Path-aware RFC 9728 metadata for the resource URL ending in /mcp.",
    responses((status = 200, body = ProtectedResourceMetadata))
)]
pub async fn oauth_protected_resource_mcp(
    State(state): State<AuthState>,
) -> Json<ProtectedResourceMetadata> {
    Json(protected_resource(state.public_base_url()))
}

#[utoipa::path(
    get,
    path = "/.well-known/oauth-authorization-server",
    operation_id = "oauthAuthorizationServer",
    tag = "oauth",
    description = "RFC 8414 metadata for this authorization server.",
    responses((status = 200, body = AuthorizationServerMetadata))
)]
pub async fn oauth_authorization_server(
    State(state): State<AuthState>,
) -> Json<AuthorizationServerMetadata> {
    Json(authorization_server(state.public_base_url()))
}

#[utoipa::path(
    post,
    path = "/oauth/register",
    operation_id = "oauthRegisterClient",
    tag = "oauth",
    description = "RFC 7591 dynamic registration for a public client.",
    request_body = OauthRegisterRequest,
    responses(
        (status = 201, body = OauthRegisterResponse),
        (status = 400, body = OauthErrorBody)
    )
)]
pub async fn oauth_register(
    State(state): State<AuthState>,
    body: Result<Json<OauthRegisterRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Ok(Json(body)) = body else {
        return oauth_failure(
            StatusCode::BAD_REQUEST,
            "invalid_client_metadata",
            "request body must be JSON",
        );
    };
    match state
        .service()
        .register_oauth_client(OauthClientRegistration {
            client_name: body.client_name,
            redirect_uris: body.redirect_uris,
            grant_types: body.grant_types,
            response_types: body.response_types,
            token_endpoint_auth_method: body.token_endpoint_auth_method,
        })
        .await
    {
        Ok(client) => (
            StatusCode::CREATED,
            [(CACHE_CONTROL, HeaderValue::from_static("no-store"))],
            Json(OauthRegisterResponse {
                client_id: client.client_id,
                client_name: client.client_name,
                redirect_uris: client.redirect_uris,
                grant_types: client.grant_types,
                response_types: client.response_types,
                token_endpoint_auth_method: client.token_endpoint_auth_method.to_owned(),
            }),
        )
            .into_response(),
        Err(error) => oauth_error_response(error),
    }
}

#[utoipa::path(
    post,
    path = "/oauth/token",
    operation_id = "oauthToken",
    tag = "oauth",
    description = "Exchanges an authorization code or rotates a refresh token. PKCE S256 is required.",
    request_body(
        content = OauthTokenRequest,
        content_type = "application/x-www-form-urlencoded"
    ),
    responses(
        (status = 200, body = OauthTokenResponse),
        (status = 400, body = OauthErrorBody),
        (status = 401, body = OauthErrorBody)
    )
)]
pub async fn oauth_token(
    State(state): State<AuthState>,
    body: Result<Form<OauthTokenRequest>, axum::extract::rejection::FormRejection>,
) -> Response {
    let Ok(Form(body)) = body else {
        return oauth_failure(
            StatusCode::BAD_REQUEST,
            "invalid_request",
            "token request must be form-encoded",
        );
    };
    let result = match body.grant_type.as_str() {
        "authorization_code" => {
            state
                .service()
                .exchange_authorization_code(
                    &body.client_id,
                    &body.code,
                    &body.redirect_uri,
                    &body.code_verifier,
                )
                .await
        }
        "refresh_token" => {
            state
                .service()
                .refresh_oauth_token(&body.client_id, &body.refresh_token)
                .await
        }
        _ => Err(OauthError::UnsupportedGrantType),
    };
    match result {
        Ok(issued) => (
            StatusCode::OK,
            [(CACHE_CONTROL, HeaderValue::from_static("no-store"))],
            Json(OauthTokenResponse {
                access_token: issued.access_token,
                token_type: "Bearer",
                expires_in: issued.expires_in,
                refresh_token: issued.refresh_token,
            }),
        )
            .into_response(),
        Err(error) => oauth_error_response(error),
    }
}

pub async fn oauth_authorize_form(
    State(state): State<AuthState>,
    Query(params): Query<AuthorizeParams>,
) -> Response {
    render_authorize(&state, &params, None, None).await
}

pub async fn oauth_authorize_submit(
    State(state): State<AuthState>,
    body: Result<Form<AuthorizeParams>, axum::extract::rejection::FormRejection>,
) -> Response {
    let Ok(Form(params)) = body else {
        return Html(page(
            "Sign-in failed",
            "<p>The sign-in form was not valid.</p>",
        ))
        .into_response();
    };
    if params.decision == "deny" {
        return match state
            .service()
            .preview_authorization(&authorization_request(&params))
            .await
        {
            Ok(AuthorizationPreview::Redirect(location)) => redirect_to(&location),
            Ok(AuthorizationPreview::Consent { .. }) => redirect_to(&denied_redirect(&params)),
            Err(error) => authorize_error(error),
        };
    }
    if let Some(wait) = state.login_throttle().retry_after(&params.email) {
        let mut response = render_authorize(
            &state,
            &params,
            Some("Too many sign-in attempts. Try again later."),
            Some(StatusCode::TOO_MANY_REQUESTS),
        )
        .await;
        if let Ok(value) = HeaderValue::from_str(&wait.as_secs().max(1).to_string()) {
            response.headers_mut().insert(RETRY_AFTER, value);
        }
        return response;
    }
    let email = params.email.clone();
    let login = state
        .service()
        .authenticate_credentials(crate::core::auth::LoginInput {
            email: params.email.clone(),
            password: params.password.clone(),
        })
        .await;
    let account = match login {
        Ok(account) => {
            state.login_throttle().record_success(&email);
            account
        }
        Err(crate::error::AppError::Unauthorized) => {
            state.login_throttle().record_failure(&email);
            return render_authorize(
                &state,
                &params,
                Some("Sign-in failed."),
                Some(StatusCode::UNAUTHORIZED),
            )
            .await;
        }
        Err(_) => {
            return authorize_error(OauthError::Internal(crate::error::AppError::internal(
                "sign-in failed",
            )));
        }
    };
    match state
        .service()
        .grant_authorization_code(&account, &authorization_request(&params))
        .await
    {
        Ok(location) => redirect_to(&location),
        Err(OauthError::AccessDenied) => {
            render_authorize(
                &state,
                &params,
                Some("An admin account is required."),
                Some(StatusCode::FORBIDDEN),
            )
            .await
        }
        Err(error) => authorize_error(error),
    }
}

async fn render_authorize(
    state: &AuthState,
    params: &AuthorizeParams,
    message: Option<&str>,
    status: Option<StatusCode>,
) -> Response {
    match state
        .service()
        .preview_authorization(&authorization_request(params))
        .await
    {
        Ok(AuthorizationPreview::Redirect(location)) => redirect_to(&location),
        Ok(AuthorizationPreview::Consent { client_name }) => (
            status.unwrap_or(StatusCode::OK),
            Html(consent_page(&client_name, params, message)),
        )
            .into_response(),
        Err(error) => authorize_error(error),
    }
}

fn authorization_request(params: &AuthorizeParams) -> AuthorizationRequest {
    AuthorizationRequest {
        response_type: params.response_type.clone(),
        client_id: params.client_id.clone(),
        redirect_uri: params.redirect_uri.clone(),
        code_challenge: params.code_challenge.clone(),
        code_challenge_method: params.code_challenge_method.clone(),
        state: Some(params.state.clone()).filter(|state| !state.is_empty()),
        scope: Some(params.scope.clone()).filter(|scope| !scope.is_empty()),
    }
}

fn denied_redirect(params: &AuthorizeParams) -> String {
    let state = if params.state.is_empty() {
        None
    } else {
        Some(params.state.as_str())
    };
    crate::core::auth::oauth::authorization_error_redirect(
        &params.redirect_uri,
        "access_denied",
        "The owner denied the request",
        state,
    )
}

fn authorize_error(error: OauthError) -> Response {
    let (status, message) = match error {
        OauthError::InvalidClient => (
            StatusCode::BAD_REQUEST,
            "This sign-in request is not valid.",
        ),
        OauthError::InvalidRequest(message) | OauthError::InvalidRedirect(message) => {
            (StatusCode::BAD_REQUEST, message)
        }
        OauthError::AccessDenied => (StatusCode::FORBIDDEN, "An admin account is required."),
        OauthError::Internal(_)
        | OauthError::InvalidGrant
        | OauthError::InvalidClientMetadata(_)
        | OauthError::UnsupportedGrantType => (
            StatusCode::INTERNAL_SERVER_ERROR,
            "Sign-in could not be completed.",
        ),
    };
    (
        status,
        Html(page("Sign-in", &format!("<p>{}</p>", escape(message)))),
    )
        .into_response()
}

fn oauth_error_response(error: OauthError) -> Response {
    let (status, code, description) = match error {
        OauthError::InvalidRequest(description) => (
            StatusCode::BAD_REQUEST,
            "invalid_request",
            description.to_owned(),
        ),
        OauthError::InvalidClient => (
            StatusCode::UNAUTHORIZED,
            "invalid_client",
            "client_id is not registered".to_owned(),
        ),
        OauthError::InvalidGrant => (
            StatusCode::BAD_REQUEST,
            "invalid_grant",
            "the code or refresh token was rejected".to_owned(),
        ),
        OauthError::InvalidRedirect(description) => (
            StatusCode::BAD_REQUEST,
            "invalid_redirect_uri",
            description.to_owned(),
        ),
        OauthError::InvalidClientMetadata(description) => (
            StatusCode::BAD_REQUEST,
            "invalid_client_metadata",
            description.to_owned(),
        ),
        OauthError::UnsupportedGrantType => (
            StatusCode::BAD_REQUEST,
            "unsupported_grant_type",
            "grant_type must be authorization_code or refresh_token".to_owned(),
        ),
        OauthError::AccessDenied => (
            StatusCode::FORBIDDEN,
            "access_denied",
            "an admin account is required".to_owned(),
        ),
        OauthError::Internal(_) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            "server_error",
            "the authorization server failed".to_owned(),
        ),
    };
    oauth_failure(status, code, &description)
}

fn oauth_failure(status: StatusCode, error: &'static str, description: &str) -> Response {
    (
        status,
        [(CACHE_CONTROL, HeaderValue::from_static("no-store"))],
        Json(OauthErrorBody {
            error,
            error_description: description.to_owned(),
        }),
    )
        .into_response()
}

fn redirect_to(location: &str) -> Response {
    match HeaderValue::try_from(location) {
        Ok(value) => (StatusCode::FOUND, [(LOCATION, value)]).into_response(),
        Err(_) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Html(page("Sign-in", "<p>Sign-in could not be completed.</p>")),
        )
            .into_response(),
    }
}

fn consent_page(client_name: &str, params: &AuthorizeParams, message: Option<&str>) -> String {
    let notice = message
        .map(|message| format!("<p>{}</p>", escape(message)))
        .unwrap_or_default();
    page(
        "Approve MCP access",
        &format!(
            "{notice}<p><strong>{}</strong> wants to connect.</p>\
             <form method=\"post\" action=\"/oauth/authorize\">\
             {hidden}\
             <label>Email <input name=\"email\" type=\"email\" autocomplete=\"username\"></label>\
             <label>Password <input name=\"password\" type=\"password\" autocomplete=\"current-password\"></label>\
             <button type=\"submit\" name=\"decision\" value=\"approve\">Approve</button>\
             <button type=\"submit\" name=\"decision\" value=\"deny\">Deny</button>\
             </form>",
            escape(client_name),
            notice = notice,
            hidden = hidden_fields(params),
        ),
    )
}

fn hidden_fields(params: &AuthorizeParams) -> String {
    [
        ("response_type", params.response_type.as_str()),
        ("client_id", params.client_id.as_str()),
        ("redirect_uri", params.redirect_uri.as_str()),
        ("code_challenge", params.code_challenge.as_str()),
        (
            "code_challenge_method",
            params.code_challenge_method.as_str(),
        ),
        ("state", params.state.as_str()),
        ("scope", params.scope.as_str()),
    ]
    .into_iter()
    .map(|(name, value)| {
        format!(
            "<input type=\"hidden\" name=\"{}\" value=\"{}\">",
            escape(name),
            escape(value)
        )
    })
    .collect()
}

fn page(title: &str, body: &str) -> String {
    format!(
        "<!doctype html><html><head><meta charset=\"utf-8\"><title>{}</title></head><body><h1>{}</h1>{}</body></html>",
        escape(title),
        escape(title),
        body
    )
}

fn escape(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len());
    for character in value.chars() {
        match character {
            '&' => escaped.push_str("&amp;"),
            '<' => escaped.push_str("&lt;"),
            '>' => escaped.push_str("&gt;"),
            '"' => escaped.push_str("&quot;"),
            '\'' => escaped.push_str("&#39;"),
            _ => escaped.push(character),
        }
    }
    escaped
}
