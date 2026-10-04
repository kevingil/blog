use std::time::Duration;

use async_trait::async_trait;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use reqwest::{Client, StatusCode, Url, header::ACCEPT};
use serde::Deserialize;

use crate::{
    core::project::{GithubImportPort, GithubProjectDraft},
    error::AppError,
};

const API_BASE: &str = "https://api.github.com";
const MAX_BODY_BYTES: usize = 2_000_000;
const MAX_TITLE_CHARS: usize = 200;
const MAX_DESCRIPTION_CHARS: usize = 500;
const MAX_TAGS: usize = 10;

pub struct GithubClient {
    http: Client,
    api_base: String,
}

impl GithubClient {
    pub fn new() -> Result<Self, AppError> {
        Self::with_api_base(API_BASE)
    }

    pub fn with_api_base(api_base: &str) -> Result<Self, AppError> {
        let api_base = api_base.trim_end_matches('/').to_owned();
        if api_base.is_empty() {
            return Err(AppError::internal("GitHub API base URL is empty"));
        }
        let http = Client::builder()
            .timeout(Duration::from_secs(20))
            .redirect(reqwest::redirect::Policy::limited(2))
            .user_agent("blog-copilot")
            .build()
            .map_err(AppError::internal)?;
        Ok(Self { http, api_base })
    }

    async fn get_bytes(&self, url: &str) -> Result<(StatusCode, Vec<u8>), AppError> {
        let response = self
            .http
            .get(url)
            .header(ACCEPT, "application/vnd.github+json")
            .header("X-GitHub-Api-Version", "2022-11-28")
            .send()
            .await
            .map_err(AppError::external)?;
        ensure_same_host(&self.api_base, response.url())?;
        let status = response.status();
        let bytes = response.bytes().await.map_err(AppError::external)?;
        if bytes.len() > MAX_BODY_BYTES {
            return Err(AppError::external(
                "GitHub response exceeded the size limit",
            ));
        }
        Ok((status, bytes.to_vec()))
    }
}

#[async_trait]
impl GithubImportPort for GithubClient {
    async fn import_repository(&self, url: &str) -> Result<GithubProjectDraft, AppError> {
        let location = parse_github_url(url)?;
        let repo_url = format!(
            "{}/repos/{}/{}",
            self.api_base, location.owner, location.repo
        );
        let (status, body) = self.get_bytes(&repo_url).await?;
        if status == StatusCode::NOT_FOUND {
            return Err(AppError::NotFound);
        }
        if !status.is_success() {
            return Err(AppError::external(format!("GitHub returned {status}")));
        }
        let repo: GithubRepo = serde_json::from_slice(&body).map_err(AppError::external)?;
        let reference = location
            .reference
            .clone()
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| {
                if repo.default_branch.is_empty() {
                    "main".to_owned()
                } else {
                    repo.default_branch.clone()
                }
            });

        let mut readme_url = Url::parse(&format!(
            "{}/repos/{}/{}/readme",
            self.api_base, location.owner, location.repo
        ))
        .map_err(AppError::internal)?;
        readme_url.query_pairs_mut().append_pair("ref", &reference);
        let readme = match self.get_bytes(readme_url.as_str()).await? {
            (status, _) if status == StatusCode::NOT_FOUND => String::new(),
            (status, _) if !status.is_success() => {
                return Err(AppError::external(format!("GitHub returned {status}")));
            }
            (_, body) => decode_readme(&body)?,
        };

        let title = readme_title(&readme).unwrap_or_else(|| prettify_name(&repo.name));
        let title = bounded(&title, MAX_TITLE_CHARS);
        let title = if title.is_empty() {
            "Project".to_owned()
        } else {
            title
        };
        let description = repo
            .description
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(ToOwned::to_owned)
            .or_else(|| readme_summary(&readme))
            .unwrap_or_else(|| title.clone());
        let description = bounded(&description, MAX_DESCRIPTION_CHARS);
        let description = if description.is_empty() {
            title.clone()
        } else {
            description
        };
        let image_url =
            first_image(&readme, &location.owner, &location.repo, &reference).unwrap_or_default();
        let project_url = if repo.html_url.trim().is_empty() {
            format!("https://github.com/{}/{}", location.owner, location.repo)
        } else {
            repo.html_url
        };

        Ok(GithubProjectDraft {
            title,
            description,
            content: readme,
            tags: project_tags(repo.language.as_deref(), &repo.topics),
            image_url,
            url: project_url,
        })
    }
}

#[derive(Debug, Deserialize)]
struct GithubRepo {
    name: String,
    #[serde(default)]
    description: Option<String>,
    #[serde(default)]
    html_url: String,
    #[serde(default)]
    language: Option<String>,
    #[serde(default)]
    topics: Vec<String>,
    #[serde(default)]
    default_branch: String,
}

#[derive(Debug, Deserialize)]
struct GithubReadme {
    #[serde(default)]
    content: String,
    #[serde(default)]
    encoding: String,
}

#[derive(Debug, Clone)]
struct RepoLocation {
    owner: String,
    repo: String,
    reference: Option<String>,
}

fn parse_github_url(input: &str) -> Result<RepoLocation, AppError> {
    let input = input.trim();
    if input.is_empty() {
        return Err(AppError::InvalidInput(
            "GitHub repository URL is required".to_owned(),
        ));
    }
    if let Some(rest) = input.strip_prefix("git@github.com:") {
        let rest = rest.split('?').next().unwrap_or(rest);
        return location_from_parts(rest, None);
    }

    let with_scheme = if input.contains("://") {
        input.to_owned()
    } else {
        format!("https://{input}")
    };
    let url = Url::parse(&with_scheme).map_err(|_| invalid_github_url())?;
    let host = url.host_str().unwrap_or("").to_ascii_lowercase();
    if host != "github.com" && host != "www.github.com" {
        return Err(invalid_github_url());
    }
    let segments: Vec<&str> = url
        .path_segments()
        .map(|segments| segments.filter(|segment| !segment.is_empty()).collect())
        .unwrap_or_default();
    if segments.len() < 2 {
        return Err(invalid_github_url());
    }
    let reference = if segments.len() >= 4 && segments[2] == "tree" {
        Some(segments[3..].join("/"))
    } else if segments.len() >= 4 && segments[2] == "blob" {
        Some(segments[3].to_owned())
    } else {
        None
    };
    let path = format!(
        "{}/{}",
        segments[0],
        segments[1].strip_suffix(".git").unwrap_or(segments[1])
    );
    location_from_parts(&path, reference)
}

fn location_from_parts(path: &str, reference: Option<String>) -> Result<RepoLocation, AppError> {
    let mut parts = path.split('/').filter(|part| !part.is_empty());
    let owner = parts.next().unwrap_or("");
    let repo = parts.next().unwrap_or("");
    let repo = repo.strip_suffix(".git").unwrap_or(repo);
    if parts.next().is_some() || !valid_name(owner) || !valid_name(repo) {
        return Err(invalid_github_url());
    }
    let reference = match reference {
        Some(value) if valid_reference(&value) => Some(value),
        Some(_) => return Err(invalid_github_url()),
        None => None,
    };
    Ok(RepoLocation {
        owner: owner.to_owned(),
        repo: repo.to_owned(),
        reference,
    })
}

fn valid_name(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 100
        && value != "."
        && value != ".."
        && value.chars().all(|character| {
            character.is_ascii_alphanumeric() || matches!(character, '-' | '_' | '.')
        })
}

fn valid_reference(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 255
        && !value.contains("..")
        && !value.starts_with('/')
        && !value.ends_with('/')
        && value.chars().all(|character| {
            character.is_ascii_alphanumeric() || matches!(character, '-' | '_' | '.' | '/')
        })
}

fn invalid_github_url() -> AppError {
    AppError::InvalidInput(
        "Enter a GitHub repository URL such as https://github.com/owner/repo".to_owned(),
    )
}

fn ensure_same_host(api_base: &str, final_url: &Url) -> Result<(), AppError> {
    let base = Url::parse(api_base).map_err(AppError::internal)?;
    if final_url.host() != base.host() {
        return Err(AppError::external("GitHub redirected to a different host"));
    }
    Ok(())
}

fn decode_readme(body: &[u8]) -> Result<String, AppError> {
    let payload: GithubReadme = serde_json::from_slice(body).map_err(AppError::external)?;
    if payload.content.trim().is_empty() {
        return Ok(String::new());
    }
    if !payload.encoding.is_empty() && !payload.encoding.eq_ignore_ascii_case("base64") {
        return Err(AppError::external("GitHub readme was not base64"));
    }
    let cleaned: String = payload
        .content
        .chars()
        .filter(|character| !character.is_whitespace())
        .collect();
    let bytes = STANDARD.decode(cleaned).map_err(AppError::external)?;
    String::from_utf8(bytes).map_err(AppError::external)
}

fn readme_title(markdown: &str) -> Option<String> {
    let mut in_fence = false;
    for line in markdown.lines() {
        let trimmed = line.trim();
        if is_fence(trimmed) {
            in_fence = !in_fence;
            continue;
        }
        if in_fence {
            continue;
        }
        let Some(rest) = trimmed.strip_prefix('#') else {
            continue;
        };
        if rest.starts_with('#') {
            continue;
        }
        let title = rest.trim().trim_end_matches('#').trim();
        if !title.is_empty() {
            return Some(title.to_owned());
        }
    }
    None
}

fn readme_summary(markdown: &str) -> Option<String> {
    let mut in_fence = false;
    let mut paragraph = String::new();
    for line in markdown.lines() {
        let trimmed = line.trim();
        if is_fence(trimmed) {
            in_fence = !in_fence;
            if !paragraph.is_empty() {
                break;
            }
            continue;
        }
        if in_fence {
            continue;
        }
        if trimmed.is_empty() {
            if !paragraph.is_empty() {
                break;
            }
            continue;
        }
        if trimmed.starts_with('#') || trimmed.starts_with("![") {
            continue;
        }
        if is_link_only(trimmed) {
            continue;
        }
        if !paragraph.is_empty() {
            paragraph.push(' ');
        }
        paragraph.push_str(trimmed);
    }
    let paragraph = paragraph.trim();
    if paragraph.is_empty() {
        None
    } else {
        Some(paragraph.to_owned())
    }
}

fn is_fence(trimmed: &str) -> bool {
    trimmed.starts_with("```") || trimmed.starts_with("~~~")
}

fn is_link_only(trimmed: &str) -> bool {
    trimmed.starts_with('[')
        && trimmed.ends_with(')')
        && trimmed.contains("](")
        && !trimmed.contains(' ')
}

fn prettify_name(name: &str) -> String {
    let mut words = Vec::new();
    for word in name.split(['-', '_', '.']).filter(|word| !word.is_empty()) {
        let mut characters = word.chars();
        let Some(first) = characters.next() else {
            continue;
        };
        let mut pretty = first.to_uppercase().collect::<String>();
        pretty.push_str(characters.as_str());
        words.push(pretty);
    }
    words.join(" ")
}

fn bounded(value: &str, max_chars: usize) -> String {
    let trimmed = value.trim();
    if trimmed.chars().count() <= max_chars {
        trimmed.to_owned()
    } else {
        trimmed.chars().take(max_chars).collect()
    }
}

fn project_tags(language: Option<&str>, topics: &[String]) -> Vec<String> {
    let mut tags = Vec::new();
    for topic in topics {
        push_tag(&mut tags, topic);
    }
    if let Some(language) = language {
        push_tag(&mut tags, language);
    }
    tags
}

fn push_tag(tags: &mut Vec<String>, value: &str) {
    if tags.len() >= MAX_TAGS {
        return;
    }
    let value = value.trim();
    let length = value.chars().count();
    if !(1..=50).contains(&length) {
        return;
    }
    if tags
        .iter()
        .any(|existing| existing.eq_ignore_ascii_case(value))
    {
        return;
    }
    tags.push(value.to_owned());
}

fn first_image(markdown: &str, owner: &str, repo: &str, branch: &str) -> Option<String> {
    let mut in_fence = false;
    for line in markdown.lines() {
        let trimmed = line.trim();
        if is_fence(trimmed) {
            in_fence = !in_fence;
            continue;
        }
        if in_fence {
            continue;
        }
        if let Some(image) = first_image_in_line(line, owner, repo, branch) {
            return Some(image);
        }
    }
    None
}

fn first_image_in_line(line: &str, owner: &str, repo: &str, branch: &str) -> Option<String> {
    let mut rest = line;
    while !rest.is_empty() {
        let markdown_at = rest.find("![");
        let html_at = find_ascii(rest, "<img");
        let next = match (markdown_at, html_at) {
            (Some(markdown), Some(html)) => Some(markdown.min(html)),
            (Some(markdown), None) => Some(markdown),
            (None, Some(html)) => Some(html),
            (None, None) => None,
        };
        let index = next?;
        let candidate = rest.get(index..).unwrap_or("");
        if candidate.starts_with("![")
            && let Some((raw, consumed)) = markdown_image_target(candidate)
        {
            if let Some(resolved) = resolve_image(owner, repo, branch, raw) {
                return Some(resolved);
            }
            rest = candidate.get(consumed.max(1)..).unwrap_or("");
            continue;
        }
        if candidate.len() >= 4
            && candidate[..4].eq_ignore_ascii_case("<img")
            && let Some((raw, consumed)) = html_image_target(candidate)
        {
            if let Some(resolved) = resolve_image(owner, repo, branch, raw) {
                return Some(resolved);
            }
            rest = candidate.get(consumed.max(1)..).unwrap_or("");
            continue;
        }
        rest = skip_char(candidate);
    }
    None
}

fn find_ascii(haystack: &str, needle: &str) -> Option<usize> {
    let haystack_bytes = haystack.as_bytes();
    let needle_bytes = needle.as_bytes();
    if needle_bytes.is_empty() || haystack_bytes.len() < needle_bytes.len() {
        return None;
    }
    haystack_bytes
        .windows(needle_bytes.len())
        .position(|window| window.eq_ignore_ascii_case(needle_bytes))
}

fn markdown_image_target(input: &str) -> Option<(&str, usize)> {
    if !input.starts_with("![") {
        return None;
    }
    let close = input.find("](")?;
    let mut url_start = close + 2;
    let bytes = input.as_bytes();
    while url_start < bytes.len() && bytes[url_start].is_ascii_whitespace() {
        url_start += 1;
    }
    let region = input.get(url_start..)?;
    if let Some(body) = region.strip_prefix('<') {
        let end = body.find('>')?;
        let raw = body.get(..end)?;
        return Some((raw, url_start + 1 + end + 1));
    }
    let end_rel = region
        .find([' ', '\t', '\n', ')', '"'])
        .unwrap_or(region.len());
    if end_rel == 0 {
        return None;
    }
    let raw = region.get(..end_rel)?;
    Some((raw, url_start + end_rel))
}

fn html_image_target(input: &str) -> Option<(&str, usize)> {
    if input.len() < 4 || !input[..4].eq_ignore_ascii_case("<img") {
        return None;
    }
    let next = input.chars().nth(4)?;
    if next.is_ascii_alphanumeric() {
        return None;
    }
    let tag_end = input.find('>').unwrap_or(input.len());
    let tag = input.get(..tag_end)?;
    let src_at = find_ascii(tag, "src=")?;
    let mut value_start = src_at + 4;
    let bytes = tag.as_bytes();
    while value_start < bytes.len() && bytes[value_start].is_ascii_whitespace() {
        value_start += 1;
    }
    if value_start >= bytes.len() {
        return None;
    }
    let quote = bytes[value_start];
    if quote == b'"' || quote == b'\'' {
        let start = value_start + 1;
        let end = tag.get(start..)?.find(quote as char)? + start;
        let raw = tag.get(start..end)?;
        return Some((raw, tag_end + 1));
    }
    let start = value_start;
    let end_rel = tag
        .get(start..)?
        .find(|character: char| character.is_whitespace() || character == '>')
        .unwrap_or(tag.len().saturating_sub(start));
    let raw = tag.get(start..start + end_rel)?;
    if raw.is_empty() {
        return None;
    }
    Some((raw, tag_end + 1))
}

fn skip_char(value: &str) -> &str {
    match value.chars().next() {
        Some(character) => value.get(character.len_utf8()..).unwrap_or(""),
        None => "",
    }
}

fn resolve_image(owner: &str, repo: &str, branch: &str, raw: &str) -> Option<String> {
    let raw = raw.trim();
    if raw.is_empty() || raw.starts_with('#') || raw.starts_with("data:") {
        return None;
    }
    let url = if raw.starts_with("http://") || raw.starts_with("https://") {
        Url::parse(raw).ok()?
    } else if let Some(rest) = raw.strip_prefix("//") {
        Url::parse(&format!("https://{rest}")).ok()?
    } else {
        let path = raw.trim_start_matches("./").trim_start_matches('/');
        if path.is_empty() || path.split('/').any(|part| part == "..") {
            return None;
        }
        raw_content_base(owner, repo, branch)?.join(path).ok()?
    };
    let url = normalize_image_url(url)?;
    if is_badge(&url) {
        return None;
    }
    Some(url.to_string())
}

fn raw_content_base(owner: &str, repo: &str, branch: &str) -> Option<Url> {
    let mut url = Url::parse("https://raw.githubusercontent.com/placeholder").ok()?;
    {
        let mut segments = url.path_segments_mut().ok()?;
        segments.clear();
        segments.push(owner);
        segments.push(repo);
        for part in branch.split('/').filter(|part| !part.is_empty()) {
            if part == "." || part == ".." {
                return None;
            }
            segments.push(part);
        }
        segments.push("");
    }
    Some(url)
}

fn normalize_image_url(url: Url) -> Option<Url> {
    if url.scheme() != "http" && url.scheme() != "https" {
        return None;
    }
    let host = url.host_str()?.to_ascii_lowercase();
    if host != "github.com" && host != "www.github.com" {
        return Some(url);
    }
    let segments: Vec<String> = url
        .path_segments()?
        .filter(|segment| !segment.is_empty())
        .map(str::to_owned)
        .collect();
    if segments.len() >= 5 && (segments[2] == "blob" || segments[2] == "raw") {
        let path = segments[4..].join("/");
        return raw_content_base(&segments[0], &segments[1], &segments[3])?
            .join(&path)
            .ok();
    }
    if segments
        .first()
        .is_some_and(|segment| segment == "user-attachments")
    {
        return Some(url);
    }
    None
}

fn is_badge(url: &Url) -> bool {
    let host = url.host_str().unwrap_or("").to_ascii_lowercase();
    const BADGE_HOSTS: &[&str] = &[
        "shields.io",
        "img.shields.io",
        "badgen.net",
        "badge.fury.io",
        "coveralls.io",
        "codecov.io",
        "travis-ci.com",
        "travis-ci.org",
        "circleci.com",
    ];
    if BADGE_HOSTS
        .iter()
        .any(|badge_host| host == *badge_host || host.ends_with(&format!(".{badge_host}")))
    {
        return true;
    }
    url.path().to_ascii_lowercase().contains("badge")
}
