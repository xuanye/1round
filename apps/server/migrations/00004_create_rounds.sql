-- +goose Up
CREATE TABLE rounds (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    round_no INTEGER NOT NULL,
    created_by_user_id TEXT NOT NULL,
    note TEXT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(created_by_user_id) REFERENCES users(id),
    UNIQUE(game_session_id, round_no)
);

CREATE INDEX idx_rounds_game_session_id_round_no
ON rounds(game_session_id, round_no DESC);

CREATE TABLE round_scores (
    id TEXT PRIMARY KEY,
    round_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    score INTEGER NOT NULL,
    FOREIGN KEY(round_id) REFERENCES rounds(id),
    FOREIGN KEY(player_id) REFERENCES players(id)
);

CREATE INDEX idx_round_scores_round_id ON round_scores(round_id);
CREATE INDEX idx_round_scores_player_id ON round_scores(player_id);

-- +goose Down
DROP TABLE round_scores;
DROP TABLE rounds;
