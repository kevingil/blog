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

const CONVERT_TIMEOUT: Duration = Duration::from_secs(60);

/// iOS 18 HEIC (iPhone 16 and later) needs libheif >= 1.18. Debian 12 ships
/// 1.15.1, which rejects those files, and Render's ImageMagick and libvips
/// are built without a HEIF decoder. `scripts/vendor-heif.sh` copies a newer
/// `heif-convert` into `backend/opt/heif` during the native build.
fn converters() -> Vec<Converter> {
    let mut converters = Vec::new();
    if let Some(vendored) = vendored_heif() {
        converters.push(Converter::Heif {
            program: vendored.program,
            library_path: Some(vendored.library_path),
            plugin_path: Some(vendored.plugin_path),
        });
    }
    converters.push(Converter::Heif {
        program: PathBuf::from("heif-convert"),
        library_path: None,
        plugin_path: None,
    });
    converters.push(Converter::Vips);
    converters.push(Converter::ImageMagick("convert"));
    converters.push(Converter::ImageMagick("magick"));
    converters
}

struct VendoredHeif {
    program: PathBuf,
    library_path: PathBuf,
    plugin_path: PathBuf,
}

fn vendored_heif() -> Option<VendoredHeif> {
    vendored_heif_in(vendored_roots())
}

fn vendored_heif_in(roots: impl IntoIterator<Item = PathBuf>) -> Option<VendoredHeif> {
    roots.into_iter().find_map(|root| {
        let program = root.join("bin/heif-convert");
        let library_path = root.join("lib");
        let plugin_path = root.join("plugins");
        if program.is_file() && library_path.is_dir() && plugin_path.is_dir() {
            Some(VendoredHeif {
                program,
                library_path,
                plugin_path,
            })
        } else {
            None
        }
    })
}

fn vendored_roots() -> Vec<PathBuf> {
    let mut roots = Vec::new();
    if let Ok(exe) = std::env::current_exe() {
        let mut dir = exe.parent().map(Path::to_path_buf);
        for _ in 0..6 {
            let Some(current) = dir else {
                break;
            };
            roots.push(current.join("opt/heif"));
            roots.push(current.join("backend/opt/heif"));
            dir = current.parent().map(Path::to_path_buf);
        }
    }
    roots.push(PathBuf::from("backend/opt/heif"));
    roots.push(PathBuf::from("opt/heif"));
    roots
}

enum Converter {
    Heif {
        program: PathBuf,
        library_path: Option<PathBuf>,
        plugin_path: Option<PathBuf>,
    },
    Vips,
    ImageMagick(&'static str),
}

impl Converter {
    fn label(&self) -> &'static str {
        match self {
            Self::Heif {
                plugin_path: Some(_),
                ..
            } => "vendored-heif-convert",
            Self::Heif { .. } => "heif-convert",
            Self::Vips => "vips",
            Self::ImageMagick("magick") => "magick",
            Self::ImageMagick(_) => "convert",
        }
    }

    fn command(&self, input: &Path, output: &Path) -> Command {
        match self {
            Self::Heif {
                program,
                library_path,
                plugin_path,
            } => {
                let mut command =
                    heif_process(program, library_path.as_deref(), plugin_path.as_deref());
                command.arg(input).arg(output);
                command
            }
            Self::Vips => {
                let mut command = Command::new("vips");
                command
                    .arg("jpegsave")
                    .arg(input)
                    .arg(output)
                    .arg("--Q")
                    .arg("90");
                command
            }
            Self::ImageMagick(program) => {
                let mut command = Command::new(program);
                command.arg(input).arg(output);
                command
            }
        }
    }

    fn probe(&self) -> Command {
        match self {
            Self::Heif {
                program,
                library_path,
                plugin_path,
            } => {
                let mut command =
                    heif_process(program, library_path.as_deref(), plugin_path.as_deref());
                command.arg("-v");
                command
            }
            Self::Vips => {
                let mut command = Command::new("vips");
                command.arg("--help");
                command
            }
            Self::ImageMagick(program) => {
                let mut command = Command::new(program);
                command.arg("--help");
                command
            }
        }
    }
}

fn heif_process(
    program: &Path,
    library_path: Option<&Path>,
    plugin_path: Option<&Path>,
) -> Command {
    let mut command = Command::new(program);
    if let Some(library_path) = library_path {
        command.env("LD_LIBRARY_PATH", library_path);
    }
    if let Some(plugin_path) = plugin_path {
        command.env("LIBHEIF_PLUGIN_PATH", plugin_path);
    }
    command
}

/// ImageMagick and libheif 1.15 fail on valid iPhone photos. Those messages
/// mean the tool cannot decode HEIC, so another converter should be tried.
fn converter_lacks_heif(detail: &str) -> bool {
    let detail = detail.to_ascii_lowercase();
    [
        "too many auxiliary",
        "unable to read file",
        "no images defined",
        "moov atom not found",
        "invalid data found when processing",
        "no decoding plugin",
        "decoding plugin",
        "unsupported codec",
        "error while loading shared libraries",
    ]
    .iter()
    .any(|needle| detail.contains(needle))
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

    let converters = converters();
    let mut rejected = false;
    let mut limited = false;
    let mut unavailable = None;
    for converter in &converters {
        match run_converter(converter, input.path(), output.path()).await {
            Ok(jpeg) => return Ok(jpeg),
            Err(ConversionError::Missing) => {}
            Err(ConversionError::TimedOut) => {
                tracing::error!(tool = converter.label(), "HEIC converter timed out");
                return Err(AppError::InvalidInput(
                    "Could not read this image".to_owned(),
                ));
            }
            Err(ConversionError::Rejected(detail)) => {
                if converter_lacks_heif(&detail) {
                    limited = true;
                } else {
                    rejected = true;
                }
                tracing::warn!(
                    tool = converter.label(),
                    %detail,
                    "HEIC converter could not read the upload"
                );
            }
            Err(ConversionError::Unavailable(detail)) => {
                unavailable = Some(format!("{}: {detail}", converter.label()));
                tracing::error!(
                    tool = converter.label(),
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
    if limited {
        tracing::error!("installed HEIC tools cannot decode this photo");
        return Err(AppError::internal(
            "HEIC decoder cannot read this photo; libheif >= 1.18 is required",
        ));
    }
    if let Some(detail) = unavailable {
        return Err(AppError::internal(detail));
    }
    let tried = converters
        .iter()
        .map(Converter::label)
        .collect::<Vec<_>>()
        .join(", ");
    tracing::error!(tried = %tried, "no HEIC converter is installed");
    Err(AppError::internal(format!(
        "no HEIC converter is installed (tried {tried})"
    )))
}

async fn run_converter(
    converter: &Converter,
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
        ConversionError, converter_lacks_heif, is_heif_payload, is_heif_upload, is_jpeg,
        vendored_heif_in, with_jpeg_extension,
    };
    use std::process::Stdio;

    use tokio::fs;

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
        for converter in super::converters() {
            if !converter_installed(&converter).await {
                continue;
            }
            let id = uuid::Uuid::new_v4();
            let input = super::TempFile::new(format!("blog-heif-test-{id}.heic"));
            let output = super::TempFile::new(format!("blog-heif-test-{id}.jpg"));
            let written = fs::write(input.path(), &bytes).await;
            assert!(written.is_ok(), "write HEIC fixture copy");
            let result = super::run_converter(&converter, input.path(), output.path()).await;
            let detail = match &result {
                Ok(_) => String::new(),
                Err(error) => failure_text(error),
            };
            assert!(
                result.as_ref().is_ok_and(|jpeg| is_jpeg(jpeg)),
                "{} failed: {detail}",
                converter.label()
            );
            converted += 1;
        }
        assert!(converted > 0, "no HEIC converter is installed");
    }

    #[test]
    fn vendored_converter_is_used_when_its_layout_is_present() {
        let root = std::env::temp_dir().join(format!("blog-heif-layout-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir_all(root.join("bin")).unwrap();
        std::fs::create_dir_all(root.join("lib")).unwrap();
        std::fs::create_dir_all(root.join("plugins")).unwrap();
        std::fs::write(root.join("bin/heif-convert"), b"#!/bin/sh\n").unwrap();
        let found = vendored_heif_in([root.clone()]);
        let _ = std::fs::remove_dir_all(&root);
        assert!(
            found.is_some(),
            "vendored heif-convert layout was not found"
        );
        assert!(vendored_heif_in([root]).is_none());
    }

    #[test]
    fn old_decoders_are_not_treated_as_a_bad_upload() {
        assert!(converter_lacks_heif(
            "exit 1: Could not read HEIF/AVIF file: Invalid input: Unspecified: Too many auxiliary image references"
        ));
        assert!(converter_lacks_heif(
            "exit 1: magick2vips: libMagick error: magick2vips: unable to read file \"/tmp/photo.heic\""
        ));
        assert!(converter_lacks_heif(
            "exit 1: convert-im6.q16: no images defined `/tmp/photo.jpg'"
        ));
        assert!(!converter_lacks_heif("exit 1: output was not a jpeg"));
    }

    async fn converter_installed(converter: &super::Converter) -> bool {
        converter
            .probe()
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
