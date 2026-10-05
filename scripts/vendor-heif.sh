#!/usr/bin/env bash
# Copy heif-convert and its libraries into backend/opt/heif.
# Render's native runtime does not keep apt packages installed at build time,
# and its ImageMagick/libvips builds cannot decode HEIC. The API runs this copy.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
dest="${root}/backend/opt/heif"

bash "${root}/scripts/install-heif.sh"

rm -rf "$dest"
mkdir -p "${dest}/bin" "${dest}/lib" "${dest}/plugins"

cp -L "$(command -v heif-convert)" "${dest}/bin/heif-convert"
chmod 755 "${dest}/bin/heif-convert"

plugin_dir="$(heif-convert -v 2>&1 | awk -F': ' '/plugin path/ { print $2; exit }')"
if [[ -z "$plugin_dir" || ! -d "$plugin_dir" ]]; then
  echo "vendor-heif: could not find the libheif plugin directory" >&2
  exit 1
fi

decoder="$(find "$plugin_dir" -name '*libde265*' \( -type f -o -type l \) -print -quit)"
if [[ -z "$decoder" ]]; then
  echo "vendor-heif: libde265 plugin is missing from ${plugin_dir}" >&2
  exit 1
fi
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
