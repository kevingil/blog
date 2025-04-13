-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- Check if the column exists before adding it
SELECT CASE 
    WHEN NOT EXISTS (SELECT 1 FROM pragma_table_info('articles') WHERE name = 'published_at') 
    THEN 'ALTER TABLE articles ADD COLUMN published_at INTEGER;'
    ELSE 'SELECT 1;' -- Do nothing if column exists
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- Check if the column exists before dropping it
SELECT CASE 
    WHEN EXISTS (SELECT 1 FROM pragma_table_info('articles') WHERE name = 'published_at') 
    THEN 'ALTER TABLE articles DROP COLUMN published_at;'
    ELSE 'SELECT 1;' -- Do nothing if column doesn't exist
END;
-- +goose StatementEnd
