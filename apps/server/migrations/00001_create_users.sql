-- +goose Up
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    open_id TEXT NOT NULL UNIQUE,
    union_id TEXT NULL,
    display_name TEXT NULL,
    avatar_url TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_users_open_id ON users(open_id);

-- +goose Down
DROP TABLE users;
