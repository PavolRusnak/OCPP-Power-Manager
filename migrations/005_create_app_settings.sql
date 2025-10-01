-- +goose Up
CREATE TABLE app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Insert default settings
INSERT INTO app_settings (key, value) VALUES 
    ('heartbeat_interval', '300'),
    ('log_level', 'info');

-- +goose Down
DROP TABLE app_settings;
