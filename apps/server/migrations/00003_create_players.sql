-- +goose Up
CREATE TABLE players (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    user_id TEXT NULL,
    display_name TEXT NOT NULL,
    total_score INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX idx_players_game_session_id ON players(game_session_id);
CREATE INDEX idx_players_user_id ON players(user_id);

-- +goose Down
DROP TABLE players;
