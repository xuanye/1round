-- +goose Up
ALTER TABLE game_sessions ADD COLUMN max_participants INTEGER NULL;
ALTER TABLE game_sessions ADD COLUMN settled_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN voided_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN last_scored_at TEXT NULL;
ALTER TABLE game_sessions ADD COLUMN public_share_token TEXT NULL;

CREATE UNIQUE INDEX idx_game_sessions_public_share_token
ON game_sessions(public_share_token)
WHERE public_share_token IS NOT NULL;

ALTER TABLE players ADD COLUMN active INTEGER NOT NULL DEFAULT 1;
ALTER TABLE players ADD COLUMN joined_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN left_at TEXT NULL;

CREATE UNIQUE INDEX idx_players_game_display_name
ON players(game_session_id, display_name);

CREATE UNIQUE INDEX idx_players_game_user
ON players(game_session_id, user_id)
WHERE user_id IS NOT NULL;

CREATE INDEX idx_players_game_active_order
ON players(game_session_id, active, joined_order);

CREATE TABLE score_transfers (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    sequence_no INTEGER NOT NULL,
    from_player_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    amount INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(from_player_id) REFERENCES players(id),
    FOREIGN KEY(created_by_user_id) REFERENCES users(id),
    UNIQUE(game_session_id, sequence_no),
    UNIQUE(game_session_id, created_by_user_id, idempotency_key),
    CHECK(amount > 0)
);

CREATE INDEX idx_score_transfers_game_sequence
ON score_transfers(game_session_id, sequence_no DESC);

CREATE TABLE score_transfer_receivers (
    id TEXT PRIMARY KEY,
    score_transfer_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    receiver_order INTEGER NOT NULL,
    FOREIGN KEY(score_transfer_id) REFERENCES score_transfers(id),
    FOREIGN KEY(player_id) REFERENCES players(id),
    UNIQUE(score_transfer_id, player_id)
);

CREATE INDEX idx_score_transfer_receivers_transfer_order
ON score_transfer_receivers(score_transfer_id, receiver_order);

CREATE INDEX idx_score_transfer_receivers_player
ON score_transfer_receivers(player_id);

CREATE TABLE finish_requests (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    requested_by_player_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    decided_at TEXT NULL,
    decided_by_player_id TEXT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(requested_by_player_id) REFERENCES players(id),
    FOREIGN KEY(decided_by_player_id) REFERENCES players(id)
);

CREATE UNIQUE INDEX idx_finish_requests_one_pending
ON finish_requests(game_session_id)
WHERE status = 'pending';

CREATE INDEX idx_finish_requests_game_created
ON finish_requests(game_session_id, created_at DESC);

-- Backfill existing rows.
UPDATE game_sessions
SET last_scored_at = CASE WHEN round_count > 0 THEN updated_at ELSE NULL END;

UPDATE players
SET joined_order = (
    SELECT COUNT(*)
    FROM players p2
    WHERE p2.game_session_id = players.game_session_id
      AND (p2.created_at < players.created_at OR (p2.created_at = players.created_at AND p2.id <= players.id))
);

-- +goose Down
DROP INDEX idx_finish_requests_game_created;
DROP INDEX idx_finish_requests_one_pending;
DROP TABLE finish_requests;
DROP INDEX idx_score_transfer_receivers_player;
DROP INDEX idx_score_transfer_receivers_transfer_order;
DROP TABLE score_transfer_receivers;
DROP INDEX idx_score_transfers_game_sequence;
DROP TABLE score_transfers;
DROP INDEX idx_players_game_active_order;
DROP INDEX idx_players_game_user;
DROP INDEX idx_players_game_display_name;
DROP INDEX idx_game_sessions_public_share_token;
