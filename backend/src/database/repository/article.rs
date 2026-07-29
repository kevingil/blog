use std::{
    collections::{BTreeMap, BTreeSet, VecDeque},
    panic::AssertUnwindSafe,
    sync::{Arc, Mutex, MutexGuard},
    time::Duration,
};

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use diesel::{
    BoolExpressionMethods, ExpressionMethods, OptionalExtension, PgArrayExpressionMethods,
    PgSortExpressionMethods, PgTextExpressionMethods, QueryDsl, SelectableHelper,
    dsl::{count_star, max},
    result::{DatabaseErrorKind, Error as DieselError},
};
use diesel_async::{AsyncConnection, RunQueryDsl};
use futures_util::FutureExt;
use pgvector::{Vector, VectorExpressionMethods};
use serde_json::{Map, Value};
use tokio::{
    sync::Notify,
    task::JoinHandle,
    time::{Instant, timeout_at},
};
use tokio_util::sync::CancellationToken;
use tracing::error;
use uuid::Uuid;

use crate::{
    core::article::{
        Article, ArticleListOptions, ArticleRepository, ArticleSearchOptions, ArticleVersion,
    },
    database::{
        models::article::{
            ArticleChangeset, ArticleRow, ArticleVersionRow, NewArticleRow, NewArticleVersionRow,
        },
        pool::PgPool,
    },
    error::AppError,
    schema::{article, article_version},
};

const MAX_BACKGROUND_TASKS: usize = 64;
const MAX_RECORDED_FAILURES: usize = 64;
type OwnedTaskHandles = Vec<(u64, Option<JoinHandle<()>>)>;

#[derive(Clone)]
pub struct DieselArticleRepository {
    pool: PgPool,
    background_tasks: Arc<BackgroundTasks>,
}

impl DieselArticleRepository {
    pub fn new(pool: PgPool) -> Self {
        Self {
            pool,
            background_tasks: Arc::new(BackgroundTasks::new()),
        }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }

    fn reserve_version(&self) -> Result<VersionReservation, AppError> {
        BackgroundTasks::reserve(&self.background_tasks)
    }
}

#[async_trait]
impl ArticleRepository for DieselArticleRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Article, AppError> {
        let mut connection = self.connection().await?;
        article::table
            .find(id)
            .select(ArticleRow::as_select())
            .first::<ArticleRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Article, AppError> {
        let mut connection = self.connection().await?;
        article::table
            .filter(article::slug.eq(slug))
            .select(ArticleRow::as_select())
            .first::<ArticleRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn list(&self, options: ArticleListOptions) -> Result<(Vec<Article>, i64), AppError> {
        let tag_id = options.tag_id.map(tag_id_to_i32).transpose()?;
        let offset = pagination_offset(options.page, options.per_page)?;
        let mut connection = self.connection().await?;

        let mut count_query = article::table.into_boxed();
        if options.published_only {
            count_query = count_query.filter(article::published_at.is_not_null());
        }
        if let Some(author_id) = options.author_id {
            count_query = count_query.filter(article::author_id.eq(author_id));
        }
        if let Some(tag_id) = tag_id {
            count_query = count_query.filter(article::tag_ids.contains(vec![Some(tag_id)]));
        }
        let total = count_query
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        let mut query = article::table.into_boxed();
        if options.published_only {
            query = query.filter(article::published_at.is_not_null());
        }
        if let Some(author_id) = options.author_id {
            query = query.filter(article::author_id.eq(author_id));
        }
        if let Some(tag_id) = tag_id {
            query = query.filter(article::tag_ids.contains(vec![Some(tag_id)]));
        }

        let sort_column = match options.sort_by.as_str() {
            "created_at" => SortColumn::Created,
            "updated_at" => SortColumn::Updated,
            _ => SortColumn::Published,
        };
        let ascending = options.sort_order == "asc";
        query = match (sort_column, ascending) {
            (SortColumn::Published, true) => query.order(article::published_at.asc().nulls_last()),
            (SortColumn::Published, false) => {
                query.order(article::published_at.desc().nulls_last())
            }
            (SortColumn::Created, true) => query.order(article::created_at.asc()),
            (SortColumn::Created, false) => query.order(article::created_at.desc()),
            (SortColumn::Updated, true) => query.order(article::updated_at.asc()),
            (SortColumn::Updated, false) => query.order(article::updated_at.desc()),
        };

        query = query.offset(offset);
        if options.per_page >= 0 {
            query = query.limit(options.per_page);
        }
        let rows = query
            .select(ArticleRow::as_select())
            .load::<ArticleRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        Ok((articles_from_rows(rows)?, total))
    }

    async fn search(&self, options: ArticleSearchOptions) -> Result<(Vec<Article>, i64), AppError> {
        let offset = pagination_offset(options.page, options.per_page)?;
        let pattern = (!options.query.is_empty()).then(|| format!("%{}%", options.query));
        let mut connection = self.connection().await?;

        let mut count_query = article::table.into_boxed();
        if let Some(pattern) = pattern.as_deref() {
            count_query = count_query.filter(
                article::draft_title
                    .ilike(pattern)
                    .or(article::draft_content.ilike(pattern))
                    .or(article::published_title.ilike(pattern))
                    .or(article::published_content.ilike(pattern)),
            );
        }
        if options.published_only {
            count_query = count_query.filter(article::published_at.is_not_null());
        }
        let total = count_query
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        let mut query = article::table.into_boxed();
        if let Some(pattern) = pattern.as_deref() {
            query = query.filter(
                article::draft_title
                    .ilike(pattern)
                    .or(article::draft_content.ilike(pattern))
                    .or(article::published_title.ilike(pattern))
                    .or(article::published_content.ilike(pattern)),
            );
        }
        if options.published_only {
            query = query.filter(article::published_at.is_not_null());
        }
        query = query
            .order(article::published_at.desc().nulls_last())
            .offset(offset);
        if options.per_page >= 0 {
            query = query.limit(options.per_page);
        }
        let rows = query
            .select(ArticleRow::as_select())
            .load::<ArticleRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        Ok((articles_from_rows(rows)?, total))
    }

    async fn search_by_embedding(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Article>, AppError> {
        let mut connection = self.connection().await?;
        let rows = article::table
            .filter(article::draft_embedding.is_not_null())
            .order(article::draft_embedding.l2_distance(Vector::from(embedding.to_vec())))
            .limit(limit)
            .select(ArticleRow::as_select())
            .load::<ArticleRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        articles_from_rows(rows)
    }

    async fn save(&self, value: &mut Article) -> Result<(), AppError> {
        let tag_ids = tag_ids_to_database(value.tag_ids.as_deref())?;
        let now = Utc::now();
        if value.id.is_nil() {
            value.id = Uuid::new_v4();
        }

        let mut connection = self.connection().await?;
        let exists = article::table
            .find(value.id)
            .select(article::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .is_some();

        if exists {
            let changeset = article_changeset(value, tag_ids, now);
            diesel::update(article::table.find(value.id))
                .set(changeset)
                .execute(&mut connection)
                .await
                .map_err(map_diesel_error)?;
        } else {
            let row = new_article_row(value, tag_ids, now);
            diesel::insert_into(article::table)
                .values(row)
                .execute(&mut connection)
                .await
                .map_err(map_diesel_error)?;
        }
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::delete(article::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if affected == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn get_popular_tags(&self, limit: i64) -> Result<Vec<i64>, AppError> {
        let mut connection = self.connection().await?;
        diesel::sql_query(
            r#"
            SELECT tag_id, COUNT(*)::BIGINT AS count
            FROM article, unnest(tag_ids) AS tag_id
            WHERE published_at IS NOT NULL
            GROUP BY tag_id
            ORDER BY count DESC
            LIMIT $1
            "#,
        )
        .bind::<diesel::sql_types::BigInt, _>(limit)
        .load::<PopularTagRow>(&mut connection)
        .await
        .map(|rows| rows.into_iter().map(|row| i64::from(row.tag_id)).collect())
        .map_err(map_diesel_error)
    }

    async fn slug_exists(&self, slug: &str, exclude_id: Option<Uuid>) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        let mut query = article::table.filter(article::slug.eq(slug)).into_boxed();
        if let Some(exclude_id) = exclude_id {
            query = query.filter(article::id.ne(exclude_id));
        }
        query
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map(|count| count > 0)
            .map_err(map_diesel_error)
    }

    async fn save_draft(&self, value: &mut Article) -> Result<(), AppError> {
        let reservation = self.reserve_version()?;
        let now = Utc::now();
        let mut connection = self.connection().await?;
        diesel::update(article::table.filter(article::id.eq(value.id)))
            .set((
                article::draft_title.eq(&value.draft_title),
                article::draft_content.eq(&value.draft_content),
                article::draft_image_url.eq(&value.draft_image_url),
                article::updated_at.eq(now),
            ))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        value.updated_at = Some(now);
        reservation.spawn(
            self.pool.clone(),
            VersionInput::draft(value, Some(value.author_id)),
        )
    }

    async fn publish(
        &self,
        value: &mut Article,
        published_at: Option<DateTime<Utc>>,
    ) -> Result<(), AppError> {
        let reservation = self.reserve_version()?;
        let now = Utc::now();
        let published_at = published_at.unwrap_or(now);
        let embedding = vector_or_none(&value.draft_embedding);
        let mut connection = self.connection().await?;
        diesel::update(article::table.filter(article::id.eq(value.id)))
            .set((
                article::published_title.eq(Some(value.draft_title.clone())),
                article::published_content.eq(Some(value.draft_content.clone())),
                article::published_image_url.eq(Some(value.draft_image_url.clone())),
                article::published_embedding.eq(embedding),
                article::published_at.eq(Some(published_at)),
                article::updated_at.eq(now),
            ))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        value.published_title = Some(value.draft_title.clone());
        value.published_content = Some(value.draft_content.clone());
        value.published_image_url = Some(value.draft_image_url.clone());
        value.published_embedding.clone_from(&value.draft_embedding);
        value.published_at = Some(published_at);
        value.updated_at = Some(now);
        reservation.spawn(
            self.pool.clone(),
            VersionInput::published(value, Some(value.author_id)),
        )
    }

    async fn unpublish(&self, value: &mut Article) -> Result<(), AppError> {
        let now = Utc::now();
        let mut connection = self.connection().await?;
        diesel::update(article::table.filter(article::id.eq(value.id)))
            .set((
                article::published_title.eq::<Option<String>>(None),
                article::published_content.eq::<Option<String>>(None),
                article::published_image_url.eq::<Option<String>>(None),
                article::published_embedding.eq::<Option<Vector>>(None),
                article::published_at.eq::<Option<DateTime<Utc>>>(None),
                article::current_published_version_id.eq::<Option<Uuid>>(None),
                article::updated_at.eq(now),
            ))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        value.published_title = None;
        value.published_content = None;
        value.published_image_url = None;
        value.published_embedding.clear();
        value.published_at = None;
        value.current_published_version_id = None;
        value.updated_at = Some(now);
        Ok(())
    }

    async fn list_versions(&self, article_id: Uuid) -> Result<Vec<ArticleVersion>, AppError> {
        let mut connection = self.connection().await?;
        article_version::table
            .filter(article_version::article_id.eq(article_id))
            .order(article_version::version_number.desc())
            .select(ArticleVersionRow::as_select())
            .load::<ArticleVersionRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(ArticleVersion::from).collect())
            .map_err(map_diesel_error)
    }

    async fn get_version(&self, version_id: Uuid) -> Result<ArticleVersion, AppError> {
        let mut connection = self.connection().await?;
        article_version::table
            .find(version_id)
            .select(ArticleVersionRow::as_select())
            .first::<ArticleVersionRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(ArticleVersion::from)
            .ok_or(AppError::NotFound)
    }

    async fn revert_to_version(&self, article_id: Uuid, version_id: Uuid) -> Result<(), AppError> {
        let version = self.get_version(version_id).await?;
        if version.article_id != article_id {
            return Err(AppError::InvalidInput(
                "Version does not belong to this article".to_owned(),
            ));
        }

        let reservation = self.reserve_version()?;
        let now = Utc::now();
        let embedding = vector_or_none(&version.embedding);
        let mut connection = self.connection().await?;
        diesel::update(article::table.filter(article::id.eq(article_id)))
            .set((
                article::draft_title.eq(&version.title),
                article::draft_content.eq(&version.content),
                article::draft_image_url.eq(&version.image_url),
                article::draft_embedding.eq(embedding),
                article::updated_at.eq(now),
            ))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        reservation.spawn(
            self.pool.clone(),
            VersionInput {
                article_id,
                title: version.title,
                content: version.content,
                image_url: version.image_url,
                embedding: version.embedding,
                status: VersionStatus::Draft,
                edited_by: None,
            },
        )
    }

    async fn create_draft_snapshot(&self, article_id: Uuid) -> Result<Uuid, AppError> {
        let mut connection = self.connection().await?;
        connection
            .transaction::<Uuid, RepositoryError, _>(async |connection| {
                lock_article_versions(connection, article_id).await?;
                let value = article::table
                    .find(article_id)
                    .for_update()
                    .select(ArticleRow::as_select())
                    .first::<ArticleRow>(connection)
                    .await
                    .optional()?
                    .ok_or(AppError::Database)?;
                let version_id = insert_version_and_update_pointer(
                    connection,
                    VersionInput {
                        article_id,
                        title: value.draft_title.unwrap_or_default(),
                        content: value.draft_content.unwrap_or_default(),
                        image_url: value.draft_image_url.unwrap_or_default(),
                        embedding: value
                            .draft_embedding
                            .map_or_else(Vec::new, |vector| vector.to_vec()),
                        status: VersionStatus::Draft,
                        edited_by: Some(value.author_id),
                    },
                )
                .await?;
                Ok(version_id)
            })
            .await
            .map_err(RepositoryError::into_app_error)
    }

    async fn update_draft_content(
        &self,
        article_id: Uuid,
        html_content: &str,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(article::table.filter(article::id.eq(article_id)))
            .set((
                article::draft_content.eq(html_content),
                article::updated_at.eq(Utc::now()),
            ))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_diesel_error)
    }

    async fn drain_background_tasks(&self) -> Result<(), AppError> {
        let target = self.background_tasks.snapshot();
        self.background_tasks.wait_for(target).await;
        self.background_tasks.report_failure(target)
    }

    async fn shutdown_background_tasks(&self, timeout: Duration) -> Result<(), AppError> {
        self.background_tasks.shutdown(timeout).await
    }
}

#[derive(Clone, Copy)]
enum SortColumn {
    Published,
    Created,
    Updated,
}

#[derive(Debug, diesel::QueryableByName)]
struct PopularTagRow {
    #[diesel(sql_type = diesel::sql_types::Integer)]
    tag_id: i32,
    #[diesel(sql_type = diesel::sql_types::BigInt)]
    #[allow(dead_code)]
    count: i64,
}

#[derive(Debug)]
enum RepositoryError {
    Diesel(DieselError),
    Domain(AppError),
}

impl RepositoryError {
    fn into_app_error(self) -> AppError {
        match self {
            Self::Diesel(error) => map_diesel_error(error),
            Self::Domain(error) => error,
        }
    }
}

impl From<DieselError> for RepositoryError {
    fn from(value: DieselError) -> Self {
        Self::Diesel(value)
    }
}

impl From<AppError> for RepositoryError {
    fn from(value: AppError) -> Self {
        Self::Domain(value)
    }
}

#[derive(Clone, Copy)]
enum VersionStatus {
    Draft,
    Published,
}

impl VersionStatus {
    const fn as_str(self) -> &'static str {
        match self {
            Self::Draft => "draft",
            Self::Published => "published",
        }
    }
}

struct VersionInput {
    article_id: Uuid,
    title: String,
    content: String,
    image_url: String,
    embedding: Vec<f32>,
    status: VersionStatus,
    edited_by: Option<Uuid>,
}

impl VersionInput {
    fn draft(value: &Article, edited_by: Option<Uuid>) -> Self {
        Self {
            article_id: value.id,
            title: value.draft_title.clone(),
            content: value.draft_content.clone(),
            image_url: value.draft_image_url.clone(),
            embedding: value.draft_embedding.clone(),
            status: VersionStatus::Draft,
            edited_by,
        }
    }

    fn published(value: &Article, edited_by: Option<Uuid>) -> Self {
        Self {
            status: VersionStatus::Published,
            ..Self::draft(value, edited_by)
        }
    }
}

async fn create_version(pool: PgPool, input: VersionInput) -> Result<(), AppError> {
    let mut connection = pool.get().await.map_err(|_| AppError::Database)?;
    connection
        .transaction::<(), RepositoryError, _>(async |connection| {
            lock_article_versions(connection, input.article_id).await?;
            article::table
                .find(input.article_id)
                .select(article::id)
                .first::<Uuid>(connection)
                .await
                .optional()?
                .ok_or(AppError::NotFound)?;
            insert_version_and_update_pointer(connection, input).await?;
            Ok(())
        })
        .await
        .map_err(RepositoryError::into_app_error)
}

async fn lock_article_versions(
    connection: &mut diesel_async::AsyncPgConnection,
    article_id: Uuid,
) -> Result<(), DieselError> {
    diesel::sql_query("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")
        .bind::<diesel::sql_types::Text, _>(article_id.to_string())
        .execute(connection)
        .await
        .map(|_| ())
}

async fn insert_version_and_update_pointer(
    connection: &mut diesel_async::AsyncPgConnection,
    input: VersionInput,
) -> Result<Uuid, RepositoryError> {
    let max_version = article_version::table
        .filter(article_version::article_id.eq(input.article_id))
        .select(max(article_version::version_number))
        .first::<Option<i32>>(connection)
        .await?
        .unwrap_or(0);
    let version_number = max_version.checked_add(1).ok_or_else(|| {
        AppError::InvalidInput("article version number exceeds PostgreSQL INTEGER".to_owned())
    })?;
    let version_id = Uuid::new_v4();
    let status = input.status.as_str().to_owned();
    let row = NewArticleVersionRow {
        id: version_id,
        article_id: input.article_id,
        version_number,
        status,
        title: input.title,
        content: Some(input.content),
        image_url: Some(input.image_url),
        embedding: vector_or_none(&input.embedding),
        edited_by: input.edited_by,
        created_at: Utc::now(),
    };
    diesel::insert_into(article_version::table)
        .values(row)
        .execute(connection)
        .await?;

    match input.status {
        VersionStatus::Draft => {
            diesel::update(article::table.find(input.article_id))
                .set(article::current_draft_version_id.eq(Some(version_id)))
                .execute(connection)
                .await?;
        }
        VersionStatus::Published => {
            diesel::update(article::table.find(input.article_id))
                .set(article::current_published_version_id.eq(Some(version_id)))
                .execute(connection)
                .await?;
        }
    }
    Ok(version_id)
}

struct BackgroundTasks {
    state: Mutex<BackgroundState>,
    notify: Notify,
    cancellation: CancellationToken,
}

#[derive(Default)]
struct BackgroundState {
    closed: bool,
    last_admitted: u64,
    finished_through: u64,
    finished_out_of_order: BTreeSet<u64>,
    reservations: BTreeSet<u64>,
    handles: BTreeMap<u64, JoinHandle<()>>,
    failures: VecDeque<TaskFailure>,
    failure_overflow: Option<FailureOverflow>,
    rejected_admissions: usize,
}

struct TaskFailure {
    task_id: u64,
    error: AppError,
}

struct FailureOverflow {
    min_task_id: u64,
    max_task_id: u64,
    count: usize,
}

struct VersionReservation {
    task_id: u64,
    tasks: Arc<BackgroundTasks>,
    released: bool,
}

impl VersionReservation {
    fn spawn(mut self, pool: PgPool, input: VersionInput) -> Result<(), AppError> {
        let result = self.tasks.spawn_reserved(self.task_id, pool, input);
        self.released = true;
        result
    }
}

impl Drop for VersionReservation {
    fn drop(&mut self) {
        if !self.released {
            self.tasks.release(self.task_id);
            self.released = true;
        }
    }
}

impl BackgroundTasks {
    fn new() -> Self {
        Self {
            state: Mutex::new(BackgroundState::default()),
            notify: Notify::new(),
            cancellation: CancellationToken::new(),
        }
    }

    fn state(&self) -> MutexGuard<'_, BackgroundState> {
        match self.state.lock() {
            Ok(state) => state,
            Err(poisoned) => poisoned.into_inner(),
        }
    }

    fn reserve(tasks: &Arc<Self>) -> Result<VersionReservation, AppError> {
        let mut state = tasks.state();
        let admission_window = state.last_admitted.saturating_sub(state.finished_through);
        if state.closed
            || admission_window >= u64::try_from(MAX_BACKGROUND_TASKS).unwrap_or(u64::MAX)
        {
            state.rejected_admissions = state.rejected_admissions.saturating_add(1);
            return Err(AppError::Internal);
        }
        let Some(task_id) = state.last_admitted.checked_add(1) else {
            state.rejected_admissions = state.rejected_admissions.saturating_add(1);
            return Err(AppError::Internal);
        };
        state.last_admitted = task_id;
        state.reservations.insert(task_id);
        drop(state);
        Ok(VersionReservation {
            task_id,
            tasks: tasks.clone(),
            released: false,
        })
    }

    fn spawn_reserved(
        self: &Arc<Self>,
        task_id: u64,
        pool: PgPool,
        input: VersionInput,
    ) -> Result<(), AppError> {
        let mut state = self.state();
        if !state.reservations.remove(&task_id) {
            return Err(AppError::Internal);
        }
        if self.cancellation.is_cancelled() {
            drop(state);
            self.complete(task_id, Err(AppError::Internal));
            return Err(AppError::Internal);
        }

        let tasks = self.clone();
        let cancellation = self.cancellation.clone();
        let handle = tokio::spawn(async move {
            let result = AssertUnwindSafe(async move {
                tokio::select! {
                    result = create_version(pool, input) => result,
                    () = cancellation.cancelled() => Err(AppError::Internal),
                }
            })
            .catch_unwind()
            .await
            .unwrap_or(Err(AppError::Internal));
            tasks.complete(task_id, result);
        });
        state.handles.insert(task_id, handle);
        drop(state);
        self.notify.notify_waiters();
        Ok(())
    }

    fn release(&self, task_id: u64) {
        self.complete(task_id, Ok(()));
    }

    fn complete(&self, task_id: u64, result: Result<(), AppError>) {
        let mut state = self.state();
        let remove_handle = !state.closed;
        if !mark_finished(&mut state, task_id, remove_handle) {
            return;
        }
        if let Err(task_error) = result {
            error!(task_id, error = %task_error, "article version task failed");
            record_failure(&mut state, task_id, task_error);
        }
        drop(state);
        self.notify.notify_waiters();
    }

    fn snapshot(&self) -> u64 {
        self.state().last_admitted
    }

    fn close(&self) -> u64 {
        let mut state = self.state();
        state.closed = true;
        state.last_admitted
    }

    fn take_handles_through(&self, target: u64) -> OwnedTaskHandles {
        let mut state = self.state();
        let task_ids = state
            .handles
            .range(..=target)
            .map(|(task_id, _)| *task_id)
            .collect::<Vec<_>>();
        task_ids
            .into_iter()
            .filter_map(|task_id| {
                state
                    .handles
                    .remove(&task_id)
                    .map(|handle| (task_id, Some(handle)))
            })
            .collect()
    }

    async fn wait_for(&self, target: u64) {
        loop {
            let notified = self.notify.notified();
            if self.state().finished_through >= target {
                return;
            }
            notified.await;
        }
    }

    fn report_failure(&self, target: u64) -> Result<(), AppError> {
        let mut state = self.state();
        if state.rejected_admissions > 0 {
            state.rejected_admissions -= 1;
            return Err(AppError::Internal);
        }
        let Some(position) = state
            .failures
            .iter()
            .position(|failure| failure.task_id <= target)
        else {
            if state
                .failure_overflow
                .as_ref()
                .is_some_and(|overflow| overflow.min_task_id <= target)
            {
                let Some(overflow) = state.failure_overflow.take() else {
                    return Err(AppError::Internal);
                };
                error!(
                    failure_count = overflow.count,
                    min_task_id = overflow.min_task_id,
                    max_task_id = overflow.max_task_id.min(target),
                    "reporting aggregated article version task failures"
                );
                if overflow.max_task_id > target {
                    state.failure_overflow = Some(FailureOverflow {
                        min_task_id: target.saturating_add(1),
                        ..overflow
                    });
                }
                return Err(AppError::Internal);
            }
            return Ok(());
        };
        state
            .failures
            .remove(position)
            .map_or(Ok(()), |failure| Err(failure.error))
    }

    async fn shutdown(self: &Arc<Self>, timeout: Duration) -> Result<(), AppError> {
        let started_at = Instant::now();
        let full_deadline = started_at + timeout;
        let abort_reserve = timeout / 10;
        let graceful_deadline = started_at + timeout.saturating_sub(abort_reserve);
        let target = self.close();
        let mut handles = Vec::new();

        let graceful = timeout_at(
            graceful_deadline,
            self.settle_through(target, &mut handles, false),
        )
        .await;
        if graceful.is_ok() {
            return self.report_failure(target);
        }

        self.cancellation.cancel();
        self.cancel_reservations_through(target);
        let _ = timeout_at(
            full_deadline,
            self.settle_through(target, &mut handles, true),
        )
        .await;
        Err(AppError::Internal)
    }

    async fn settle_through(&self, target: u64, handles: &mut OwnedTaskHandles, abort: bool) {
        loop {
            handles.extend(self.take_handles_through(target));
            if abort {
                for (_, handle) in &mut *handles {
                    if let Some(handle) = handle {
                        handle.abort();
                    }
                }
            }
            self.await_handles(handles).await;

            let notified = self.notify.notified();
            handles.extend(self.take_handles_through(target));
            if handles.iter().any(|(_, handle)| handle.is_some()) {
                continue;
            }
            if self.state().finished_through >= target {
                return;
            }
            notified.await;
        }
    }

    fn cancel_reservations_through(&self, target: u64) {
        let reservation_ids = {
            let state = self.state();
            state
                .reservations
                .range(..=target)
                .copied()
                .collect::<Vec<_>>()
        };
        for task_id in reservation_ids {
            self.complete(task_id, Err(AppError::Internal));
        }
    }

    async fn await_handles(&self, handles: &mut [(u64, Option<JoinHandle<()>>)]) {
        for (task_id, handle) in handles {
            let Some(join_handle) = handle.as_mut() else {
                continue;
            };
            let join_result = join_handle.await;
            *handle = None;
            if join_result.is_err() {
                self.complete(*task_id, Err(AppError::Internal));
            } else {
                self.complete(*task_id, Ok(()));
            }
        }
    }
}

fn mark_finished(state: &mut BackgroundState, task_id: u64, remove_handle: bool) -> bool {
    if state.finished_through >= task_id || state.finished_out_of_order.contains(&task_id) {
        return false;
    }
    state.reservations.remove(&task_id);
    if remove_handle {
        state.handles.remove(&task_id);
    }
    state.finished_out_of_order.insert(task_id);
    while state
        .finished_out_of_order
        .remove(&state.finished_through.saturating_add(1))
    {
        state.finished_through = state.finished_through.saturating_add(1);
    }
    true
}

fn record_failure(state: &mut BackgroundState, task_id: u64, error: AppError) {
    if state.failures.len() < MAX_RECORDED_FAILURES {
        state.failures.push_back(TaskFailure { task_id, error });
    } else {
        match &mut state.failure_overflow {
            Some(overflow) => {
                overflow.min_task_id = overflow.min_task_id.min(task_id);
                overflow.max_task_id = overflow.max_task_id.max(task_id);
                overflow.count = overflow.count.saturating_add(1);
            }
            None => {
                state.failure_overflow = Some(FailureOverflow {
                    min_task_id: task_id,
                    max_task_id: task_id,
                    count: 1,
                });
            }
        }
    }
}

fn pagination_offset(page: i64, per_page: i64) -> Result<i64, AppError> {
    page.checked_sub(1)
        .and_then(|page_index| page_index.checked_mul(per_page))
        .map(|offset| offset.max(0))
        .ok_or_else(|| AppError::InvalidInput("pagination offset is too large".to_owned()))
}

fn tag_id_to_i32(tag_id: i64) -> Result<i32, AppError> {
    i32::try_from(tag_id).map_err(|_| {
        AppError::InvalidInput(format!("tag id {tag_id} does not fit PostgreSQL INTEGER"))
    })
}

fn tag_ids_to_database(tag_ids: Option<&[i64]>) -> Result<Option<Vec<Option<i32>>>, AppError> {
    tag_ids
        .map(|tag_ids| {
            tag_ids
                .iter()
                .copied()
                .map(tag_id_to_i32)
                .map(|value| value.map(Some))
                .collect()
        })
        .transpose()
}

fn tag_ids_from_database(tag_ids: Option<Vec<Option<i32>>>) -> Result<Option<Vec<i64>>, AppError> {
    tag_ids
        .map(|tag_ids| {
            tag_ids
                .into_iter()
                .map(|tag_id| tag_id.map(i64::from).ok_or(AppError::Database))
                .collect()
        })
        .transpose()
}

fn vector_or_none(embedding: &[f32]) -> Option<Vector> {
    (!embedding.is_empty()).then(|| Vector::from(embedding.to_vec()))
}

fn session_memory_to_database(memory: &Option<BTreeMap<String, Value>>) -> Option<Value> {
    memory.as_ref().map(|memory| {
        Value::Object(
            memory
                .iter()
                .map(|(key, value)| (key.clone(), value.clone()))
                .collect::<Map<String, Value>>(),
        )
    })
}

fn session_memory_from_database(value: Option<Value>) -> Option<BTreeMap<String, Value>> {
    match value {
        Some(Value::Object(memory)) => Some(memory.into_iter().collect()),
        _ => None,
    }
}

fn new_article_row(
    value: &Article,
    tag_ids: Option<Vec<Option<i32>>>,
    now: DateTime<Utc>,
) -> NewArticleRow {
    NewArticleRow {
        id: value.id,
        slug: value.slug.clone(),
        author_id: value.author_id,
        tag_ids: Some(tag_ids),
        created_at: value.created_at.unwrap_or(now),
        updated_at: now,
        published_at: value.published_at,
        imagen_request_id: value.imagen_request_id,
        session_memory: session_memory_to_database(&value.session_memory)
            .unwrap_or_else(|| Value::Object(Map::new())),
        draft_title: value.draft_title.clone(),
        draft_content: value.draft_content.clone(),
        draft_image_url: value.draft_image_url.clone(),
        draft_embedding: vector_or_none(&value.draft_embedding),
        published_title: value.published_title.clone(),
        published_content: value.published_content.clone(),
        published_image_url: value.published_image_url.clone(),
        published_embedding: vector_or_none(&value.published_embedding),
        current_draft_version_id: value.current_draft_version_id,
        current_published_version_id: value.current_published_version_id,
    }
}

fn article_changeset(
    value: &Article,
    tag_ids: Option<Vec<Option<i32>>>,
    now: DateTime<Utc>,
) -> ArticleChangeset {
    ArticleChangeset {
        slug: value.slug.clone(),
        author_id: value.author_id,
        tag_ids: Some(tag_ids),
        created_at: value.created_at,
        updated_at: now,
        published_at: Some(value.published_at),
        imagen_request_id: Some(value.imagen_request_id),
        session_memory: Some(session_memory_to_database(&value.session_memory)),
        draft_title: value.draft_title.clone(),
        draft_content: value.draft_content.clone(),
        draft_image_url: value.draft_image_url.clone(),
        draft_embedding: (!value.draft_embedding.is_empty())
            .then(|| vector_or_none(&value.draft_embedding)),
        published_title: Some(value.published_title.clone()),
        published_content: Some(value.published_content.clone()),
        published_image_url: Some(value.published_image_url.clone()),
        published_embedding: (!value.published_embedding.is_empty())
            .then(|| vector_or_none(&value.published_embedding)),
        current_draft_version_id: Some(value.current_draft_version_id),
        current_published_version_id: Some(value.current_published_version_id),
    }
}

fn articles_from_rows(rows: Vec<ArticleRow>) -> Result<Vec<Article>, AppError> {
    rows.into_iter().map(Article::try_from).collect()
}

impl TryFrom<ArticleRow> for Article {
    type Error = AppError;

    fn try_from(row: ArticleRow) -> Result<Self, Self::Error> {
        Ok(Self {
            id: row.id,
            slug: row.slug,
            author_id: row.author_id,
            tag_ids: tag_ids_from_database(row.tag_ids)?,
            draft_title: row.draft_title.unwrap_or_default(),
            draft_content: row.draft_content.unwrap_or_default(),
            draft_image_url: row.draft_image_url.unwrap_or_default(),
            draft_embedding: row
                .draft_embedding
                .map_or_else(Vec::new, |vector| vector.to_vec()),
            published_title: row.published_title,
            published_content: row.published_content,
            published_image_url: row.published_image_url,
            published_embedding: row
                .published_embedding
                .map_or_else(Vec::new, |vector| vector.to_vec()),
            published_at: row.published_at,
            current_draft_version_id: row.current_draft_version_id,
            current_published_version_id: row.current_published_version_id,
            imagen_request_id: row.imagen_request_id,
            session_memory: session_memory_from_database(row.session_memory),
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}

impl From<ArticleVersionRow> for ArticleVersion {
    fn from(row: ArticleVersionRow) -> Self {
        Self {
            id: row.id,
            article_id: row.article_id,
            version_number: row.version_number,
            status: row.status,
            title: row.title,
            content: row.content.unwrap_or_default(),
            image_url: row.image_url.unwrap_or_default(),
            embedding: row
                .embedding
                .map_or_else(Vec::new, |vector| vector.to_vec()),
            edited_by: row.edited_by,
            created_at: row.created_at,
        }
    }
}

fn map_diesel_error(error: DieselError) -> AppError {
    match error {
        DieselError::NotFound => AppError::NotFound,
        DieselError::DatabaseError(DatabaseErrorKind::UniqueViolation, _) => {
            AppError::Conflict("resource already exists".to_owned())
        }
        _ => AppError::Database,
    }
}

#[cfg(test)]
mod background_task_tests {
    use super::*;

    #[tokio::test]
    async fn admission_window_and_failure_storage_stay_bounded() -> Result<(), AppError> {
        let tasks = Arc::new(BackgroundTasks::new());
        let mut reservations = Vec::with_capacity(MAX_BACKGROUND_TASKS);
        for _ in 0..MAX_BACKGROUND_TASKS {
            reservations.push(BackgroundTasks::reserve(&tasks)?);
        }

        let stuck_first = reservations.remove(0);
        drop(reservations);
        assert!(matches!(
            BackgroundTasks::reserve(&tasks),
            Err(AppError::Internal)
        ));
        {
            let state = tasks.state();
            assert_eq!(state.finished_out_of_order.len(), MAX_BACKGROUND_TASKS - 1);
            assert!(state.finished_out_of_order.len() < MAX_BACKGROUND_TASKS);
        }

        drop(stuck_first);
        let released = BackgroundTasks::reserve(&tasks)?;
        drop(released);
        let target = tasks.snapshot();
        tasks.wait_for(target).await;
        assert!(matches!(
            tasks.report_failure(target),
            Err(AppError::Internal)
        ));
        tasks.report_failure(target)?;

        for _ in 0..(MAX_RECORDED_FAILURES + 5) {
            let reservation = BackgroundTasks::reserve(&tasks)?;
            let task_id = reservation.task_id;
            tasks.complete(task_id, Err(AppError::Internal));
            drop(reservation);
        }
        let (overflow_min, overflow_max) = {
            let mut state = tasks.state();
            assert_eq!(state.failures.len(), MAX_RECORDED_FAILURES);
            let overflow = state.failure_overflow.as_ref().ok_or(AppError::Internal)?;
            assert_eq!(overflow.count, 5);
            let bounds = (overflow.min_task_id, overflow.max_task_id);
            state.failures.clear();
            bounds
        };
        assert!(overflow_min < overflow_max);
        assert!(matches!(
            tasks.report_failure(overflow_max.saturating_sub(1)),
            Err(AppError::Internal)
        ));
        {
            let state = tasks.state();
            let remainder = state.failure_overflow.as_ref().ok_or(AppError::Internal)?;
            assert_eq!(remainder.min_task_id, overflow_max);
            assert_eq!(remainder.max_task_id, overflow_max);
            assert_eq!(remainder.count, 5);
        }
        assert!(matches!(
            tasks.report_failure(overflow_max),
            Err(AppError::Internal)
        ));
        tasks.report_failure(overflow_max)?;
        Ok(())
    }

    #[tokio::test]
    async fn shutdown_reaps_a_handle_registered_after_close() -> Result<(), AppError> {
        use crate::database::pool::create_pool;
        use secrecy::SecretString;

        let tasks = Arc::new(BackgroundTasks::new());
        let reservation = BackgroundTasks::reserve(&tasks)?;
        let shutdown_tasks = tasks.clone();
        let shutdown =
            tokio::spawn(async move { shutdown_tasks.shutdown(Duration::from_secs(1)).await });

        loop {
            if tasks.state().closed {
                break;
            }
            tokio::task::yield_now().await;
        }

        let pool = create_pool(&SecretString::from(
            "postgres://invalid:invalid@127.0.0.1:1/invalid",
        ))
        .map_err(|_| AppError::Internal)?;
        reservation.spawn(
            pool,
            VersionInput {
                article_id: Uuid::new_v4(),
                title: "late".to_owned(),
                content: String::new(),
                image_url: String::new(),
                embedding: Vec::new(),
                status: VersionStatus::Draft,
                edited_by: None,
            },
        )?;

        let shutdown_result = shutdown.await.map_err(|_| AppError::Internal)?;
        assert!(matches!(shutdown_result, Err(AppError::Database)));
        let state = tasks.state();
        assert!(state.handles.is_empty());
        assert!(state.reservations.is_empty());
        assert_eq!(state.finished_through, state.last_admitted);
        Ok(())
    }
}
