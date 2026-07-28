use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use blog_backend::{
    core::ml::llm::{
        AnswerCitation, AnswerResponse, AskQuestionTool, GetRelevantSourcesTool, ResearchPort,
        SearchWebSourcesTool, SelectSourcesForEditTool, SourceResource, SourceResourcePort,
        SourceSelection, Tool, ToolCallRequest, ToolContext, WebSearchResponse, WebSearchResult,
    },
    error::AppError,
};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

struct FixtureResearch;

#[async_trait]
impl ResearchPort for FixtureResearch {
    fn is_configured(&self) -> bool {
        true
    }

    async fn search(&self, query: &str) -> Result<WebSearchResponse, AppError> {
        Ok(WebSearchResponse {
            results: vec![WebSearchResult {
                id: "result-1".to_owned(),
                title: "Primary source".to_owned(),
                url: "https://example.com/source".to_owned(),
                text: format!("Evidence for {query}"),
                summary: "Summary".to_owned(),
                author: "Author".to_owned(),
                published_date: "2026-01-01".to_owned(),
                highlights: vec!["Evidence".to_owned()],
                score: 0.9,
                favicon: String::new(),
            }],
            request_id: "exa-request".to_owned(),
            resolved_search_type: "auto".to_owned(),
            cost_dollars: None,
        })
    }

    async fn answer(&self, question: &str) -> Result<AnswerResponse, AppError> {
        Ok(AnswerResponse {
            answer: format!("Answer to {question}"),
            citations: vec![AnswerCitation {
                url: "https://example.com/citation".to_owned(),
                title: "Citation".to_owned(),
                author: "Author".to_owned(),
                published_date: "2026-01-01".to_owned(),
                favicon: String::new(),
                text: "Citation text".to_owned(),
            }],
            cost_dollars: None,
        })
    }
}

#[derive(Default)]
struct FixtureSources {
    values: Mutex<Vec<SourceResource>>,
}

impl FixtureSources {
    fn source(article_id: Uuid, content: &str) -> SourceResource {
        SourceResource {
            id: Uuid::new_v4(),
            article_id,
            title: "Source".to_owned(),
            content: content.to_owned(),
            url: "https://example.com".to_owned(),
            source_type: "web".to_owned(),
            meta_data: Default::default(),
            created_at: None,
        }
    }
}

#[async_trait]
impl SourceResourcePort for FixtureSources {
    async fn create_web_source(
        &self,
        article_id: Uuid,
        _query: &str,
        result: &WebSearchResult,
        _request_id: &str,
    ) -> Result<SourceResource, AppError> {
        let source = Self::source(article_id, &result.text);
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .push(source.clone());
        Ok(source)
    }

    async fn list(&self, article_id: Uuid) -> Result<Vec<SourceResource>, AppError> {
        Ok(self
            .values
            .lock()
            .map_err(|_| AppError::Internal)?
            .iter()
            .filter(|source| source.article_id == article_id)
            .cloned()
            .collect())
    }

    async fn search_similar(
        &self,
        article_id: Uuid,
        _query: &str,
        _limit: i64,
    ) -> Result<Vec<SourceResource>, AppError> {
        self.list(article_id).await
    }

    async fn select(
        &self,
        article_id: Uuid,
        selection: SourceSelection,
        _request_id: &str,
    ) -> Result<SourceResource, AppError> {
        let mut source = Self::source(article_id, &selection.excerpt_text);
        source.title = selection.title;
        self.values
            .lock()
            .map_err(|_| AppError::Internal)?
            .push(source.clone());
        Ok(source)
    }
}

fn context(article_id: Option<Uuid>) -> ToolContext {
    ToolContext::new(
        "session",
        "message",
        "request",
        article_id,
        "",
        "",
        CancellationToken::new(),
    )
}

#[tokio::test]
async fn answer_and_search_tools_preserve_structured_artifacts() {
    let research = Arc::new(FixtureResearch);
    let sources = Arc::new(FixtureSources::default());
    let article_id = Uuid::new_v4();
    let answer = AskQuestionTool::new(research.clone())
        .run(
            context(Some(article_id)),
            ToolCallRequest {
                id: "answer".to_owned(),
                name: "ask_question".to_owned(),
                input: r#"{"question":"What changed?"}"#.to_owned(),
            },
        )
        .await;
    assert!(answer.is_ok());
    let Ok(answer) = answer else {
        return;
    };
    assert_eq!(
        answer
            .artifact
            .as_ref()
            .map(|artifact| artifact.artifact_type.as_str()),
        Some("answer")
    );
    assert_eq!(answer.result["citation_count"], 1);

    let search = SearchWebSourcesTool::new(research, sources.clone())
        .run(
            context(Some(article_id)),
            ToolCallRequest {
                id: "search".to_owned(),
                name: "search_web_sources".to_owned(),
                input: r#"{"query":"Rust migration"}"#.to_owned(),
            },
        )
        .await;
    assert!(search.is_ok());
    let Ok(search) = search else {
        return;
    };
    assert_eq!(search.result["sources_successful"], 1);
    assert_eq!(sources.list(article_id).await.unwrap_or_default().len(), 1);
}

#[tokio::test]
async fn relevant_and_selection_tools_delegate_ranking_and_persist_exact_excerpt() {
    let article_id = Uuid::new_v4();
    let sources = Arc::new(FixtureSources::default());
    sources
        .values
        .lock()
        .map(|mut values| values.push(FixtureSources::source(article_id, "vector-ranked evidence")))
        .unwrap_or_default();
    let relevant = GetRelevantSourcesTool::new(sources.clone())
        .run(
            context(Some(article_id)),
            ToolCallRequest {
                id: "relevant".to_owned(),
                name: "get_relevant_sources".to_owned(),
                input: r#"{"query":"evidence","limit":5}"#.to_owned(),
            },
        )
        .await;
    assert!(relevant.is_ok());
    let Ok(relevant) = relevant else {
        return;
    };
    assert_eq!(relevant.result["total_found"], 1);
    assert_eq!(
        relevant.result["relevant_sources"][0]["excerpt_text"],
        "vector-ranked evidence"
    );

    let selected = SelectSourcesForEditTool::new(sources.clone())
        .run(
            context(Some(article_id)),
            ToolCallRequest {
                id: "select".to_owned(),
                name: "select_sources_for_edit".to_owned(),
                input: r#"{"sources":[{"title":"Chosen","excerpt_text":"exact excerpt","origin_tool":"ask_question"}]}"#.to_owned(),
            },
        )
        .await;
    assert!(selected.is_ok());
    let Ok(selected) = selected else {
        return;
    };
    assert_eq!(selected.result["selected_count"], 1);
    assert!(
        sources
            .list(article_id)
            .await
            .unwrap_or_default()
            .iter()
            .any(|source| source.content == "exact excerpt")
    );
}
