-- Unique key for project upsert: one row per URL, compared case-insensitively.
-- Null and empty URLs are not keys (POST /projects allows a project with no URL),
-- so they stay out of the index.
--
-- Existing case-insensitive duplicates are not merged or deleted. The migration
-- stops with a named error that lists a sample of the conflicting keys. Resolve
-- those rows, then run the migration again. Creating the index is a second
-- guard if a duplicate appears between the check and the build.

DO $$
DECLARE
    duplicate_groups integer;
    sample text;
BEGIN
    SELECT count(*)
    INTO duplicate_groups
    FROM (
        SELECT lower(url) AS url_key
        FROM project
        WHERE url IS NOT NULL AND url <> ''
        GROUP BY lower(url)
        HAVING count(*) > 1
    ) grouped;

    IF duplicate_groups > 0 THEN
        SELECT string_agg(url_key, ', ' ORDER BY url_key)
        INTO sample
        FROM (
            SELECT lower(url) AS url_key
            FROM project
            WHERE url IS NOT NULL AND url <> ''
            GROUP BY lower(url)
            HAVING count(*) > 1
            ORDER BY lower(url)
            LIMIT 5
        ) grouped;

        RAISE EXCEPTION
            'refusing unique index on lower(project.url): % case-insensitive duplicate URL group(s) exist (sample: %). This migration does not delete rows; resolve the duplicates, then rerun.',
            duplicate_groups,
            sample;
    END IF;
END $$;

CREATE UNIQUE INDEX idx_project_url_lower
    ON project (lower(url))
    WHERE url IS NOT NULL AND url <> '';
