use std::{collections::BTreeSet, sync::Arc};

use reqwest::Url;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    DataSource, DataSourceDiscoveryRecommendationRequest, DataSourceRecommendationRequest,
    DataSourceRecommendationResponse, DataSourceRecommendationsResponse, DataSourceRepository,
    RecommendationSearchPort, SearchOptions, SearchResult, SimilarOptions,
};

const DEFAULT_LIMIT: i32 = 8;
const MAX_LIMIT: i32 = 12;
const MAX_SEEDS: usize = 5;
const RESULTS_PER_SEED: i32 = 6;

#[derive(Clone)]
pub struct RecommendationService {
    data_sources: Arc<dyn DataSourceRepository>,
    search: Arc<dyn RecommendationSearchPort>,
}

impl RecommendationService {
    pub fn new(
        data_sources: Arc<dyn DataSourceRepository>,
        search: Arc<dyn RecommendationSearchPort>,
    ) -> Self {
        Self {
            data_sources,
            search,
        }
    }

    pub async fn recommend(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
        request: DataSourceRecommendationRequest,
    ) -> Result<DataSourceRecommendationsResponse, AppError> {
        require_owner(organization_id, user_id)?;
        if !self.search.is_configured() {
            return Err(AppError::External);
        }
        let limit = normalize_limit(request.limit);
        let existing = self.existing(organization_id, user_id).await?;
        let response = self
            .search
            .search(
                &request.query,
                SearchOptions {
                    num_results: limit * 3,
                    use_autoprompt: true,
                    include_text: true,
                    include_highlights: true,
                    include_summary: true,
                    ..SearchOptions::default()
                },
            )
            .await
            .map_err(|_| AppError::External)?;
        Ok(DataSourceRecommendationsResponse {
            mode: "query".to_owned(),
            query: request.query.trim().to_owned(),
            seed_count: 0,
            recommendations: build_recommendations(response.results, &existing, limit as usize),
        })
    }

    pub async fn recommend_from_existing_sources(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
        request: DataSourceDiscoveryRecommendationRequest,
    ) -> Result<DataSourceRecommendationsResponse, AppError> {
        require_owner(organization_id, user_id)?;
        if !self.search.is_configured() {
            return Err(AppError::External);
        }
        let existing = self.existing(organization_id, user_id).await?;
        let seeds: Vec<_> = existing
            .iter()
            .filter(|source| source.is_enabled && !source.is_discovered)
            .take(MAX_SEEDS)
            .cloned()
            .collect();
        if seeds.is_empty() {
            return Ok(DataSourceRecommendationsResponse {
                mode: "discovery".to_owned(),
                query: String::new(),
                seed_count: 0,
                recommendations: Vec::new(),
            });
        }

        let mut candidates = Vec::new();
        let mut failures = 0;
        for seed in &seeds {
            match self
                .search
                .find_similar(
                    &seed.url,
                    SimilarOptions {
                        num_results: RESULTS_PER_SEED,
                        exclude_source_domain: true,
                        include_text: true,
                        include_highlights: true,
                        include_summary: true,
                        ..SimilarOptions::default()
                    },
                )
                .await
            {
                Ok(response) => candidates.extend(
                    response
                        .results
                        .into_iter()
                        .map(|result| (result, seed.name.clone())),
                ),
                Err(_) => failures += 1,
            }
        }
        if candidates.is_empty() && failures > 0 {
            return Err(AppError::External);
        }
        candidates.sort_by(|left, right| {
            right
                .0
                .score
                .partial_cmp(&left.0.score)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        Ok(DataSourceRecommendationsResponse {
            mode: "discovery".to_owned(),
            query: String::new(),
            seed_count: seeds.len() as i32,
            recommendations: build_discovery_recommendations(
                candidates,
                &existing,
                &seeds,
                normalize_limit(request.limit) as usize,
            ),
        })
    }

    async fn existing(
        &self,
        organization_id: Option<Uuid>,
        user_id: Option<Uuid>,
    ) -> Result<Vec<DataSource>, AppError> {
        if let Some(id) = organization_id {
            self.data_sources.find_by_organization_id(id).await
        } else if let Some(id) = user_id {
            self.data_sources.find_by_user_id(id).await
        } else {
            Err(AppError::InvalidInput(
                "Either organization_id or user_id must be provided".to_owned(),
            ))
        }
    }
}

fn build_recommendations(
    results: Vec<SearchResult>,
    existing: &[DataSource],
    limit: usize,
) -> Vec<DataSourceRecommendationResponse> {
    let existing_urls: BTreeSet<_> = existing
        .iter()
        .filter_map(|source| normalize_url(&source.url).map(|pair| pair.0))
        .collect();
    let existing_domains: BTreeSet<_> = existing
        .iter()
        .filter_map(|source| normalize_url(&source.url).map(|pair| pair.1))
        .collect();
    let mut seen = BTreeSet::new();
    results
        .into_iter()
        .filter_map(|result| {
            let (url, domain) = normalize_url(&result.url)?;
            if existing_urls.contains(&url)
                || existing_domains.contains(&domain)
                || !seen.insert(domain.clone())
            {
                return None;
            }
            Some(recommendation(result, url, domain, None))
        })
        .take(limit)
        .collect()
}

fn build_discovery_recommendations(
    candidates: Vec<(SearchResult, String)>,
    existing: &[DataSource],
    seeds: &[DataSource],
    limit: usize,
) -> Vec<DataSourceRecommendationResponse> {
    let existing_urls: BTreeSet<_> = existing
        .iter()
        .filter_map(|source| normalize_url(&source.url).map(|pair| pair.0))
        .collect();
    let existing_roots: BTreeSet<_> = existing
        .iter()
        .filter_map(|source| normalize_url(&source.url).map(|pair| domain_root(&pair.1)))
        .collect();
    let seed_roots: BTreeSet<_> = seeds
        .iter()
        .filter_map(|source| normalize_url(&source.url).map(|pair| domain_root(&pair.1)))
        .collect();
    let mut seen = BTreeSet::new();
    candidates
        .into_iter()
        .filter_map(|(result, seed)| {
            let (url, domain) = normalize_url(&result.url)?;
            let root = domain_root(&domain);
            if existing_urls.contains(&url)
                || existing_roots.contains(&root)
                || seed_roots.contains(&root)
                || !seen.insert(root)
            {
                return None;
            }
            Some(recommendation(result, url, domain, Some(seed)))
        })
        .take(limit)
        .collect()
}

fn recommendation(
    result: SearchResult,
    url: String,
    domain: String,
    seed: Option<String>,
) -> DataSourceRecommendationResponse {
    let reason = match seed {
        Some(seed) if seed.is_empty() => "Adjacent site related to your current sources".to_owned(),
        Some(seed) if result.title.trim().is_empty() => format!("Similar to {seed}"),
        Some(seed) => format!("Similar to {seed} - {}", result.title.trim()),
        None if result.title.is_empty() && result.highlights.is_empty() => {
            "Relevant source from AI search results".to_owned()
        }
        None => [
            Some(result.title.as_str()),
            result.highlights.first().map(String::as_str),
        ]
        .into_iter()
        .flatten()
        .map(str::trim)
        .filter(|part| !part.is_empty())
        .collect::<Vec<_>>()
        .join(" - "),
    };
    DataSourceRecommendationResponse {
        name: humanize_domain(&domain),
        url,
        domain,
        summary: summarize(&result),
        reason,
        source_type: infer_source_type(&result),
        score: result.score,
        favicon: result.favicon.clone(),
        sample_url: result.url.clone(),
        sample_title: result.title.clone(),
    }
}

fn normalize_limit(limit: i32) -> i32 {
    if limit <= 0 {
        DEFAULT_LIMIT
    } else {
        limit.min(MAX_LIMIT)
    }
}

fn normalize_url(raw: &str) -> Option<(String, String)> {
    let parsed = Url::parse(raw.trim()).ok()?;
    if !matches!(parsed.scheme(), "http" | "https") {
        return None;
    }
    let host = parsed.host_str()?.to_lowercase();
    Some((format!("{}://{host}", parsed.scheme()), host))
}

fn summarize(result: &SearchResult) -> String {
    if !result.summary.trim().is_empty() {
        return result.summary.trim().to_owned();
    }
    if let Some(value) = result.highlights.first() {
        return value.trim().to_owned();
    }
    let text = result.text.trim();
    if text.len() <= 220 {
        return text.to_owned();
    }
    format!(
        "{}...",
        String::from_utf8_lossy(&text.as_bytes()[..220]).trim()
    )
}

fn infer_source_type(result: &SearchResult) -> String {
    let combined = format!("{} {} {}", result.url, result.title, result.summary).to_lowercase();
    if combined.contains("substack") || combined.contains("newsletter") {
        "newsletter"
    } else if combined.contains("/feed") || combined.contains(" rss") || combined.contains("rss ") {
        "rss"
    } else if [
        "forum",
        "community",
        "discuss",
        "reddit",
        "news.ycombinator.com",
    ]
    .iter()
    .any(|value| combined.contains(value))
    {
        "forum"
    } else if combined.contains("news") || combined.contains("press") {
        "news"
    } else {
        "blog"
    }
    .to_owned()
}

fn humanize_domain(domain: &str) -> String {
    domain
        .trim_start_matches("www.")
        .split('.')
        .next()
        .unwrap_or(domain)
        .split(['-', '_'])
        .filter(|part| !part.is_empty())
        .map(|part| {
            let mut chars = part.chars();
            match chars.next() {
                Some(first) => first.to_uppercase().chain(chars).collect(),
                None => String::new(),
            }
        })
        .collect::<Vec<String>>()
        .join(" ")
}

fn domain_root(domain: &str) -> String {
    let parts: Vec<_> = domain
        .trim()
        .to_lowercase()
        .trim_start_matches("www.")
        .split('.')
        .map(str::to_owned)
        .collect();
    if parts.len() < 2 {
        parts.first().cloned().unwrap_or_default()
    } else {
        format!("{}.{}", parts[parts.len() - 2], parts[parts.len() - 1])
    }
}

fn require_owner(organization_id: Option<Uuid>, user_id: Option<Uuid>) -> Result<(), AppError> {
    if organization_id.is_none() && user_id.is_none() {
        Err(AppError::InvalidInput(
            "Either organization_id or user_id must be provided".to_owned(),
        ))
    } else {
        Ok(())
    }
}
