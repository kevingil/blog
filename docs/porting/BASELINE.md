# Go Reference Baseline

## Source and toolchain

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
- Goose builder: `golang:1.25.7-bookworm` at
  `sha256:564e366a28ad1d70f460a2b97d1d299a562f08707eb0ecb24b659e5bd6c108e1`.

The renamed Go tree built successfully with `go build ./...`. The renamed root
Dockerfile also built successfully as `blog-go-rename-baseline`.

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

## Baseline commands and results

```sh
cd backend-go
go build ./...
```

Passed after the rename.

```sh
docker build -t blog-go-rename-baseline .
```

Passed on Docker Desktop arm64 using the Go 1.24.4 builder.

```sh
cd backend-go
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

## Pending baseline evidence

Performance, request/provider snapshots, fixture checksums, and the Supabase
settings/fingerprint remain pending. Acceptance budgets must be recorded before
bulk performance changes and cannot be invented after the Rust results are
known.
