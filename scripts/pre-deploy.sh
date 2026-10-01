#!/bin/sh
set -eu

# This deployment adopts the existing Goose database. Stamping verifies the
# historical schema once; subsequent deploys require its seven ledger entries.
# Never seed production or reapply the historical table-creation SQL here.
: "${DATABASE_URL:?DATABASE_URL must point to the existing blog database}"

# Render native builds pass backend/target/release; Docker installs into the
# default directory. Both use the same adoption and migration sequence.
bin_dir="${1:-/usr/local/bin}"
"$bin_dir/stamp-diesel-migrations" --if-needed
exec "$bin_dir/migrate"
