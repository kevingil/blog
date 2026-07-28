mod ports;
mod service;
mod types;

pub use ports::{
    ContentTopicMatchRepository, EmbeddingPort, InsightContentRepository, InsightRepository,
    InsightTopicRepository, UserInsightStatusRepository,
};
pub use service::InsightService;
pub use types::{
    ContentTopicMatch, Insight, InsightResponse, InsightSearchRequest, InsightTopic,
    InsightTopicCreateRequest, InsightTopicResponse, InsightTopicUpdateRequest, InsightWithSources,
    InsightWithUserStatus, MetaData, UserInsightStatus, UserInsightStatusResponse,
};
