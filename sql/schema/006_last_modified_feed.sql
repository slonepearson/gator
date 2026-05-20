-- +goose Up
ALTER TABLE feeds
ADD COLUMN last_modified TIMESTAMP;

-- +goose Down
ALTER TABLE feeds
DROP COLUMN last_modified;