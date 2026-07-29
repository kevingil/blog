# Go Reference Baseline

## Source and toolchain

- Canonical source: <https://github.com/kevingil/blog-go>
- Pinned upstream revision:
  [`67e918381c0331d29be39f88b0a62e8f8d6f1d10`](https://github.com/kevingil/blog-go/commit/67e918381c0331d29be39f88b0a62e8f8d6f1d10)
- Pinned upstream source directory: `backend/`
- Imported content commit: `3df5ccd823b49cbb157dbbeb091bec906bcdc1a8`
- Behavior-preserving rename commit: `a286dac2fc14035c6e1c87953f5d1c6d99d7a6b5`
- Required reference toolchain: Go `1.24.4`
- Local host toolchain observed during preparation: Go `1.24.1`; it is not the
  acceptance oracle.
- Docker reference: `golang:1.24.4-bookworm`, resolved during the rename build
  to manifest digest
  `sha256:10f549dc8489597aa7ed2b62008199bb96717f52a8e8434ea035d5b44368f8a6`.
- Rust builder: `rust:1.92.0-bookworm` at
  `sha256:e90e846de4124376164ddfbaab4b0774c7bdeef5e738866295e5a90a34a307a2`.
- Frontend builder: `oven/bun:1.3.14` at
  `sha256:e10577f0db68676a7024391c6e5cb4b879ebd17188ab750cf10024a6d700e5c4`.
- Frontend runtime: `nginx:1.29.5-alpine` at
  `sha256:1eff5a5f3fcf8431a0abb7eddf5471fec24e5e1905a2581aeacdb07a4479b92b`.
- PostgreSQL runtime: `postgres:17.4-bookworm` at
  `sha256:304ab813518754228f9f792f79d6da36359b82d8ecf418096c636725f8c930ad`.
- Goose reference: official `v3.27.1` Linux release binaries, verified against
  the upstream SHA-256 checksums for `arm64` and `x86_64`.

Before its deletion, the renamed Go tree built successfully with
`go build ./...`. The renamed root Dockerfile also built successfully as
`blog-go-rename-baseline`. The preserved Go suite passed all 101 top-level
cases with an isolated writable build cache:

```sh
git clone https://github.com/kevingil/blog-go.git ../blog-go-reference
git -C ../blog-go-reference checkout --detach 67e918381c0331d29be39f88b0a62e8f8d6f1d10
cd ../blog-go-reference/backend
GOCACHE=/tmp/blog-go-build-cache go test ./...
```

The imported article-service test was updated only to pass the explicit
`publishedAt` argument already required by the imported service signature; its
mock expectation and behavioral assertion remain unchanged.

Immediately before deletion, the pinned upstream `backend/` directory and the
former local `backend-go/` directory both contained 235 files. Their only diff
was that documented test-only compatibility update. The runtime source,
migrations, generated Swagger documents, dependencies, and fixtures matched.

## Inventory

- 222 Go source files.
- 19 Go test files with 101 top-level `Test*` functions.
- 98 registered HTTP routes and one WebSocket upgrade route.
- 63 checked Swagger operations, all without stable operation IDs.
- 21 database model structs, 17 repository interfaces, and seven Goose SQL
  migration files.
- Checked Swagger JSON SHA-256:
  `720ee922df74933028c3e35d784aee42f6012343c762b5b5823202bfec36e663`.

## Known contract drift

- Swagger omits all data-source, insight, task-run, worker, WebSocket, and
  root/meta routes.
- Runtime auth differs from Swagger on organization reads, site-settings GET,
  and logout.
- Active frontend image-generation calls and
  `PUT /blog/articles/{id}/context` have no registered Go route.
- Generated frontend code has no live non-generated importer; handwritten
  services remain authoritative consumers until each row is reconciled.
- Detailed rows and unresolved decisions live in `CONTRACTS.tsv`.

## Historical baseline commands and results

These commands record the acceptance evidence produced while the local
`backend-go/` copy existed. Use the pinned external checkout above for any new
Go-only investigation.

```sh
cd ../blog-go-reference/backend
go build ./...
```

Passed after the rename.

```sh
docker build -t blog-go-rename-baseline ../blog-go-reference
```

Passed on Docker Desktop arm64 using the Go 1.24.4 builder.

```sh
cd ../blog-go-reference/backend
go test ./pkg/core/insight -run '^TestService_' -count=1 -v
```

All 11 top-level insight service tests passed. They remain a non-blocking Rust
behavior lane only; compile, schema, route, OpenAPI, auth, and lifecycle remain
blocking.

## Disposable PostgreSQL parity

Both migration systems were exercised against clean databases built from the
same `postgres:17.4-bookworm` runtime with pgvector `0.8.2`:

- Goose `v3.27.1` applied the seven original migrations and recorded the
  version-zero sentinel plus the seven applied versions.
- Diesel `2.3.1` applied the seven mechanically extracted migrations and
  recorded the same seven version strings.
- The canonical schema fingerprint, excluding only migration ledgers, matched:
  `81fa7f13268ae949c1c627f62ea860d4fe7dfb72698a4f40c5b4706cadd07b29`.

The fingerprint includes application columns, constraints, indexes, and
extension metadata. This establishes local Goose-to-Diesel DDL equivalence; it
does not substitute for the required read-only Supabase fingerprint before
production adoption.

The Compose parity gate now creates both databases from empty PostgreSQL 17.4
volumes, runs Goose and Diesel independently, requires the exact fingerprint
above, and only then starts the language-neutral contract suite. The final
parity run passed all 93 HTTP, authentication-boundary, and WebSocket cases
against both live servers.

## Rust and full-stack acceptance evidence

The Rust blocking matrix passed every selected library, repository, service,
HTTP, provider, storage, worker, WebSocket, lifecycle, and database test:

```sh
TEST_DATABASE_URL=postgres://blog:blog@127.0.0.1:55432/blog \
TEST_S3_ENDPOINT=http://127.0.0.1:9000 \
./scripts/test-rust.sh blocking
```

The exact insight exception lane also ran and passed all 11 recorded cases:

```sh
./scripts/test-rust.sh insights
```

The release/compiler and contract-generation gates pass:

```sh
cd backend
cargo fmt --all -- --check
cargo check --locked
cargo clippy --locked --lib --bins -- -D warnings
cargo build --locked --release

cd ..
./scripts/generate-client.sh
node scripts/verify-openapi-contracts.mjs

cd frontend
bun run build
```

The verifier reports 100 OpenAPI operations with 100 unique stable operation
IDs and complete coverage of settled HTTP ledger rows. The generated client is
the active frontend transport. The frontend release build succeeds; its
existing CSS minifier and large-chunk messages remain warnings.

The default Compose stack was started from empty volumes, served a healthy Rust
API and frontend, and completed disposable register/login, authenticated
article creation, agent submission, and image-generation/object-storage smoke
flows. The Rust process runs as UID 10001. Graceful and forced shutdown are
also covered by the server and upgraded-WebSocket lifecycle tests.

External provider behavior is exercised through deterministic local OpenAI,
Groq-compatible Responses/SSE, Anthropic, Gemini, Vertex AI, Exa, image, and
S3-compatible fixtures. The active insight worker keeps the Go composition's
separate `GROQ_API_KEY` configuration, `openai/gpt-oss-120b` model, and
2,000-token output cap instead of substituting the copilot's OpenAI client. MCP
remains inventory-only because the active Go configuration contains no MCP
servers. No parity test contacts production services.

## Pending baseline evidence

No further static source information is required from the original Go
repository before deleting the local copy: every one of its 222 Go files has a
recorded disposition, every contract row is classified, all seven migration
bodies are retained in Diesel form, and the exact source revision remains
available upstream.

The read-only Supabase settings/fingerprint remains pending because no
production database credentials were made available to this checkout.
Production adoption must stop until that capture and the documented
schema-identical staging rehearsal pass.

The original plan also requires a pre-port Go performance baseline before
setting latency, throughput, memory, pool, and binary-size acceptance budgets.
That evidence was not captured before bulk implementation, so no retrospective
budget is invented here. Performance comparison and the actual Fly deployment
or later Render staging/cutover remain explicit production-release gates rather
than local implementation claims.
