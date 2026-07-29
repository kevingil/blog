# Go-to-Rust Porting Rules

This is the durable semantic guide for the atomic backend rewrite. The Go
oracle is the `backend/` directory at
[`kevingil/blog-go@67e9183`](https://github.com/kevingil/blog-go/tree/67e918381c0331d29be39f88b0a62e8f8d6f1d10/backend);
the Rust implementation is `backend/` in this repository. Mechanical parity
precedes refactoring.

## Legacy source reference

- Canonical repository: <https://github.com/kevingil/blog-go>
- Pinned source revision:
  `67e918381c0331d29be39f88b0a62e8f8d6f1d10`
- Pinned source directory: `backend/`
- Local import commit: `3df5ccd823b49cbb157dbbeb091bec906bcdc1a8`
- Local rename commit: `a286dac2fc14035c6e1c87953f5d1c6d99d7a6b5`

The `backend-go/...` paths in `MODULE_MAP.tsv` and `OWNERSHIP.tsv` are stable
historical identifiers for files under the pinned source's `backend/...`
directory; they do not imply that a local `backend-go/` directory exists.

The upstream snapshot was compared against the former local tree before
deletion. All 235 files matched except
`pkg/core/article/service_test.go`, whose local copy contained only the
documented compatibility update that passes the already-required
`publishedAt` argument. No runtime implementation differed.

`docker-compose.parity.yml` fetches this exact revision as a named Docker build
context for reproducible Go/Diesel migration and HTTP/WebSocket parity checks.
Changing the pinned revision requires reviewing and updating the porting
ledgers first.

## Layer and dependency direction

```text
api (Axum, DTOs, OpenAPI, transport error mapping)
  -> core (domain values, use cases, consumer-owned I/O ports)
       <- database and integrations (Diesel/provider adapters)

bootstrap is the only production composition root.
```

Handlers never construct repositories or clients. `AppState` contains
already-constructed services and managers. Narrow state types implement
`FromRef<AppState>`. Core modules cannot import `api`, `bootstrap`, Diesel, or
concrete integrations.

## Value and serialization mapping

| Go | Rust | Rule |
| --- | --- | --- |
| value with meaningful zero | value/newtype | Preserve the zero only when the Go contract distinguishes it. |
| pointer / nullable SQL | `Option<T>` | SQL nullability comes from migrations, not Go pointer declarations. |
| `omitempty` | `#[serde(skip_serializing_if = "Option::is_none")]` or collection predicate | Apply per observed JSON contract; never blanket-skip false/zero. |
| slice | `Vec<T>` | Preserve `null` versus `[]` per response evidence. |
| map | `BTreeMap` or `HashMap` | Use `BTreeMap` where deterministic JSON/order is externally observed. |
| `uuid.UUID` | typed ID newtype over `uuid::Uuid` | Serialize lowercase hyphenated UUID; nil UUID is not automatically `None`. |
| `time.Time` | `chrono::DateTime<Utc>` | Preserve RFC3339 precision and omission behavior. |
| JSON number | explicit integer/float type | Do not silently narrow or change integer-to-float encoding. |

Transport DTOs, domain values, and database rows are distinct when validation,
nullability, or representation differs. Conversions are explicit and fallible
where data can be invalid.

## Errors

Core returns typed `thiserror` errors. Repositories map not-found, uniqueness,
foreign-key, and serialization errors to domain errors. API code alone maps
domain errors to the status/envelope recorded in `CONTRACTS.tsv`. Bootstrap and
binaries may use `anyhow` for contextual startup errors.

Request and worker paths do not use `unwrap`, `expect`, panic, `todo!`,
`unimplemented!`, or fake success values for external input.

## Database and migrations

- Runtime persistence and schema use Diesel; async request paths use
  `diesel-async` with Deadpool.
- SQL nullability/defaults are taken from the seven migration bodies.
- `vector(1536)` uses `pgvector` Diesel types. PostgreSQL `INTEGER[]` maps to
  `i32`, including nullable array elements where generated schema requires it.
- Transactions stay in the repository/use case that owns the invariant.
- GORM not-found becomes an explicit `Option` or domain `NotFound`, according
  to each interface. `RowsAffected` checks remain explicit.
- Raw query column order is represented by a named `QueryableByName` row.
- Fire-and-forget version/task writes become owned tasks with cancellation,
  observed results, and bounded shutdown.
- Goose SQL bodies are copied byte-for-byte after removing only control
  comments. Existing databases are ledger-adopted; DDL is never replayed.

## Async and ownership

- `context.Context` cancellation becomes a `CancellationToken` child of the
  owning request, WebSocket session, copilot request, worker run, or
  `Application`.
- Deadlines use `tokio::time::timeout` or `select!` with the same duration.
- Every spawned task is stored in an owning `JoinSet`/`JoinHandle`; results are
  observed and shutdown joins are bounded.
- Use `mpsc` for one consumer, `broadcast` for multiple subscribers, and
  `oneshot` for one result. Capacity and lag/drop behavior come from
  `OWNERSHIP.tsv`.
- Never hold any lock guard across `.await`. Prefer a single owner task or
  snapshot under a short synchronous lock.
- WebSocket writes are serialized by exactly one writer task.

## HTTP and OpenAPI

- Axum/Tower middleware preserves runtime order, auth, body limits, CORS,
  security headers, timeouts, and error JSON.
- Every operation is registered through `utoipa_axum::OpenApiRouter`, with an
  explicit stable `operationId`, tag, actual DTO schemas, status codes, and
  security requirements.
- Offline export and runtime serving call the same OpenAPI constructor.
- Runtime router evidence wins only for rows classified `active-contract`.
  `missing-implementation` and `approved-change` rows require an explicit
  product decision.

## Approved auth security corrections

The Rust auth trial intentionally corrects these defects in the Go reference;
parity tests must treat them as explicit improvements rather than accidental
drift:

- Registration validates name, RFC email syntax, and password length even
  though the live Go `RegisterRequest` accidentally lacks validation tags.
- Passwords are bcrypt cost 10 for existing capacity parity, reject inputs
  longer than 72 bytes instead of truncating them, and run on a bounded blocking
  pool rather than Tokio executor threads.
- JWTs require `exp`, use zero expiry leeway, and accept HS256 only. Missing or
  malformed `sub` claims return 401 instead of reaching Go's panic path.
- Empty `AUTH_SECRET` values fail startup.
- Identity/password/delete writes are endpoint-scoped and conditional so
  concurrent requests cannot restore stale password, profile, role, or
  organization fields.
- Database failures are sanitized at the HTTP boundary; raw SQL/driver errors
  are never returned to clients.

These corrections do not change successful response envelopes. Account,
password, and deletion requests continue to accept both JSON and the
multipart-form payloads used by the active frontend.

The authentication representative trial passed both independent read-only
reviews after its local and Docker PostgreSQL gates: 15 service/HTTP tests, one
transactional repository test, and the dependency-light health test.

## Article repository trial findings

The article repository trial preserves the Go database contract for lookup,
search, pagination, order fallback, tags, nullable arrays, JSON values,
`vector(1536)`, drafts, publish state, snapshots, versions, and reverts. The
PostgreSQL integration case exercises both null and empty representations and
concurrent version creation against the parity schema.

Version side effects reserve a bounded task identifier before the parent write,
use a parameterized per-article advisory transaction lock, retain real task
handles, and report every failure intersecting the caller's snapshot. Shutdown
closes admission, cancels late producers, joins admitted work through one
deadline, aborts overdue work, and fully reaps all handles. Failure retention
and out-of-order completion tracking are fixed-size and conservatively report
ambiguous overflow rather than returning false success.

The application HTTP server likewise owns every accepted connection in a
`JoinSet`, stops acceptance on cancellation, requests Hyper HTTP/1 graceful
shutdown with upgrade support, then aborts and fully reaps overdue connections.
Axum upgrade callbacks are outside the Hyper connection future; the WebSocket
trial therefore owns upgraded sessions in a separate application supervisor
before repository shutdown.

## WebSocket trial findings

The public Axum upgrade callback performs one nonblocking handoff and returns;
it never owns a provider, repository, or long-lived session. An
application-owned supervisor retains every session, writer, and stream task in
one `JoinSet`. Root cancellation closes admission, cancels all child tokens,
joins through a checked deadline, aborts overdue work, and fully reaps before
repository shutdown.

Exactly one writer owns the socket sink. Producers share a capacity-256 `mpsc`
sender and use `try_send`, so a full queue drops the newest frame without
reordering existing frames. Cancellation remains the highest-priority event.
When both a ping and queued data are ready, writer preference alternates so
neither can starve; text frames retain FIFO order. Worker updates subscribe
before snapshot collection, then enqueue the acknowledgment, every initial
status, and queued updates in that order.

`WebSocketConfig` is validated at construction. Zero capacities/durations,
channel sizes beyond Tokio's permit limit, and durations that cannot form an
`Instant` deadline are typed startup errors. Runtime deadline construction is
also checked. Critical supervisor success, error, or panic outside root
cancellation cancels the application root and remains observable during the
bounded application-task drain.

The representative trial passed both independent reviews after 16 focused
contract cases, repeated slow-writer and race stress runs, a real network
upgrade/shutdown case in Docker, tracked HTTP shutdown tests, strict Clippy,
and all-target compilation.

## Provider and tool adapters

Provider-specific DTOs and stream parsers live behind consumer-owned ports.
Preserve delta order, lazy/eager behavior, tool-call grouping, usage fields,
request payloads, cancellation, terminal errors, and retry behavior.
Deterministic fixture services—not live providers—are parity oracles.

## Review rule

A row becomes `parity-passed` only after the Rust destination compiles, its
targeted tests execute, and its settled contract case passes against Go and
Rust. Deletion requires an evidence-backed, explicitly approved disposition.
