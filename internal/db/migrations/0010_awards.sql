CREATE TABLE awards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    issuer TEXT NOT NULL,
    date TEXT NOT NULL,
    picture_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    game_id INTEGER REFERENCES games(id) ON DELETE SET NULL,
    link TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
