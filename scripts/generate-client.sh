#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/../backend"
cargo run --locked --bin export-openapi -- --output ../openapi/openapi.json
cp ../openapi/openapi.json ../frontend/openapi.json
cd ../frontend
bun run generate-client
