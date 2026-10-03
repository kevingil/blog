use async_trait::async_trait;
use chrono::Utc;
use diesel::{
    BoolExpressionMethods, ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper,
    TextExpressionMethods,
};
use diesel_async::RunQueryDsl;
use std::collections::HashMap;
use uuid::Uuid;

use crate::{
    core::storage::{
        ImageAsset, UploadFile, UploadRepository, public_url, unlinked_image_like_patterns,
        urls_match_upload,
    },
    database::{
        models::upload::{NewUploadFileRow, UploadFileRow},
        pool::PgPool,
    },
    error::AppError,
    schema::{
        account, article, article_version, organization, page, project, upload_file_refs,
        upload_files,
    },
};

#[derive(Clone)]
pub struct DieselUploadRepository {
    pool: PgPool,
}

impl DieselUploadRepository {
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
impl UploadRepository for DieselUploadRepository {
    async fn upsert(&self, file: UploadFile) -> Result<UploadFile, AppError> {
        let mut connection = self.connection().await?;
        let row = diesel::insert_into(upload_files::table)
            .values(NewUploadFileRow {
                id: file.id,
                s3_key: file.s3_key.clone(),
                public_url: file.public_url.clone(),
                filename: file.filename.clone(),
                directory_path: file.directory_path.clone(),
                content_type: file.content_type.clone(),
                byte_size: file.byte_size,
                width: file.width,
                height: file.height,
                blurhash: file.blurhash.clone(),
                created_by: file.created_by,
            })
            .on_conflict(upload_files::s3_key)
            .do_update()
            .set((
                upload_files::public_url.eq(&file.public_url),
                upload_files::filename.eq(&file.filename),
                upload_files::directory_path.eq(&file.directory_path),
                upload_files::content_type.eq(&file.content_type),
                upload_files::byte_size.eq(file.byte_size),
                upload_files::width.eq(file.width),
                upload_files::height.eq(file.height),
                upload_files::blurhash.eq(&file.blurhash),
                upload_files::updated_at.eq(Utc::now()),
            ))
            .returning(UploadFileRow::as_returning())
            .get_result::<UploadFileRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        Ok(row.into())
    }

    async fn find_by_id(&self, id: Uuid) -> Result<Option<UploadFile>, AppError> {
        let mut connection = self.connection().await?;
        upload_files::table
            .find(id)
            .select(UploadFileRow::as_select())
            .first::<UploadFileRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)
            .map(|row| row.map(Into::into))
    }

    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<UploadFile>, AppError> {
        if ids.is_empty() {
            return Ok(Vec::new());
        }
        let mut connection = self.connection().await?;
        upload_files::table
            .filter(upload_files::id.eq_any(ids))
            .select(UploadFileRow::as_select())
            .load::<UploadFileRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)
            .map(|rows| rows.into_iter().map(Into::into).collect())
    }

    async fn find_by_keys(&self, keys: &[String]) -> Result<Vec<UploadFile>, AppError> {
        if keys.is_empty() {
            return Ok(Vec::new());
        }
        let mut connection = self.connection().await?;
        upload_files::table
            .filter(upload_files::s3_key.eq_any(keys))
            .select(UploadFileRow::as_select())
            .load::<UploadFileRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)
            .map(|rows| rows.into_iter().map(Into::into).collect())
    }

    async fn find_by_public_url(&self, url: &str) -> Result<Option<UploadFile>, AppError> {
        let mut connection = self.connection().await?;
        upload_files::table
            .filter(upload_files::public_url.eq(url))
            .select(UploadFileRow::as_select())
            .first::<UploadFileRow>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)
            .map(|row| row.map(Into::into))
    }

    async fn find_by_public_urls(&self, urls: &[String]) -> Result<Vec<UploadFile>, AppError> {
        if urls.is_empty() {
            return Ok(Vec::new());
        }
        let mut connection = self.connection().await?;
        upload_files::table
            .filter(upload_files::public_url.eq_any(urls))
            .select(UploadFileRow::as_select())
            .load::<UploadFileRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)
            .map(|rows| rows.into_iter().map(Into::into).collect())
    }

    async fn delete_by_key(&self, key: &str) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(upload_files::table.filter(upload_files::s3_key.eq(key)))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(|_| AppError::Database)
    }

    async fn is_referenced(&self, id: Uuid) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        let article_hit = article::table
            .filter(
                article::draft_upload_file_id
                    .eq(id)
                    .or(article::published_upload_file_id.eq(id)),
            )
            .select(article::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if article_hit.is_some() {
            return Ok(true);
        }
        let version_hit = article_version::table
            .filter(article_version::upload_file_id.eq(id))
            .select(article_version::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if version_hit.is_some() {
            return Ok(true);
        }
        let page_hit = page::table
            .filter(page::upload_file_id.eq(id))
            .select(page::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if page_hit.is_some() {
            return Ok(true);
        }
        let project_hit = project::table
            .filter(project::upload_file_id.eq(id))
            .select(project::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if project_hit.is_some() {
            return Ok(true);
        }
        let account_hit = account::table
            .filter(account::profile_upload_file_id.eq(id))
            .select(account::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if account_hit.is_some() {
            return Ok(true);
        }
        let organization_hit = organization::table
            .filter(organization::logo_upload_file_id.eq(id))
            .select(organization::id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)?;
        if organization_hit.is_some() {
            return Ok(true);
        }
        upload_file_refs::table
            .filter(upload_file_refs::upload_file_id.eq(id))
            .select(upload_file_refs::upload_file_id)
            .first::<Uuid>(&mut connection)
            .await
            .optional()
            .map_err(|_| AppError::Database)
            .map(|row| row.is_some())
    }

    async fn rename_prefix(
        &self,
        old_prefix: &str,
        new_prefix: &str,
        url_prefix: &str,
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        let rows = upload_files::table
            .filter(upload_files::s3_key.like(format!("{old_prefix}%")))
            .select(UploadFileRow::as_select())
            .load::<UploadFileRow>(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        for row in rows {
            if row.s3_key.ends_with('/') {
                continue;
            }
            let new_key = row.s3_key.replacen(old_prefix, new_prefix, 1);
            let new_url = public_url(url_prefix, &new_key);
            let directory = match new_key.rsplit_once('/') {
                Some((directory, file)) if !directory.is_empty() && !file.is_empty() => {
                    directory.to_owned()
                }
                _ => String::new(),
            };
            let filename = new_key
                .rsplit('/')
                .next()
                .unwrap_or(new_key.as_str())
                .to_owned();
            diesel::update(upload_files::table.find(row.id))
                .set((
                    upload_files::s3_key.eq(&new_key),
                    upload_files::public_url.eq(&new_url),
                    upload_files::filename.eq(&filename),
                    upload_files::directory_path.eq(&directory),
                    upload_files::updated_at.eq(Utc::now()),
                ))
                .execute(&mut connection)
                .await
                .map_err(|_| AppError::Database)?;
            rewrite_stored_url(&mut connection, &row.public_url, &new_url).await?;
        }
        Ok(())
    }

    async fn replace_body_refs(
        &self,
        owner_kind: &str,
        owner_id: Uuid,
        upload_ids: &[Uuid],
    ) -> Result<(), AppError> {
        let mut connection = self.connection().await?;
        diesel::delete(
            upload_file_refs::table.filter(
                upload_file_refs::owner_kind
                    .eq(owner_kind)
                    .and(upload_file_refs::owner_id.eq(owner_id)),
            ),
        )
        .execute(&mut connection)
        .await
        .map_err(|_| AppError::Database)?;
        for upload_id in upload_ids {
            diesel::insert_into(upload_file_refs::table)
                .values((
                    upload_file_refs::upload_file_id.eq(upload_id),
                    upload_file_refs::owner_kind.eq(owner_kind),
                    upload_file_refs::owner_id.eq(owner_id),
                ))
                .execute(&mut connection)
                .await
                .map_err(|_| AppError::Database)?;
        }
        Ok(())
    }

    async fn attach_unlinked_images(&self, file: &UploadFile) -> Result<(), AppError> {
        let patterns = unlinked_image_like_patterns(file);
        let mut connection = self.connection().await?;
        let draft_ids = matching_image_ids(
            &mut connection,
            file,
            &patterns,
            "SELECT id, coalesce(draft_image_url, '') AS image_url \
             FROM article \
             WHERE draft_upload_file_id IS NULL \
               AND coalesce(draft_image_url, '') <> '' \
               AND (draft_image_url = $1 \
                    OR draft_image_url LIKE $2 ESCAPE '\\' \
                    OR draft_image_url LIKE $3 ESCAPE '\\')",
        )
        .await?;
        if !draft_ids.is_empty() {
            diesel::update(
                article::table.filter(
                    article::id
                        .eq_any(&draft_ids)
                        .and(article::draft_upload_file_id.is_null()),
                ),
            )
            .set(article::draft_upload_file_id.eq(file.id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        }
        let published_ids = matching_image_ids(
            &mut connection,
            file,
            &patterns,
            "SELECT id, coalesce(published_image_url, '') AS image_url \
             FROM article \
             WHERE published_upload_file_id IS NULL \
               AND coalesce(published_image_url, '') <> '' \
               AND (published_image_url = $1 \
                    OR published_image_url LIKE $2 ESCAPE '\\' \
                    OR published_image_url LIKE $3 ESCAPE '\\')",
        )
        .await?;
        if !published_ids.is_empty() {
            diesel::update(
                article::table.filter(
                    article::id
                        .eq_any(&published_ids)
                        .and(article::published_upload_file_id.is_null()),
                ),
            )
            .set(article::published_upload_file_id.eq(file.id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        }
        let version_ids = matching_image_ids(
            &mut connection,
            file,
            &patterns,
            "SELECT id, coalesce(image_url, '') AS image_url \
             FROM article_version \
             WHERE upload_file_id IS NULL \
               AND coalesce(image_url, '') <> '' \
               AND (image_url = $1 OR image_url LIKE $2 ESCAPE '\\' OR image_url LIKE $3 ESCAPE '\\')",
        )
        .await?;
        if !version_ids.is_empty() {
            diesel::update(
                article_version::table.filter(
                    article_version::id
                        .eq_any(&version_ids)
                        .and(article_version::upload_file_id.is_null()),
                ),
            )
            .set(article_version::upload_file_id.eq(file.id))
            .execute(&mut connection)
            .await
            .map_err(|_| AppError::Database)?;
        }
        Ok(())
    }
}

#[derive(Debug, diesel::QueryableByName)]
struct UnlinkedImageRow {
    #[diesel(sql_type = diesel::sql_types::Uuid)]
    id: Uuid,
    #[diesel(sql_type = diesel::sql_types::Text)]
    image_url: String,
}

async fn matching_image_ids(
    connection: &mut diesel_async::AsyncPgConnection,
    file: &UploadFile,
    patterns: &[String; 2],
    query: &str,
) -> Result<Vec<Uuid>, AppError> {
    let rows = diesel::sql_query(query)
        .bind::<diesel::sql_types::Text, _>(&file.public_url)
        .bind::<diesel::sql_types::Text, _>(&patterns[0])
        .bind::<diesel::sql_types::Text, _>(&patterns[1])
        .get_results::<UnlinkedImageRow>(connection)
        .await
        .map_err(|_| AppError::Database)?;
    Ok(rows
        .into_iter()
        .filter(|row| urls_match_upload(&row.image_url, file))
        .map(|row| row.id)
        .collect())
}

impl From<UploadFileRow> for UploadFile {
    fn from(row: UploadFileRow) -> Self {
        Self {
            id: row.id,
            s3_key: row.s3_key,
            public_url: row.public_url,
            filename: row.filename,
            directory_path: row.directory_path,
            content_type: row.content_type,
            byte_size: row.byte_size,
            width: row.width,
            height: row.height,
            blurhash: row.blurhash,
            created_by: row.created_by,
        }
    }
}

pub fn asset_map(files: Vec<UploadFile>) -> HashMap<Uuid, ImageAsset> {
    files
        .into_iter()
        .map(|file| {
            let id = file.id;
            (id, file.asset())
        })
        .collect()
}

pub fn markdown_image_urls(content: &str) -> Vec<String> {
    let mut urls = Vec::new();
    let mut rest = content;
    while let Some(start) = rest.find("![") {
        rest = &rest[start + 2..];
        let Some(alt_end) = rest.find("](") else {
            break;
        };
        rest = &rest[alt_end + 2..];
        let Some(url_end) = rest.find(')') else {
            break;
        };
        let url = rest[..url_end].trim();
        if !url.is_empty() {
            urls.push(url.to_owned());
        }
        rest = &rest[url_end + 1..];
    }
    urls
}

async fn rewrite_stored_url(
    connection: &mut diesel_async::AsyncPgConnection,
    old_url: &str,
    new_url: &str,
) -> Result<(), AppError> {
    diesel::sql_query("UPDATE article SET draft_image_url = $2 WHERE draft_image_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE article SET published_image_url = $2 WHERE published_image_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query(
        "UPDATE article SET draft_content = replace(draft_content, $1, $2) WHERE strpos(draft_content, $1) > 0",
    )
    .bind::<diesel::sql_types::Text, _>(old_url)
    .bind::<diesel::sql_types::Text, _>(new_url)
    .execute(connection)
    .await
    .map_err(|_| AppError::Database)?;
    diesel::sql_query(
        "UPDATE article SET published_content = replace(published_content, $1, $2) WHERE strpos(published_content, $1) > 0",
    )
    .bind::<diesel::sql_types::Text, _>(old_url)
    .bind::<diesel::sql_types::Text, _>(new_url)
    .execute(connection)
    .await
    .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE article_version SET image_url = $2 WHERE image_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query(
        "UPDATE article_version SET content = replace(content, $1, $2) WHERE strpos(content, $1) > 0",
    )
    .bind::<diesel::sql_types::Text, _>(old_url)
    .bind::<diesel::sql_types::Text, _>(new_url)
    .execute(connection)
    .await
    .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE page SET image_url = $2 WHERE image_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query(
        "UPDATE page SET content = replace(content, $1, $2) WHERE strpos(content, $1) > 0",
    )
    .bind::<diesel::sql_types::Text, _>(old_url)
    .bind::<diesel::sql_types::Text, _>(new_url)
    .execute(connection)
    .await
    .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE project SET image_url = $2 WHERE image_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE account SET profile_image = $2 WHERE profile_image = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    diesel::sql_query("UPDATE organization SET logo_url = $2 WHERE logo_url = $1")
        .bind::<diesel::sql_types::Text, _>(old_url)
        .bind::<diesel::sql_types::Text, _>(new_url)
        .execute(connection)
        .await
        .map_err(|_| AppError::Database)?;
    Ok(())
}
