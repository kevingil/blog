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
    core::organization::{Organization, OrganizationAccountRepository, OrganizationRepository},
    database::{
        models::organization::{NewOrganizationRow, OrganizationRow},
        pool::PgPool,
    },
    error::AppError,
    schema::{account, organization},
};

#[derive(Clone)]
pub struct DieselOrganizationRepository {
    pool: PgPool,
}

impl DieselOrganizationRepository {
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
impl OrganizationRepository for DieselOrganizationRepository {
    async fn find_by_id(&self, id: Uuid) -> Result<Organization, AppError> {
        let mut connection = self.connection().await?;
        organization::table
            .find(id)
            .select(OrganizationRow::as_select())
            .first::<OrganizationRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn find_by_slug(&self, slug: &str) -> Result<Organization, AppError> {
        let mut connection = self.connection().await?;
        organization::table
            .filter(organization::slug.eq(slug))
            .select(OrganizationRow::as_select())
            .first::<OrganizationRow>(&mut connection)
            .await
            .optional()
            .map_err(map_diesel_error)?
            .ok_or(AppError::NotFound)?
            .try_into()
    }

    async fn list(&self) -> Result<Vec<Organization>, AppError> {
        let mut connection = self.connection().await?;
        let rows = organization::table
            .order(organization::created_at.desc())
            .select(OrganizationRow::as_select())
            .load::<OrganizationRow>(&mut connection)
            .await
            .map_err(map_diesel_error)?;
        rows.into_iter()
            .map(Organization::try_from)
            .collect::<Result<Vec<_>, _>>()
    }

    async fn save(&self, value: &mut Organization) -> Result<(), AppError> {
        if value.id.is_nil() {
            value.id = Uuid::new_v4();
        }
        let social_links = social_links_to_value(value);
        let mut connection = self.connection().await?;
        diesel::insert_into(organization::table)
            .values(NewOrganizationRow {
                id: value.id,
                name: &value.name,
                slug: &value.slug,
                bio: value.bio.as_deref(),
                logo_url: value.logo_url.as_deref(),
                website_url: value.website_url.as_deref(),
                email_public: value.email_public.as_deref(),
                social_links,
                meta_description: value.meta_description.as_deref(),
            })
            .execute(&mut connection)
            .await
            .map(|_| ())
            .map_err(map_diesel_error)
    }

    async fn update(&self, value: &Organization) -> Result<(), AppError> {
        let social_links = social_links_to_value(value);
        let mut connection = self.connection().await?;
        let affected = diesel::update(organization::table.find(value.id))
            .set((
                organization::name.eq(&value.name),
                organization::slug.eq(&value.slug),
                organization::bio.eq(value.bio.as_deref()),
                organization::logo_url.eq(value.logo_url.as_deref()),
                organization::website_url.eq(value.website_url.as_deref()),
                organization::email_public.eq(value.email_public.as_deref()),
                organization::social_links.eq(social_links),
                organization::meta_description.eq(value.meta_description.as_deref()),
                organization::updated_at.eq(Utc::now()),
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
        let affected = diesel::delete(organization::table.find(id))
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
impl OrganizationAccountRepository for DieselOrganizationRepository {
    async fn set_organization(
        &self,
        account_id: Uuid,
        organization_id: Option<Uuid>,
    ) -> Result<bool, AppError> {
        let mut connection = self.connection().await?;
        diesel::update(account::table.find(account_id))
            .set((
                account::organization_id.eq(organization_id),
                account::updated_at.eq(Utc::now()),
            ))
            .execute(&mut connection)
            .await
            .map(|affected| affected > 0)
            .map_err(map_diesel_error)
    }
}

fn social_links_to_value(organization: &Organization) -> Option<Value> {
    organization
        .social_links
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
