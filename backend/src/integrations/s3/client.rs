use async_trait::async_trait;
use aws_config::BehaviorVersion;
use aws_sdk_s3::{
    Client,
    config::{Credentials, Region},
    primitives::ByteStream,
};
use chrono::{TimeZone, Utc};
use secrecy::{ExposeSecret, SecretString};

use crate::{
    core::storage::{ObjectEntry, ObjectListing, ObjectStore},
    error::AppError,
};

#[derive(Clone)]
pub struct S3ObjectStore {
    client: Client,
    bucket: String,
}

impl S3ObjectStore {
    pub fn new(client: Client, bucket: impl Into<String>) -> Result<Self, AppError> {
        let bucket = bucket.into();
        if bucket.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "S3 bucket must not be empty".to_owned(),
            ));
        }
        Ok(Self { client, bucket })
    }

    pub async fn for_s3_compatible(
        endpoint: impl Into<String>,
        access_key_id: impl Into<String>,
        secret_access_key: impl Into<String>,
        bucket: impl Into<String>,
    ) -> Result<Self, AppError> {
        let endpoint = endpoint.into();
        let access_key_id = access_key_id.into();
        let secret_access_key = SecretString::from(secret_access_key.into());
        if endpoint.trim().is_empty()
            || access_key_id.trim().is_empty()
            || secret_access_key.expose_secret().is_empty()
        {
            return Err(AppError::InvalidInput(
                "S3 endpoint and credentials must not be empty".to_owned(),
            ));
        }
        let credentials = Credentials::new(
            access_key_id,
            secret_access_key.expose_secret(),
            None,
            None,
            "blog-backend",
        );
        let shared = aws_config::defaults(BehaviorVersion::latest())
            .region(Region::new("auto"))
            .credentials_provider(credentials)
            .endpoint_url(endpoint)
            .load()
            .await;
        let config = aws_sdk_s3::config::Builder::from(&shared)
            .force_path_style(true)
            .build();
        Self::new(Client::from_conf(config), bucket)
    }
}

#[async_trait]
impl ObjectStore for S3ObjectStore {
    async fn list(&self, prefix: &str, delimiter: Option<&str>) -> Result<ObjectListing, AppError> {
        let mut request = self
            .client
            .list_objects_v2()
            .bucket(&self.bucket)
            .prefix(prefix);
        if let Some(delimiter) = delimiter {
            request = request.delimiter(delimiter);
        }
        let response = request.send().await.map_err(|_| AppError::External)?;
        let mut objects = Vec::with_capacity(response.contents().len());
        for object in response.contents() {
            let key = object.key().ok_or(AppError::External)?;
            let modified = object.last_modified().ok_or(AppError::External)?;
            let last_modified = Utc
                .timestamp_opt(modified.secs(), modified.subsec_nanos())
                .single()
                .ok_or(AppError::External)?;
            objects.push(ObjectEntry {
                key: key.to_owned(),
                last_modified,
                size: object.size().unwrap_or_default(),
            });
        }
        let common_prefixes = response
            .common_prefixes()
            .iter()
            .filter_map(|prefix| prefix.prefix().map(ToOwned::to_owned))
            .collect();
        Ok(ObjectListing {
            objects,
            common_prefixes,
        })
    }

    async fn put(&self, key: &str, data: Vec<u8>) -> Result<(), AppError> {
        self.client
            .put_object()
            .bucket(&self.bucket)
            .key(key)
            .body(ByteStream::from(data))
            .send()
            .await
            .map(|_| ())
            .map_err(|_| AppError::External)
    }

    async fn delete(&self, key: &str) -> Result<(), AppError> {
        self.client
            .delete_object()
            .bucket(&self.bucket)
            .key(key)
            .send()
            .await
            .map(|_| ())
            .map_err(|_| AppError::External)
    }

    async fn copy(&self, source_key: &str, destination_key: &str) -> Result<(), AppError> {
        self.client
            .copy_object()
            .bucket(&self.bucket)
            .copy_source(format!("{}/{source_key}", self.bucket))
            .key(destination_key)
            .send()
            .await
            .map(|_| ())
            .map_err(|_| AppError::External)
    }
}
