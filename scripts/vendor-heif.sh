#!/usr/bin/env bash
# Copy heif-convert (>= 1.18) and its libraries into backend/opt/heif.
#
# Render's native root filesystem is read-only, so apt-get cannot create
# /var/lib/apt/lists/partial and the build exits 100. Download the Debian
# bookworm-backports packages into a temp directory and extract them. The API
# runs this copy; ImageMagick and libvips on that image cannot decode HEIC.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
dest="${root}/backend/opt/heif"
min_version="1.18.0"

heif_version() {
  "$1" -v 2>&1 | tr ' ' '\n' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true
}

version_at_least() {
  local version="$1"
  local minimum="$2"
  [[ -n "$version" ]] || return 1
  [[ "$(printf '%s\n%s\n' "$version" "$minimum" | sort -V | head -n 1)" == "$minimum" ]]
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "vendor-heif: $1 is required" >&2
    exit 1
  fi
}

debian_triplet() {
  case "$1" in
    amd64) printf '%s\n' x86_64-linux-gnu ;;
    arm64) printf '%s\n' aarch64-linux-gnu ;;
    armhf) printf '%s\n' arm-linux-gnueabihf ;;
    i386) printf '%s\n' i386-linux-gnu ;;
    *) printf '%s\n' "$1-linux-gnu" ;;
  esac
}

# Print a colon-separated library path for an extracted Debian prefix.
prefix_library_path() {
  local prefix="$1"
  local triplet="$2"
  local dir path=""
  for dir in \
    "${prefix}/usr/lib/${triplet}" \
    "${prefix}/lib/${triplet}" \
    "${prefix}/usr/lib" \
    "${prefix}/lib"
  do
    [[ -d "$dir" ]] || continue
    if [[ -z "$path" ]]; then
      path="$dir"
    else
      path="${path}:${dir}"
    fi
  done
  printf '%s\n' "$path"
}

# Download libheif-examples from bookworm-backports without writing to /var or /usr.
fetch_heif_prefix() {
  local stage="$1"
  local arch triplet apt_root prefix key url pkg deb
  require_cmd apt-get
  require_cmd curl
  require_cmd dpkg-deb
  require_cmd gpg

  if command -v dpkg >/dev/null 2>&1; then
    arch="$(dpkg --print-architecture)"
  else
    case "$(uname -m)" in
      x86_64) arch="amd64" ;;
      aarch64) arch="arm64" ;;
      *)
        echo "vendor-heif: unsupported architecture $(uname -m)" >&2
        exit 1
        ;;
    esac
  fi
  triplet="$(debian_triplet "$arch")"

  apt_root="${stage}/apt"
  prefix="${stage}/prefix"
  mkdir -p \
    "${apt_root}/etc/apt/sources.list.d" \
    "${apt_root}/etc/apt/apt.conf.d" \
    "${apt_root}/etc/apt/preferences.d" \
    "${apt_root}/etc/apt/trusted.gpg.d" \
    "${apt_root}/var/lib/apt/lists/partial" \
    "${apt_root}/var/cache/apt/archives/partial" \
    "${apt_root}/var/lib/dpkg" \
    "${apt_root}/var/log/apt" \
    "$prefix"
  : > "${apt_root}/var/lib/dpkg/status"

  # The apt root below does not use the host keyring. Debian publishes the
  # bookworm archive keys at these URLs.
  for url in \
    https://ftp-master.debian.org/keys/archive-key-12.asc \
    https://ftp-master.debian.org/keys/archive-key-12-security.asc
  do
    key="${apt_root}/etc/apt/trusted.gpg.d/$(basename "${url%.asc}").gpg"
    curl -fsSL "$url" | gpg --batch --dearmor --output "$key"
  done

  cat > "${apt_root}/etc/apt/sources.list" <<EOF
deb http://deb.debian.org/debian bookworm main
deb http://deb.debian.org/debian bookworm-updates main
deb http://deb.debian.org/debian-security bookworm-security main
deb http://deb.debian.org/debian bookworm-backports main
EOF

  cat > "${apt_root}/etc/apt/apt.conf" <<EOF
APT::Architecture "${arch}";
APT::Architectures "${arch}";
APT::Install-Recommends "false";
Acquire::Retries "3";
Dir "${apt_root}";
Dir::State "${apt_root}/var/lib/apt";
Dir::State::status "${apt_root}/var/lib/dpkg/status";
Dir::Cache "${apt_root}/var/cache/apt";
Dir::Log "${apt_root}/var/log/apt";
Dir::Etc "${apt_root}/etc/apt";
Dir::Etc::TrustedParts "${apt_root}/etc/apt/trusted.gpg.d";
Debug::NoLocking "true";
EOF

  export APT_CONFIG="${apt_root}/etc/apt/apt.conf"
  apt-get update
  # libheif1 depends on libheif-plugin-libde265, which reads iOS 18 HEIC.
  apt-get install -y --download-only --no-install-recommends \
    -t bookworm-backports libheif-examples

  shopt -s nullglob
  for deb in "${apt_root}/var/cache/apt/archives"/*.deb; do
    pkg="$(dpkg-deb -f "$deb" Package)"
    case "$pkg" in
      libc6|libgcc-s1|libstdc++6|gcc-*-base)
        continue
        ;;
    esac
    dpkg-deb -x "$deb" "$prefix"
  done
  shopt -u nullglob

  if [[ ! -x "${prefix}/usr/bin/heif-convert" && ! -L "${prefix}/usr/bin/heif-convert" ]]; then
    echo "vendor-heif: extracted packages did not include heif-convert" >&2
    exit 1
  fi

  heif_prefix="$prefix"
  heif_library_path="$(prefix_library_path "$prefix" "$triplet")"
  heif_plugin_dir="$(find "$prefix" -type d -path '*/libheif/plugins' -print -quit)"
}

stage=""
cleanup() {
  if [[ -n "$stage" ]]; then
    rm -rf "$stage"
  fi
}
trap cleanup EXIT

heif_prefix=""
heif_library_path=""
heif_plugin_dir=""
program=""

if command -v heif-convert >/dev/null 2>&1; then
  system_program="$(command -v heif-convert)"
  system_version="$(heif_version "$system_program")"
  if version_at_least "$system_version" "$min_version"; then
    program="$system_program"
    heif_plugin_dir="$(heif-convert -v 2>&1 | awk -F': ' '/plugin path/ { print $2; exit }')"
    echo "vendor-heif: using installed heif-convert ${system_version}"
  fi
fi

if [[ -z "$program" ]]; then
  stage="$(mktemp -d)"
  fetch_heif_prefix "$stage"
  program="${heif_prefix}/usr/bin/heif-convert"
  export LD_LIBRARY_PATH="$heif_library_path"
  if [[ -n "$heif_plugin_dir" ]]; then
    export LIBHEIF_PLUGIN_PATH="$heif_plugin_dir"
  fi
  fetched_version="$(heif_version "$program")"
  if ! version_at_least "$fetched_version" "$min_version"; then
    echo "vendor-heif: downloaded heif-convert ${fetched_version:-unknown} is too old (need >= ${min_version})" >&2
    exit 1
  fi
  echo "vendor-heif: downloaded heif-convert ${fetched_version}"
fi

if [[ -z "$heif_plugin_dir" || ! -d "$heif_plugin_dir" ]]; then
  echo "vendor-heif: could not find the libheif plugin directory" >&2
  exit 1
fi

decoder="$(find "$heif_plugin_dir" -name '*libde265*' \( -type f -o -type l \) -print -quit)"
if [[ -z "$decoder" ]]; then
  echo "vendor-heif: libde265 plugin is missing from ${heif_plugin_dir}" >&2
  exit 1
fi

rm -rf "$dest"
mkdir -p "${dest}/bin" "${dest}/lib" "${dest}/plugins"
cp -L "$program" "${dest}/bin/heif-convert"
chmod 755 "${dest}/bin/heif-convert"
cp -L "$decoder" "${dest}/plugins/$(basename "$decoder")"

copy_needed() {
  local file="$1"
  local lib name
  while read -r lib; do
    [[ -n "$lib" && -e "$lib" ]] || continue
    name="$(basename "$lib")"
    case "$name" in
      libc.so.*|libm.so.*|libdl.so.*|libpthread.so.*|librt.so.*|libresolv.so.*|ld-linux*.so*|libstdc++.so.*|libgcc_s.so.*)
        continue
        ;;
    esac
    if [[ ! -e "${dest}/lib/${name}" ]]; then
      cp -L "$lib" "${dest}/lib/${name}"
      copy_needed "${dest}/lib/${name}"
    fi
  done < <(ldd "$file" 2>/dev/null | awk '/=> \// { print $3 }')
}

copy_needed "${dest}/bin/heif-convert"
for plugin in "${dest}/plugins/"*; do
  [[ -f "$plugin" ]] || continue
  copy_needed "$plugin"
done

sample="${root}/backend/tests/fixtures/solid-red.heic"
if [[ -f "$sample" ]]; then
  LD_LIBRARY_PATH="${dest}/lib" \
    LIBHEIF_PLUGIN_PATH="${dest}/plugins" \
    "${dest}/bin/heif-convert" "$sample" "${dest}/smoke.jpg"
  rm -f "${dest}/smoke.jpg"
fi

echo "vendor-heif: ${dest}"
