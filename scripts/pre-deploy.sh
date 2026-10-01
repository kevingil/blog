#!/bin/sh
set -eu

# This deployment adopts the existing Goose database. Stamping verifies the
# historical schema once; subsequent deploys require its seven ledger entries.
# Never seed production or reapply the historical table-creation SQL here.
: "${DATABASE_URL:?DATABASE_URL must point to the existing blog database}"

stamp-diesel-migrations --if-needed
exec migrate
