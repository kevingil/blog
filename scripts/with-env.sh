#!/usr/bin/env sh
set -eu

if [ "$#" -lt 2 ]; then
  echo "usage: $0 ENV_FILE COMMAND [ARG ...]" >&2
  exit 2
fi

env_file="$1"
shift

if [ ! -f "$env_file" ]; then
  echo "environment file not found: $env_file" >&2
  exit 2
fi

set -a
# The selected profile is a trusted, local shell-compatible KEY=value file.
# shellcheck disable=SC1090
. "$env_file"
set +a

exec "$@"
