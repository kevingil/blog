use async_trait::async_trait;
use diesel::{ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper};
use diesel_async::RunQueryDsl;
use serde_json::Value;
use uuid::Uuid;

use crate::{
    core::image::{IMAGE_STATUS_PENDING, ImageGeneration, ImageRepository},
    database::{
        models::image::{ImageChangeset, ImageRow, NewImageRow},
        pool::PgPool,
    },
    error::AppError,
    schema::imagen_request,
};

#[derive(Clone)]
pub struct DieselImageRepository {
    pool: PgPool,
}

impl DieselImageRepository {
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
impl ImageRepository for DieselImageRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<ImageGeneration, AppError> {
        let mut connection = self.connection().await?;
        imagen_request::table
            .find(id)
            .select(ImageRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn find_by_request_id(&self, request_id: &str) -> Result<ImageGeneration, AppError> {
        let mut connection = self.connection().await?;
        imagen_request::table
            .filter(imagen_request::request_id.eq(request_id))
            .select(ImageRow::as_select())
            .first(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?
            .ok_or(AppError::NotFound)
            .map(Into::into)
    }

    async fn save(&self, image: &mut ImageGeneration) -> Result<(), AppError> {
        if image.id.is_nil() {
            image.id = Uuid::new_v4();
        }
        let mut connection = self.connection().await?;
        diesel::insert_into(imagen_request::table)
            .values(new_row(image))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn update(&self, image: &ImageGeneration) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let affected = diesel::update(imagen_request::table.find(image.id))
            .set(changeset(image))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        if affected == 0 {
            diesel::insert_into(imagen_request::table)
                .values(new_row(image))
                .execute(&mut connection)
                .await
                .map_err(|_| AppError::Database)?;
        }
        Ok(())
    }
}

impl From<ImageRow> for ImageGeneration {
    fn from(row: ImageRow) -> Self {
        Self {
            id: row.id,
            prompt: row.prompt,
            provider: row.provider,
            model_name: row.model_name,
            request_id: row.request_id.unwrap_or_default(),
            status: row
                .status
                .unwrap_or_else(|| IMAGE_STATUS_PENDING.to_owned()),
            output_url: row.output_url.unwrap_or_default(),
            file_index_id: row.file_index_id,
            error_message: row.error_message.unwrap_or_default(),
            meta_data: row.meta_data.and_then(|value| match value {
                Value::Object(map) => Some(map.into_iter().collect()),
                _ => None,
            }),
            created_at: row.created_at,
            completed_at: row.completed_at,
        }
    }
}

fn new_row(image: &ImageGeneration) -> NewImageRow {
    NewImageRow {
        id: image.id,
        prompt: image.prompt.clone(),
        provider: image.provider.clone(),
        model_name: image.model_name.clone(),
        request_id: Some(image.request_id.clone()),
        status: image.status.clone(),
        output_url: Some(image.output_url.clone()),
        file_index_id: image.file_index_id,
        error_message: Some(image.error_message.clone()),
        meta_data: metadata_value(image),
        created_at: image.created_at.unwrap_or_else(chrono::Utc::now),
        completed_at: image.completed_at,
    }
}

fn changeset(image: &ImageGeneration) -> ImageChangeset {
    ImageChangeset {
        prompt: image.prompt.clone(),
        provider: image.provider.clone(),
        model_name: image.model_name.clone(),
        request_id: Some(image.request_id.clone()),
        status: Some(image.status.clone()),
        output_url: Some(image.output_url.clone()),
        file_index_id: image.file_index_id,
        error_message: Some(image.error_message.clone()),
        meta_data: image
            .meta_data
            .as_ref()
            .map(|map| Value::Object(map.clone().into_iter().collect())),
        completed_at: image.completed_at,
    }
}

fn metadata_value(image: &ImageGeneration) -> Value {
    Value::Object(
        image
            .meta_data
            .clone()
            .map(IntoIterator::into_iter)
            .map(Iterator::collect)
            .unwrap_or_default(),
    )
}
