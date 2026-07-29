use std::sync::{Arc, Mutex, MutexGuard};

use async_trait::async_trait;
use blog_backend::{
    core::storage::{ObjectEntry, ObjectListing, ObjectStore, StorageService},
    error::AppError,
};
use chrono::{TimeZone, Utc};
use tokio_util::sync::CancellationToken;

#[derive(Debug, Clone, PartialEq, Eq)]
enum Operation {
    List(String, Option<String>),
    Put(String, Vec<u8>),
    Copy(String, String),
    Delete(String),
}

#[derive(Default)]
struct ObjectStoreState {
    listing: ObjectListing,
    operations: Vec<Operation>,
    fail_copy_destination: Option<String>,
}

#[derive(Default)]
struct MemoryObjectStore {
    state: Mutex<ObjectStoreState>,
}

impl MemoryObjectStore {
    fn state(&self) -> MutexGuard<'_, ObjectStoreState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl ObjectStore for MemoryObjectStore {
    async fn list(&self, prefix: &str, delimiter: Option<&str>) -> Result<ObjectListing, AppError> {
        let mut state = self.state();
        state.operations.push(Operation::List(
            prefix.to_owned(),
            delimiter.map(str::to_owned),
        ));
        Ok(state.listing.clone())
    }

    async fn put(&self, key: &str, data: Vec<u8>) -> Result<(), AppError> {
        self.state()
            .operations
            .push(Operation::Put(key.to_owned(), data));
        Ok(())
    }

    async fn delete(&self, key: &str) -> Result<(), AppError> {
        self.state()
            .operations
            .push(Operation::Delete(key.to_owned()));
        Ok(())
    }

    async fn copy(&self, source_key: &str, destination_key: &str) -> Result<(), AppError> {
        let mut state = self.state();
        state.operations.push(Operation::Copy(
            source_key.to_owned(),
            destination_key.to_owned(),
        ));
        if state.fail_copy_destination.as_deref() == Some(destination_key) {
            return Err(AppError::External);
        }
        Ok(())
    }
}

#[tokio::test]
async fn listing_preserves_go_format_sort_url_and_folder_rules() {
    let store = Arc::new(MemoryObjectStore::default());
    {
        let mut state = store.state();
        state.listing = ObjectListing {
            objects: vec![
                ObjectEntry {
                    key: "images/old.jpg".to_owned(),
                    last_modified: Utc.timestamp_opt(10, 0).single().unwrap_or_else(Utc::now),
                    size: 1536,
                },
                ObjectEntry {
                    key: "images/new.PNG".to_owned(),
                    last_modified: Utc.timestamp_opt(20, 0).single().unwrap_or_else(Utc::now),
                    size: 1024,
                },
            ],
            common_prefixes: vec![
                "images/zebra/".to_owned(),
                "images/.hidden/".to_owned(),
                "images/alpha/".to_owned(),
            ],
        };
    }
    let service = StorageService::new(
        store.clone(),
        "https://cdn.example.test",
        CancellationToken::new(),
    );
    let result = service.list_files("images/").await;
    assert!(result.is_ok());
    let Ok(result) = result else {
        return;
    };
    assert_eq!(
        result
            .files
            .iter()
            .map(|file| file.key.as_str())
            .collect::<Vec<_>>(),
        vec!["images/new.PNG", "images/old.jpg"]
    );
    assert_eq!(result.files[0].size, "1.00 KiB");
    assert_eq!(
        result.files[0].url,
        "https://cdn.example.test/images/new.PNG"
    );
    assert!(!result.files[0].is_image);
    assert_eq!(result.files[1].size, "1.00 KiB");
    assert!(result.files[1].is_image);
    assert_eq!(
        result
            .folders
            .iter()
            .map(|folder| folder.name.as_str())
            .collect::<Vec<_>>(),
        vec![".hidden", "alpha", "zebra"]
    );
    assert!(result.folders[0].is_hidden);
    assert_eq!(result.folders[0].path, "images/.hidden/");
    assert!(result.folders.iter().all(|folder| folder.file_count == 0));
    assert_eq!(
        store.state().operations,
        vec![Operation::List("images/".to_owned(), Some("/".to_owned()))]
    );
}

#[tokio::test]
async fn folder_move_is_sequential_and_preserves_partial_failure_boundary() {
    let store = Arc::new(MemoryObjectStore::default());
    {
        let mut state = store.state();
        state.listing.objects = vec![
            ObjectEntry {
                key: "old/a.txt".to_owned(),
                last_modified: Utc::now(),
                size: 1,
            },
            ObjectEntry {
                key: "old/b.txt".to_owned(),
                last_modified: Utc::now(),
                size: 1,
            },
        ];
        state.fail_copy_destination = Some("new/b.txt".to_owned());
    }
    let service = StorageService::new(
        store.clone(),
        "https://cdn.example.test",
        CancellationToken::new(),
    );
    let result = service.update_folder("old/", "new/").await;
    assert!(matches!(result, Err(AppError::External)));
    assert_eq!(
        store.state().operations,
        vec![
            Operation::List("old/".to_owned(), None),
            Operation::Copy("old/a.txt".to_owned(), "new/a.txt".to_owned()),
            Operation::Delete("old/a.txt".to_owned()),
            Operation::Copy("old/b.txt".to_owned(), "new/b.txt".to_owned()),
        ]
    );
}

#[tokio::test]
async fn folder_creation_upload_and_url_prefix_preserve_exact_keys() {
    let store = Arc::new(MemoryObjectStore::default());
    let service = StorageService::new(
        store.clone(),
        "https://cdn.example.test/",
        CancellationToken::new(),
    );
    assert!(service.create_folder("drafts").await.is_ok());
    assert!(service.create_folder("published/").await.is_ok());
    assert!(
        service
            .upload_file("drafts/post.md", b"post".to_vec())
            .await
            .is_ok()
    );
    assert_eq!(service.url_prefix(), "https://cdn.example.test/");
    assert_eq!(
        store.state().operations,
        vec![
            Operation::Put("drafts/".to_owned(), Vec::new()),
            Operation::Put("published/".to_owned(), Vec::new()),
            Operation::Put("drafts/post.md".to_owned(), b"post".to_vec()),
        ]
    );
}

#[tokio::test]
async fn cancellation_stops_object_store_admission() {
    let store = Arc::new(MemoryObjectStore::default());
    let cancellation = CancellationToken::new();
    cancellation.cancel();
    let service = StorageService::new(store.clone(), "", cancellation);
    assert!(matches!(
        service.delete_file("key").await,
        Err(AppError::Internal)
    ));
    assert!(store.state().operations.is_empty());
}
