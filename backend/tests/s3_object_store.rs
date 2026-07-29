use std::{env, error::Error, io};

use blog_backend::{core::storage::ObjectStore, integrations::s3::S3ObjectStore};
use uuid::Uuid;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[tokio::test]
async fn s3_compatible_adapter_round_trips_list_copy_and_delete() -> TestResult {
    let endpoint = env::var("TEST_S3_ENDPOINT").map_err(|_| {
        io::Error::other("TEST_S3_ENDPOINT is required; start the Docker object-storage fixture")
    })?;
    let store =
        S3ObjectStore::for_s3_compatible(endpoint, "blog", "blog-local-secret", "blog").await?;
    let prefix = format!("adapter-test-{}/", Uuid::new_v4());
    let source = format!("{prefix}source.txt");
    let copy = format!("{prefix}copy.txt");

    store.put(&source, b"fixture body".to_vec()).await?;
    let listing = store.list(&prefix, Some("/")).await?;
    assert_eq!(listing.objects.len(), 1);
    assert_eq!(listing.objects[0].key, source);
    assert_eq!(listing.objects[0].size, 12);

    store.copy(&source, &copy).await?;
    let listing = store.list(&prefix, None).await?;
    assert_eq!(listing.objects.len(), 2);

    store.delete(&source).await?;
    store.delete(&copy).await?;
    assert!(store.list(&prefix, None).await?.objects.is_empty());
    Ok(())
}
