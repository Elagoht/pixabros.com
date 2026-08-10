CREATE TABLE homepage_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri'))
);
