use super::UploadFile;

/// True when an article image URL refers to this stored object.
///
/// Older articles keep the public URL and leave `upload_file_id` empty.
/// Matching has to accept that URL, a percent-encoded copy of it, and the
/// same object key served from another host in the same bucket.
pub fn urls_match_upload(url: &str, file: &UploadFile) -> bool {
    let article_url = percent_decode(url.trim());
    if article_url.is_empty() {
        return false;
    }
    let public_url = percent_decode(file.public_url.trim());
    if article_url == public_url {
        return true;
    }
    let key = file.s3_key.trim_matches('/');
    if !key.contains('/') {
        return false;
    }
    let Some(article_path) = path_after_host(&article_url) else {
        return false;
    };
    let Some(public_path) = path_after_host(&public_url) else {
        return false;
    };
    let Some((article_bucket, article_key)) = article_path.split_once('/') else {
        return false;
    };
    let Some((public_bucket, _)) = public_path.split_once('/') else {
        return false;
    };
    article_bucket == public_bucket && article_key == key
}

pub fn header_upload<'a>(url: &str, files: &'a [UploadFile]) -> Option<&'a UploadFile> {
    files.iter().find(|file| urls_match_upload(url, file))
}

/// Object keys that could belong to a stored image URL.
pub fn candidate_object_keys(url: &str) -> Vec<String> {
    let decoded = percent_decode(url.trim());
    let Some(path) = path_after_host(&decoded) else {
        return Vec::new();
    };
    let mut keys = Vec::new();
    let mut rest = path;
    while let Some((_, tail)) = rest.split_once('/') {
        if tail.contains('/') {
            keys.push(tail.to_owned());
        }
        rest = tail;
    }
    keys
}

/// `LIKE` patterns for rows that might reference this file. Callers still
/// confirm the match with [`urls_match_upload`].
pub fn unlinked_image_like_patterns(file: &UploadFile) -> [String; 2] {
    let key = file.s3_key.trim_matches('/');
    if !key.contains('/') {
        let exact = escape_like(&file.public_url);
        return [exact.clone(), exact];
    }
    let encoded = percent_encode_path(key);
    let primary = escape_like(&format!("/{key}"));
    let secondary = if encoded == key {
        primary.clone()
    } else {
        escape_like(&format!("/{encoded}"))
    };
    [format!("%{primary}"), format!("%{secondary}")]
}

pub fn percent_decode(value: &str) -> String {
    let bytes = value.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut index = 0;
    while index < bytes.len() {
        if bytes[index] == b'%' && index + 2 < bytes.len() {
            if let Some(decoded) = hex_byte(bytes[index + 1], bytes[index + 2]) {
                out.push(decoded);
                index += 3;
                continue;
            }
        }
        out.push(bytes[index]);
        index += 1;
    }
    String::from_utf8_lossy(&out).into_owned()
}

fn percent_encode_path(path: &str) -> String {
    let mut out = String::new();
    for (index, segment) in path.split('/').enumerate() {
        if index > 0 {
            out.push('/');
        }
        for byte in segment.bytes() {
            if is_unreserved(byte) {
                out.push(char::from(byte));
            } else {
                out.push('%');
                out.push(hex_digit(byte >> 4));
                out.push(hex_digit(byte & 0x0f));
            }
        }
    }
    out
}

fn path_after_host(url: &str) -> Option<&str> {
    let rest = url.split_once("://").map(|(_, rest)| rest).unwrap_or(url);
    let (_, path) = rest.split_once('/')?;
    let path = path.split(['?', '#']).next().unwrap_or(path);
    if path.is_empty() { None } else { Some(path) }
}

fn escape_like(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len());
    for ch in value.chars() {
        if matches!(ch, '%' | '_' | '\\') {
            escaped.push('\\');
        }
        escaped.push(ch);
    }
    escaped
}

fn is_unreserved(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b'.' | b'~')
}

fn hex_digit(value: u8) -> char {
    char::from(b"0123456789ABCDEF"[usize::from(value)])
}

fn hex_byte(high: u8, low: u8) -> Option<u8> {
    Some((hex_value(high)? << 4) | hex_value(low)?)
}

fn hex_value(byte: u8) -> Option<u8> {
    match byte {
        b'0'..=b'9' => Some(byte - b'0'),
        b'a'..=b'f' => Some(byte - b'a' + 10),
        b'A'..=b'F' => Some(byte - b'A' + 10),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::{candidate_object_keys, header_upload, percent_decode, urls_match_upload};
    use crate::core::storage::UploadFile;
    use uuid::Uuid;

    fn file(key: &str, url: &str) -> UploadFile {
        UploadFile {
            id: Uuid::nil(),
            s3_key: key.to_owned(),
            public_url: url.to_owned(),
            filename: "header.png".to_owned(),
            directory_path: "images".to_owned(),
            content_type: "image/png".to_owned(),
            byte_size: 8,
            width: Some(8),
            height: Some(8),
            blurhash: Some("LTPJVz|_fQ|_|_sofQsofQfQfQfQ".to_owned()),
            created_by: None,
        }
    }

    #[test]
    fn exact_and_encoded_urls_match_the_stored_image() {
        let stored = file(
            "images/header.png",
            "http://localhost:9000/blog/images/my header.png",
        );
        assert!(urls_match_upload(
            "http://localhost:9000/blog/images/my header.png",
            &stored
        ));
        assert!(urls_match_upload(
            "http://localhost:9000/blog/images/my%20header.png",
            &stored
        ));
        assert_eq!(
            percent_decode("http://localhost:9000/blog/images/my%20header.png"),
            "http://localhost:9000/blog/images/my header.png"
        );
    }

    #[test]
    fn same_bucket_and_key_match_across_hosts() {
        let stored = file(
            "images/header.png",
            "http://localhost:9000/blog/images/header.png",
        );
        assert!(urls_match_upload(
            "http://127.0.0.1:9000/blog/images/header.png",
            &stored
        ));
        assert_eq!(
            candidate_object_keys("http://127.0.0.1:9000/blog/images/header.png"),
            vec!["images/header.png".to_owned()]
        );
        assert!(header_upload("http://127.0.0.1:9000/blog/images/header.png", &[stored]).is_some());
    }

    #[test]
    fn external_urls_do_not_match_a_local_object() {
        let stored = file(
            "images/header.png",
            "http://localhost:9000/blog/images/header.png",
        );
        assert!(!urls_match_upload(
            "https://cdn.example.test/photos/header.png",
            &stored
        ));
        assert!(!urls_match_upload(
            "https://example.test/header.png",
            &stored
        ));
    }
}
