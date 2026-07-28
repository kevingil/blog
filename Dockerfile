FROM rust:1.92.0-bookworm@sha256:e90e846de4124376164ddfbaab4b0774c7bdeef5e738866295e5a90a34a307a2 AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends libpq-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/backend
COPY backend/Cargo.toml backend/Cargo.lock backend/rust-toolchain.toml backend/diesel.toml ./
COPY backend/migrations ./migrations
COPY backend/src ./src
COPY backend/tests ./tests
RUN cargo build --locked --release --bin blog-backend --bin migrate

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl libpq5 \
    && useradd --system --uid 10001 --create-home blog \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/backend/target/release/blog-backend /usr/local/bin/blog-backend
COPY --from=builder /app/backend/target/release/migrate /usr/local/bin/migrate

USER 10001
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/blog-backend"]
