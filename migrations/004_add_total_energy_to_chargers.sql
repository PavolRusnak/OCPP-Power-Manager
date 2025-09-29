-- +goose Up
ALTER TABLE chargers ADD COLUMN total_energy_wh INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE chargers DROP COLUMN total_energy_wh;
