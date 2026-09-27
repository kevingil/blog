use blog_backend::core::mcp::{
    McpOauth, PRESETS, authorization_server_metadata_url, authorize_url, code_challenge,
    preset_by_id, resource_metadata_url, split_origin_path, validate_mcp_server,
};

#[test]
fn presets_use_the_official_oauth_servers() {
    let notion = preset_by_id("notion").expect("notion");
    assert_eq!(notion.official_url, "https://mcp.notion.com/mcp");
    assert_eq!(preset_by_id("granola").expect("granola").name, "Granola");
    assert_eq!(
        preset_by_id("fireflies").expect("fireflies").official_url,
        "https://api.fireflies.ai/mcp"
    );
    assert_eq!(PRESETS.len(), 3);
    assert!(preset_by_id("unknown").is_err());
}

#[test]
fn custom_server_requires_a_name_and_http_url() {
    let (name, url) = validate_mcp_server(" Team wiki ", "https://example.com/mcp").expect("ok");
    assert_eq!(name, "Team wiki");
    assert_eq!(url, "https://example.com/mcp");
    assert!(validate_mcp_server("  ", "https://example.com/mcp").is_err());
    assert!(validate_mcp_server("Docs", "stdio://local").is_err());
}

#[test]
fn pkce_s256_matches_the_rfc_vector() {
    let challenge = code_challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk");
    assert_eq!(challenge, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM");
}

#[test]
fn discovery_reads_the_resource_metadata_challenge() {
    let header = concat!(
        r#"Bearer realm="OAuth", resource_metadata="https://mcp.notion.com/.well-known/oauth-protected-resource/mcp", "#,
        r#"error="invalid_token""#
    );
    assert_eq!(
        resource_metadata_url(header).as_deref(),
        Some("https://mcp.notion.com/.well-known/oauth-protected-resource/mcp")
    );
    assert!(resource_metadata_url(r#"Bearer realm="OAuth""#).is_none());
}

#[test]
fn authorize_url_uses_pkce_and_the_mcp_resource() {
    let url = authorize_url(
        "https://mcp.notion.com/authorize",
        "client",
        "http://localhost:8080/agent/connectors/oauth/callback",
        "challenge",
        "state",
        &["default".to_owned()],
        "https://mcp.notion.com/mcp",
    );
    assert!(url.starts_with("https://mcp.notion.com/authorize?"));
    assert!(url.contains("response_type=code"));
    assert!(url.contains("code_challenge_method=S256"));
    assert!(url.contains("code_challenge=challenge"));
    assert!(url.contains("resource=https%3A%2F%2Fmcp.notion.com%2Fmcp"));
    assert!(url.contains("scope=default"));
    assert!(url.contains(
        "redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fagent%2Fconnectors%2Foauth%2Fcallback"
    ));
}

#[test]
fn metadata_urls_follow_the_issuer_and_resource_path() {
    assert_eq!(
        authorization_server_metadata_url("https://api.fireflies.ai/"),
        "https://api.fireflies.ai/.well-known/oauth-authorization-server"
    );
    assert_eq!(
        split_origin_path("https://mcp.notion.com/mcp").expect("split"),
        ("https://mcp.notion.com".to_owned(), "/mcp".to_owned())
    );
    assert_eq!(
        split_origin_path("https://mcp.granola.ai/mcp?x=1").expect("split"),
        ("https://mcp.granola.ai".to_owned(), "/mcp".to_owned())
    );
}

#[test]
fn redirect_targets_the_public_api_and_app() {
    let oauth = McpOauth::new("http://localhost:8080/", "http://localhost:3000/");
    assert_eq!(
        oauth.redirect_uri(),
        "http://localhost:8080/agent/connectors/oauth/callback"
    );
    assert_eq!(
        oauth.app_connectors_url(),
        "http://localhost:3000/dashboard/connectors"
    );
}
