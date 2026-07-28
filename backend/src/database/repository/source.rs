use async_trait::async_trait;
use diesel::{ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper, dsl::count_star};
use diesel_async::RunQueryDsl;
use pgvector::{Vector, VectorExpressionMethods};
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::{
        article::ArticleRepository,
        source::{
            ArticleLookupPort, MetaData, Source, SourceListOptions, SourceRepository,
            SourceWithArticle,
        },
    },
    database::{
        models::source::{NewSourceRow, SourceChangeset, SourceRow},
        pool::PgPool,
        repository::article::DieselArticleRepository,
    },
    error::AppError,
    schema::{article, article_source},
};

#[derive(Clone)]
pub struct DieselSourceRepository {
    pool: PgPool,
}

#[async_trait]
impl ArticleLookupPort for DieselArticleRepository {
    async fn ensure_exists(&self, article_id: Uuid) -> Result<(), AppError> {
        ArticleRepository::find_by_id(self, article_id)
            .await
            .map(|_| ())
    }
}

impl DieselSourceRepository {
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
impl SourceRepository for DieselSourceRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Source, AppError> {
        let mut connection = self.connection().await?;
        article_source::table
            .find(id)
            .select(SourceRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn find_by_article_id(&self, article_id: Uuid) -> Result<Vec<Source>, AppError> {
        let mut connection = self.connection().await?;
        article_source::table
            .filter(article_source::article_id.eq(article_id))
            .order(article_source::created_at.desc())
            .select(SourceRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<SourceRow>| rows.into_iter().map(Into::into).collect())
            .map_err(|_| AppError::Database)
    }

    async fn list(
        &self,
        options: SourceListOptions,
    ) -> Result<(Vec<SourceWithArticle>, i64), AppError> {
        let page = options.page.max(1);
        let per_page = if !(1..=100).contains(&options.per_page) {
            20
        } else {
            options.per_page
        };
        let mut connection = self.connection().await?;
        let total = article_source::table
            .select(count_star())
            .first(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        let rows = article_source::table
            .order(article_source::created_at.desc())
            .offset((page - 1) * per_page)
            .limit(per_page)
            .select(SourceRow::as_select())
            .load::<SourceRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        let mut result = Vec::with_capacity(rows.len());
        for row in rows {
            let article_data = article::table
                .find(row.article_id)
                .select((article::draft_title, article::slug))
                .first::<(Option<String>, String)>(&mut connection)
                .await
                .optional()
                .map_err(|_| AppError::Database)?;
            let (title, slug) = article_data.unwrap_or_default();
            result.push(SourceWithArticle {
                source: row.into(),
                article_title: title.unwrap_or_default(),
                article_slug: slug,
            });
        }
        Ok((result, total))
    }

    async fn save(&self, source: &mut Source) -> Result<(), AppError> {
        if source.id.is_nil() {
            source.id = Uuid::new_v4();
        }
        let mut connection = self.connection().await?;
        diesel::insert_into(article_source::table)
            .values(new_row(source))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn update(&self, source: &Source) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::update(article_source::table.find(source.id))
            .set(changeset(source))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        if affected == 0 {
            diesel::insert_into(article_source::table)
                .values(new_row(source))
                .execute(&mut connection)
                .await
                .map_err(|_| AppError::Database)?;
        }
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        match diesel::delete(article_source::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?
        {
            0 => Err(AppError::NotFound),
            _ => Ok(()),
        }
    }

    async fn search_similar(
        &self,
        article_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Source>, AppError> {
        let mut connection = self.connection().await?;
        let mut query = article_source::table
            .filter(article_source::article_id.eq(article_id))
            .filter(article_source::embedding.is_not_null())
            .order(article_source::embedding.l2_distance(Vector::from(embedding.to_vec())))
            .into_boxed();
        if limit >= 0 {
            query = query.limit(limit);
        }
        query
            .select(SourceRow::as_select())
            .load(&mut connection)
            .await
            .map(|rows: Vec<SourceRow>| rows.into_iter().map(Into::into).collect())
            .map_err(|_| AppError::Database)
    }
}

impl From<SourceRow> for Source {
    fn from(row: SourceRow) -> Self {
        Self {
            id: row.id,
            article_id: row.article_id,
            title: row.title.unwrap_or_default(),
            content: row.content,
            url: row.url.unwrap_or_default(),
            source_type: row.source_type.unwrap_or_default(),
            embedding: row.embedding.map(|value| value.to_vec()),
            meta_data: row.meta_data.and_then(metadata),
            created_at: row.created_at,
        }
    }
}

fn new_row(source: &Source) -> NewSourceRow {
    NewSourceRow {
        id: source.id,
        article_id: source.article_id,
        title: Some(source.title.clone()),
        content: source.content.clone(),
        url: Some(source.url.clone()),
        source_type: Some(source.source_type.clone()),
        embedding: source.embedding.clone().map(Vector::from),
        meta_data: metadata_value(source.meta_data.as_ref()),
        created_at: source.created_at.unwrap_or_else(chrono::Utc::now),
    }
}

fn changeset(source: &Source) -> SourceChangeset {
    SourceChangeset {
        article_id: source.article_id,
        title: Some(source.title.clone()),
        content: source.content.clone(),
        url: Some(source.url.clone()),
        source_type: Some(source.source_type.clone()),
        embedding: source.embedding.clone().map(Vector::from),
        meta_data: source
            .meta_data
            .as_ref()
            .map(|value| metadata_value(Some(value))),
    }
}

fn metadata(value: Value) -> Option<MetaData> {
    match value {
        Value::Object(map) => Some(map.into_iter().collect()),
        _ => None,
    }
}

fn metadata_value(value: Option<&MetaData>) -> Value {
    Value::Object(
        value
            .map(|map| map.clone().into_iter().collect())
            .unwrap_or_default(),
    )
}
