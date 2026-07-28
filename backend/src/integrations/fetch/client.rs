use std::time::Duration;

use async_trait::async_trait;
use reqwest::{Client, Url, header::CONTENT_TYPE};
use scraper::{Html, Selector};

use crate::{
    core::source::{FetchExtractPort, ScrapedContent},
    error::AppError,
};

const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);
const USER_AGENT: &str = "Mozilla/5.0 (compatible; BlogAgent/1.0)";
const MAX_CONTENT_CHARS: usize = 5000;

#[derive(Clone)]
pub struct HttpFetchExtract {
    client: Client,
}

impl HttpFetchExtract {
    pub fn new() -> Result<Self, AppError> {
        Ok(Self {
            client: Client::builder()
                .timeout(REQUEST_TIMEOUT)
                .user_agent(USER_AGENT)
                .build()
                .map_err(|_| AppError::Internal)?,
        })
    }

    fn normalize_url(target: &str) -> Result<Url, AppError> {
        let target = target.trim();
        if target.is_empty() {
            return Err(AppError::InvalidInput("URL is required".to_owned()));
        }
        Url::parse(target)
            .or_else(|_| Url::parse(&format!("https://{target}")))
            .map_err(|_| AppError::InvalidInput("invalid URL".to_owned()))
    }

    fn extract_pdf(body: &[u8], url: String) -> Result<ScrapedContent, AppError> {
        let content = pdf_extract::extract_text_from_mem(body)
            .map_err(|_| AppError::External)?
            .trim()
            .to_owned();
        if content.is_empty() {
            return Err(AppError::InvalidInput(
                "no text content found in PDF".to_owned(),
            ));
        }
        let title = content
            .lines()
            .map(str::trim)
            .find(|line| !line.is_empty() && line.chars().count() < 200)
            .unwrap_or("PDF Document")
            .to_owned();
        Ok(ScrapedContent {
            title,
            content,
            url,
        })
    }

    fn extract_html(body: &[u8], url: String) -> Result<ScrapedContent, AppError> {
        let html = String::from_utf8_lossy(body);
        let document = Html::parse_document(&html);
        let title_selector = Selector::parse("title").map_err(|_| AppError::Internal)?;
        let heading_selector = Selector::parse("h1").map_err(|_| AppError::Internal)?;
        let title = document
            .select(&title_selector)
            .next()
            .or_else(|| document.select(&heading_selector).next())
            .map(|element| collapse_whitespace(element.text()))
            .unwrap_or_default();

        let content_selectors = [
            "article",
            "main",
            ".content",
            ".post-content",
            ".entry-content",
            ".article-content",
            "#content",
            ".main-content",
            "body",
        ];
        let text_selector =
            Selector::parse("p, h1, h2, h3, h4, h5, h6, li").map_err(|_| AppError::Internal)?;
        let mut content = String::new();
        for selector in content_selectors {
            let selector = Selector::parse(selector).map_err(|_| AppError::Internal)?;
            let Some(root) = document.select(&selector).next() else {
                continue;
            };
            for element in root.select(&text_selector) {
                let text = collapse_whitespace(element.text());
                if text.chars().count() > 10 {
                    if !content.is_empty() {
                        content.push_str("\n\n");
                    }
                    content.push_str(&text);
                }
            }
            break;
        }
        if content.chars().count() > MAX_CONTENT_CHARS {
            content = content.chars().take(MAX_CONTENT_CHARS).collect();
            content.push_str("...");
        }
        Ok(ScrapedContent {
            title,
            content,
            url,
        })
    }
}

#[async_trait]
impl FetchExtractPort for HttpFetchExtract {
    async fn fetch_extract(&self, target: &str) -> Result<ScrapedContent, AppError> {
        let url = Self::normalize_url(target)?;
        let response = self
            .client
            .get(url.clone())
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let is_pdf = response
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .is_some_and(|value| value.contains("application/pdf"));
        let body = response.bytes().await.map_err(|_| AppError::External)?;
        if is_pdf || body.starts_with(b"%PDF") {
            Self::extract_pdf(&body, url.to_string())
        } else {
            Self::extract_html(&body, url.to_string())
        }
    }
}

fn collapse_whitespace<'a>(parts: impl Iterator<Item = &'a str>) -> String {
    parts
        .flat_map(str::split_whitespace)
        .collect::<Vec<_>>()
        .join(" ")
}
