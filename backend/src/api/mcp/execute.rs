use std::{
    cell::RefCell,
    rc::Rc,
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU32, Ordering},
        mpsc,
    },
    time::{Duration, Instant},
};

use axum::{
    body::Body,
    http::{Request, Response, header},
};
use http_body_util::{BodyExt, LengthLimitError, Limited};
use rquickjs::{
    CaughtError, Context, Ctx, Exception, Function, Object, Promise, Runtime, Value,
    context::intrinsic, function::Rest,
};
use serde::Deserialize;
use serde_json::{Map, Value as Json};
use tower::ServiceExt;

use super::catalog;

const MEMORY_LIMIT_BYTES: usize = 16 * 1024 * 1024;
const STACK_LIMIT_BYTES: usize = 256 * 1024;
const WALL_LIMIT: Duration = Duration::from_secs(20);
const MAX_CALLS: u32 = 50;
const MAX_CODE_BYTES: usize = 64 * 1024;
const MAX_RESULT_BYTES: usize = 1024 * 1024;
const MAX_RESPONSE_BYTES: usize = 1024 * 1024;
const MAX_REQUEST_BYTES: usize = 1024 * 1024;
const MAX_CONSOLE_LINES: usize = 200;
const MAX_CONSOLE_BYTES: usize = 64 * 1024;
const MAX_CONSOLE_LINE: usize = 2_000;
const METHODS_WITH_BODY: &[&str] = &["DELETE", "PATCH", "POST", "PUT"];
const METHODS: &[&str] = &["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"];

pub(super) struct Job {
    pub api: axum::Router,
    pub authorization: String,
    pub code: String,
}

pub(super) struct Outcome {
    pub result: Json,
    pub console: Vec<String>,
}

/// Run `job` on this thread. The caller puts it on a dedicated thread so QuickJS
/// cannot stall the tokio workers. `blog.request` hops back onto `handle`.
pub(super) fn run(handle: tokio::runtime::Handle, job: Job) -> Result<Outcome, String> {
    if job.code.len() > MAX_CODE_BYTES {
        return Err(format!("code exceeded {MAX_CODE_BYTES} bytes"));
    }
    if job.code.as_bytes().contains(&0) {
        return Err("code must not contain null bytes".to_owned());
    }
    let runtime = Runtime::new().map_err(|error| format!("could not start JavaScript: {error}"))?;
    runtime.set_memory_limit(MEMORY_LIMIT_BYTES);
    runtime.set_max_stack_size(STACK_LIMIT_BYTES);
    let timed_out = Arc::new(AtomicBool::new(false));
    let deadline = Instant::now() + WALL_LIMIT;
    runtime.set_interrupt_handler(Some(Box::new({
        let timed_out = Arc::clone(&timed_out);
        move || {
            if Instant::now() >= deadline {
                timed_out.store(true, Ordering::Relaxed);
                true
            } else {
                false
            }
        }
    })));
    let context = Context::custom::<(
        intrinsic::Eval,
        intrinsic::Json,
        intrinsic::MapSet,
        intrinsic::Promise,
        intrinsic::RegExp,
    )>(&runtime)
    .map_err(|error| format!("could not start JavaScript: {error}"))?;
    context.with(|ctx| evaluate(ctx, handle, job, &timed_out, deadline))
}

fn evaluate(
    ctx: Ctx<'_>,
    handle: tokio::runtime::Handle,
    job: Job,
    timed_out: &AtomicBool,
    deadline: Instant,
) -> Result<Outcome, String> {
    let logs = Rc::new(RefCell::new(Vec::new()));
    install_host(&ctx, &logs, handle, job.api, job.authorization, deadline)
        .map_err(|error| explain(&ctx, error, timed_out.load(Ordering::Relaxed)))?;
    let source = format!("(async () => {{\n{}\n}})()", job.code);
    let promise = ctx
        .eval::<Promise, _>(source)
        .map_err(|error| explain(&ctx, error, timed_out.load(Ordering::Relaxed)))?;
    let value = drive(&ctx, &promise, timed_out)?;
    let result = json_value(&ctx, value, timed_out)?;
    Ok(Outcome {
        result,
        console: logs.borrow().clone(),
    })
}

pub(super) fn tool_text(outcome: &Outcome) -> Result<String, String> {
    let payload = serde_json::json!({
        "result": outcome.result,
        "console": outcome.console,
    });
    let text = serde_json::to_string_pretty(&payload)
        .map_err(|_| "result could not be encoded".to_owned())?;
    if text.len() > MAX_RESULT_BYTES {
        return Err(format!("result exceeded {MAX_RESULT_BYTES} bytes"));
    }
    Ok(text)
}

struct Host {
    handle: tokio::runtime::Handle,
    api: axum::Router,
    authorization: String,
    calls: Arc<AtomicU32>,
    deadline: Instant,
}

fn install_host<'js>(
    ctx: &Ctx<'js>,
    logs: &Rc<RefCell<Vec<String>>>,
    handle: tokio::runtime::Handle,
    api: axum::Router,
    authorization: String,
    deadline: Instant,
) -> rquickjs::Result<()> {
    install_console(ctx, logs)?;
    let host = Host {
        handle,
        api,
        authorization,
        calls: Arc::new(AtomicU32::new(0)),
        deadline,
    };
    let request = Function::new(
        ctx.clone(),
        move |ctx: Ctx<'js>, input: Value<'js>| -> rquickjs::Result<Value<'js>> {
            blog_request(&ctx, input, &host)
        },
    )?
    .with_name("request")?;
    let blog = Object::new(ctx.clone())?;
    blog.set("request", request)?;
    ctx.globals().set("blog", blog)?;
    Ok(())
}

fn install_console<'js>(ctx: &Ctx<'js>, logs: &Rc<RefCell<Vec<String>>>) -> rquickjs::Result<()> {
    let console = Object::new(ctx.clone())?;
    for name in ["debug", "error", "info", "log", "warn"] {
        let logs = Rc::clone(logs);
        let function = Function::new(ctx.clone(), move |ctx: Ctx<'js>, args: Rest<Value<'js>>| {
            record_console(&logs, &ctx, &args);
        })?
        .with_name(name)?;
        console.set(name, function)?;
    }
    ctx.globals().set("console", console)?;
    Ok(())
}

fn record_console<'js>(logs: &Rc<RefCell<Vec<String>>>, ctx: &Ctx<'js>, args: &Rest<Value<'js>>) {
    let Ok(mut logs) = logs.try_borrow_mut() else {
        return;
    };
    if logs.len() >= MAX_CONSOLE_LINES
        || logs
            .last()
            .is_some_and(|line| line == "[console truncated]")
    {
        return;
    }
    let mut line = String::new();
    for (index, arg) in args.0.iter().enumerate() {
        if index > 0 {
            line.push(' ');
        }
        line.push_str(&display_value(ctx, arg));
        if line.len() > MAX_CONSOLE_LINE {
            line.truncate(MAX_CONSOLE_LINE);
            line.push_str("...");
            break;
        }
    }
    let used: usize = logs.iter().map(String::len).sum();
    if used + line.len() > MAX_CONSOLE_BYTES {
        logs.push("[console truncated]".to_owned());
        return;
    }
    logs.push(line);
}

fn display_value<'js>(ctx: &Ctx<'js>, value: &Value<'js>) -> String {
    if value.is_undefined() {
        return "undefined".to_owned();
    }
    if value.is_null() {
        return "null".to_owned();
    }
    if let Some(text) = value.as_string() {
        return text
            .to_string()
            .unwrap_or_else(|_| value.type_name().to_owned());
    }
    if let Some(boolean) = value.as_bool() {
        return boolean.to_string();
    }
    if let Some(number) = value.as_int() {
        return number.to_string();
    }
    if let Some(number) = value.as_float()
        && number.is_finite()
    {
        return number.to_string();
    }
    match ctx.json_stringify(value.clone()) {
        Ok(Some(text)) => text
            .to_string()
            .unwrap_or_else(|_| value.type_name().to_owned()),
        _ => {
            // A failed stringify leaves the exception pending.
            let _ = ctx.catch();
            value.type_name().to_owned()
        }
    }
}

fn blog_request<'js>(
    ctx: &Ctx<'js>,
    input: Value<'js>,
    host: &Host,
) -> rquickjs::Result<Value<'js>> {
    let mut call = parse_call(ctx, input)?;
    call.method = call.method.trim().to_ascii_uppercase();
    call.path = call.path.trim().to_owned();
    if Instant::now() >= host.deadline {
        return Err(fail(ctx, "execution interrupted after 20 seconds"));
    }
    let attempt = host.calls.fetch_add(1, Ordering::Relaxed);
    if attempt >= MAX_CALLS {
        return Err(fail(
            ctx,
            &format!("blog.request exceeded {MAX_CALLS} calls"),
        ));
    }
    if let Some(reason) = catalog::execute_path_block(&call.path) {
        let shown = truncate(&call.path, 200);
        return Err(fail(
            ctx,
            &format!("blog.request refused {shown}: {reason}"),
        ));
    }
    let payload = dispatch(
        &host.handle,
        host.api.clone(),
        &host.authorization,
        &call,
        host.deadline,
    )
    .map_err(|message| fail(ctx, &message))?;
    let bytes = serde_json::to_vec(&payload).map_err(|error| {
        fail(
            ctx,
            &format!("blog.request result could not be encoded: {error}"),
        )
    })?;
    ctx.json_parse(bytes)
}

#[derive(Deserialize)]
struct BlogCall {
    method: String,
    path: String,
    #[serde(default)]
    query: Option<Map<String, Json>>,
    #[serde(default)]
    body: Option<Json>,
}

fn parse_call<'js>(ctx: &Ctx<'js>, input: Value<'js>) -> rquickjs::Result<BlogCall> {
    if !input.is_object() {
        return Err(fail(ctx, "blog.request expects an object"));
    }
    let Some(text) = ctx.json_stringify(input)? else {
        return Err(fail(ctx, "blog.request arguments must be JSON"));
    };
    let text = text
        .to_string()
        .map_err(|error| fail(ctx, &error.to_string()))?;
    serde_json::from_str(&text)
        .map_err(|error| fail(ctx, &format!("blog.request arguments are invalid: {error}")))
}

fn dispatch(
    handle: &tokio::runtime::Handle,
    api: axum::Router,
    authorization: &str,
    call: &BlogCall,
    deadline: Instant,
) -> Result<Json, String> {
    let request = build_request(authorization, call)?;
    let (sender, receiver) = mpsc::sync_channel(1);
    handle.spawn(async move {
        let outcome = perform(api, request).await;
        let _ = sender.send(outcome);
    });
    let remaining = deadline.saturating_duration_since(Instant::now());
    if remaining.is_zero() {
        return Err("execution interrupted after 20 seconds".to_owned());
    }
    match receiver.recv_timeout(remaining) {
        Ok(result) => result,
        Err(mpsc::RecvTimeoutError::Timeout) => {
            Err("execution interrupted after 20 seconds".to_owned())
        }
        Err(mpsc::RecvTimeoutError::Disconnected) => {
            Err("blog.request failed before the API responded".to_owned())
        }
    }
}

fn build_request(authorization: &str, call: &BlogCall) -> Result<Request<Body>, String> {
    let method = call.method.trim().to_ascii_uppercase();
    if !METHODS.contains(&method.as_str()) {
        return Err(format!("blog.request method {method} is not allowed"));
    }
    let query = match &call.query {
        Some(query) => encode_query(query)?,
        None => String::new(),
    };
    let uri = format!("{}{query}", call.path.trim());
    let body = request_body(&method, call.body.as_ref())?;
    let mut builder = Request::builder()
        .method(method.as_str())
        .uri(uri)
        .header(header::AUTHORIZATION, authorization)
        .header(header::ACCEPT, "application/json")
        .header(header::HOST, "localhost");
    if body.is_some() {
        builder = builder.header(header::CONTENT_TYPE, "application/json");
    }
    builder
        .body(body.unwrap_or_else(Body::empty))
        .map_err(|error| format!("blog.request could not build the request: {error}"))
}

fn request_body(method: &str, body: Option<&Json>) -> Result<Option<Body>, String> {
    let Some(body) = body else {
        return Ok(None);
    };
    if body.is_null() {
        return Ok(None);
    }
    if !METHODS_WITH_BODY.contains(&method) {
        return Err("blog.request only sends a body for POST, PUT, PATCH, and DELETE".to_owned());
    }
    let bytes = serde_json::to_vec(body)
        .map_err(|error| format!("blog.request body is not JSON: {error}"))?;
    if bytes.len() > MAX_REQUEST_BYTES {
        return Err(format!(
            "blog.request body exceeded {MAX_REQUEST_BYTES} bytes"
        ));
    }
    Ok(Some(Body::from(bytes)))
}

fn encode_query(query: &Map<String, Json>) -> Result<String, String> {
    let mut encoded = String::new();
    for (key, value) in query {
        if key.is_empty() || key.bytes().any(|byte| byte.is_ascii_control()) {
            return Err(
                "query names must be non-empty and not contain control characters".to_owned(),
            );
        }
        let text = match value {
            Json::String(text) => text.clone(),
            Json::Number(number) => number.to_string(),
            Json::Bool(boolean) => boolean.to_string(),
            Json::Null | Json::Array(_) | Json::Object(_) => {
                return Err("query values must be strings, numbers, or booleans".to_owned());
            }
        };
        if !encoded.is_empty() {
            encoded.push('&');
        }
        encoded.push_str(&percent_encode(key));
        encoded.push('=');
        encoded.push_str(&percent_encode(&text));
    }
    if encoded.is_empty() {
        Ok(String::new())
    } else {
        Ok(format!("?{encoded}"))
    }
}

fn percent_encode(value: &str) -> String {
    let mut encoded = String::with_capacity(value.len());
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                encoded.push(byte as char);
            }
            _ => encoded.push_str(&format!("%{byte:02X}")),
        }
    }
    encoded
}

async fn perform(api: axum::Router, request: Request<Body>) -> Result<Json, String> {
    let response = match api.oneshot(request).await {
        Ok(response) => response,
        Err(error) => match error {},
    };
    read_response(response).await
}

async fn read_response(response: Response<Body>) -> Result<Json, String> {
    let status = u16::from(response.status());
    let limited = Limited::new(response.into_body(), MAX_RESPONSE_BYTES + 1);
    let collected = limited.collect().await.map_err(|error| {
        if error.downcast_ref::<LengthLimitError>().is_some() {
            format!("blog.request response exceeded {MAX_RESPONSE_BYTES} bytes")
        } else {
            "blog.request response could not be read".to_owned()
        }
    })?;
    let bytes = collected.to_bytes();
    if bytes.len() > MAX_RESPONSE_BYTES {
        return Err(format!(
            "blog.request response exceeded {MAX_RESPONSE_BYTES} bytes"
        ));
    }
    let body = if bytes.is_empty() {
        Json::Null
    } else {
        serde_json::from_slice(&bytes)
            .unwrap_or_else(|_| Json::String(String::from_utf8_lossy(&bytes).into_owned()))
    };
    Ok(serde_json::json!({ "status": status, "body": body }))
}

fn drive<'js>(
    ctx: &Ctx<'js>,
    promise: &Promise<'js>,
    timed_out: &AtomicBool,
) -> Result<Value<'js>, String> {
    loop {
        if timed_out.load(Ordering::Relaxed) {
            return Err("execution interrupted after 20 seconds".to_owned());
        }
        match promise.result::<Value>() {
            Some(Ok(value)) => return Ok(value),
            Some(Err(error)) => {
                return Err(explain(ctx, error, timed_out.load(Ordering::Relaxed)));
            }
            None => {}
        }
        if !ctx.execute_pending_job() {
            if timed_out.load(Ordering::Relaxed) {
                return Err("execution interrupted after 20 seconds".to_owned());
            }
            return Err("snippet is waiting on a promise that will not resolve".to_owned());
        }
    }
}

fn json_value<'js>(
    ctx: &Ctx<'js>,
    value: Value<'js>,
    timed_out: &AtomicBool,
) -> Result<Json, String> {
    if value.is_undefined() || value.is_null() {
        return Ok(Json::Null);
    }
    let text = match ctx.json_stringify(value) {
        Ok(Some(text)) => text
            .to_string()
            .map_err(|error| format!("return value is not JSON: {error}"))?,
        Ok(None) => return Err("return value is not JSON".to_owned()),
        Err(error) => return Err(explain(ctx, error, timed_out.load(Ordering::Relaxed))),
    };
    serde_json::from_str(&text).map_err(|_| "return value is not JSON".to_owned())
}

fn explain<'js>(ctx: &Ctx<'js>, error: rquickjs::Error, timed_out: bool) -> String {
    if timed_out {
        return "execution interrupted after 20 seconds".to_owned();
    }
    if matches!(error, rquickjs::Error::Allocation) {
        return memory_limit();
    }
    if matches!(error, rquickjs::Error::WouldBlock) {
        return "snippet is waiting on a promise that will not resolve".to_owned();
    }
    let message = match CaughtError::from_error(ctx, error) {
        CaughtError::Exception(exception) => exception
            .message()
            .unwrap_or_else(|| "JavaScript exception".to_owned()),
        CaughtError::Value(value) => format!("JavaScript exception ({})", value.type_name()),
        CaughtError::Error(error) => error.to_string(),
    };
    if message.to_ascii_lowercase().contains("out of memory") {
        return memory_limit();
    }
    truncate(&message, 1_500)
}

fn memory_limit() -> String {
    "execution exceeded the 16 MiB memory limit".to_owned()
}

fn fail<'js>(ctx: &Ctx<'js>, message: &str) -> rquickjs::Error {
    Exception::throw_message(ctx, message)
}

fn truncate(value: &str, max: usize) -> String {
    if value.len() <= max {
        return value.to_owned();
    }
    let mut end = max;
    while !value.is_char_boundary(end) && end > 0 {
        end -= 1;
    }
    format!("{}...", &value[..end])
}
