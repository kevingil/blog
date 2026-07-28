mod ports;
mod recommendation_service;
mod service;
mod types;

pub use ports::{
    CrawledContentRepository, DataSourceRepository, RecommendationSearchPort, SearchOptions,
    SearchResponse, SearchResult, SimilarOptions,
};
pub use recommendation_service::RecommendationService;
pub use service::DataSourceService;
pub use types::{
    CrawledContent, CrawledContentResponse, DataSource, DataSourceCreateRequest,
    DataSourceDiscoveryRecommendationRequest, DataSourceRecommendationRequest,
    DataSourceRecommendationResponse, DataSourceRecommendationsResponse, DataSourceResponse,
    DataSourceUpdateRequest, MetaData,
};
