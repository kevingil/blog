use blog_backend::core::mcp::{PRESETS, connect_custom, connect_preset, preset_server_url};

#[test]
fn fixture_base_connects_presets_without_a_browser() {
    let notion = connect_preset("notion", "http://external-fixtures:8090/").expect("notion");
    assert_eq!(notion.url, "http://external-fixtures:8090/mcp/notion");
    assert_eq!(
        notion.authorization_header.as_deref(),
        Some("Bearer fixture")
    );
    assert!(!notion.needs_authorization);

    let granola = connect_preset("granola", "http://external-fixtures:8090").expect("granola");
    assert_eq!(granola.name, "Granola");
    assert_eq!(granola.url, "http://external-fixtures:8090/mcp/granola");

    let fireflies =
        connect_preset("fireflies", "http://external-fixtures:8090").expect("fireflies");
    assert_eq!(fireflies.url, "http://external-fixtures:8090/mcp/fireflies");
}

#[test]
fn official_presets_ask_the_provider_to_sign_in() {
    let granola = connect_preset("granola", "").expect("granola");
    assert_eq!(granola.url, "https://mcp.granola.ai/mcp");
    assert!(granola.needs_authorization);
    assert_eq!(granola.authorization_url, "https://mcp.granola.ai/mcp");
    assert!(granola.authorization_header.is_none());

    let notion = PRESETS
        .iter()
        .find(|preset| preset.id == "notion")
        .expect("notion");
    assert_eq!(preset_server_url(notion, ""), "https://mcp.notion.com/mcp");
}

#[test]
fn custom_oauth_server_uses_the_fixture_only_on_that_host() {
    let local = connect_custom(
        "Docs",
        "http://external-fixtures:8090/mcp/docs",
        "http://external-fixtures:8090",
    )
    .expect("local");
    assert!(!local.needs_authorization);
    assert_eq!(
        local.authorization_header.as_deref(),
        Some("Bearer fixture")
    );

    let remote = connect_custom(
        "Linear",
        "https://mcp.linear.app/mcp",
        "http://external-fixtures:8090",
    )
    .expect("remote");
    assert!(remote.needs_authorization);
    assert_eq!(remote.authorization_url, "https://mcp.linear.app/mcp");
    assert_eq!(remote.url, "https://mcp.linear.app/mcp");
}

#[test]
fn custom_oauth_rejects_a_missing_name_or_url() {
    assert!(connect_custom("  ", "https://example.com/mcp", "").is_err());
    assert!(connect_custom("Docs", "stdio://local", "").is_err());
    assert!(connect_preset("unknown", "").is_err());
}
