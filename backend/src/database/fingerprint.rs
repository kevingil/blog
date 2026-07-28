use diesel::{
    PgConnection, QueryableByName, RunQueryDsl,
    sql_types::{Nullable, Text},
};
use sha2::{Digest, Sha256};

#[derive(QueryableByName)]
struct FingerprintSource {
    #[diesel(sql_type = Nullable<Text>)]
    source: Option<String>,
}

pub fn schema_fingerprint(connection: &mut PgConnection) -> anyhow::Result<String> {
    let row = diesel::sql_query(
        r#"
        SELECT jsonb_build_object(
            'columns', COALESCE((
                SELECT jsonb_agg(to_jsonb(c) ORDER BY c.table_name, c.ordinal_position)
                FROM (
                    SELECT table_name, ordinal_position, column_name, data_type, udt_schema,
                           udt_name, is_nullable, column_default, is_identity
                    FROM information_schema.columns
                    WHERE table_schema = 'public'
                      AND table_name NOT IN ('goose_db_version', '__diesel_schema_migrations')
                ) c
            ), '[]'::jsonb),
            'constraints', COALESCE((
                SELECT jsonb_agg(to_jsonb(c) ORDER BY c.table_name, c.name)
                FROM (
                    SELECT rel.relname AS table_name, con.conname AS name,
                           con.contype AS kind,
                           pg_get_constraintdef(con.oid, true) AS definition
                    FROM pg_constraint con
                    JOIN pg_class rel ON rel.oid = con.conrelid
                    JOIN pg_namespace ns ON ns.oid = rel.relnamespace
                    WHERE ns.nspname = 'public'
                      AND rel.relname NOT IN ('goose_db_version', '__diesel_schema_migrations')
                ) c
            ), '[]'::jsonb),
            'indexes', COALESCE((
                SELECT jsonb_agg(to_jsonb(i) ORDER BY i.tablename, i.indexname)
                FROM (
                    SELECT tablename, indexname, indexdef
                    FROM pg_indexes
                    WHERE schemaname = 'public'
                      AND tablename NOT IN ('goose_db_version', '__diesel_schema_migrations')
                ) i
            ), '[]'::jsonb),
            'extensions', COALESCE((
                SELECT jsonb_agg(to_jsonb(e) ORDER BY e.extname)
                FROM (
                    SELECT ext.extname, ext.extversion, ns.nspname AS schema
                    FROM pg_extension ext
                    JOIN pg_namespace ns ON ns.oid = ext.extnamespace
                    WHERE ext.extname IN ('uuid-ossp', 'vector')
                ) e
            ), '[]'::jsonb)
        )::text AS source
        "#,
    )
    .get_result::<FingerprintSource>(connection)?;

    let source = row
        .source
        .ok_or_else(|| anyhow::anyhow!("database returned no schema fingerprint source"))?;
    Ok(hex::encode(Sha256::digest(source.as_bytes())))
}
