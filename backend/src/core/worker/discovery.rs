use std::sync::Arc;

use async_trait::async_trait;
use reqwest::Url;
use serde_json::json;

use crate::{
    core::datasource::{
        DataSourceResponse, DataSourceService, RecommendationSearchPort, SimilarOptions,
    },
    error::AppError,
};

use super::{StatusService, Worker, WorkerContext, WorkerFailure, WorkerResult, WorkerState};

const DISCOVERY_WORKER_NAME: &str = "discovery";
const DEFAULT_MAX_DISCOVERIES: i32 = 5;
const DEFAULT_MAX_SOURCES: usize = 5;

pub struct DiscoveryWorker {
    status: Arc<StatusService>,
    data_sources: Arc<DataSourceService>,
    search: Option<Arc<dyn RecommendationSearchPort>>,
    max_discoveries: i32,
    max_sources: usize,
}

impl DiscoveryWorker {
    pub fn new(
        status: Arc<StatusService>,
        data_sources: Arc<DataSourceService>,
        search: Option<Arc<dyn RecommendationSearchPort>>,
    ) -> Self {
        Self {
            status,
            data_sources,
            search,
            max_discoveries: DEFAULT_MAX_DISCOVERIES,
            max_sources: DEFAULT_MAX_SOURCES,
        }
    }

    async fn discover_similar(
        &self,
        search: &dyn RecommendationSearchPort,
        source: &DataSourceResponse,
    ) -> Result<i32, WorkerFailure> {
        let response = search
            .find_similar(
                &source.url,
                SimilarOptions {
                    num_results: self.max_discoveries + 5,
                    exclude_source_domain: true,
                    ..SimilarOptions::default()
                },
            )
            .await
            .map_err(|error| WorkerFailure::new(format!("Exa findSimilar failed: {error}")))?;
        let mut discovered = 0_i32;
        for result in response.results {
            if discovered >= self.max_discoveries {
                break;
            }
            let Some(url) = normalize_url(&result.url) else {
                continue;
            };
            if is_same_domain_root(&source.url, &url) {
                continue;
            }
            let name = if result.title.is_empty() {
                extract_domain_name(&url)
            } else {
                result.title
            };
            match self
                .data_sources
                .create_discovered_source(
                    source.organization_id,
                    source.user_id,
                    source.id,
                    name,
                    url,
                )
                .await
            {
                Ok(_) => discovered += 1,
                Err(AppError::Conflict(_)) => {}
                Err(_) => {}
            }
        }
        Ok(discovered)
    }
}

#[async_trait]
impl Worker for DiscoveryWorker {
    fn name(&self) -> &str {
        DISCOVERY_WORKER_NAME
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        let Some(search) = self.search.as_ref().filter(|search| search.is_configured()) else {
            self.status.update_status(
                self.name(),
                WorkerState::Running,
                100,
                "Exa not configured, skipping",
            );
            return Ok(WorkerResult::warning(
                "Exa not configured, discovery skipped",
                vec!["Exa client not configured".to_owned()],
            ));
        };
        self.status.update_status(
            self.name(),
            WorkerState::Running,
            0,
            "Fetching data sources...",
        );
        let (sources, _) = tokio::select! {
            biased;
            () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.data_sources.list_all(1, 100) => {
                result.map_err(|error| WorkerFailure::new(format!("failed to get data sources: {error}")))?
            }
        };
        let sources = sources
            .into_iter()
            .filter(|source| source.is_enabled && !source.is_discovered)
            .take(self.max_sources)
            .collect::<Vec<_>>();
        if sources.is_empty() {
            self.status.update_status(
                self.name(),
                WorkerState::Running,
                100,
                "No manual sources to discover from",
            );
            return Ok(WorkerResult::warning(
                "No manual sources to discover from",
                vec!["No enabled manual sources were available".to_owned()],
            ));
        }

        let total = i32::try_from(sources.len())
            .map_err(|_| WorkerFailure::new("too many discovery sources"))?;
        self.status
            .set_progress(self.name(), 0, total, format!("Processing {total} sources"));
        let mut total_discovered = 0_i32;
        for (index, source) in sources.into_iter().enumerate() {
            if context.cancellation().is_cancelled() {
                return Err(WorkerFailure::new("operation cancelled"));
            }
            let done = i32::try_from(index)
                .map_err(|_| WorkerFailure::new("too many discovery sources"))?;
            self.status.set_progress(
                self.name(),
                done,
                total,
                format!("Finding similar sites for: {}", source.name),
            );
            if let Ok(discovered) = self.discover_similar(search.as_ref(), &source).await {
                total_discovered = total_discovered.saturating_add(discovered);
            }
        }
        self.status.set_progress(
            self.name(),
            total,
            total,
            format!("Discovered {total_discovered} new sites"),
        );
        let mut result = if total_discovered == 0 {
            WorkerResult::warning(
                "Discovery completed without new sites",
                vec!["No new adjacent sites were discovered".to_owned()],
            )
        } else {
            WorkerResult::completed(format!("Discovered {total_discovered} new sites"))
        };
        result
            .metrics
            .insert("seed_sources".to_owned(), json!(total));
        result
            .metrics
            .insert("discovered_sources".to_owned(), json!(total_discovered));
        if total_discovered > 0 {
            result
                .output_summary
                .insert("discovered_sources".to_owned(), json!(total_discovered));
        }
        Ok(result)
    }
}

pub fn normalize_url(raw_url: &str) -> Option<String> {
    let mut url = Url::parse(raw_url).ok()?;
    if !matches!(url.scheme(), "http" | "https") {
        return None;
    }
    let trimmed_path = url.path().trim_end_matches('/').to_owned();
    url.set_path(&trimmed_path);
    url.set_fragment(None);
    let query = url
        .query_pairs()
        .filter(|(key, _)| {
            let key = key.to_ascii_lowercase();
            !key.starts_with("utm_") && !matches!(key.as_str(), "ref" | "source" | "campaign")
        })
        .map(|(key, value)| (key.into_owned(), value.into_owned()))
        .collect::<Vec<_>>();
    url.query_pairs_mut().clear().extend_pairs(query);
    if url.query() == Some("") {
        url.set_query(None);
    }
    Some(url.to_string())
}

fn is_same_domain_root(left: &str, right: &str) -> bool {
    match (domain_root(left), domain_root(right)) {
        (Some(left), Some(right)) => left == right,
        _ => false,
    }
}

fn domain_root(value: &str) -> Option<String> {
    let host = Url::parse(value).ok()?.host_str()?.to_ascii_lowercase();
    let host = host.strip_prefix("www.").unwrap_or(&host);
    let parts = host.split('.').collect::<Vec<_>>();
    if parts.len() >= 2 {
        Some(format!(
            "{}.{}",
            parts[parts.len() - 2],
            parts[parts.len() - 1]
        ))
    } else {
        Some(host.to_owned())
    }
}

fn extract_domain_name(value: &str) -> String {
    let Some(host) = Url::parse(value)
        .ok()
        .and_then(|url| url.host_str().map(str::to_owned))
    else {
        return value.to_owned();
    };
    let host = host.strip_prefix("www.").unwrap_or(&host);
    let name = host.split('.').next().unwrap_or(host);
    let mut characters = name.chars();
    match characters.next() {
        Some(first) => first.to_uppercase().chain(characters).collect(),
        None => host.to_owned(),
    }
}
