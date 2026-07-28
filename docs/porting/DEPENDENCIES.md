# Dependency Map

Exact versions live in `backend/Cargo.toml`/`Cargo.lock`. A version is retained
only after its representative behavior passes.

## Settled architecture

| Go responsibility/dependency | Rust target | Behavior boundary |
| --- | --- | --- |
| Fiber + Fiber WebSocket | Axum, Tower, Tower HTTP, Tokio | Routing, middleware order, extraction, serialized WS writes, shutdown |
| GORM + postgres driver + `lib/pq` | Diesel, `diesel-async`, Deadpool | PostgreSQL rows, arrays/JSONB, transactions, pool behavior |
| pgvector-go | `pgvector` Diesel feature | `vector(1536)`, distance/operator classes |
| Goose | Diesel CLI + `diesel_migrations` | generate/diff, run/revert/redo, embedded runner, ledger adoption |
| swaggo/Fiber Swagger | Utoipa, `utoipa-axum`, Swagger UI | Same route declaration emits runtime route and OpenAPI |
| golang-jwt | `jsonwebtoken` | claim names, algorithms, expiry and error semantics |
| x/crypto bcrypt | `bcrypt` | existing password hashes remain verifiable |
| AWS Go S3 | AWS SDK for Rust S3 | R2 endpoint/path style, credentials, object operations |
| MCP Go | official `rmcp` | stdio and streamable HTTP transports |
| OpenAI/Anthropic/Google SDKs | local `reqwest` adapters | exact payloads, SSE order, tools, usage, errors, cancellation |
| OpenTelemetry | `tracing`, OpenTelemetry Rust, OTLP | boundary spans, propagation, OTLP HTTP |

## Trial-dependent choices

| Go dependency | Candidate | Must prove before pinning |
| --- | --- | --- |
| goquery | `scraper` | selectors, text extraction, malformed HTML |
| html-to-markdown | `html2md` or local adapter | whitespace, links, headings, lists |
| validator | `garde` or `validator` | every tag/custom rule and field error shape |
| PDF reader | `pdf-extract` or `lopdf` adapter | page order, text, malformed/encrypted errors |
| robfig/cron | application-owned Tokio scheduler | six-field seconds syntax, no overlap, disabled default, shutdown |
| sergi/go-diff | `similar` or `diffy` | exact edit/patch semantics |
| goldmark | `comrak` or `pulldown-cmark` | required GFM output |
| libsql client | `libsql` | conversion binary batching, resume and errors |

Provider SDK replacement is intentionally not one Rust SDK per Go SDK. The
streaming trial decides the minimum local adapters and features.
