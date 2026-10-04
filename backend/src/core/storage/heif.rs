use std::{io::ErrorKind, path::PathBuf, process::Stdio, time::Duration};

use tokio::{fs, process::Command, time::timeout};
use uuid::Uuid;

use crate::error::AppError;

const HEIC_BRANDS: &[&[u8]] = &[
    b"heic", b"heix", b"hevc", b"hevx", b"heim", b"heis", b"hevm", b"hevs",
];

/// iPhone photos arrive as HEIC/HEIF. Browsers other than Safari cannot render
/// that container, so uploads are rewritten to JPEG before they are stored.
pub(super) fn is_heif_upload(key: &str, content_type: &str, bytes: &[u8]) -> bool {
    if bytes.is_empty() || key.ends_with('/') {
        return false;
    }
    is_heif_type(content_type) || is_heif_extension(key) || is_heif_payload(bytes)
}

pub(super) fn with_jpeg_extension(key: &str) -> String {
    let (directory, name) = match key.rsplit_once('/') {
        Some((directory, name)) => (Some(directory), name),
        None => (None, key),
    };
    let stem = match name.rsplit_once('.') {
        Some((stem, extension)) if !stem.is_empty() && !extension.is_empty() => stem,
        _ => name,
    };
    match directory {
        Some(directory) if !directory.is_empty() => format!("{directory}/{stem}.jpg"),
        _ => format!("{stem}.jpg"),
    }
}

pub(super) async fn transcode_heif_to_jpeg(bytes: &[u8]) -> Result<Vec<u8>, AppError> {
    let id = Uuid::new_v4();
    let input = TempFile::new(format!("blog-heif-{id}.heic"));
    let output = TempFile::new(format!("blog-heif-{id}.jpg"));
    fs::write(input.path(), bytes).await.map_err(|error| {
        tracing::error!(%error, "failed to write HEIC upload for conversion");
        AppError::internal(error)
    })?;
    let mut child = Command::new("heif-convert")
        .arg(input.path())
        .arg(output.path())
        .kill_on_drop(true)
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|error| {
            if error.kind() == ErrorKind::NotFound {
                tracing::error!("heif-convert is not installed");
                AppError::internal("heif-convert is not installed")
            } else {
                tracing::error!(%error, "failed to start heif-convert");
                AppError::internal(error)
            }
        })?;
    let status = timeout(Duration::from_secs(20), child.wait())
        .await
        .map_err(|_| {
            tracing::error!("heif-convert timed out");
            AppError::InvalidInput("Could not read this image".to_owned())
        })?
        .map_err(|error| {
            tracing::error!(%error, "heif-convert failed");
            AppError::InvalidInput("Could not read this image".to_owned())
        })?;
    if !status.success() {
        tracing::warn!(code = status.code(), "heif-convert rejected an upload");
        return Err(AppError::InvalidInput(
            "Could not read this image".to_owned(),
        ));
    }
    let jpeg = fs::read(output.path()).await.map_err(|error| {
        tracing::error!(%error, "heif-convert produced no jpeg");
        AppError::InvalidInput("Could not read this image".to_owned())
    })?;
    if jpeg.len() < 3 || jpeg[0] != 0xFF || jpeg[1] != 0xD8 || jpeg[2] != 0xFF {
        return Err(AppError::InvalidInput(
            "Could not read this image".to_owned(),
        ));
    }
    Ok(jpeg)
}

fn is_heif_type(content_type: &str) -> bool {
    let declared = content_type
        .split(';')
        .next()
        .unwrap_or("")
        .trim()
        .to_ascii_lowercase();
    matches!(
        declared.as_str(),
        "image/heic" | "image/heif" | "image/heic-sequence" | "image/heif-sequence"
    )
}

fn is_heif_extension(key: &str) -> bool {
    let name = key.rsplit('/').next().unwrap_or(key);
    let extension = match name.rsplit_once('.') {
        Some((_, extension)) => extension,
        None => return false,
    };
    matches!(extension.to_ascii_lowercase().as_str(), "heic" | "heif")
}

fn is_heif_payload(bytes: &[u8]) -> bool {
    let brands = ftyp_brands(bytes);
    if brands.iter().any(|brand| HEIC_BRANDS.contains(brand)) {
        return true;
    }
    let generic = brands
        .iter()
        .any(|brand| *brand == b"mif1" || *brand == b"msf1");
    let avif = brands
        .iter()
        .any(|brand| *brand == b"avif" || *brand == b"avis");
    generic && !avif
}

fn ftyp_brands(bytes: &[u8]) -> Vec<&[u8]> {
    if bytes.len() < 16 || &bytes[4..8] != b"ftyp" {
        return Vec::new();
    }
    let declared = u32::from_be_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]) as usize;
    let end = declared.min(bytes.len());
    if end < 16 {
        return Vec::new();
    }
    let mut brands = vec![&bytes[8..12]];
    let mut offset = 16;
    while offset + 4 <= end {
        brands.push(&bytes[offset..offset + 4]);
        offset += 4;
    }
    brands
}

struct TempFile {
    path: PathBuf,
}

impl TempFile {
    fn new(name: String) -> Self {
        Self {
            path: std::env::temp_dir().join(name),
        }
    }

    fn path(&self) -> &std::path::Path {
        &self.path
    }
}

impl Drop for TempFile {
    fn drop(&mut self) {
        let _ = std::fs::remove_file(&self.path);
    }
}

#[cfg(test)]
mod tests {
    use super::{is_heif_payload, is_heif_upload, with_jpeg_extension};

    #[test]
    fn heic_container_and_declared_type_are_uploads_to_convert() {
        let mut bytes = vec![0, 0, 0, 24];
        bytes.extend_from_slice(b"ftypheic");
        bytes.extend_from_slice(&[0, 0, 0, 0]);
        bytes.extend_from_slice(b"mif1");
        assert!(is_heif_payload(&bytes));
        assert!(is_heif_upload(
            "articles/IMG_8021.HEIC",
            "image/heic",
            &bytes
        ));
        assert!(is_heif_upload(
            "articles/photo.heif",
            "application/octet-stream",
            b"not-a-container-but-named-heif"
        ));
        assert!(!is_heif_upload("notes/draft.md", "text/plain", b"hello"));
        assert!(!is_heif_payload(b"not heic"));
    }

    #[test]
    fn avif_container_stays_untouched() {
        let mut bytes = vec![0, 0, 0, 24];
        bytes.extend_from_slice(b"ftypavif");
        bytes.extend_from_slice(&[0, 0, 0, 0]);
        bytes.extend_from_slice(b"mif1");
        assert!(!is_heif_payload(&bytes));
        assert!(!is_heif_upload("images/cover.avif", "image/avif", &bytes));
    }

    #[test]
    fn jpeg_key_keeps_the_directory_and_stem() {
        assert_eq!(
            with_jpeg_extension("articles/1791098747690-IMG_8021.HEIC"),
            "articles/1791098747690-IMG_8021.jpg"
        );
        assert_eq!(with_jpeg_extension("photo"), "photo.jpg");
    }
}
