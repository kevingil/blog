use serde_json::{Value, json};
use utoipa::openapi::{
    OpenApi,
    path::{Operation, Parameter, PathItem},
};

const ALLOWED_TAGS: &[&str] = &["articles", "pages", "projects", "sources", "storage"];
const EXCLUDED_TAGS: &[&str] = &[
    "insights",
    "data-sources",
    "workers",
    "task-runs",
    "agent",
    "auth",
    "mcp",
    "oauth",
];
const ALLOWED_PREFIXES: &[&str] = &[
    "/blog",
    "/pages",
    "/dashboard/pages",
    "/projects",
    "/sources",
    "/dashboard/sources",
    "/storage",
];
const EXCLUDED_PREFIXES: &[&str] = &[
    "/insights",
    "/data-sources",
    "/workers",
    "/task-runs",
    "/agent",
    "/auth",
    "/mcp",
    "/oauth",
    "/.well-known",
];
const MAX_SCHEMA_DEPTH: usize = 8;

/// Operations from the live OpenAPI document that match `query`.
pub fn search(document: &OpenApi, query: &str, limit: usize) -> Vec<Value> {
    let needle = query.trim().to_ascii_lowercase();
    if needle.is_empty() || limit == 0 {
        return Vec::new();
    }
    let components = json!({
        "components": serde_json::to_value(&document.components).unwrap_or(Value::Null)
    });
    let mut matches = Vec::new();
    for (path, item) in &document.paths.paths {
        let path_parameters = item.parameters.as_deref();
        for (method, operation) in operations(item) {
            let Some(operation) = operation else {
                continue;
            };
            let tags = operation.tags.as_deref().unwrap_or(&[]);
            if !include_operation(path, tags) {
                continue;
            }
            let rank = score(&needle, path, method, operation);
            if rank == 0 {
                continue;
            }
            matches.push((rank, path.clone(), method, operation, path_parameters));
        }
    }
    matches.sort_by(|left, right| {
        right
            .0
            .cmp(&left.0)
            .then_with(|| left.1.cmp(&right.1))
            .then_with(|| left.2.cmp(right.2))
    });
    matches.truncate(limit);
    matches
        .into_iter()
        .map(|(_, path, method, operation, path_parameters)| {
            render(method, &path, operation, path_parameters, &components)
        })
        .collect()
}

pub(crate) fn include_operation(path: &str, tags: &[String]) -> bool {
    if excluded_path(path) || tags.iter().any(|tag| EXCLUDED_TAGS.contains(&tag.as_str())) {
        return false;
    }
    tags.iter().any(|tag| ALLOWED_TAGS.contains(&tag.as_str())) || allowed_path(path)
}

fn excluded_path(path: &str) -> bool {
    EXCLUDED_PREFIXES
        .iter()
        .any(|prefix| path_has_prefix(path, prefix))
}

fn allowed_path(path: &str) -> bool {
    ALLOWED_PREFIXES
        .iter()
        .any(|prefix| path_has_prefix(path, prefix))
}

fn path_has_prefix(path: &str, prefix: &str) -> bool {
    path == prefix
        || path
            .strip_prefix(prefix)
            .is_some_and(|rest| rest.starts_with('/'))
}

/// Why `execute` must not dispatch `path`, or `None` when the path is on the
/// same articles, pages, projects, sources, and storage allowlist as search.
pub(crate) fn execute_path_block(path: &str) -> Option<&'static str> {
    if path.is_empty() || !path.starts_with('/') || !path.is_ascii() {
        return Some("path must be an absolute ASCII path starting with /");
    }
    if path.len() > 2048
        || path
            .bytes()
            .any(|byte| byte.is_ascii_control() || matches!(byte, b' ' | b'\\' | b'?' | b'#'))
    {
        return Some("path must be a single absolute path without a query or fragment");
    }
    let Some(decoded) = decode_path(path) else {
        return Some("path is not valid percent-encoding");
    };
    if !decoded.is_ascii()
        || decoded
            .bytes()
            .any(|byte| byte.is_ascii_control() || matches!(byte, b' ' | b'\\' | b'?' | b'#'))
        || decoded.contains("://")
    {
        return Some("path must be a single absolute path without a query or fragment");
    }
    let mut segments: Vec<&str> = decoded.split('/').skip(1).collect();
    if segments.last() == Some(&"") {
        segments.pop();
    }
    if segments
        .iter()
        .any(|segment| segment.is_empty() || *segment == "." || *segment == "..")
    {
        return Some("path must not contain '.', '..', or empty segments");
    }
    if excluded_path(&decoded) || !allowed_path(&decoded) {
        return Some("path is outside articles, pages, projects, sources, and storage");
    }
    None
}

fn decode_path(path: &str) -> Option<String> {
    let bytes = path.as_bytes();
    let mut decoded = Vec::with_capacity(bytes.len());
    let mut index = 0;
    while index < bytes.len() {
        if bytes[index] != b'%' {
            decoded.push(bytes[index]);
            index += 1;
            continue;
        }
        if index + 2 >= bytes.len() {
            return None;
        }
        let hex = std::str::from_utf8(&bytes[index + 1..index + 3]).ok()?;
        let byte = u8::from_str_radix(hex, 16).ok()?;
        if byte == 0 {
            return None;
        }
        decoded.push(byte);
        index += 3;
    }
    String::from_utf8(decoded).ok()
}

fn operations(item: &PathItem) -> [(&'static str, Option<&Operation>); 8] {
    [
        ("DELETE", item.delete.as_ref()),
        ("GET", item.get.as_ref()),
        ("HEAD", item.head.as_ref()),
        ("OPTIONS", item.options.as_ref()),
        ("PATCH", item.patch.as_ref()),
        ("POST", item.post.as_ref()),
        ("PUT", item.put.as_ref()),
        ("TRACE", item.trace.as_ref()),
    ]
}

fn score(query: &str, path: &str, method: &str, operation: &Operation) -> u32 {
    let mut rank = 0;
    if path.to_ascii_lowercase().contains(query) {
        rank += 50;
    }
    if method.to_ascii_lowercase().contains(query) {
        rank += 10;
    }
    if let Some(operation_id) = operation.operation_id.as_deref() {
        let operation_id = operation_id.to_ascii_lowercase();
        if operation_id == query {
            rank += 80;
        } else if operation_id.contains(query) {
            rank += 40;
        }
    }
    if contains_text(operation.summary.as_deref(), query) {
        rank += 30;
    }
    if contains_text(operation.description.as_deref(), query) {
        rank += 15;
    }
    if operation
        .tags
        .iter()
        .flatten()
        .any(|tag| tag.to_ascii_lowercase().contains(query))
    {
        rank += 10;
    }
    if parameter_names(operation.parameters.as_deref()).any(|name| name.contains(query)) {
        rank += 15;
    }
    rank
}

fn contains_text(value: Option<&str>, query: &str) -> bool {
    value.is_some_and(|text| text.to_ascii_lowercase().contains(query))
}

fn parameter_names(parameters: Option<&[Parameter]>) -> impl Iterator<Item = String> + '_ {
    parameters
        .into_iter()
        .flatten()
        .map(|parameter| parameter.name.to_ascii_lowercase())
}

fn render(
    method: &str,
    path: &str,
    operation: &Operation,
    path_parameters: Option<&[Parameter]>,
    components: &Value,
) -> Value {
    let mut rendered = serde_json::to_value(operation).unwrap_or(Value::Null);
    merge_path_parameters(&mut rendered, path_parameters);
    let mut stack = Vec::new();
    resolve_refs(&mut rendered, components, &mut stack);
    let summary = rendered
        .get("summary")
        .cloned()
        .filter(|value| !value.is_null())
        .or_else(|| rendered.get("description").cloned())
        .unwrap_or(Value::Null);
    json!({
        "method": method,
        "path": path,
        "operationId": rendered.get("operationId").cloned().unwrap_or(Value::Null),
        "summary": summary,
        "parameters": rendered.get("parameters").cloned().unwrap_or_else(|| Value::Array(Vec::new())),
        "requestBody": rendered.get("requestBody").cloned().unwrap_or(Value::Null),
        "responses": rendered.get("responses").cloned().unwrap_or(Value::Null),
    })
}

fn merge_path_parameters(rendered: &mut Value, path_parameters: Option<&[Parameter]>) {
    let Some(path_parameters) = path_parameters else {
        return;
    };
    let Ok(Value::Array(extra)) = serde_json::to_value(path_parameters) else {
        return;
    };
    let Some(object) = rendered.as_object_mut() else {
        return;
    };
    let parameters = object
        .entry("parameters".to_owned())
        .or_insert_with(|| Value::Array(Vec::new()));
    let Value::Array(existing) = parameters else {
        return;
    };
    for parameter in extra {
        let name = parameter.get("name").and_then(Value::as_str);
        let location = parameter.get("in").and_then(Value::as_str);
        let already_present = existing.iter().any(|current| {
            current.get("name").and_then(Value::as_str) == name
                && current.get("in").and_then(Value::as_str) == location
        });
        if !already_present {
            existing.insert(0, parameter);
        }
    }
}

fn resolve_refs(value: &mut Value, components: &Value, stack: &mut Vec<String>) {
    if stack.len() > MAX_SCHEMA_DEPTH {
        return;
    }
    if let Some(reference) = value.get("$ref").and_then(Value::as_str).map(str::to_owned) {
        if !stack.iter().any(|seen| seen == &reference)
            && let Some(mut resolved) = lookup_ref(&reference, components)
        {
            stack.push(reference);
            resolve_refs(&mut resolved, components, stack);
            stack.pop();
            *value = resolved;
        }
        return;
    }
    match value {
        Value::Object(map) => {
            for child in map.values_mut() {
                resolve_refs(child, components, stack);
            }
        }
        Value::Array(items) => {
            for item in items {
                resolve_refs(item, components, stack);
            }
        }
        _ => {}
    }
}

fn lookup_ref(reference: &str, document: &Value) -> Option<Value> {
    let pointer = reference.strip_prefix('#')?;
    let mut cursor = document;
    for segment in pointer.split('/').filter(|segment| !segment.is_empty()) {
        cursor = cursor.get(decode_pointer(segment))?;
    }
    Some(cursor.clone())
}

fn decode_pointer(segment: &str) -> String {
    segment.replace("~1", "/").replace("~0", "~")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::openapi;

    fn paths_for(query: &str) -> Vec<String> {
        search(&openapi::document(), query, 10_000)
            .into_iter()
            .filter_map(|operation| {
                operation
                    .get("path")
                    .and_then(Value::as_str)
                    .map(str::to_owned)
            })
            .collect()
    }

    fn assert_content_paths_only(paths: &[String]) {
        for path in paths {
            assert!(
                !excluded_path(path),
                "{path} is outside articles, pages, projects, sources, and storage"
            );
        }
    }

    #[test]
    fn draft_search_returns_the_draft_operation_and_its_body() {
        let matches = search(&openapi::document(), "draft", 10);
        let mut found = false;
        for operation in &matches {
            if operation.get("method").and_then(Value::as_str) == Some("POST")
                && operation.get("path").and_then(Value::as_str) == Some("/blog/drafts")
            {
                found = true;
                assert_eq!(operation["operationId"], "createDraft");
                let body = operation["requestBody"].to_string();
                assert!(body.contains("title"), "{body}");
                assert!(body.contains("markdown"), "{body}");
                assert!(!body.contains("$ref"), "{body}");
            }
        }
        assert!(found, "POST /blog/drafts missing from {matches:?}");
    }

    #[test]
    fn search_never_returns_excluded_paths() {
        for query in [
            "draft", "insight", "agent", "auth", "login", "worker", "token", "page", "upload",
            "source", "project", "health", "search", "oauth", "task", "data", "e", "a",
        ] {
            let paths = paths_for(query);
            assert_content_paths_only(&paths);
            assert!(
                paths.iter().all(|path| include_operation(path, &[])),
                "{query} returned a path outside the content API: {paths:?}"
            );
        }
    }

    #[test]
    fn the_live_document_hides_excluded_operations() {
        let document = openapi::document();
        let mut visible = 0;
        for (path, item) in &document.paths.paths {
            for (_, operation) in operations(item) {
                let Some(operation) = operation else {
                    continue;
                };
                let tags = operation.tags.as_deref().unwrap_or(&[]);
                if include_operation(path, tags) {
                    visible += 1;
                    assert!(!excluded_path(path), "{path}");
                    assert!(
                        tags.iter()
                            .all(|tag| !EXCLUDED_TAGS.contains(&tag.as_str())),
                        "{path} tags {tags:?}"
                    );
                }
            }
        }
        assert!(visible > 0, "content API produced no operations");
    }

    #[test]
    fn execute_follows_the_search_allowlist() {
        for path in [
            "/blog/drafts",
            "/blog/articles/verification-note",
            "/pages/home",
            "/dashboard/pages",
            "/projects",
            "/sources",
            "/dashboard/sources/1",
            "/storage/upload",
            "/blog/",
        ] {
            assert!(execute_path_block(path).is_none(), "{path}");
        }
        for path in [
            "/auth/me",
            "/mcp",
            "/oauth/token",
            "/insights",
            "/workers/jobs",
            "/agent/chat",
            "/health",
            "/blogs",
            "/blog/../auth/me",
            "/blog/%2e%2e/auth/me",
            "/blog/%2e%2e%2fauth/me",
            "blog/drafts",
            "https://example.test/blog/drafts",
            "/blog/drafts?x=1",
            "/blog//drafts",
        ] {
            assert!(
                execute_path_block(path).is_some(),
                "{path} should be refused"
            );
        }
    }
}
