use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};

use crate::{core::tag::Tag, schema::tag};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = tag)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TagRow {
    pub id: i32,
    pub name: String,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = tag)]
pub struct NewTagRow<'a> {
    pub name: &'a str,
}

impl From<TagRow> for Tag {
    fn from(row: TagRow) -> Self {
        Self {
            id: row.id,
            name: row.name,
            created_at: row.created_at,
        }
    }
}
