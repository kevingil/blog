use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::{core::page::Page, error::AppError, schema::page};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = page)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct PageRow {
    pub id: Uuid,
    pub slug: String,
    pub title: String,
    pub content: Option<String>,
    pub description: Option<String>,
    pub image_url: Option<String>,
    pub meta_data: Option<Value>,
    pub is_published: Option<bool>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = page)]
pub struct NewPageRow<'a> {
    pub id: Uuid,
    pub slug: &'a str,
    pub title: &'a str,
    pub content: &'a str,
    pub description: &'a str,
    pub image_url: &'a str,
    pub meta_data: Option<Value>,
    pub is_published: Option<bool>,
}

impl TryFrom<PageRow> for Page {
    type Error = AppError;

    fn try_from(row: PageRow) -> Result<Self, Self::Error> {
        let meta_data = match row.meta_data {
            Some(Value::Object(values)) => Some(values.into_iter().collect()),
            Some(_) => None,
            None => None,
        };
        Ok(Self {
            id: row.id,
            slug: row.slug,
            title: row.title,
            content: row.content.unwrap_or_default(),
            description: row.description.unwrap_or_default(),
            image_url: row.image_url.unwrap_or_default(),
            meta_data,
            is_published: row.is_published.unwrap_or(false),
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}
