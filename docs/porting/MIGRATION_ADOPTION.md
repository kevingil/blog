# Diesel Migration Adoption

The seven Goose SQL bodies are mechanically copied to same-version Diesel
directories. The extraction convention hashes exact bytes after
`-- +goose StatementBegin\n` through the byte before the corresponding
`-- +goose StatementEnd`, preserving leading/trailing blank lines.

| Version | Whole file SHA-256 | Up body SHA-256 | Down body SHA-256 |
| --- | --- | --- | --- |
| `20250723064003` | `767275bcab7ce74a13d24993c1cd024e15a46f26e18ea26e3eaf6f3cfda7b760` | `2c2b81ab78fec5a0f77b02bb8d86a2766a7a00ff4737ff37d77d2c1bf471a7d5` | `1ad8a0e2746e97bf89d2f4d0a6d2d897f0fe5ea0e083b9f0f8a832f2f7618e1e` |
| `20250813062742` | `a7fd12c40e7bc0395bd682deb92b5e228ec38d68665ea75fba69883ebdcc238f` | `4f0041a668f78a10388ab05729653cb27989d87db2cea2b2d627543acb860f4b` | `d8d5e8ae41c4b4c85ef43d5387f48ed5351f9e712cd69b06722a0483f9720675` |
| `20260119004828` | `a9cf4f294e3f32c95dab0a75222f41365e7068f48ce410091d7e0b9f94791467` | `82850f75d666fba377ce3a5a31c8fab77aa4bc914145f7263578227dcc37c116` | `06bdf62f1cc0f8ba10177aa267d5940abf0fd529f8881b87826ce3cf015049e6` |
| `20260125045327` | `e4b389f5ca678479d4b6f1b0ba367c7eb5b93bbcd381a0d62d12cd01f4f3abcc` | `2ca19b27b6646f40fc13a3a51dea91bd06505614b987737bc8eb29c5a35ed8e8` | `092bcd35809d64521a7d7eb65ecea36147b2e3179f9eb3ec2ad8969144b21087` |
| `20260129000000` | `7a4f49049229f6ab15757c69532ed3c110daf3dd4a8582e61944650585a40ee6` | `2e559eeb96bc131151db2d4b08b7edf95f346e1adce1535e916dfd1acfc780f7` | `34a78e122c8e28c958f0c34ae414adc1b854b0d80fa7ccfe309aa8702b23d31f` |
| `20260130000000` | `1cfa433691713923cfe10b770760438e9c373a9d89d65709bc419cb5a4117b5a` | `7cc94d4613a39117163aead072435b9a63d81217200a5277ad13a2907538624b` | `8228405116b05ca6f6d33c0c0fcf3de1c175c300b62d210f2132369f71bcbb1a` |
| `20260315000000` | `8252b6b6954e3658ff2c4682f909fd4d12b833d7646cdafb60a957af221c6b9b` | `6b99c00615c41a2e96782248d00ba6858d7962ded6630ad8f0746770fcdac1e5` | `344caf477f177a183f8a1e07d04dc98fb867f6e81a6e5318fd17da9ab0000faa` |

## Fresh database

1. Start two clean databases from the same pinned PostgreSQL 17.4/pgvector
   image and settings.
2. Apply original bodies with a pinned Goose runner to one and Diesel bodies to
   the other.
3. Assert the 21 application tables, columns/types/nullability/defaults,
   identity/sequence ownership, constraints, indexes/predicates/operator
   classes, and extension name/version/namespace.
4. Exclude only migration ledgers and their owned sequences.
5. Produce schema-only dumps with the same 17.4 client. Remove only version
   headers and randomized `\restrict` lines; require identical canonical bytes
   and SHA-256.
6. Verify Diesel pending is empty, redo each migration, then fully
   revert/reapply on disposable databases.

The clean-database comparison passed with canonical fingerprint
`81fa7f13268ae949c1c627f62ea860d4fe7dfb72698a4f40c5b4706cadd07b29`
on both sides. The reference runner was Goose `v3.27.1`; the Rust runner used
Diesel migrations `2.3.1`. The parity PostgreSQL runtime is 17.4 and the local
pgvector extension is 0.8.2. The stamping command was then rehearsed against
the disposable database; it inserted all seven Diesel ledger rows without
altering the application schema, and `diesel migration pending` returned
`false`.

The legacy article-version Down restores `article.title` without `NOT NULL`.
This known source defect means that intermediate down-state does not
fingerprint-equal the exact pre-Up state. Preserve it; do not silently repair
the historical SQL during the port.

## Existing Supabase database stamping

`stamp-diesel-migrations` is a one-time command for the existing database. It
does not run the application migrations because their tables already exist. It
must:

1. Acquire a transaction-scoped PostgreSQL advisory lock.
2. Refuse to continue if Diesel's migration ledger already exists.
3. Validate the existing tables, columns, types, nullability, defaults,
   constraints, indexes, and required extensions against the canonical
   fingerprint produced by applying the checked-in Diesel migrations to a clean
   database.
4. In the same transaction, create Diesel's exact ledger and insert the seven
   version strings:

   ```sql
   CREATE TABLE IF NOT EXISTS __diesel_schema_migrations (
     version VARCHAR(50) PRIMARY KEY NOT NULL,
     run_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
   );
   ```

5. Require `migrate` to report no pending work before the first Render deploy.

Any legacy migration ledger is ignored and left untouched; it is historical
metadata, not an input to Diesel stamping. If the application schema differs in
any checked detail, the transaction aborts before creating the Diesel ledger.
Rehearse against an approved schema-identical disposable clone before Supabase.
Run the stamp once against Supabase before the first Render deploy:

```sh
cd backend
DATABASE_URL='postgresql://...' cargo run --locked --release \
  --bin stamp-diesel-migrations
```

The later Fly-to-Render compute move reuses this database and does no second
adoption or data transfer.
