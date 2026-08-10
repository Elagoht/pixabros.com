CREATE TABLE rendered_pages (
    page_key TEXT PRIMARY KEY,
    etag TEXT NOT NULL,
    rendered_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
