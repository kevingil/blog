#!/bin/sh
set -eu

# Render stored the Supabase CA as a secret file. Fly secrets are environment
# variables, so the PEM is base64-encoded and written here before the process
# starts. rustls-native-certs reads SSL_CERT_FILE and SSL_CERT_DIR.
if [ -n "${SUPABASE_CA_CERT_B64:-}" ]; then
  umask 077
  printf '%s' "$SUPABASE_CA_CERT_B64" | base64 -d > /tmp/supabase-ca.crt
  export SSL_CERT_FILE=/tmp/supabase-ca.crt
  export SSL_CERT_DIR="${SSL_CERT_DIR:-/etc/ssl/certs}"
fi

if [ "$#" -eq 0 ]; then
  set -- /usr/local/bin/blog-backend
fi
exec "$@"
