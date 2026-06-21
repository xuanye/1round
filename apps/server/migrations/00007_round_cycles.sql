-- +goose Up
CREATE TABLE round_cycles (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    round_no INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'complete')),
    created_at TEXT NOT NULL,
    completed_at TEXT NULL,
    UNIQUE(game_session_id, round_no),
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id)
);

CREATE UNIQUE INDEX idx_round_cycles_active_unique ON round_cycles(game_session_id) WHERE status = 'active';

CREATE TABLE round_participation_statuses (
    id TEXT PRIMARY KEY,
    round_cycle_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'satisfied', 'exempt')),
    satisfied_by_transfer_id TEXT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(round_cycle_id, player_id),
    FOREIGN KEY(round_cycle_id) REFERENCES round_cycles(id),
    FOREIGN KEY(player_id) REFERENCES players(id),
    FOREIGN KEY(satisfied_by_transfer_id) REFERENCES score_transfers(id)
);

ALTER TABLE score_transfers ADD COLUMN round_cycle_id TEXT NULL;
ALTER TABLE score_transfers ADD COLUMN transfer_kind TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE score_transfers ADD COLUMN reversal_of_transfer_id TEXT NULL;
ALTER TABLE score_transfers ADD COLUMN reversed_at TEXT NULL;

CREATE INDEX idx_round_cycles_game_status ON round_cycles(game_session_id, status);
CREATE INDEX idx_round_participation_round_status ON round_participation_statuses(round_cycle_id, status);
CREATE INDEX idx_score_transfers_round_cycle ON score_transfers(round_cycle_id, sequence_no DESC);
CREATE UNIQUE INDEX idx_score_transfers_reversal_of ON score_transfers(reversal_of_transfer_id) WHERE reversal_of_transfer_id IS NOT NULL;

-- Backfill existing games that are active
INSERT INTO round_cycles (id, game_session_id, round_no, status, created_at, completed_at)
SELECT hex(randomblob(16)), id, 1, 'complete', updated_at, updated_at
FROM game_sessions
WHERE status = 'active' AND round_count > 0;

INSERT INTO round_cycles (id, game_session_id, round_no, status, created_at, completed_at)
SELECT hex(randomblob(16)), id, 1, 'active', created_at, NULL
FROM game_sessions
WHERE status = 'active' AND round_count = 0;

-- Backfill round_participation_statuses for active players in pre-existing active sessions
INSERT INTO round_participation_statuses (id, round_cycle_id, player_id, status, updated_at)
SELECT hex(randomblob(16)), rc.id, p.id, 'pending', rc.created_at
FROM round_cycles rc
JOIN players p ON p.game_session_id = rc.game_session_id
WHERE rc.status = 'active' AND p.active = 1;

UPDATE score_transfers
SET round_cycle_id = (
    SELECT rc.id FROM round_cycles rc
    WHERE rc.game_session_id = score_transfers.game_session_id AND rc.round_no = 1
)
WHERE round_cycle_id IS NULL;

-- +goose Down
DROP INDEX idx_score_transfers_reversal_of;
DROP INDEX idx_score_transfers_round_cycle;
DROP INDEX idx_round_participation_round_status;
DROP INDEX idx_round_cycles_game_status;
DROP TABLE round_participation_statuses;
DROP TABLE round_cycles;
