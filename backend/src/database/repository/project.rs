use async_trait::async_trait;
use chrono::Utc;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper,
    dsl::count_star,
    result::{DatabaseErrorKind, Error as DieselError},
};
use diesel_async::RunQueryDsl;
use uuid::Uuid;

use crate::{
    core::project::{Project, ProjectListOptions, ProjectRepository},
    database::{
        models::project::{NewProjectRow, ProjectRow},
        pool::PgPool,
    },
    error::AppError,
    schema::project,
};

#[derive(Clone)]
pub struct DieselProjectRepository {
    pool: PgPool,
}

impl DieselProjectRepository {
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
impl ProjectRepository for DieselProjectRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Project, AppError> {
        let mut connection = self.connection().await?;
        project::table
            .find(id)
            .select(ProjectRow::as_select())
            .first::<ProjectRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn list(&self, options: ProjectListOptions) -> Result<(Vec<Project>, i64), AppError> {
        let offset = pagination_offset(options)?;
        let mut connection = self.connection().await?;
        let total = project::table
            .select(count_star())
            .first::<i64>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        let rows = project::table
            .order(project::created_at.desc())
            .offset(offset)
            .limit(options.per_page)
            .select(ProjectRow::as_select())
            .load::<ProjectRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        let projects = rows
            .into_iter()
            .map(Project::try_from)
            .collect::<Result<Vec<_>, _>>()?;
        Ok((projects, total))
    }

    async fn save(&self, value: &mut Project) -> Result<(), AppError> {
        if value.id.is_nil() {
            value.id = Uuid::new_v4();
        }
        let tag_ids = tag_ids_to_database(&value.tag_ids)?;
        let mut connection = self.connection().await?;
        let row = diesel::insert_into(project::table)
            .values(NewProjectRow {
                id: value.id,
                title: &value.title,
                description: &value.description,
                image_url: &value.image_url,
                url: &value.url,
                content: &value.content,
                tag_ids,
            })
            .returning(ProjectRow::as_returning())
            .get_result::<ProjectRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        *value = row.try_into()?;
        Ok(())
    }

    async fn update(&self, value: &Project) -> Result<(), AppError> {
        let tag_ids = tag_ids_to_database(&value.tag_ids)?;
        let mut connection = self.connection().await?;
        let affected = diesel::update(project::table.find(value.id))
            .set((
                project::title.eq(&value.title),
                project::description.eq(&value.description),
                project::content.eq(&value.content),
                project::tag_ids.eq(tag_ids),
                project::image_url.eq(&value.image_url),
                project::url.eq(&value.url),
                project::updated_at.eq(value.updated_at.unwrap_or_else(Utc::now)),
            ))
            .execute(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        if affected == 0 {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::delete(project::table.find(id))
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

fn pagination_offset(options: ProjectListOptions) -> Result<i64, AppError> {
    options
        .page
        .checked_sub(1)
        .and_then(|page| page.checked_mul(options.per_page))
        .ok_or_else(|| AppError::InvalidInput("pagination is out of range".to_owned()))
}

fn tag_ids_to_database(ids: &[i64]) -> Result<Vec<Option<i32>>, AppError> {
    ids.iter()
        .copied()
        .map(|id| {
            i32::try_from(id)
                .map(Some)
                .map_err(|_| AppError::InvalidInput("tag ID is out of range".to_owned()))
        })
        .collect()
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
