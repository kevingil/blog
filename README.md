# Blog Copilot

An agentic blog editor with a React/Bun frontend and an Axum/Rust backend.

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

Put these in `backend/.env` and `frontend/.env`. Each process loads the file from its own directory.

Backend, required:

```env
DATABASE_URL=postgres://blog:blog@localhost:55432/blog
AUTH_SECRET=local-test-secret-that-is-not-used-in-production
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY_ID=blog
S3_ACCESS_KEY_SECRET=blog-local-secret
S3_BUCKET=blog
S3_URL_PREFIX=http://localhost:9000/blog
```

Backend, optional:

```env
HOST=0.0.0.0
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000
OPENAI_API_KEY=local-fixture-key
OPENAI_BASE_URL=http://localhost:8090/v1
GROQ_API_KEY=local-fixture-key
GROQ_BASE_URL=http://localhost:8090/v1
EXA_API_KEY=local-fixture-key
EXA_BASE_URL=http://localhost:8090
PUBLIC_API_URL=http://localhost:8080
PUBLIC_APP_URL=http://localhost:3000
```

Unset model URLs fall back to `https://api.openai.com/v1`, `https://api.groq.com/openai/v1`, and `https://api.exa.ai`. `PUBLIC_API_URL` and `PUBLIC_APP_URL` are the browser-visible origins used when an MCP connector starts OAuth.

Frontend. `VITE_API_BASE_URL` and `VITE_WS_URL` are optional and already default to the values below. Upload previews need `VITE_PUBLIC_S3_URL_PREFIX`.

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/websocket
VITE_PUBLIC_S3_URL_PREFIX=http://localhost:9000/blog
```

Apply pending Diesel migrations. `migrate` reads `DATABASE_URL` from the environment or `backend/.env` and records applied versions in `__diesel_schema_migrations`:

```bash
cd backend
cargo run --locked --bin migrate
```

`docker compose up` runs that migrator before the API starts. A database that already has the Goose schema through `20260315000000` gets a Diesel ledger once, then `migrate` applies anything newer:

```bash
cd backend
cargo run --locked --bin stamp-diesel-migrations
cargo run --locked --bin migrate
```

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
