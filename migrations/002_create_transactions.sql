-- +goose Up
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    charger_id INTEGER NOT NULL,
    tx_id TEXT NOT NULL,
    start_ts DATETIME NOT NULL,
    stop_ts DATETIME,
    start_meter_wh INTEGER NOT NULL,
    stop_meter_wh INTEGER,
    energy_wh INTEGER,
    FOREIGN KEY (charger_id) REFERENCES chargers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS tx_ch_ts ON transactions(charger_id, start_ts);

-- +goose Down
DROP TABLE transactions;
