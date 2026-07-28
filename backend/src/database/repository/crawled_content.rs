use async_trait::async_trait;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper, dsl::count_star,
    result::Error as DieselError,
};
use diesel_async::RunQueryDsl;
use pgvector::{Vector, VectorExpressionMethods};
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::{
        datasource::{CrawledContent, CrawledContentRepository, MetaData},
        insight::InsightContentRepository,
    },
    database::{
        models::crawled_content::{
            CrawledContentChangeset, CrawledContentRow, NewCrawledContentRow,
        },
        pool::PgPool,
    },
    error::AppError,
    schema::{crawled_content, data_source},
};

#[derive(Clone)]
pub struct DieselCrawledContentRepository {
    pool: PgPool,
}

impl DieselCrawledContentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn connection(
        &self,
    ) -> Result<
        diesel_async::pooled_connection::deadpool::Object<diesel_async::AsyncPgConnection>,
        AppError,
    > {
        self.pool.get().await.map_err(|_| AppError::Database)
    }

    async fn find_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        let mut connection = self.connection().await?;
        crawled_content::table
            .filter(crawled_content::id.eq_any(ids))
            .select(CrawledContentRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(map_error)
    }

    async fn similar(
        &self,
        organization_id: Option<Uuid>,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        let mut connection = self.connection().await?;
        let vector = Vector::from(embedding.to_vec());
        let mut query = crawled_content::table
            .inner_join(data_source::table)
            .filter(crawled_content::embedding.is_not_null())
            .into_boxed();
        if let Some(id) = organization_id {
            query = query.filter(data_source::organization_id.eq(id));
        }
        if limit >= 0 {
            query = query.limit(limit);
        }
        query
            .order(crawled_content::embedding.cosine_distance(vector))
            .select(CrawledContentRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(map_error)
    }

    async fn recent(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        let mut connection = self.connection().await?;
        let mut query = crawled_content::table
            .inner_join(data_source::table)
            .filter(data_source::organization_id.eq(organization_id))
            .order(crawled_content::created_at.desc())
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        query
            .select(CrawledContentRow::as_select())
            .load(&mut connection)
            .await
            .map(rows_into_domain)
            .map_err(map_error)
    }
}

#[async_trait]
impl CrawledContentRepository for DieselCrawledContentRepository {
    async fn find_by_data_source_id(
        &self,
        id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<CrawledContent>, i64), AppError> {
        let mut connection = self.connection().await?;
        let total = crawled_content::table
            .filter(crawled_content::data_source_id.eq(id))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(map_error)?;
        let mut query = crawled_content::table
            .filter(crawled_content::data_source_id.eq(id))
            .order(crawled_content::created_at.desc())
            .offset(offset)
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        let rows = query
            .select(CrawledContentRow::as_select())
            .load(&mut connection)
            .await
            .map_err(map_error)?;
        Ok((rows_into_domain(rows), total))
    }

    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        self.find_ids(ids).await
    }

    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.similar(None, embedding, limit).await
    }

    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.similar(Some(organization_id), embedding, limit).await
    }

    async fn find_recent_by_org(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.recent(organization_id, limit).await
    }

    async fn find_by_id(&self, id: Uuid) -> Result<CrawledContent, AppError> {
        let mut connection = self.connection().await?;
        crawled_content::table
            .find(id)
            .select(CrawledContentRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(map_error)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn find_by_url(
        &self,
        data_source_id: Uuid,
        url: &str,
    ) -> Result<Option<CrawledContent>, AppError> {
        let mut connection = self.connection().await?;
        crawled_content::table
            .filter(crawled_content::data_source_id.eq(data_source_id))
            .filter(crawled_content::url.eq(url))
            .select(CrawledContentRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map(|row| row.map(Into::into))
            .map_err(map_error)
    }

    async fn save(&self, content: &mut CrawledContent) -> Result<(), AppError> {
        if content.id.is_nil() {
            content.id = Uuid::new_v4();
        }
        let now = chrono::Utc::now();
        let row = new_row(content, now);
        let changes = changeset(content);
        let mut connection = self.connection().await?;
        diesel::insert_into(crawled_content::table)
            .values(row)
            .on_conflict((crawled_content::data_source_id, crawled_content::url))
            .do_update()
            .set(changes)
            .returning(crawled_content::id)
            .get_result::<Uuid>(&mut connection)
            .await
            .map(|persisted_id| {
                content.id = persisted_id;
            })
            .map_err(map_error)
    }

    async fn update(&self, content: &CrawledContent) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::update(crawled_content::table.find(content.id))
            .set(changeset(content))
            .execute(&mut connection)
            .await
            .map_err(map_error)?;
        if affected == 0 {
            diesel::insert_into(crawled_content::table)
                .values(new_row(content, chrono::Utc::now()))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        }
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(crawled_content::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_error)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }

    async fn delete_by_data_source_id(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(crawled_content::table.filter(crawled_content::data_source_id.eq(id)))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn count_by_data_source_id(&self, id: Uuid) -> Result<i64, AppError> {
        let mut connection = self.connection().await?;
        crawled_content::table
            .filter(crawled_content::data_source_id.eq(id))
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(map_error)
    }
}

#[async_trait]
impl InsightContentRepository for DieselCrawledContentRepository {
    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        self.find_ids(ids).await
    }

    async fn search_similar(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.similar(None, embedding, limit).await
    }

    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.similar(Some(organization_id), embedding, limit).await
    }

    async fn find_recent_by_org(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        self.recent(organization_id, limit).await
    }
}

impl From<CrawledContentRow> for CrawledContent {
    fn from(row: CrawledContentRow) -> Self {
        Self {
            id: row.id,
            data_source_id: row.data_source_id,
            url: row.url,
            title: row.title,
            content: row.content,
            summary: row.summary,
            author: row.author,
            published_at: row.published_at,
            embedding: row.embedding.map(|value| value.to_vec()),
            meta_data: metadata(row.meta_data),
            created_at: row.created_at,
        }
    }
}

fn new_row(content: &CrawledContent, now: chrono::DateTime<chrono::Utc>) -> NewCrawledContentRow {
    NewCrawledContentRow {
        id: content.id,
        data_source_id: content.data_source_id,
        url: content.url.clone(),
        title: content.title.clone(),
        content: content.content.clone(),
        summary: content.summary.clone(),
        author: content.author.clone(),
        published_at: content.published_at,
        embedding: content.embedding.clone().map(Vector::from),
        meta_data: metadata_value(content.meta_data.as_ref()),
        created_at: content.created_at.unwrap_or(now),
    }
}

fn changeset(content: &CrawledContent) -> CrawledContentChangeset {
    CrawledContentChangeset {
        data_source_id: content.data_source_id,
        url: content.url.clone(),
        title: content.title.clone(),
        content: content.content.clone(),
        summary: content.summary.clone(),
        author: content.author.clone(),
        published_at: content.published_at,
        embedding: content.embedding.clone().map(Vector::from),
        meta_data: content
            .meta_data
            .as_ref()
            .map(|value| metadata_value(Some(value))),
    }
}

fn rows_into_domain(rows: Vec<CrawledContentRow>) -> Vec<CrawledContent> {
    rows.into_iter().map(Into::into).collect()
}

fn metadata(value: Option<Value>) -> Option<MetaData> {
    value.and_then(|value| match value {
        Value::Object(map) => Some(map.into_iter().collect()),
        _ => None,
    })
}

fn metadata_value(value: Option<&MetaData>) -> Value {
    Value::Object(
        value
            .map(|map| map.clone().into_iter().collect())
            .unwrap_or_default(),
    )
}

fn map_error(_: DieselError) -> AppError {
    AppError::Database
}
