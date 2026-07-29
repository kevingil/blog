use axum::{
    Json,
    body::Bytes,
    extract::{Path, Query, State},
    http::StatusCode,
};
use uuid::Uuid;

use crate::{
    api::{auth::AuthenticatedAccount, request::JsonBody, response::SuccessResponse},
    core::article::{CreateArticle, UpdateArticle},
    error::AppError,
};

use super::{
    dto::{
        ArticleListQuery, ArticleSearchQuery, DeleteArticleResponse, GenerateArticleRequest,
        GenerateArticleResponse, PopularTagsResponse, PublishArticleRequest, timestamp,
    },
    state::{ArticleState, GenerationRequest},
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

#[utoipa::path(
    post,
    path = "/blog/generate",
    request_body = GenerateArticleRequest,
    responses(
        (status = 200, body = SuccessResponse<GenerateArticleResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "generateArticle"
)]
pub async fn generate_article(
    authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    JsonBody(request): JsonBody<GenerateArticleRequest>,
) -> ApiResult<GenerateArticleResponse> {
    let author_id = authenticated.into_inner().into_inner();
    let prompt = request.prompt.trim();
    let title = request.title.trim();
    if prompt.is_empty() && title.is_empty() {
        return Err(AppError::InvalidInput(
            "either title or prompt is required".to_owned(),
        ));
    }
    let draft_title = if title.is_empty() {
        "Untitled Article"
    } else {
        title
    };
    let message = match (prompt.is_empty(), title.is_empty()) {
        (true, false) => format!("Write a complete blog article about: {title}"),
        (false, false) => format!("Title: {title}\n\n{prompt}"),
        _ => prompt.to_owned(),
    };
    let service = state.service()?;
    let article = service.create_draft_shell(draft_title, author_id).await?;
    let request_id = match state
        .generation_queue()?
        .submit(GenerationRequest {
            message,
            article_id: article.id,
        })
        .await
    {
        Ok(request_id) => request_id,
        Err(error) => {
            if let Err(rollback_error) = service.delete(article.id).await {
                tracing::error!(
                    article_id = %article.id,
                    %rollback_error,
                    "failed to roll back draft after generation queue rejection"
                );
            }
            return Err(error);
        }
    };
    Ok(Json(SuccessResponse::new(GenerateArticleResponse {
        article,
        request_id,
    })))
}

#[utoipa::path(
    post,
    path = "/blog/articles",
    request_body = CreateArticle,
    responses(
        (status = 201, body = SuccessResponse<crate::core::article::ArticleListItem>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "createArticle"
)]
pub async fn create_article(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    JsonBody(request): JsonBody<CreateArticle>,
) -> Result<
    (
        StatusCode,
        Json<SuccessResponse<crate::core::article::ArticleListItem>>,
    ),
    AppError,
> {
    let article = state.service()?.create(request).await?;
    Ok((StatusCode::CREATED, Json(SuccessResponse::new(article))))
}

#[utoipa::path(
    post,
    path = "/blog/articles/{slug}/update",
    params(("slug" = String, Path)),
    request_body = UpdateArticle,
    responses(
        (status = 200, body = SuccessResponse<crate::core::article::ArticleListItem>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "updateArticle"
)]
pub async fn update_article(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(slug): Path<String>,
    JsonBody(request): JsonBody<UpdateArticle>,
) -> ApiResult<crate::core::article::ArticleListItem> {
    if slug.is_empty() {
        return Err(AppError::InvalidInput(
            "article slug is required".to_owned(),
        ));
    }
    let service = state.service()?;
    let article_id = service.get_id_by_slug(&slug).await?;
    Ok(Json(SuccessResponse::new(
        service.update(article_id, request).await?,
    )))
}

#[utoipa::path(
    put,
    path = "/blog/{id}/update",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<crate::core::article::Article>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "updateArticleWithContext"
)]
pub async fn update_article_with_context(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(id): Path<Uuid>,
) -> ApiResult<crate::core::article::Article> {
    Ok(Json(SuccessResponse::new(
        state.service()?.update_with_context(id).await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/articles",
    params(ArticleListQuery),
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleListResponse>)),
    tag = "articles",
    operation_id = "getArticles"
)]
pub async fn get_articles(
    State(state): State<ArticleState>,
    Query(query): Query<ArticleListQuery>,
) -> ApiResult<crate::core::article::ArticleListResponse> {
    Ok(Json(SuccessResponse::new(
        state
            .service()?
            .list(
                query.page.unwrap_or(1),
                query.tag.as_deref().unwrap_or_default(),
                query.status.as_deref().unwrap_or("published"),
                query.articles_per_page.unwrap_or(6),
                query.sort_by.as_deref().unwrap_or_default(),
                query.sort_order.as_deref().unwrap_or_default(),
            )
            .await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/articles/search",
    params(ArticleSearchQuery),
    responses(
        (status = 200, body = SuccessResponse<crate::core::article::ArticleListResponse>),
        (status = 400, body = crate::error::ErrorEnvelope)
    ),
    tag = "articles",
    operation_id = "searchArticles"
)]
pub async fn search_articles(
    State(state): State<ArticleState>,
    Query(query): Query<ArticleSearchQuery>,
) -> ApiResult<crate::core::article::ArticleListResponse> {
    if query.query.is_empty() {
        return Err(AppError::InvalidInput(
            "query parameter is required".to_owned(),
        ));
    }
    // The Go service accepted these compatibility parameters but did not apply
    // them to repository search options.
    let _compatibility_parameters = (&query.tag, &query.sort_by, &query.sort_order);
    Ok(Json(SuccessResponse::new(
        state
            .service()?
            .search(
                &query.query,
                query.page.unwrap_or(1),
                query.status.as_deref().unwrap_or("published"),
            )
            .await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/tags/popular",
    responses((status = 200, body = SuccessResponse<PopularTagsResponse>)),
    tag = "articles",
    operation_id = "getPopularTags"
)]
pub async fn get_popular_tags(State(state): State<ArticleState>) -> ApiResult<PopularTagsResponse> {
    Ok(Json(SuccessResponse::new(PopularTagsResponse {
        tags: state.service()?.get_popular_tags().await?,
    })))
}

#[utoipa::path(
    get,
    path = "/blog/articles/{slug}",
    params(("slug" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<crate::core::article::ArticleListItem>),
        (status = 404, body = crate::error::ErrorEnvelope)
    ),
    tag = "articles",
    operation_id = "getArticleData"
)]
pub async fn get_article_data(
    State(state): State<ArticleState>,
    Path(slug): Path<String>,
) -> ApiResult<crate::core::article::ArticleListItem> {
    if slug.is_empty() {
        return Err(AppError::InvalidInput("slug is required".to_owned()));
    }
    Ok(Json(SuccessResponse::new(
        state.service()?.get_by_slug(&slug).await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/articles/{id}/recommended",
    params(("id" = Uuid, Path)),
    responses((status = 200, body = SuccessResponse<Option<Vec<crate::core::article::RecommendedArticle>>>)),
    tag = "articles",
    operation_id = "getRecommendedArticles"
)]
pub async fn get_recommended_articles(
    State(state): State<ArticleState>,
    Path(id): Path<Uuid>,
) -> ApiResult<Option<Vec<crate::core::article::RecommendedArticle>>> {
    let articles = state.service()?.get_recommended(id).await?;
    Ok(Json(SuccessResponse::new(
        (!articles.is_empty()).then_some(articles),
    )))
}

#[utoipa::path(
    delete,
    path = "/blog/articles/{slug}",
    params(("slug" = Uuid, Path, description = "Article ID")),
    responses(
        (status = 200, body = SuccessResponse<DeleteArticleResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "deleteArticle"
)]
pub async fn delete_article(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(id): Path<Uuid>,
) -> ApiResult<DeleteArticleResponse> {
    state.service()?.delete(id).await?;
    Ok(Json(SuccessResponse::new(DeleteArticleResponse {
        success: true,
    })))
}

#[utoipa::path(
    post,
    path = "/blog/articles/{slug}/publish",
    params(("slug" = String, Path)),
    request_body = Option<PublishArticleRequest>,
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleListItem>)),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "publishArticle"
)]
pub async fn publish_article(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(slug): Path<String>,
    body: Bytes,
) -> ApiResult<crate::core::article::ArticleListItem> {
    let service = state.service()?;
    let article_id = service.get_id_by_slug(&slug).await?;
    let request = if body.is_empty() {
        PublishArticleRequest::default()
    } else {
        serde_json::from_slice(&body).unwrap_or_default()
    };
    let published_at = match request.published_at {
        Some(value) => Some(
            timestamp(value)
                .ok_or_else(|| AppError::InvalidInput("published_at is out of range".to_owned()))?,
        ),
        None => None,
    };
    Ok(Json(SuccessResponse::new(
        service.publish(article_id, published_at).await?,
    )))
}

#[utoipa::path(
    post,
    path = "/blog/articles/{slug}/unpublish",
    params(("slug" = String, Path)),
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleListItem>)),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "unpublishArticle"
)]
pub async fn unpublish_article(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(slug): Path<String>,
) -> ApiResult<crate::core::article::ArticleListItem> {
    let service = state.service()?;
    let article_id = service.get_id_by_slug(&slug).await?;
    Ok(Json(SuccessResponse::new(
        service.unpublish(article_id).await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/articles/{slug}/versions",
    params(("slug" = String, Path)),
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleVersionListResponse>)),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "listArticleVersions"
)]
pub async fn list_versions(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(slug): Path<String>,
) -> ApiResult<crate::core::article::ArticleVersionListResponse> {
    let service = state.service()?;
    let article_id = service.get_id_by_slug(&slug).await?;
    Ok(Json(SuccessResponse::new(
        service.list_versions(article_id).await?,
    )))
}

#[utoipa::path(
    get,
    path = "/blog/articles/versions/{versionId}",
    params(("versionId" = Uuid, Path)),
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleVersionResponse>)),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "getArticleVersion"
)]
pub async fn get_version(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path(version_id): Path<Uuid>,
) -> ApiResult<crate::core::article::ArticleVersionResponse> {
    Ok(Json(SuccessResponse::new(
        state.service()?.get_version(version_id).await?,
    )))
}

#[utoipa::path(
    post,
    path = "/blog/articles/{slug}/revert/{versionId}",
    params(("slug" = String, Path), ("versionId" = Uuid, Path)),
    responses((status = 200, body = SuccessResponse<crate::core::article::ArticleListItem>)),
    security(("bearerAuth" = [])),
    tag = "articles",
    operation_id = "revertArticleToVersion"
)]
pub async fn revert_to_version(
    _authenticated: AuthenticatedAccount,
    State(state): State<ArticleState>,
    Path((slug, version_id)): Path<(String, Uuid)>,
) -> ApiResult<crate::core::article::ArticleListItem> {
    let service = state.service()?;
    let article_id = service.get_id_by_slug(&slug).await?;
    Ok(Json(SuccessResponse::new(
        service.revert_to_version(article_id, version_id).await?,
    )))
}
