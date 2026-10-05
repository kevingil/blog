use std::{
    io::ErrorKind,
    path::{Path, PathBuf},
    process::Stdio,
    time::Duration,
};

use tokio::{fs, io::AsyncReadExt, process::Command, time::timeout};
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

const CONVERT_TIMEOUT: Duration = Duration::from_secs(20);

/// Render's native image does not include `heif-convert`. It does include
/// libvips and ImageMagick, both built with HEIF support on Debian 12.
const CONVERTERS: &[Converter] = &[
    Converter::HeifConvert,
    Converter::Vips,
    Converter::ImageMagick("convert"),
    Converter::ImageMagick("magick"),
];

#[derive(Clone, Copy)]
enum Converter {
    HeifConvert,
    Vips,
    ImageMagick(&'static str),
}

impl Converter {
    fn name(self) -> &'static str {
        match self {
            Self::HeifConvert => "heif-convert",
            Self::Vips => "vips",
            Self::ImageMagick(program) => program,
        }
    }

    fn command(self, input: &Path, output: &Path) -> Command {
        let mut command = Command::new(self.name());
        match self {
            Self::HeifConvert | Self::ImageMagick(_) => {
                command.arg(input).arg(output);
            }
            Self::Vips => {
                command
                    .arg("jpegsave")
                    .arg(input)
                    .arg(output)
                    .arg("--Q")
                    .arg("90");
            }
        }
        command
    }
}

enum ConversionError {
    Missing,
    TimedOut,
    Rejected(String),
    Unavailable(String),
}

pub(super) async fn transcode_heif_to_jpeg(bytes: &[u8]) -> Result<Vec<u8>, AppError> {
    let id = Uuid::new_v4();
    let input = TempFile::new(format!("blog-heif-{id}.heic"));
    let output = TempFile::new(format!("blog-heif-{id}.jpg"));
    fs::write(input.path(), bytes).await.map_err(|error| {
        tracing::error!(%error, "failed to write HEIC upload for conversion");
        AppError::internal(error)
    })?;

    let mut rejected = false;
    let mut unavailable = None;
    for converter in CONVERTERS {
        match run_converter(*converter, input.path(), output.path()).await {
            Ok(jpeg) => return Ok(jpeg),
            Err(ConversionError::Missing) => {}
            Err(ConversionError::TimedOut) => {
                tracing::error!(tool = converter.name(), "HEIC converter timed out");
                return Err(AppError::InvalidInput(
                    "Could not read this image".to_owned(),
                ));
            }
            Err(ConversionError::Rejected(detail)) => {
                rejected = true;
                tracing::warn!(
                    tool = converter.name(),
                    %detail,
                    "HEIC converter could not read the upload"
                );
            }
            Err(ConversionError::Unavailable(detail)) => {
                unavailable = Some(format!("{}: {detail}", converter.name()));
                tracing::error!(
                    tool = converter.name(),
                    %detail,
                    "failed to start HEIC converter"
                );
            }
        }
    }
    if rejected {
        return Err(AppError::InvalidInput(
            "Could not read this image".to_owned(),
        ));
    }
    if let Some(detail) = unavailable {
        return Err(AppError::internal(detail));
    }
    let tried = CONVERTERS
        .iter()
        .map(|converter| converter.name())
        .collect::<Vec<_>>()
        .join(", ");
    tracing::error!(tried = %tried, "no HEIC converter is installed");
    Err(AppError::internal(format!(
        "no HEIC converter is installed (tried {tried})"
    )))
}

async fn run_converter(
    converter: Converter,
    input: &Path,
    output: &Path,
) -> Result<Vec<u8>, ConversionError> {
    let _ = fs::remove_file(output).await;
    let mut child = converter
        .command(input, output)
        .kill_on_drop(true)
        .stdout(Stdio::null())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|error| {
            if error.kind() == ErrorKind::NotFound {
                ConversionError::Missing
            } else {
                ConversionError::Unavailable(error.to_string())
            }
        })?;
    let stderr = child.stderr.take();
    let status = match timeout(CONVERT_TIMEOUT, child.wait()).await {
        Ok(Ok(status)) => status,
        Ok(Err(error)) => return Err(ConversionError::Unavailable(error.to_string())),
        Err(_) => return Err(ConversionError::TimedOut),
    };
    let detail = stderr_detail(stderr).await;
    if !status.success() {
        let code = status
            .code()
            .map(|code| code.to_string())
            .unwrap_or_else(|| "signal".to_owned());
        return Err(ConversionError::Rejected(format!("exit {code}: {detail}")));
    }
    let jpeg = fs::read(output)
        .await
        .map_err(|error| ConversionError::Rejected(format!("no jpeg output: {error}; {detail}")))?;
    if !is_jpeg(&jpeg) {
        return Err(ConversionError::Rejected(format!(
            "output was not a jpeg; {detail}"
        )));
    }
    Ok(jpeg)
}

async fn stderr_detail(stderr: Option<tokio::process::ChildStderr>) -> String {
    let Some(mut stderr) = stderr else {
        return String::new();
    };
    let mut bytes = Vec::new();
    if stderr.read_to_end(&mut bytes).await.is_err() {
        return String::new();
    }
    let collapsed = String::from_utf8_lossy(&bytes)
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ");
    const LIMIT: usize = 300;
    if collapsed.chars().count() <= LIMIT {
        return collapsed;
    }
    let end = collapsed
        .char_indices()
        .nth(LIMIT)
        .map(|(index, _)| index)
        .unwrap_or(collapsed.len());
    format!("{}…", &collapsed[..end])
}

fn is_jpeg(bytes: &[u8]) -> bool {
    bytes.len() >= 3 && bytes[0] == 0xFF && bytes[1] == 0xD8 && bytes[2] == 0xFF
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
    use super::{
        CONVERTERS, ConversionError, is_heif_payload, is_heif_upload, is_jpeg, with_jpeg_extension,
    };
    use std::process::Stdio;

    use tokio::{fs, process::Command};

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

    #[tokio::test]
    async fn each_installed_converter_writes_a_jpeg() {
        let path = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/solid-red.heic");
        let bytes = fs::read(path).await.unwrap_or_default();
        assert!(!bytes.is_empty(), "solid-red.heic fixture is missing");
        let mut converted = 0_u32;
        for converter in CONVERTERS {
            if !converter_installed(*converter).await {
                continue;
            }
            let id = uuid::Uuid::new_v4();
            let input = super::TempFile::new(format!("blog-heif-test-{id}.heic"));
            let output = super::TempFile::new(format!("blog-heif-test-{id}.jpg"));
            let written = fs::write(input.path(), &bytes).await;
            assert!(written.is_ok(), "write HEIC fixture copy");
            let result = super::run_converter(*converter, input.path(), output.path()).await;
            let detail = match &result {
                Ok(_) => String::new(),
                Err(error) => failure_text(error),
            };
            assert!(
                result.as_ref().is_ok_and(|jpeg| is_jpeg(jpeg)),
                "{} failed: {detail}",
                converter.name()
            );
            converted += 1;
        }
        assert!(converted > 0, "no HEIC converter is installed");
    }

    async fn converter_installed(converter: super::Converter) -> bool {
        Command::new(converter.name())
            .arg("--help")
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .await
            .is_ok()
    }

    fn failure_text(error: &ConversionError) -> String {
        match error {
            ConversionError::Missing => "missing".to_owned(),
            ConversionError::TimedOut => "timed out".to_owned(),
            ConversionError::Rejected(detail) | ConversionError::Unavailable(detail) => {
                detail.clone()
            }
        }
    }
}
