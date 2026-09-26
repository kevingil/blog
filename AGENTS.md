# Agent Guide

Blog Copilot is an agentic blog editor: an Axum/Rust backend, a React/Bun
frontend, PostgreSQL 17.4 with pgvector, MinIO object storage, and a
deterministic OpenAI/Exa/Groq fixture service, all orchestrated with Docker
Compose.

This file is the shared, tool-agnostic entry point (Cursor, Claude Code, Codex,
and humans). Backend-specific rules live in `backend/AGENTS.md`; full docs are
in `README.md`.

## Run the whole stack

Prerequisite: a working Docker Engine with the daemon running.

```bash
docker compose up --build
```

- Frontend: <http://localhost:3000>
- API: <http://localhost:8080>
- Health: <http://localhost:8080/health>
- Swagger UI: <http://localhost:8080/swagger>
- MinIO console: <http://localhost:9001>

No real API keys are required. `docker-compose.yml` supplies safe local values
for every setting (`DATABASE_URL`, `AUTH_SECRET`, `OPENAI_*`, `GROQ_*`, `EXA_*`,
`S3_*`): the database and object store run locally, and all model/search calls
hit the bundled `external-fixtures` service. The full product flow (auth,
articles, AI search, copilot, image generation) is exercisable offline.

## Bare VM without Docker (cloud agent sandboxes)

Some agent sandboxes boot without Docker installed or without the daemon
running. Two idempotent helpers under `.agents/` bootstrap the host and start
the stack. On a machine that already has a working Docker daemon they leave the
host configuration untouched and simply build/run the stack.

```bash
bash .agents/install.sh   # install Docker Engine (if missing) + build images
bash .agents/start.sh     # ensure the daemon is up, bring the stack up, wait for health
```

`install.sh` is one-time host/dependency setup; `start.sh` is the per-boot
command that launches the stack and fails if `/health` never comes up.
`verify.sh` re-checks API, frontend, Postgres (`localhost:55432`), and MinIO.

Host-side Rust tests that talk to the running stack:

```bash
TEST_DATABASE_URL=postgres://blog:blog@localhost:55432/blog \
TEST_S3_ENDPOINT=http://localhost:9000 \
  ./scripts/test-rust.sh blocking
```

## Test

```bash
make test-docker          # blocking Rust matrix in the pinned Docker environment
./scripts/test-rust.sh blocking
./scripts/test-rust.sh insights
```

Local, non-Docker commands (require a reachable Postgres and the settings shown
in `docker-compose.yml`):

```bash
make run                  # cargo run --locked --bin blog-backend
make test                 # ./scripts/test-rust.sh blocking
```

## OpenAPI and generated client

```bash
./scripts/generate-client.sh
node ./scripts/verify-openapi-contracts.mjs
```

Generated files under `frontend/src/client/` are not edited by hand.
