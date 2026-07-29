use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::{core::project::Project, error::AppError, schema::project};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = project)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ProjectRow {
    pub id: Uuid,
    pub title: String,
    pub description: String,
    pub image_url: Option<String>,
    pub url: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub content: Option<String>,
    pub tag_ids: Option<Vec<Option<i32>>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = project)]
pub struct NewProjectRow<'a> {
    pub id: Uuid,
    pub title: &'a str,
    pub description: &'a str,
    pub image_url: &'a str,
    pub url: &'a str,
    pub content: &'a str,
    pub tag_ids: Vec<Option<i32>>,
}

impl TryFrom<ProjectRow> for Project {
    type Error = AppError;

    fn try_from(row: ProjectRow) -> Result<Self, Self::Error> {
        let tag_ids = row
            .tag_ids
            .unwrap_or_default()
            .into_iter()
            .map(|id| id.map(i64::from).ok_or(AppError::Database))
            .collect::<Result<Vec<_>, _>>()?;
        Ok(Self {
            id: row.id,
            title: row.title,
            description: row.description,
            content: row.content.unwrap_or_default(),
            tag_ids,
            image_url: row.image_url.unwrap_or_default(),
            url: row.url.unwrap_or_default(),
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}
