-- +goose Up
CREATE TABLE logs_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    enabled BOOLEAN NOT NULL DEFAULT 0,
    directory TEXT NOT NULL DEFAULT '',
    frequency TEXT NOT NULL DEFAULT 'hours',
    frequency_value INTEGER NOT NULL DEFAULT 1,
    last_export DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default configuration
INSERT INTO logs_config (enabled, directory, frequency, frequency_value, last_export) 
VALUES (0, '', 'hours', 1, NULL);

-- +goose Down
DROP TABLE logs_config;
