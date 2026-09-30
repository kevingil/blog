use std::{net::IpAddr, time::Duration};

use async_trait::async_trait;
use chrono::{DateTime, NaiveDate, Utc};
use reqwest::{Url, header::LOCATION};
use scraper::{Html, Selector};

use crate::{
    core::article::{ExternalImage, ExternalPage, ExternalPagePort},
    error::AppError,
};

const REQUEST_TIMEOUT: Duration = Duration::from_secs(20);
const USER_AGENT: &str = "Mozilla/5.0 (compatible; BlogAgent/1.0)";
const MAX_DOCUMENT_BYTES: usize = 2_000_000;
const MAX_IMAGE_BYTES: usize = 8_000_000;
const MAX_REDIRECTS: usize = 5;
const EXCERPT_CHARS: usize = 1_200;

#[derive(Clone)]
pub struct HttpExternalPages {
    client: reqwest::Client,
}

impl HttpExternalPages {
    pub fn new() -> Result<Self, AppError> {
        Ok(Self {
            client: reqwest::Client::builder()
                .timeout(REQUEST_TIMEOUT)
                .user_agent(USER_AGENT)
                .redirect(reqwest::redirect::Policy::none())
                .build()
                .map_err(|_| AppError::Internal)?,
        })
    }

    async fn get_limited(
        &self,
        start: &Url,
        max_bytes: usize,
    ) -> Result<(Url, String, Vec<u8>), AppError> {
        let mut url = start.clone();
        for _ in 0..MAX_REDIRECTS {
            ensure_public_target(&url).await?;
            let response =
                self.client.get(url.clone()).send().await.map_err(|_| {
                    AppError::InvalidInput("that link could not be reached".to_owned())
                })?;
            let status = response.status();
            if status.is_redirection() {
                let location = response
                    .headers()
                    .get(LOCATION)
                    .and_then(|value| value.to_str().ok())
                    .ok_or_else(|| {
                        AppError::InvalidInput("that link could not be reached".to_owned())
                    })?;
                url = url.join(location).map_err(|_| {
                    AppError::InvalidInput("that link could not be reached".to_owned())
                })?;
                url.set_fragment(None);
                continue;
            }
            if !status.is_success() {
                return Err(AppError::InvalidInput(
                    "that link could not be reached".to_owned(),
                ));
            }
            let content_type = response
                .headers()
                .get(reqwest::header::CONTENT_TYPE)
                .and_then(|value| value.to_str().ok())
                .unwrap_or_default()
                .to_owned();
            let body = read_limited(response, max_bytes).await?;
            return Ok((url, content_type, body));
        }
        Err(AppError::InvalidInput(
            "that link could not be reached".to_owned(),
        ))
    }
}

#[async_trait]
impl ExternalPagePort for HttpExternalPages {
    async fn load(&self, target: &str) -> Result<ExternalPage, AppError> {
        let start = parse_public_url(target)?;
        let (final_url, content_type, body) = self.get_limited(&start, MAX_DOCUMENT_BYTES).await?;
        if content_type.contains("application/pdf") || body.starts_with(b"%PDF") {
            return Err(AppError::InvalidInput(
                "that link is not an HTML article".to_owned(),
            ));
        }
        let html = String::from_utf8_lossy(&body);
        let mut parsed = parse_html_page(&final_url, &html)?;
        if parsed.excerpt.chars().count() < 80 {
            if let Some(prose) = self.script_excerpt(&final_url, &html, &parsed.slug).await {
                if prose.chars().count() > parsed.excerpt.chars().count() {
                    parsed.excerpt = prose;
                }
            }
        }
        if parsed.excerpt.chars().count() < 40 {
            if let Some(description) = parsed.description.clone() {
                parsed.excerpt = cap_excerpt(&description);
            }
        }
        if parsed.excerpt.chars().count() < 10 {
            let host = final_url.host_str().unwrap_or("the original site");
            parsed.excerpt = format!("Published on {host}.");
        }
        let image = match parsed.image_url {
            Some(image_url) => Some(self.cacheable_image(&image_url).await?),
            None => None,
        };
        Ok(ExternalPage {
            url: parsed.canonical_url,
            title: parsed.title,
            excerpt: parsed.excerpt,
            image,
            published_at: parsed.published_at,
        })
    }
}

impl HttpExternalPages {
    async fn script_excerpt(&self, page_url: &Url, html: &str, slug: &str) -> Option<String> {
        if slug.len() < 3 {
            return None;
        }
        let mut scripts = {
            let document = Html::parse_document(html);
            script_urls(page_url, &document)
        };
        scripts.retain(|url| same_site(page_url, url) && url.path().ends_with(".js"));
        if let Some(direct) = scripts.iter().find(|url| url.path().contains(slug)) {
            return self.prose_from_script(direct).await;
        }
        for entry in scripts.into_iter().take(2) {
            let Ok((_, _, body)) = self.get_limited(&entry, MAX_DOCUMENT_BYTES).await else {
                continue;
            };
            let source = String::from_utf8_lossy(&body);
            let Some(token) = module_path_for_slug(&source, slug) else {
                continue;
            };
            let Some(module_url) = resolve_module(page_url, &entry, &token) else {
                continue;
            };
            if !same_site(page_url, &module_url) {
                continue;
            }
            if let Some(prose) = self.prose_from_script(&module_url).await {
                return Some(prose);
            }
        }
        None
    }

    async fn prose_from_script(&self, url: &Url) -> Option<String> {
        let (_, _, body) = self.get_limited(url, MAX_DOCUMENT_BYTES).await.ok()?;
        let source = String::from_utf8_lossy(&body);
        let prose = prose_excerpt(&source);
        (prose.chars().count() >= 40).then_some(prose)
    }

    async fn cacheable_image(&self, image_url: &str) -> Result<ExternalImage, AppError> {
        let start = parse_public_url(image_url)?;
        let (_, content_type, body) = self.get_limited(&start, MAX_IMAGE_BYTES).await?;
        let sniffed = image_type(&content_type, &body).ok_or_else(|| {
            AppError::InvalidInput("could not cache the article image".to_owned())
        })?;
        Ok(ExternalImage {
            bytes: body,
            content_type: sniffed.0.to_owned(),
            extension: sniffed.1.to_owned(),
        })
    }
}

struct ParsedHtml {
    canonical_url: String,
    title: String,
    excerpt: String,
    description: Option<String>,
    image_url: Option<String>,
    published_at: Option<DateTime<Utc>>,
    slug: String,
}

fn parse_html_page(page_url: &Url, html: &str) -> Result<ParsedHtml, AppError> {
    let document = Html::parse_document(html);
    let site_name = meta_content(&document, "og:site_name");
    let title = strip_site_suffix(
        &meta_content(&document, "og:title")
            .or_else(|| meta_content(&document, "twitter:title"))
            .or_else(|| element_text(&document, "title"))
            .unwrap_or_default(),
        site_name.as_deref(),
    );
    if title.chars().count() < 3 {
        return Err(AppError::InvalidInput(
            "could not read a title from that page".to_owned(),
        ));
    }
    let description = meta_content(&document, "og:description")
        .or_else(|| meta_content(&document, "description"))
        .map(|value| cap_excerpt(&value));
    let canonical_url = meta_content(&document, "og:url")
        .and_then(|value| page_url.join(value.trim()).ok())
        .filter(|url| same_site(page_url, url))
        .map(|url| canonical_string(&url))
        .unwrap_or_else(|| canonical_string(page_url));
    Ok(ParsedHtml {
        canonical_url,
        title,
        excerpt: body_excerpt(&document),
        description,
        image_url: meta_content(&document, "og:image")
            .or_else(|| meta_content(&document, "twitter:image"))
            .and_then(|value| page_url.join(value.trim()).ok())
            .filter(|url| matches!(url.scheme(), "http" | "https"))
            .map(|url| url.to_string()),
        published_at: meta_content(&document, "article:published_time")
            .and_then(|value| parse_published(&value)),
        slug: slug_from_url(page_url),
    })
}

fn script_urls(page_url: &Url, document: &Html) -> Vec<Url> {
    let Ok(selector) = Selector::parse("script[src]") else {
        return Vec::new();
    };
    let mut modules = Vec::new();
    let mut others = Vec::new();
    for element in document.select(&selector) {
        let Some(src) = element.attr("src") else {
            continue;
        };
        let Ok(url) = page_url.join(src) else {
            continue;
        };
        if element
            .attr("type")
            .is_some_and(|value| value.eq_ignore_ascii_case("module"))
        {
            modules.push(url);
        } else {
            others.push(url);
        }
    }
    modules.extend(others);
    modules
}

fn resolve_module(page_url: &Url, entry: &Url, token: &str) -> Option<Url> {
    if token.starts_with("https://") || token.starts_with("http://") {
        return Url::parse(token).ok();
    }
    if token.starts_with('/') {
        return page_url.join(token).ok();
    }
    if token.starts_with("./") || token.starts_with("../") {
        return entry.join(token).ok();
    }
    page_url.join(&format!("/{token}")).ok()
}

fn module_path_for_slug(source: &str, slug: &str) -> Option<String> {
    let mut search_from = 0;
    while let Some(relative) = source[search_from..].find(slug) {
        let absolute = search_from + relative;
        let start = source[..absolute]
            .rfind(['"', '\'', '`', ' ', ','])
            .map(|index| index + 1)
            .unwrap_or(0);
        let end_offset = source[absolute..]
            .find(['"', '\'', '`', ' ', ','])
            .unwrap_or(source.len() - absolute);
        let token = &source[start..absolute + end_offset];
        if token.ends_with(".js") && !token.contains(' ') && token.contains(slug) {
            return Some(token.to_owned());
        }
        search_from = absolute + slug.len();
    }
    None
}

fn prose_excerpt(source: &str) -> String {
    let bytes = source.as_bytes();
    let mut index = 0;
    let mut parts = Vec::new();
    while index < bytes.len() {
        let quote = bytes[index];
        if quote == b'`' || quote == b'"' {
            let start = index + 1;
            let mut cursor = start;
            let mut escaped = false;
            let mut found = None;
            while cursor < bytes.len() {
                let next = bytes[cursor];
                if escaped {
                    escaped = false;
                    cursor += 1;
                    continue;
                }
                if next == b'\\' && quote == b'"' {
                    escaped = true;
                    cursor += 1;
                    continue;
                }
                if next == quote {
                    found = Some(cursor);
                    break;
                }
                cursor += 1;
            }
            if let Some(end) = found {
                if let Ok(raw) = std::str::from_utf8(&bytes[start..end])
                    && let Some(text) = accept_prose(raw)
                {
                    parts.push(text);
                }
                index = end + 1;
                continue;
            }
        }
        index += 1;
    }
    cap_excerpt(&parts.join("\n\n"))
}

fn accept_prose(raw: &str) -> Option<String> {
    let text = collapse_whitespace(std::iter::once(raw));
    let count = text.chars().count();
    if !(40..=2_000).contains(&count) {
        return None;
    }
    if text.contains("${")
        || text.contains("=>")
        || text.contains("function")
        || text.contains("import ")
        || text.contains("://")
        || text.starts_with('/')
        || text.starts_with("./")
    {
        return None;
    }
    let lower = text.to_ascii_lowercase();
    if lower.ends_with(".jpg")
        || lower.ends_with(".jpeg")
        || lower.ends_with(".png")
        || lower.ends_with(".webp")
        || lower.ends_with(".gif")
        || lower.ends_with(".avif")
    {
        return None;
    }
    let sentence = text.contains(". ")
        || text.contains("? ")
        || text.contains("! ")
        || text.ends_with(['.', '?', '!']);
    if !sentence {
        return None;
    }
    let letters = text
        .chars()
        .filter(|character| character.is_alphabetic())
        .count();
    if letters * 100 / count < 55 {
        return None;
    }
    Some(text)
}

fn body_excerpt(document: &Html) -> String {
    let Ok(text_selector) = Selector::parse("p, h1, h2, h3") else {
        return String::new();
    };
    for selector in [
        "article",
        "main",
        ".post-content",
        ".entry-content",
        ".article-content",
        "#content",
    ] {
        let Ok(root_selector) = Selector::parse(selector) else {
            continue;
        };
        let Some(root) = document.select(&root_selector).next() else {
            continue;
        };
        let parts: Vec<String> = root
            .select(&text_selector)
            .map(|element| collapse_whitespace(element.text()))
            .filter(|text| text.chars().count() > 40)
            .collect();
        if !parts.is_empty() {
            return cap_excerpt(&parts.join("\n\n"));
        }
    }
    String::new()
}

fn meta_content(document: &Html, key: &str) -> Option<String> {
    let selector = Selector::parse("meta").ok()?;
    for element in document.select(&selector) {
        let Some(property) = element.attr("property").or_else(|| element.attr("name")) else {
            continue;
        };
        if property.eq_ignore_ascii_case(key) {
            let content = element.attr("content")?.trim();
            if !content.is_empty() {
                return Some(content.to_owned());
            }
        }
    }
    None
}

fn element_text(document: &Html, selector: &str) -> Option<String> {
    let selector = Selector::parse(selector).ok()?;
    document
        .select(&selector)
        .next()
        .map(|element| collapse_whitespace(element.text()))
        .filter(|text| !text.is_empty())
}

fn strip_site_suffix(title: &str, site_name: Option<&str>) -> String {
    let title = collapse_whitespace(std::iter::once(title));
    let Some(site) = site_name
        .map(|value| collapse_whitespace(std::iter::once(value)))
        .filter(|value| !value.is_empty())
    else {
        return title;
    };
    for separator in [" | ", " — ", " – ", " - ", " · "] {
        let suffix = format!("{separator}{site}");
        if let Some(stripped) = title.strip_suffix(&suffix) {
            let stripped = stripped.trim();
            if stripped.chars().count() >= 3 {
                return stripped.to_owned();
            }
        }
    }
    title
}

fn parse_published(value: &str) -> Option<DateTime<Utc>> {
    let value = value.trim();
    if let Ok(parsed) = DateTime::parse_from_rfc3339(value) {
        return Some(parsed.with_timezone(&Utc));
    }
    NaiveDate::parse_from_str(value, "%Y-%m-%d")
        .ok()
        .and_then(|date| date.and_hms_opt(0, 0, 0))
        .map(|naive| DateTime::<Utc>::from_naive_utc_and_offset(naive, Utc))
}

fn slug_from_url(url: &Url) -> String {
    url.path_segments()
        .and_then(|segments| {
            segments
                .rev()
                .find(|segment| !segment.is_empty())
                .map(ToOwned::to_owned)
        })
        .unwrap_or_default()
        .trim_end_matches(".html")
        .trim_end_matches(".htm")
        .to_owned()
}

fn canonical_string(url: &Url) -> String {
    let mut url = url.clone();
    url.set_fragment(None);
    let mut text = url.to_string();
    if text.ends_with('/') && url.path() != "/" {
        text.pop();
    }
    text
}

fn same_site(left: &Url, right: &Url) -> bool {
    left.scheme() == right.scheme() && left.host() == right.host()
}

fn cap_excerpt(text: &str) -> String {
    let trimmed = text.trim();
    if trimmed.chars().count() <= EXCERPT_CHARS {
        return trimmed.to_owned();
    }
    let mut end = trimmed
        .char_indices()
        .nth(EXCERPT_CHARS)
        .map(|(index, _)| index)
        .unwrap_or(trimmed.len());
    if let Some(space) = trimmed[..end].rfind(' ')
        && space > 800
    {
        end = space;
    }
    format!("{}...", trimmed[..end].trim_end())
}

fn collapse_whitespace<'a>(parts: impl Iterator<Item = &'a str>) -> String {
    parts
        .flat_map(str::split_whitespace)
        .collect::<Vec<_>>()
        .join(" ")
}

fn image_type(content_type: &str, body: &[u8]) -> Option<(&'static str, &'static str)> {
    let declared = content_type.split(';').next().unwrap_or("").trim();
    match declared {
        "image/jpeg" => Some(("image/jpeg", "jpg")),
        "image/png" => Some(("image/png", "png")),
        "image/gif" => Some(("image/gif", "gif")),
        "image/webp" => Some(("image/webp", "webp")),
        "image/avif" => Some(("image/avif", "avif")),
        _ => sniff_image(body),
    }
}

fn sniff_image(body: &[u8]) -> Option<(&'static str, &'static str)> {
    if body.starts_with(&[0xFF, 0xD8, 0xFF]) {
        Some(("image/jpeg", "jpg"))
    } else if body.starts_with(&[0x89, b'P', b'N', b'G']) {
        Some(("image/png", "png"))
    } else if body.starts_with(b"GIF8") {
        Some(("image/gif", "gif"))
    } else if body.len() > 12 && &body[0..4] == b"RIFF" && &body[8..12] == b"WEBP" {
        Some(("image/webp", "webp"))
    } else {
        None
    }
}

pub(crate) fn parse_public_url(raw: &str) -> Result<Url, AppError> {
    let url = Url::parse(raw.trim())
        .map_err(|_| AppError::InvalidInput("enter a valid http or https link".to_owned()))?;
    if !matches!(url.scheme(), "http" | "https") || url.host_str().is_none() {
        return Err(AppError::InvalidInput(
            "enter a valid http or https link".to_owned(),
        ));
    }
    if !url.username().is_empty() || url.password().is_some() {
        return Err(AppError::InvalidInput(
            "links with embedded credentials are not allowed".to_owned(),
        ));
    }
    if let Some(host) = url.host_str()
        && blocked_host_name(host)
    {
        return Err(AppError::InvalidInput(
            "that link is not a public web page".to_owned(),
        ));
    }
    if let Some(host) = url.host_str()
        && let Ok(ip) = host.parse::<IpAddr>()
        && is_blocked_ip(ip)
    {
        return Err(AppError::InvalidInput(
            "that link is not a public web page".to_owned(),
        ));
    }
    Ok(url)
}

async fn ensure_public_target(url: &Url) -> Result<(), AppError> {
    let url = parse_public_url(url.as_str())?;
    if url
        .host_str()
        .is_some_and(|host| host.parse::<IpAddr>().is_ok())
    {
        return Ok(());
    }
    let host = url
        .host_str()
        .ok_or_else(|| AppError::InvalidInput("enter a valid http or https link".to_owned()))?;
    let port = url.port_or_known_default().unwrap_or(443);
    let mut found = false;
    let addresses = tokio::net::lookup_host((host, port))
        .await
        .map_err(|_| AppError::InvalidInput("that link could not be reached".to_owned()))?;
    for address in addresses {
        found = true;
        if is_blocked_ip(address.ip()) {
            return Err(AppError::InvalidInput(
                "that link is not a public web page".to_owned(),
            ));
        }
    }
    if !found {
        return Err(AppError::InvalidInput(
            "that link could not be reached".to_owned(),
        ));
    }
    Ok(())
}

fn blocked_host_name(host: &str) -> bool {
    let host = host.trim_end_matches('.').to_ascii_lowercase();
    host == "localhost"
        || host.ends_with(".localhost")
        || host.ends_with(".local")
        || host.ends_with(".internal")
        || host == "metadata.google.internal"
}

fn is_blocked_ip(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(ip) => {
            let [first, second, _, _] = ip.octets();
            ip.is_private()
                || ip.is_loopback()
                || ip.is_link_local()
                || ip.is_broadcast()
                || ip.is_unspecified()
                || ip.is_documentation()
                || ip.is_multicast()
                || first == 0
                || (first == 100 && (64..128).contains(&second))
        }
        IpAddr::V6(ip) => {
            ip.is_loopback()
                || ip.is_unspecified()
                || ip.is_unique_local()
                || ip.is_unicast_link_local()
                || ip.is_multicast()
                || ip
                    .to_ipv4_mapped()
                    .is_some_and(|mapped| is_blocked_ip(IpAddr::V4(mapped)))
        }
    }
}

async fn read_limited(
    mut response: reqwest::Response,
    max_bytes: usize,
) -> Result<Vec<u8>, AppError> {
    if response
        .content_length()
        .is_some_and(|length| length > max_bytes as u64)
    {
        return Err(AppError::InvalidInput(
            "that page is too large to import".to_owned(),
        ));
    }
    let mut body = Vec::new();
    while let Some(chunk) = response.chunk().await.map_err(|_| AppError::External)? {
        if body.len().saturating_add(chunk.len()) > max_bytes {
            return Err(AppError::InvalidInput(
                "that page is too large to import".to_owned(),
            ));
        }
        body.extend_from_slice(&chunk);
    }
    Ok(body)
}

#[cfg(test)]
mod tests {
    use super::*;

    const SHELL: &str = r#"
        <html><head>
          <meta charset="UTF-8" />
          <title>Our Agentic Engineering Org | SellScale</title>
          <meta property="og:site_name" content="SellScale" />
          <meta property="og:title" content="Our Agentic Engineering Org | SellScale" />
          <meta property="og:description" content="SellScale runs outbound end to end: research, messaging, sending and follow-up." />
          <meta property="og:url" content="https://www.sellscale.com/blog-posts/our-agentic-engineering-org" />
          <meta property="og:image" content="https://www.sellscale.com/blog/covers/our-agentic-engineering-org-velocity-tunnel.jpg" />
          <meta property="article:published_time" content="2026-09-16" />
          <script type="module" src="/assets/index-CU1ZSoEL.js"></script>
        </head><body><div id="root"></div></body></html>
    "#;

    const MODULE: &str = r#"
        var r={title:`Our Agentic Engineering Org`,coverImage:{src:`/blog/covers/our-agentic-engineering-org-velocity-tunnel.jpg`,alt:`A front-facing race car surrounded by violet and magenta motion blur with green accents`}};
        function i(){return (0,n.jsx)(r.p,{children:`Every engineering team runs some version of the same cycle. Something breaks or slows down. Somebody notices. Somebody establishes what happened, writes it up, decides how urgent it is, and eventually a fix lands. Repeat.`})}
    "#;

    #[test]
    fn shell_page_keeps_title_image_and_canonical_link() {
        let url = Url::parse("https://www.sellscale.com/blog-posts/our-agentic-engineering-org/")
            .unwrap();
        let parsed = parse_html_page(&url, SHELL).unwrap();
        assert_eq!(parsed.title, "Our Agentic Engineering Org");
        assert_eq!(
            parsed.canonical_url,
            "https://www.sellscale.com/blog-posts/our-agentic-engineering-org"
        );
        assert_eq!(
            parsed.image_url.as_deref(),
            Some(
                "https://www.sellscale.com/blog/covers/our-agentic-engineering-org-velocity-tunnel.jpg"
            )
        );
        assert!(parsed.excerpt.is_empty());
        assert_eq!(parsed.slug, "our-agentic-engineering-org");
        assert_eq!(
            parsed.published_at.unwrap().date_naive().to_string(),
            "2026-09-16"
        );
    }

    #[test]
    fn module_source_supplies_preview_prose() {
        let excerpt = prose_excerpt(MODULE);
        assert!(excerpt.starts_with("Every engineering team runs some version of the same cycle"));
        assert!(!excerpt.contains(".jpg"));
    }

    #[test]
    fn module_path_matches_the_article_slug() {
        let bundle = r#""assets/our-agentic-engineering-org-Bi60eHG6.js""#;
        assert_eq!(
            module_path_for_slug(bundle, "our-agentic-engineering-org").as_deref(),
            Some("assets/our-agentic-engineering-org-Bi60eHG6.js")
        );
    }

    #[test]
    fn html_article_uses_body_paragraphs() {
        let html = r#"
            <html><head><title>Notes from the field</title></head>
            <body><article>
              <p>This paragraph is long enough to stand in as the preview text for the card.</p>
            </article></body>
        </html>
        "#;
        let url = Url::parse("https://example.com/notes").unwrap();
        let parsed = parse_html_page(&url, html).unwrap();
        assert!(parsed.excerpt.contains("preview text for the card"));
    }

    #[test]
    fn private_and_local_links_are_rejected() {
        assert!(parse_public_url("http://127.0.0.1/admin").is_err());
        assert!(parse_public_url("http://169.254.169.254/latest").is_err());
        assert!(parse_public_url("http://localhost/blog").is_err());
        assert!(parse_public_url("http://10.0.0.8/post").is_err());
        assert!(parse_public_url("file:///etc/passwd").is_err());
        assert!(parse_public_url("https://user:pass@example.com/post").is_err());
        assert!(
            parse_public_url("https://www.sellscale.com/blog-posts/our-agentic-engineering-org")
                .is_ok()
        );
    }
}
