use std::collections::HashMap;
use std::sync::{Arc, Mutex, MutexGuard};

use async_trait::async_trait;
use blog_backend::{
    core::storage::{
        ObjectEntry, ObjectListing, ObjectStore, StorageService, UploadFile, UploadRepository,
    },
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
    objects: HashMap<String, Vec<u8>>,
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
        let mut state = self.state();
        state.objects.insert(key.to_owned(), data.clone());
        state.operations.push(Operation::Put(key.to_owned(), data));
        Ok(())
    }

    async fn get(&self, key: &str) -> Result<Vec<u8>, AppError> {
        self.state()
            .objects
            .get(key)
            .cloned()
            .ok_or(AppError::NotFound)
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

fn solid_red_png() -> Vec<u8> {
    let image = image::RgbaImage::from_pixel(8, 8, image::Rgba([220, 40, 40, 255]));
    let mut bytes = Vec::new();
    image::DynamicImage::ImageRgba8(image)
        .write_to(
            &mut std::io::Cursor::new(&mut bytes),
            image::ImageFormat::Png,
        )
        .unwrap();
    bytes
}

#[tokio::test]
async fn generate_blurhash_reads_the_stored_image() {
    let store = Arc::new(MemoryObjectStore::default());
    let png = solid_red_png();
    assert!(store.put("images/red.png", png).await.is_ok());
    let service = StorageService::new(store, "https://cdn.example.test", CancellationToken::new());
    let recorded = service.generate_blurhash("images/red.png").await.unwrap();
    assert_eq!(
        recorded.blurhash.as_deref(),
        Some("LTPJVz|_fQ|_|_sofQsofQfQfQfQ")
    );
    assert_eq!(recorded.width, Some(8));
    assert_eq!(recorded.height, Some(8));
    assert_eq!(recorded.url, "https://cdn.example.test/images/red.png");
    assert!(recorded.id.is_none());
    assert!(matches!(
        service.generate_blurhash("notes/draft.md").await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        service.generate_blurhash("images/missing.png").await,
        Err(AppError::NotFound)
    ));
}

#[derive(Default)]
struct MemoryUploads {
    files: Mutex<Vec<UploadFile>>,
    attached: Mutex<Vec<UploadFile>>,
}

#[async_trait]
impl UploadRepository for MemoryUploads {
    async fn upsert(&self, file: UploadFile) -> Result<UploadFile, AppError> {
        let mut files = self
            .files
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        if let Some(existing) = files.iter_mut().find(|stored| stored.s3_key == file.s3_key) {
            *existing = file.clone();
        } else {
            files.push(file.clone());
        }
        Ok(file)
    }

    async fn find_by_id(&self, id: uuid::Uuid) -> Result<Option<UploadFile>, AppError> {
        Ok(self
            .files
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .iter()
            .find(|file| file.id == id)
            .cloned())
    }

    async fn find_by_ids(&self, _ids: &[uuid::Uuid]) -> Result<Vec<UploadFile>, AppError> {
        Ok(Vec::new())
    }

    async fn find_by_keys(&self, keys: &[String]) -> Result<Vec<UploadFile>, AppError> {
        Ok(self
            .files
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .iter()
            .filter(|file| keys.iter().any(|key| key == &file.s3_key))
            .cloned()
            .collect())
    }

    async fn find_by_public_url(&self, _url: &str) -> Result<Option<UploadFile>, AppError> {
        Ok(None)
    }

    async fn find_by_public_urls(&self, _urls: &[String]) -> Result<Vec<UploadFile>, AppError> {
        Ok(Vec::new())
    }

    async fn delete_by_key(&self, _key: &str) -> Result<(), AppError> {
        Ok(())
    }

    async fn is_referenced(&self, _id: uuid::Uuid) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn rename_prefix(
        &self,
        _old_prefix: &str,
        _new_prefix: &str,
        _url_prefix: &str,
    ) -> Result<(), AppError> {
        Ok(())
    }

    async fn replace_body_refs(
        &self,
        _owner_kind: &str,
        _owner_id: uuid::Uuid,
        _upload_ids: &[uuid::Uuid],
    ) -> Result<(), AppError> {
        Ok(())
    }

    async fn attach_unlinked_images(&self, file: &UploadFile) -> Result<(), AppError> {
        self.attached
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .push(file.clone());
        Ok(())
    }
}

#[tokio::test]
async fn generate_blurhash_saves_the_hash_on_the_upload() {
    let store = Arc::new(MemoryObjectStore::default());
    let png = solid_red_png();
    assert!(store.put("images/red.png", png).await.is_ok());
    let uploads = Arc::new(MemoryUploads::default());
    let owner = uuid::Uuid::new_v4();
    uploads.files.lock().unwrap().push(UploadFile {
        id: owner,
        s3_key: "images/red.png".to_owned(),
        public_url: "https://cdn.example.test/images/red.png".to_owned(),
        filename: "red.png".to_owned(),
        directory_path: "images".to_owned(),
        content_type: "image/png".to_owned(),
        byte_size: 12,
        width: None,
        height: None,
        blurhash: None,
        created_by: Some(owner),
    });
    let service = StorageService::new(store, "https://cdn.example.test", CancellationToken::new())
        .with_uploads(uploads.clone());
    let recorded = service.generate_blurhash_for_id(owner).await.unwrap();
    assert_eq!(recorded.id, Some(owner));
    assert_eq!(
        recorded.blurhash.as_deref(),
        Some("LTPJVz|_fQ|_|_sofQsofQfQfQfQ")
    );
    let saved = uploads.find_by_id(owner).await.unwrap().unwrap();
    assert_eq!(
        saved.blurhash.as_deref(),
        Some("LTPJVz|_fQ|_|_sofQsofQfQfQfQ")
    );
    assert_eq!(saved.created_by, Some(owner));
    assert_eq!(saved.byte_size, 12);
    let attached = uploads
        .attached
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    assert_eq!(attached.len(), 1);
    assert_eq!(attached[0].id, owner);
    assert_eq!(
        attached[0].blurhash.as_deref(),
        Some("LTPJVz|_fQ|_|_sofQsofQfQfQfQ")
    );
}

#[tokio::test]
async fn heic_upload_is_stored_as_jpeg() {
    let store = Arc::new(MemoryObjectStore::default());
    let service = StorageService::new(
        store.clone(),
        "https://cdn.example.test",
        CancellationToken::new(),
    );
    let png = solid_red_png();
    let kept = service
        .put_recorded("images/red.png", "image/png", png, None)
        .await
        .unwrap();
    assert_eq!(kept.key, "images/red.png");
    assert_eq!(kept.content_type, "image/png");
    assert_eq!(kept.width, Some(8));

    let heic = std::fs::read(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/tests/fixtures/solid-red.heic"
    ))
    .unwrap();
    let recorded = service
        .put_recorded(
            "articles/1791098747690-IMG_8021.HEIC",
            "image/heic",
            heic,
            None,
        )
        .await
        .unwrap();
    assert_eq!(recorded.key, "articles/1791098747690-IMG_8021.jpg");
    assert_eq!(recorded.content_type, "image/jpeg");
    assert_eq!(recorded.width, Some(8));
    assert_eq!(recorded.height, Some(8));
    assert!(recorded.blurhash.is_some());
    assert!(
        recorded
            .url
            .ends_with("/articles/1791098747690-IMG_8021.jpg")
    );
    let stored = store
        .state()
        .objects
        .get("articles/1791098747690-IMG_8021.jpg")
        .cloned()
        .unwrap();
    assert!(stored.starts_with(&[0xFF, 0xD8, 0xFF]));

    let rejected = service
        .put_recorded(
            "articles/bad.heic",
            "image/heic",
            b"not a heic image".to_vec(),
            None,
        )
        .await;
    assert!(matches!(rejected, Err(AppError::InvalidInput(_))));
}
