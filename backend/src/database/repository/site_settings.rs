use async_trait::async_trait;
use chrono::Utc;
use diesel::{
    ExpressionMethods, OptionalExtension, QueryDsl, SelectableHelper,
    result::{DatabaseErrorKind, Error as DieselError},
};
use diesel_async::RunQueryDsl;
use serde_json::{Map, Value};
use uuid::Uuid;

use crate::{
    core::profile::{
        ProfileAccount, ProfileAccountRepository, ProfileRepository, PublicProfile, SiteSettings,
        SiteSettingsRepository,
    },
    database::{
        models::{
            account::AccountRow,
            organization::OrganizationRow,
            site_settings::{NewSiteSettingsRow, SiteSettingsRow},
        },
        pool::PgPool,
    },
    error::AppError,
    schema::{account, organization, site_settings},
};

#[derive(Clone)]
pub struct DieselSiteSettingsRepository {
    pool: PgPool,
}

impl DieselSiteSettingsRepository {
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
impl SiteSettingsRepository for DieselSiteSettingsRepository {
    async fn get(&self) -> Result<SiteSettings, AppError> {
        let mut connection = self.connection().await?;
        site_settings::table
            .find(1)
            .select(SiteSettingsRow::as_select())
            .first::<SiteSettingsRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(SiteSettings::from)
            .ok_or(AppError::NotFound)
    }

    async fn save(&self, settings: &mut SiteSettings) -> Result<(), AppError> {
        settings.id = 1;
        let mut connection = self.connection().await?;
        diesel::insert_into(site_settings::table)
            .values(NewSiteSettingsRow {
                id: 1,
                public_profile_type: &settings.public_profile_type,
                public_user_id: settings.public_user_id,
                public_organization_id: settings.public_organization_id,
            })
            .on_conflict(site_settings::id)
            .do_update()
            .set((
                site_settings::public_profile_type.eq(&settings.public_profile_type),
                site_settings::public_user_id.eq(settings.public_user_id),
                site_settings::public_organization_id.eq(settings.public_organization_id),
                site_settings::updated_at.eq(Utc::now()),
            ))
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_diesel_error)
    }
}

#[async_trait]
impl ProfileAccountRepository for DieselSiteSettingsRepository {
    async fn find_profile_account(&self, id: Uuid) -> Result<ProfileAccount, AppError> {
        let mut connection = self.connection().await?;
        account::table
            .find(id)
            .select(AccountRow::as_select())
            .first::<AccountRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)
            .and_then(profile_account_from_row)
    }

    async fn update_profile_account(&self, value: &ProfileAccount) -> Result<(), AppError> {
        let social_links = value
            .social_links
            .clone()
            .map(|values| Value::Object(values.into_iter().collect::<Map<_, _>>()));
        let mut connection = self.connection().await?;
        let affected = diesel::update(account::table.find(value.id))
            .set((
                account::name.eq(&value.name),
                account::bio.eq(value.bio.as_deref()),
                account::profile_image.eq(value.profile_image.as_deref()),
                account::email_public.eq(value.email_public.as_deref()),
                account::social_links.eq(social_links),
                account::meta_description.eq(value.meta_description.as_deref()),
                account::organization_id.eq(value.organization_id),
                account::updated_at.eq(Utc::now()),
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
}

#[async_trait]
impl ProfileRepository for DieselSiteSettingsRepository {
    async fn get_public_profile(&self) -> Result<PublicProfile, AppError> {
        let mut connection = self.connection().await?;
        let settings = site_settings::table
            .find(1)
            .select(SiteSettingsRow::as_select())
            .first::<SiteSettingsRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?;
        let profile_type = settings.public_profile_type.unwrap_or_default();

        if profile_type == "organization"
            && let Some(organization_id) = settings.public_organization_id
            && let Some(row) = organization::table
                .find(organization_id)
                .select(OrganizationRow::as_select())
                .first::<OrganizationRow>(&mut connection)
                .await
                .optional()
                .map_err(map_diesel_error)?
        {
            let social_links = json_object(row.social_links)?;
            return Ok(PublicProfile {
                profile_type,
                name: row.name,
                bio: row.bio,
                image_url: row.logo_url,
                website_url: row.website_url,
                email_public: row.email_public,
                social_links,
                meta_description: row.meta_description,
            });
        }

        if let Some(user_id) = settings.public_user_id
            && let Some(row) = account::table
                .find(user_id)
                .select(AccountRow::as_select())
                .first::<AccountRow>(&mut connection)
                .await
                .optional()
                .map_err(map_diesel_error)?
        {
            let social_links = json_object(row.social_links)?;
            return Ok(PublicProfile {
                profile_type,
                name: row.name,
                bio: row.bio,
                image_url: row.profile_image,
                website_url: None,
                email_public: row.email_public,
                social_links,
                meta_description: row.meta_description,
            });
        }

        Ok(PublicProfile {
            profile_type,
            name: String::new(),
            bio: None,
            image_url: None,
            website_url: None,
            email_public: None,
            social_links: None,
            meta_description: None,
        })
    }

    async fn is_user_admin(&self, user_id: Uuid) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        account::table
            .find(user_id)
            .select(account::role)
            .first::<String>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .map(|role| role == "admin")
            .ok_or(AppError::NotFound)
    }
}

fn profile_account_from_row(row: AccountRow) -> Result<ProfileAccount, AppError> {
    Ok(ProfileAccount {
        id: row.id,
        name: row.name,
        bio: row.bio,
        profile_image: row.profile_image,
        email_public: row.email_public,
        social_links: json_object(row.social_links)?,
        meta_description: row.meta_description,
        organization_id: row.organization_id,
    })
}

fn json_object(
    value: Option<Value>,
) -> Result<Option<std::collections::BTreeMap<String, Value>>, AppError> {
    match value {
        Some(Value::Object(values)) => Ok(Some(values.into_iter().collect())),
        Some(_) => Err(AppError::Database),
        None => Ok(None),
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
