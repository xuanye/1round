-- +goose Up
CREATE TABLE game_sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    owner_user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    zero_sum_required INTEGER NOT NULL,
    round_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(owner_user_id) REFERENCES users(id)
);

CREATE INDEX idx_game_sessions_owner_user_id ON game_sessions(owner_user_id);
CREATE INDEX idx_game_sessions_invite_code ON game_sessions(invite_code);
CREATE INDEX idx_game_sessions_updated_at ON game_sessions(updated_at);

-- +goose Down
DROP TABLE game_sessions;
