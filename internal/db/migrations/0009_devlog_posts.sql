CREATE TABLE devlog_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_markdown TEXT NOT NULL DEFAULT '',
    game_id INTEGER REFERENCES games(id) ON DELETE SET NULL,
    og_image_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    is_published INTEGER NOT NULL DEFAULT 0,
    published_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
