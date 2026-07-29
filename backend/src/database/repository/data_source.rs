use async_trait::async_trait;
use chrono::{DateTime, Utc};
use diesel::{
    BoolExpressionMethods, ExpressionMethods, OptionalExtension, PgSortExpressionMethods, QueryDsl,
    SelectableHelper, dsl::count_star, result::Error as DieselError,
};
use diesel_async::RunQueryDsl;
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::datasource::{DataSource, DataSourceRepository, MetaData},
    database::{
        models::data_source::{DataSourceChangeset, DataSourceRow, NewDataSourceRow},
        pool::PgPool,
    },
    error::AppError,
    schema::data_source,
};

#[derive(Clone)]
pub struct DieselDataSourceRepository {
    pool: PgPool,
}

impl DieselDataSourceRepository {
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
}

#[async_trait]
impl DataSourceRepository for DieselDataSourceRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<DataSource, AppError> {
        let mut connection = self.connection().await?;
        data_source::table
            .find(id)
            .select(DataSourceRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(map_error)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn find_by_organization_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        let mut connection = self.connection().await?;
        data_source::table
            .filter(data_source::organization_id.eq(id))
            .order(data_source::created_at.desc())
            .select(DataSourceRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<DataSourceRow>| rows.into_iter().map(Into::into).collect())
            .map_err(map_error)
    }

    async fn find_by_user_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        let mut connection = self.connection().await?;
        data_source::table
            .filter(data_source::user_id.eq(id))
            .order(data_source::created_at.desc())
            .select(DataSourceRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<DataSourceRow>| rows.into_iter().map(Into::into).collect())
            .map_err(map_error)
    }

    async fn find_by_url(&self, url: &str) -> Result<Option<DataSource>, AppError> {
        let mut connection = self.connection().await?;
        data_source::table
            .filter(data_source::url.eq(url))
            .select(DataSourceRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map(|row| row.map(Into::into))
            .map_err(map_error)
    }

    async fn find_due_to_crawl(&self, limit: i64) -> Result<Vec<DataSource>, AppError> {
        let mut connection = self.connection().await?;
        let mut query = data_source::table
            .filter(data_source::is_enabled.eq(true))
            .filter(
                data_source::next_crawl_at
                    .is_null()
                    .or(data_source::next_crawl_at.le(Utc::now())),
            )
            .filter(data_source::crawl_status.ne("crawling"))
            .order(data_source::next_crawl_at.asc().nulls_first())
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        query
            .select(DataSourceRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<DataSourceRow>| rows.into_iter().map(Into::into).collect())
            .map_err(map_error)
    }

    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<DataSource>, i64), AppError> {
        let mut connection = self.connection().await?;
        let total = data_source::table
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(map_error)?;
        let mut query = data_source::table
            .order(data_source::created_at.desc())
            .offset(offset)
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        let rows = query
            .select(DataSourceRow::as_select())
            .load(&mut connection)
            .await
            .map_err(map_error)?;
        Ok((rows.into_iter().map(DataSource::from).collect(), total))
    }

    async fn save(&self, source: &mut DataSource) -> Result<(), AppError> {
        if source.id.is_nil() {
            source.id = Uuid::new_v4();
        }
        let mut connection = self.connection().await?;
        diesel::insert_into(data_source::table)
            .values(new_row(source))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn update(&self, source: &DataSource) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::update(data_source::table.find(source.id))
            .set(changeset(source))
            .execute(&mut connection)
            .await
            .map_err(map_error)?;
        if affected == 0 {
            diesel::insert_into(data_source::table)
                .values(new_row(source))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        }
        Ok(())
    }

    async fn update_crawl_status(
        &self,
        id: Uuid,
        status: &str,
        error_message: Option<&str>,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let now = Utc::now();
        if status == "success" {
            diesel::update(data_source::table.find(id))
                .set((
                    data_source::crawl_status.eq(status),
                    data_source::updated_at.eq(now),
                    data_source::last_crawled_at.eq(Some(now)),
                    data_source::error_message.eq(None::<String>),
                ))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        } else if let Some(message) = error_message {
            diesel::update(data_source::table.find(id))
                .set((
                    data_source::crawl_status.eq(status),
                    data_source::updated_at.eq(now),
                    data_source::error_message.eq(message),
                ))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        } else {
            diesel::update(data_source::table.find(id))
                .set((
                    data_source::crawl_status.eq(status),
                    data_source::updated_at.eq(now),
                ))
                .execute(&mut connection)
                .await
                .map_err(map_error)?;
        }
        Ok(())
    }

    async fn update_next_crawl_at(
        &self,
        id: Uuid,
        next_crawl_at: DateTime<Utc>,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(data_source::table.find(id))
            .set(data_source::next_crawl_at.eq(next_crawl_at))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn increment_content_count(&self, id: Uuid, delta: i32) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::update(data_source::table.find(id))
            .set(data_source::content_count.eq(data_source::content_count + delta))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_error)
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(data_source::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_error)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }
}

impl From<DataSourceRow> for DataSource {
    fn from(row: DataSourceRow) -> Self {
        Self {
            id: row.id,
            organization_id: row.organization_id,
            user_id: row.user_id,
            name: row.name,
            url: row.url,
            feed_url: row.feed_url,
            source_type: row.source_type.unwrap_or_default(),
            crawl_frequency: row.crawl_frequency.unwrap_or_default(),
            is_enabled: row.is_enabled.unwrap_or_default(),
            is_discovered: row.is_discovered.unwrap_or_default(),
            discovered_from_id: row.discovered_from_id,
            last_crawled_at: row.last_crawled_at,
            next_crawl_at: row.next_crawl_at,
            crawl_status: row.crawl_status.unwrap_or_default(),
            error_message: row.error_message,
            content_count: row.content_count.unwrap_or_default(),
            subscriber_count: row.subscriber_count.unwrap_or_default(),
            meta_data: metadata(row.meta_data),
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

fn new_row(source: &DataSource) -> NewDataSourceRow {
    let now = Utc::now();
    NewDataSourceRow {
        id: source.id,
        organization_id: source.organization_id,
        name: source.name.clone(),
        url: source.url.clone(),
        feed_url: source.feed_url.clone(),
        source_type: source.source_type.clone(),
        crawl_frequency: source.crawl_frequency.clone(),
        is_enabled: source.is_enabled,
        is_discovered: source.is_discovered,
        discovered_from_id: source.discovered_from_id,
        last_crawled_at: source.last_crawled_at,
        next_crawl_at: source.next_crawl_at,
        crawl_status: source.crawl_status.clone(),
        error_message: source.error_message.clone(),
        content_count: source.content_count,
        meta_data: metadata_value(source.meta_data.as_ref()),
        created_at: source.created_at.unwrap_or(now),
        updated_at: source.updated_at.unwrap_or(now),
        user_id: source.user_id,
        subscriber_count: source.subscriber_count,
    }
}

fn changeset(source: &DataSource) -> DataSourceChangeset {
    DataSourceChangeset {
        organization_id: source.organization_id,
        name: source.name.clone(),
        url: source.url.clone(),
        feed_url: source.feed_url.clone(),
        source_type: Some(source.source_type.clone()),
        crawl_frequency: Some(source.crawl_frequency.clone()),
        is_enabled: Some(source.is_enabled),
        is_discovered: Some(source.is_discovered),
        discovered_from_id: source.discovered_from_id,
        last_crawled_at: source.last_crawled_at,
        next_crawl_at: source.next_crawl_at,
        crawl_status: Some(source.crawl_status.clone()),
        error_message: source.error_message.clone(),
        content_count: Some(source.content_count),
        meta_data: source
            .meta_data
            .as_ref()
            .map(|value| metadata_value(Some(value))),
        updated_at: Some(Utc::now()),
        user_id: source.user_id,
        subscriber_count: Some(source.subscriber_count),
    }
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
