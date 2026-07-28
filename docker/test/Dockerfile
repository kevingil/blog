FROM rust:1.92.0-bookworm@sha256:e90e846de4124376164ddfbaab4b0774c7bdeef5e738866295e5a90a34a307a2

RUN apt-get update \
    && apt-get install -y --no-install-recommends libpq-dev lld pkg-config \
    && rm -rf /var/lib/apt/lists/*

ENV CARGO_BUILD_JOBS=1 \
    CARGO_INCREMENTAL=0 \
    RUSTFLAGS="-C link-arg=-fuse-ld=lld"

WORKDIR /app/backend
COPY backend/Cargo.toml backend/Cargo.lock backend/rust-toolchain.toml backend/diesel.toml ./
COPY backend/migrations ./migrations
COPY backend/src ./src
COPY backend/tests ./tests
COPY scripts/test-rust.sh /app/scripts/test-rust.sh

CMD ["/app/scripts/test-rust.sh", "blocking"]
