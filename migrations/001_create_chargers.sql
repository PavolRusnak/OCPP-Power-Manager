-- +goose Up
CREATE TABLE chargers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    identity TEXT UNIQUE NOT NULL,
    name TEXT,
    model TEXT,
    vendor TEXT,
    max_output_kw REAL,
    firmware TEXT,
    last_seen DATETIME
);

CREATE INDEX IF NOT EXISTS ch_identity ON chargers(identity);

-- +goose Down
DROP TABLE chargers;
