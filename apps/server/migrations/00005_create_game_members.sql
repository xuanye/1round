-- +goose Up
CREATE TABLE game_members (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    joined_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(user_id) REFERENCES users(id),
    UNIQUE(game_session_id, user_id)
);

CREATE INDEX idx_game_members_game_session_id ON game_members(game_session_id);
CREATE INDEX idx_game_members_user_id ON game_members(user_id);

-- +goose Down
DROP TABLE game_members;
