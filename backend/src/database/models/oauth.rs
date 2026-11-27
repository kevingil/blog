use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::{
    core::auth::{AccountId, AuthorizationCode, NewAuthorizationCode, NewOauthClient, OauthClient},
    schema::{oauth_authorization_code, oauth_client},
};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = oauth_client, primary_key(client_id))]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct OauthClientRow {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = oauth_client)]
pub struct NewOauthClientRow {
    pub client_id: String,
    pub client_name: String,
    pub redirect_uris: Vec<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = oauth_authorization_code)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct OauthAuthorizationCodeRow {
    pub id: Uuid,
    pub code_hash: String,
    pub client_id: String,
    pub account_id: Uuid,
    pub redirect_uri: String,
    pub code_challenge: String,
    pub expires_at: DateTime<Utc>,
    pub used_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = oauth_authorization_code)]
pub struct NewOauthAuthorizationCodeRow {
    pub code_hash: String,
    pub client_id: String,
    pub account_id: Uuid,
    pub redirect_uri: String,
    pub code_challenge: String,
    pub expires_at: DateTime<Utc>,
}

impl From<&NewOauthClient> for NewOauthClientRow {
    fn from(client: &NewOauthClient) -> Self {
        Self {
            client_id: client.client_id.clone(),
            client_name: client.client_name.clone(),
            redirect_uris: client.redirect_uris.clone(),
            created_at: client.created_at,
        }
    }
}

impl From<OauthClientRow> for OauthClient {
    fn from(row: OauthClientRow) -> Self {
        Self {
            client_id: row.client_id,
            client_name: row.client_name,
            redirect_uris: row.redirect_uris,
        }
    }
}

impl From<&NewAuthorizationCode> for NewOauthAuthorizationCodeRow {
    fn from(code: &NewAuthorizationCode) -> Self {
        Self {
            code_hash: code.code_hash.clone(),
            client_id: code.client_id.clone(),
            account_id: code.account_id.0,
            redirect_uri: code.redirect_uri.clone(),
            code_challenge: code.code_challenge.clone(),
            expires_at: code.expires_at,
        }
    }
}

impl From<OauthAuthorizationCodeRow> for AuthorizationCode {
    fn from(row: OauthAuthorizationCodeRow) -> Self {
        Self {
            client_id: row.client_id,
            account_id: AccountId(row.account_id),
            redirect_uri: row.redirect_uri,
            code_challenge: row.code_challenge,
        }
    }
}
