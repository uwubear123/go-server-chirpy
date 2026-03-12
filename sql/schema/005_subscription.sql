-- +goose Up
ALTER table users
ADD COLUMN is_chirpy_red boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER table users
DROP COLUMN is_chirpy_red;