# Blog Copilot

An agentic blog editor with a React/Bun frontend and an Axum/Rust backend.
The legacy Go backend is no longer vendored. Its pinned source reference and
the retained porting evidence live in `docs/porting/`.

![Blog Copilot](frontend/public/IMG_2718.png)

## Run the full stack

Docker Compose provides PostgreSQL 17.4 with pgvector, Diesel migrations,
deterministic OpenAI/Exa fixtures, MinIO, the Rust API, and the frontend:

```bash
docker compose up --build
```

- Frontend: <http://localhost:3000>
- API: <http://localhost:8080>
- Health: <http://localhost:8080/health>
- Swagger UI: <http://localhost:8080/swagger>
- OpenAPI: <http://localhost:8080/api/openapi.json>
- MinIO console: <http://localhost:9001>

The local database and object store use named volumes. Tests never connect to
the production Supabase database.

## Development

The Rust backend requires the settings shown in `docker-compose.yml`:
`DATABASE_URL`, `AUTH_SECRET`, `OPENAI_API_KEY`, `OPENAI_BASE_URL`,
`EXA_API_KEY`, `EXA_BASE_URL`, `S3_ENDPOINT`, `S3_ACCESS_KEY_ID`,
`S3_ACCESS_KEY_SECRET`, `S3_BUCKET`, and `S3_URL_PREFIX`.

```bash
cd backend
cargo run --locked --bin blog-backend
```

```bash
cd frontend
bun install --frozen-lockfile
bun run dev
```

## Verification

The blocking Rust matrix is partitioned into exact targets so failures identify
their owning domain:

```bash
./scripts/test-rust.sh blocking
./scripts/test-rust.sh insights
```

Run the same blocking matrix in the pinned Docker environment:

```bash
docker compose --profile test run --build --rm test
```

The insight behavior target is always compiled and executed, but CI reports its
11 exact cases separately because that subsystem remains work in progress.

The Go/Rust parity environment uses independent PostgreSQL 17.4 databases. It
fetches the pinned [kevingil/blog-go](https://github.com/kevingil/blog-go)
source as a Docker build context, so the reference implementation does not
remain in this repository:

```bash
docker compose -f docker-compose.parity.yml up --build \
  --abort-on-container-exit --exit-code-from contract-tests contract-tests
```

Porting evidence, contract classifications, migration adoption instructions,
and task ownership live in `docs/porting/`.

## OpenAPI and frontend client

Axum routes and the OpenAPI document share the same Utoipa construction path.
The frontend client is generated from that document:

```bash
./scripts/generate-client.sh
node ./scripts/verify-openapi-contracts.mjs
```

Generated files under `frontend/src/client/` must not be edited by hand.
Frontend service adapters use the generated SDK and add only application-level
authentication, envelope, and view-model behavior.

## Architecture

Dependencies point inward and production dependencies are assembled once in
`backend/src/bootstrap.rs`:

```text
api (Axum DTOs, handlers, OpenAPI)
  -> core (domain services and consumer-owned ports)
       <- database (Diesel/PostgreSQL)
       <- integrations (OpenAI, Exa, S3, fetch/extract)

bootstrap -> AppState -> typed FromRef substates
```

Application-owned cancellation tokens and task sets cover the HTTP server,
article/image queues, copilot requests and WebSocket bridges, workers, and
graceful SIGINT/SIGTERM shutdown.
