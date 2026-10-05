#!/usr/bin/env bash
# Install a libheif new enough to read iPhone photos from iOS 18.
# Debian 12's default 1.15.1 rejects those files ("Too many auxiliary image references").
set -euo pipefail

if ! command -v apt-get >/dev/null 2>&1; then
  echo "install-heif: apt-get is required" >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

if [[ -r /etc/os-release ]]; then
  # shellcheck disable=SC1091
  . /etc/os-release
fi

apt-get update
if [[ "${VERSION_CODENAME:-}" == "bookworm" ]]; then
  echo "deb http://deb.debian.org/debian bookworm-backports main" \
    > /etc/apt/sources.list.d/heif-backports.list
  apt-get update
  apt-get install -y --no-install-recommends -t bookworm-backports libheif-examples
else
  apt-get install -y --no-install-recommends libheif-examples
fi

version="$(heif-convert -v 2>&1 | tr ' ' '\n' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)"
if [[ -z "$version" ]]; then
  echo "install-heif: could not read the heif-convert version" >&2
  exit 1
fi
oldest="$(printf '%s\n%s\n' "$version" "1.18.0" | sort -V | head -n 1)"
if [[ "$oldest" != "1.18.0" ]]; then
  echo "install-heif: libheif ${version} is too old to read iOS 18 HEIC photos (need >= 1.18.0)" >&2
  exit 1
fi

echo "install-heif: heif-convert ${version}"
