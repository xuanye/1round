-- +goose Up
ALTER TABLE game_sessions ADD COLUMN preset_scores TEXT NOT NULL DEFAULT '[20,30,40,60]'
    CHECK (json_valid(preset_scores) AND json_type(preset_scores) = 'array' AND json_array_length(preset_scores) BETWEEN 1 AND 8);

-- +goose Down
ALTER TABLE game_sessions DROP COLUMN preset_scores;
