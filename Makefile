# Build, test, and generation entry points.

# Build the application
all: build test

build-go:
	@echo "Building..."
	@cd backend-go && go build -o ../main .

run-go:
	@cd backend-go && go run .

test-go:
	@echo "Testing..."
	@cd backend-go && go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Generate Swagger documentation
swagger-go:
	@echo "Generating Swagger docs..."
	@cd backend-go && swag init --parseDependency --parseInternal --generalInfo main.go

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

# Go reference live reload
watch-go:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build build-go run run-go test test-insights test-database test-docker test-go clean watch-go swagger-go generate-client
