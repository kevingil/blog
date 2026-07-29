#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/../backend"
cargo run --locked --bin export-openapi -- --output ../openapi/openapi.json
cp ../openapi/openapi.json ../frontend/openapi.json
cd ../frontend
bun run generate-client
find src/client -type f -name '*.ts' -exec sed -i.bak -e 's/[[:space:]]*$//' {} \;
find src/client -type f -name '*.bak' -delete
