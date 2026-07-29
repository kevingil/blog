mod ports;
mod service;
mod types;

pub use ports::{ArticleLookupPort, EmbeddingPort, FetchExtractPort, SourceRepository};
pub use service::SourceService;
pub use types::{
    AgentResourceSelection, CreateSourceRequest, MetaData, ScrapedContent, Source,
    SourceListOptions, SourceListResponse, SourceWithArticle, UpdateSourceRequest,
};
