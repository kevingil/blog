use async_trait::async_trait;
use chrono::Utc;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper,
    dsl::count_star,
    result::{DatabaseErrorKind, Error as DieselError},
};
use diesel_async::RunQueryDsl;
use serde_json::{Map, Value};
use uuid::Uuid;

use crate::{
    core::page::{Page, PageListOptions, PageRepository},
    database::{
        models::page::{NewPageRow, PageRow},
        pool::PgPool,
    },
    error::AppError,
    schema::page,
};

#[derive(Clone)]
pub struct DieselPageRepository {
    pool: PgPool,
}

impl DieselPageRepository {
    pub const fn new(pool: PgPool) -> Self {
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
impl PageRepository for DieselPageRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Page, AppError> {
        let mut connection = self.connection().await?;
        page::table
            .find(id)
            .select(PageRow::as_select())
            .first::<PageRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Page, AppError> {
        let mut connection = self.connection().await?;
        page::table
            .filter(page::slug.eq(slug))
            .select(PageRow::as_select())
            .first::<PageRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn list(&self, options: PageListOptions) -> Result<(Vec<Page>, i64), AppError> {
        let offset = pagination_offset(options)?;
        let mut connection = self.connection().await?;

        let mut count_query = page::table.into_boxed();
        if let Some(is_published) = options.is_published {
            count_query = count_query.filter(page::is_published.eq(is_published));
        }
        let total = count_query
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map_err(map_diesel_error)?;

        let mut query = page::table.into_boxed();
        if let Some(is_published) = options.is_published {
            query = query.filter(page::is_published.eq(is_published));
        }
        let rows = query
            .order(page::created_at.desc())
            .offset(offset)
            .limit(options.per_page)
            .select(PageRow::as_select())
            .load::<PageRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        let pages = rows
            .into_iter()
            .map(Page::try_from)
            .collect::<Result<Vec<_>, _>>()?;
        Ok((pages, total))
    }

    async fn save(&self, value: &mut Page) -> Result<(), AppError> {
        if value.id.is_nil() {
            value.id = Uuid::new_v4();
        }
        let meta_data = meta_data_to_value(value);
        let mut connection = self.connection().await?;
        let exists = page::table
            .find(value.id)
            .select(page::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .is_some();

        if exists {
            let row = diesel::update(page::table.find(value.id))
                .set((
                    page::slug.eq(&value.slug),
                    page::title.eq(&value.title),
                    page::content.eq(&value.content),
                    page::description.eq(&value.description),
                    page::image_url.eq(&value.image_url),
                    page::meta_data.eq(meta_data),
                    page::is_published.eq(value.is_published),
                    page::updated_at.eq(Utc::now()),
                ))
                .returning(PageRow::as_returning())
                .get_result::<PageRow>(&mut connection)
                .await
                .map_err(map_diesel_error)?;
            *value = row.try_into()?;
            Ok(())
        } else {
            let row = diesel::insert_into(page::table)
                .values(NewPageRow {
                    id: value.id,
                    slug: &value.slug,
                    title: &value.title,
                    content: &value.content,
                    description: &value.description,
                    image_url: &value.image_url,
                    meta_data,
                    // GORM's `default:true` tag substitutes the database default
                    // when a newly-created model carries bool's false zero value.
                    is_published: value.is_published.then_some(true),
                })
                .returning(PageRow::as_returning())
                .get_result::<PageRow>(&mut connection)
                .await
                .map_err(map_diesel_error)?;
            *value = row.try_into()?;
            Ok(())
        }
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::delete(page::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if affected == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}

fn pagination_offset(options: PageListOptions) -> Result<i64, AppError> {
    options
        .page
        .checked_sub(1)
        .and_then(|page| page.checked_mul(options.per_page))
        .ok_or_else(|| AppError::InvalidInput("pagination is out of range".to_owned()))
}

fn meta_data_to_value(page: &Page) -> Option<Value> {
    page.meta_data
        .clone()
        .map(|values| Value::Object(values.into_iter().collect::<Map<_, _>>()))
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
