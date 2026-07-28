use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::{core::profile::SiteSettings, schema::site_settings};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = site_settings)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct SiteSettingsRow {
    pub id: i32,
    pub public_profile_type: Option<String>,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = site_settings)]
pub struct NewSiteSettingsRow<'a> {
    pub id: i32,
    pub public_profile_type: &'a str,
    pub public_user_id: Option<Uuid>,
    pub public_organization_id: Option<Uuid>,
}

impl From<SiteSettingsRow> for SiteSettings {
    fn from(row: SiteSettingsRow) -> Self {
        Self {
            id: row.id,
            public_profile_type: row.public_profile_type.unwrap_or_default(),
            public_user_id: row.public_user_id,
            public_organization_id: row.public_organization_id,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}
