FROM rust:1.92.0-bookworm@sha256:e90e846de4124376164ddfbaab4b0774c7bdeef5e738866295e5a90a34a307a2

RUN apt-get update \
    && apt-get install -y --no-install-recommends libpq-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/backend
COPY backend/Cargo.toml backend/Cargo.lock backend/rust-toolchain.toml backend/diesel.toml ./
COPY backend/migrations ./migrations
COPY backend/src ./src
COPY backend/tests ./tests

CMD ["cargo", "test", "--locked", "--lib", "--test", "auth", "--test", "auth_database", "--test", "article_repository", "--test", "health", "--test", "websocket_contract", "--test", "websocket_network", "--", "--test-threads=1"]
