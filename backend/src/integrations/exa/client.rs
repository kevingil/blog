use std::time::Duration;

use async_trait::async_trait;
use reqwest::{Client, StatusCode};
use secrecy::{ExposeSecret, SecretString};
use serde::{Deserialize, Serialize};

use crate::{
    core::datasource::{
        RecommendationSearchPort, SearchOptions, SearchResponse, SearchResult, SimilarOptions,
    },
    core::ml::llm::{
        AnswerCitation, AnswerResponse, ResearchPort, WebSearchResponse, WebSearchResult,
    },
    error::AppError,
};

const DEFAULT_BASE_URL: &str = "https://api.exa.ai";
const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);

#[derive(Clone)]
pub struct ExaClient {
    client: Client,
    api_key: SecretString,
    base_url: String,
}

impl ExaClient {
    pub fn new(api_key: impl Into<String>) -> Result<Self, AppError> {
        Self::with_base_url(api_key, DEFAULT_BASE_URL)
    }

    pub fn with_base_url(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
    ) -> Result<Self, AppError> {
        let base_url = base_url.into().trim_end_matches('/').to_owned();
        if base_url.is_empty() {
            return Err(AppError::InvalidInput(
                "Exa base URL must not be empty".to_owned(),
            ));
        }
        let client = Client::builder()
            .timeout(REQUEST_TIMEOUT)
            .build()
            .map_err(|_| AppError::Internal)?;
        Ok(Self {
            client,
            api_key: SecretString::from(api_key.into()),
            base_url,
        })
    }

    pub fn is_configured(&self) -> bool {
        !self.api_key.expose_secret().is_empty()
    }

    async fn post<Request, Response>(
        &self,
        path: &str,
        request: &Request,
    ) -> Result<Response, AppError>
    where
        Request: Serialize + Sync,
        Response: for<'de> Deserialize<'de>,
    {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        let response = self
            .client
            .post(format!("{}{}", self.base_url, path))
            .header("x-api-key", self.api_key.expose_secret())
            .json(request)
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if response.status() != StatusCode::OK {
            return Err(AppError::External);
        }
        response.json().await.map_err(|_| AppError::External)
    }
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct ExaSearchRequest<'a> {
    query: &'a str,
    r#type: &'static str,
    num_results: i32,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    include_domains: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    exclude_domains: Vec<String>,
    #[serde(skip_serializing_if = "str::is_empty")]
    start_crawl_date: &'a str,
    #[serde(skip_serializing_if = "str::is_empty")]
    end_crawl_date: &'a str,
    #[serde(skip_serializing_if = "str::is_empty")]
    start_published_date: &'a str,
    #[serde(skip_serializing_if = "str::is_empty")]
    end_published_date: &'a str,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    use_autoprompt: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    text: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    highlights: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    summary: bool,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct ExaSimilarRequest<'a> {
    url: &'a str,
    num_results: i32,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    include_domains: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    exclude_domains: Vec<String>,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    exclude_source_domain: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    text: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    highlights: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    summary: bool,
}

#[derive(Debug, Deserialize)]
struct ExaSearchResponse {
    #[serde(default)]
    results: Vec<ExaSearchResult>,
    #[serde(default, rename = "requestId")]
    request_id: String,
    #[serde(default, rename = "resolvedSearchType")]
    resolved_search_type: String,
    #[serde(default, rename = "costDollars")]
    cost_dollars: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct ExaSearchResult {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    url: String,
    #[serde(default, rename = "publishedDate")]
    published_date: String,
    #[serde(default)]
    author: String,
    #[serde(default)]
    score: f64,
    #[serde(default)]
    text: String,
    #[serde(default)]
    highlights: Vec<String>,
    #[serde(default)]
    summary: String,
    #[serde(default)]
    image: String,
    #[serde(default)]
    favicon: String,
    #[serde(default)]
    extras: serde_json::Map<String, serde_json::Value>,
}

#[derive(Debug, Serialize)]
struct ExaAnswerRequest<'a> {
    query: &'a str,
    text: bool,
}

#[derive(Debug, Deserialize)]
struct ExaAnswerResponse {
    #[serde(default)]
    answer: String,
    #[serde(default)]
    citations: Vec<ExaCitation>,
    #[serde(default, rename = "costDollars")]
    cost_dollars: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct ExaCitation {
    #[serde(default)]
    url: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    author: String,
    #[serde(default, rename = "publishedDate")]
    published_date: String,
    #[serde(default)]
    favicon: String,
    #[serde(default)]
    text: String,
}

impl From<ExaSearchResponse> for SearchResponse {
    fn from(response: ExaSearchResponse) -> Self {
        Self {
            results: response
                .results
                .into_iter()
                .map(|result| SearchResult {
                    id: result.id,
                    title: result.title,
                    url: result.url,
                    published_date: result.published_date,
                    author: result.author,
                    score: result.score,
                    text: result.text,
                    highlights: result.highlights,
                    summary: result.summary,
                    image: result.image,
                    favicon: result.favicon,
                    extras: result.extras,
                })
                .collect(),
        }
    }
}

#[async_trait]
impl RecommendationSearchPort for ExaClient {
    async fn search(
        &self,
        query: &str,
        options: SearchOptions,
    ) -> Result<SearchResponse, AppError> {
        if query.is_empty() {
            return Err(AppError::InvalidInput(
                "search query cannot be empty".to_owned(),
            ));
        }
        let response: ExaSearchResponse = self
            .post(
                "/search",
                &ExaSearchRequest {
                    query,
                    r#type: "auto",
                    num_results: if options.num_results == 0 {
                        10
                    } else {
                        options.num_results
                    },
                    include_domains: options.include_domains,
                    exclude_domains: options.exclude_domains,
                    start_crawl_date: &options.start_date,
                    end_crawl_date: &options.end_date,
                    start_published_date: &options.start_date,
                    end_published_date: &options.end_date,
                    use_autoprompt: options.use_autoprompt,
                    text: options.include_text,
                    highlights: options.include_highlights,
                    summary: options.include_summary,
                },
            )
            .await?;
        Ok(response.into())
    }

    async fn find_similar(
        &self,
        url: &str,
        options: SimilarOptions,
    ) -> Result<SearchResponse, AppError> {
        if url.is_empty() {
            return Err(AppError::InvalidInput("URL cannot be empty".to_owned()));
        }
        let response: ExaSearchResponse = self
            .post(
                "/findSimilar",
                &ExaSimilarRequest {
                    url,
                    num_results: if options.num_results == 0 {
                        10
                    } else {
                        options.num_results
                    },
                    include_domains: options.include_domains,
                    exclude_domains: options.exclude_domains,
                    exclude_source_domain: options.exclude_source_domain,
                    text: options.include_text,
                    highlights: options.include_highlights,
                    summary: options.include_summary,
                },
            )
            .await?;
        Ok(response.into())
    }

    fn is_configured(&self) -> bool {
        ExaClient::is_configured(self)
    }
}

#[async_trait]
impl ResearchPort for ExaClient {
    fn is_configured(&self) -> bool {
        ExaClient::is_configured(self)
    }

    async fn search(&self, query: &str) -> Result<WebSearchResponse, AppError> {
        if query.is_empty() {
            return Err(AppError::InvalidInput(
                "search query cannot be empty".to_owned(),
            ));
        }
        let response: ExaSearchResponse = self
            .post(
                "/search",
                &ExaSearchRequest {
                    query,
                    r#type: "auto",
                    num_results: 10,
                    include_domains: Vec::new(),
                    exclude_domains: Vec::new(),
                    start_crawl_date: "",
                    end_crawl_date: "",
                    start_published_date: "",
                    end_published_date: "",
                    use_autoprompt: true,
                    text: true,
                    highlights: true,
                    summary: true,
                },
            )
            .await?;
        Ok(WebSearchResponse {
            results: response
                .results
                .into_iter()
                .map(|result| WebSearchResult {
                    id: result.id,
                    title: result.title,
                    url: result.url,
                    text: result.text,
                    summary: result.summary,
                    author: result.author,
                    published_date: result.published_date,
                    highlights: result.highlights,
                    score: result.score,
                    favicon: result.favicon,
                })
                .collect(),
            request_id: response.request_id,
            resolved_search_type: response.resolved_search_type,
            cost_dollars: response.cost_dollars,
        })
    }

    async fn answer(&self, question: &str) -> Result<AnswerResponse, AppError> {
        if question.is_empty() {
            return Err(AppError::InvalidInput(
                "question cannot be empty".to_owned(),
            ));
        }
        let response: ExaAnswerResponse = self
            .post(
                "/answer",
                &ExaAnswerRequest {
                    query: question,
                    text: true,
                },
            )
            .await?;
        Ok(AnswerResponse {
            answer: response.answer,
            citations: response
                .citations
                .into_iter()
                .map(|citation| AnswerCitation {
                    url: citation.url,
                    title: citation.title,
                    author: citation.author,
                    published_date: citation.published_date,
                    favicon: citation.favicon,
                    text: citation.text,
                })
                .collect(),
            cost_dollars: response.cost_dollars,
        })
    }
}
