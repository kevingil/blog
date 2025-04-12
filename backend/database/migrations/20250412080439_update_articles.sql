-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE articles ADD COLUMN published_at INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE articles DROP COLUMN published_at;
-- +goose StatementEnd
