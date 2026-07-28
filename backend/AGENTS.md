# Rust Backend Instructions

- Preserve the observable behavior recorded in `docs/porting/CONTRACTS.tsv`.
- Keep dependencies one-way: `api -> core <- database/integrations`; only
  `bootstrap` assembles concrete implementations.
- Use constructor injection and narrow Axum `State<T>` substates. Do not add
  globals, service locators, or runtime DI containers.
- Keep transport DTOs, domain values, and Diesel row models separate when
  their constraints differ.
- Request paths must not use `unwrap`, `expect`, `todo!`, `unimplemented!`, or
  panic for external input.
- Every spawned task needs an owner, cancellation path, observed result, and
  bounded shutdown.
- Do not edit generated OpenAPI/client files by hand.
- Run targeted Rust tests and checks by module or test target; do not invoke an
  unscoped full-project test command.
