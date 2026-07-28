use async_trait::async_trait;
use diesel::{
    ExpressionMethods, OptionalExtension, PgArrayExpressionMethods, QueryDsl, SelectableHelper,
    dsl::count_star,
    result::{DatabaseErrorKind, Error as DieselError},
    sql_types::Text,
};
use diesel_async::{AsyncConnection, RunQueryDsl};

use crate::{
    core::tag::{Tag, TagRepository},
    database::{
        models::tag::{NewTagRow, TagRow},
        pool::PgPool,
    },
    error::AppError,
    schema::{article, project, tag},
};

diesel::define_sql_function!(fn lower(value: Text) -> Text);

#[derive(Clone)]
pub struct DieselTagRepository {
    pool: PgPool,
}

impl DieselTagRepository {
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
impl TagRepository for DieselTagRepository {
    async fn find_by_id(&self, id: i32) -> Result<Tag, AppError> {
        let mut connection = self.connection().await?;
        tag::table
            .find(id)
            .select(TagRow::as_select())
            .first::<TagRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Tag::from)
            .ok_or(AppError::NotFound)
    }

    async fn find_by_name(&self, name: &str) -> Result<Tag, AppError> {
        let mut connection = self.connection().await?;
        tag::table
            .filter(lower(tag::name).eq(lower(name)))
            .order(tag::id.asc())
            .select(TagRow::as_select())
            .first::<TagRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(Tag::from)
            .ok_or(AppError::NotFound)
    }

    async fn find_by_ids(&self, ids: &[i64]) -> Result<Vec<Tag>, AppError> {
        let ids = ids
            .iter()
            .copied()
            .map(|id| {
                i32::try_from(id)
                    .map_err(|_| AppError::InvalidInput("tag ID is out of range".to_owned()))
            })
            .collect::<Result<Vec<_>, _>>()?;
        let mut connection = self.connection().await?;
        tag::table
            .filter(tag::id.eq_any(ids))
            .order(tag::id.asc())
            .select(TagRow::as_select())
            .load::<TagRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Tag::from).collect())
            .map_err(map_diesel_error)
    }

    async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError> {
        let mut connection = self.connection().await?;
        connection
            .transaction::<Vec<i64>, DieselError, _>(async |connection| {
                let mut ids = Vec::with_capacity(names.len());
                for raw_name in names {
                    let name = raw_name.trim();
                    if name.is_empty() {
                        continue;
                    }

                    diesel::sql_query(
                        "SELECT pg_advisory_xact_lock(hashtextextended(LOWER($1), 0))",
                    )
                    .bind::<Text, _>(name)
                    .execute(connection)
                    .await?;

                    let existing = tag::table
                        .filter(lower(tag::name).eq(lower(name)))
                        .order(tag::id.asc())
                        .select(TagRow::as_select())
                        .first::<TagRow>(connection)
                        .await
                        .optional()?;

                    let id = if let Some(existing) = existing {
                        existing.id
                    } else {
                        diesel::insert_into(tag::table)
                            .values(NewTagRow { name })
                            .returning(tag::id)
                            .get_result::<i32>(connection)
                            .await?
                    };
                    ids.push(i64::from(id));
                }
                Ok(ids)
            })
            .await
            .map_err(map_diesel_error)
    }

    async fn list(&self) -> Result<Vec<Tag>, AppError> {
        let mut connection = self.connection().await?;
        tag::table
            .order(tag::name.asc())
            .select(TagRow::as_select())
            .load::<TagRow>(&mut connection)
            .await
            .map(|rows| rows.into_iter().map(Tag::from).collect())
            .map_err(map_diesel_error)
    }

    async fn save(&self, value: &mut Tag) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        if value.id == 0 {
            let row = diesel::insert_into(tag::table)
                .values(NewTagRow { name: &value.name })
                .returning(TagRow::as_returning())
                .get_result::<TagRow>(&mut connection)
                .await
                .map_err(map_diesel_error)?;
            value.id = row.id;
            return Ok(());
        }

        let affected = diesel::update(tag::table.find(value.id))
            .set(tag::name.eq(&value.name))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if affected == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn delete(&self, id: i32) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::delete(tag::table.find(id))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if affected == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn is_used(&self, id: i32) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        let article_count = article::table
            .filter(article::tag_ids.contains(vec![Some(id)]))
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if article_count > 0 {
            return Ok(true);
        }
        project::table
            .filter(project::tag_ids.contains(vec![Some(id)]))
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map(|count| count > 0)
            .map_err(map_diesel_error)
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
