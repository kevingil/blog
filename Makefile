# Build, test, and generation entry points.

all: build test

# Generate frontend API client from OpenAPI spec
generate-client:
	@echo "Generating frontend API client..."
	@./scripts/generate-client.sh

build:
	@cd backend && cargo build --locked

test:
	@./scripts/test-rust.sh blocking

test-insights:
	@./scripts/test-rust.sh insights

test-database:
	@cd backend && cargo test --locked --test auth_database --test article_repository --test websocket_network -- --test-threads=1

test-docker:
	@docker compose --profile test run --build --rm test

run:
	@cd backend && cargo run --locked --bin blog-backend

.PHONY: all build run test test-insights test-database test-docker generate-client
