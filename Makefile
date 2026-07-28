# Build and generation entry points during the Go-to-Rust port.

# Build the application
all: build test

build-go:
	@echo "Building..."
	@cd backend-go && go build -o ../main .

run-go:
	@cd backend-go && go run .

# Test the application
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
	@cd backend && cargo test --locked --lib --test auth --test health --test websocket_contract

test-database:
	@cd backend && cargo test --locked --test auth_database --test article_repository --test websocket_network -- --test-threads=1

test-docker:
	@docker compose --profile test run --build --rm test

run:
	@cd backend && cargo run --locked --bin blog-backend

# Live Reload
watch:
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

.PHONY: all build build-go run run-go test test-database test-docker test-go clean watch swagger-go generate-client
