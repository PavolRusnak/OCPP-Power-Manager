-- +goose Up
CREATE TABLE meter_values (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id INTEGER NOT NULL,
    ts DATETIME NOT NULL,
    measurand TEXT NOT NULL,
    value REAL NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS mv_tx_ts ON meter_values(transaction_id, ts);

-- +goose Down
DROP TABLE meter_values;
