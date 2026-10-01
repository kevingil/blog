# Local environment profiles

The application itself reads ordinary process environment variables. Render is
the production source of those variables and does not read files from this
directory.

For local commands, copy only the profile you need:

```sh
cp env/local.env.example env/local.env
cp env/testing.env.example env/testing.env
cp env/production.env.example env/production.env
```

The resulting `*.env` files are ignored by Git and excluded from Docker build
contexts. The `*.env.example` files are the tracked configuration contract.

Run a command with an explicit profile from the repository root:

```sh
./scripts/with-env.sh env/local.env \
  cargo run --manifest-path backend/Cargo.toml --locked --bin blog-backend
```

Profile selection is intentionally explicit. In particular, tests must use
`testing.env`; `production.env` must never define `TEST_DATABASE_URL`. The
Compose test profile provides the separate `blog_test` database expected by the
testing example:

```sh
docker compose --profile test up -d \
  test-db test-migrate external-fixtures object-storage object-storage-init
```

Profile files use shell-compatible `KEY=value` declarations. URL-encode special
characters in connection-string credentials. Do not put commands or other shell
code in a profile.
