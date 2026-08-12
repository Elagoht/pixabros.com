-- Re-key every content table from an autoincrementing INTEGER to a
-- 24-character text id (see internal/id). These ids appear in admin URLs and
-- API payloads, where sequential integers would leak row counts and let a
-- caller enumerate the table by counting up.
--
-- admins and sessions are deliberately left alone: neither id ever leaves the
-- server -- sessions are addressed by token hash, and the admin id only
-- travels inside a request context.
--
-- SQLite cannot generate a CUID2, so rows that already exist are backfilled
-- with lower(hex(randomblob(12))). That yields the same shape the Go
-- generator produces -- 24 lowercase alphanumeric characters -- so no
-- consumer can tell a backfilled id from a generated one. Every row created
-- from now on gets a real CUID2 from internal/id.
--
-- Foreign keys are remapped through per-table mapping tables rather than
-- being recreated blind, so existing relationships survive the re-key.
--
-- db.Migrate disables foreign keys around the whole migration run. That is
-- load-bearing here: DROP TABLE performs an implicit DELETE, so dropping
-- games or media with enforcement on would fire ON DELETE CASCADE and empty
-- game_screenshots before it could be copied across. db.Migrate runs
-- PRAGMA foreign_key_check afterwards to prove nothing was left dangling.

CREATE TABLE _media_id_map (old_id INTEGER PRIMARY KEY, new_id TEXT NOT NULL);
INSERT INTO _media_id_map (old_id, new_id)
SELECT id, lower(hex(randomblob(12))) FROM media;

CREATE TABLE _games_id_map (old_id INTEGER PRIMARY KEY, new_id TEXT NOT NULL);
INSERT INTO _games_id_map (old_id, new_id)
SELECT id, lower(hex(randomblob(12))) FROM games;

-- media ----------------------------------------------------------------
CREATE TABLE media_new (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    format TEXT NOT NULL DEFAULT 'webp',
    alt_text TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO media_new (id, path, width, height, format, alt_text, created_at)
SELECT m.new_id, o.path, o.width, o.height, o.format, o.alt_text, o.created_at
FROM media o JOIN _media_id_map m ON m.old_id = o.id;
DROP TABLE media;
ALTER TABLE media_new RENAME TO media;

-- games ----------------------------------------------------------------
CREATE TABLE games_new (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    short_description TEXT NOT NULL DEFAULT '',
    full_description TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    is_browser_playable INTEGER NOT NULL DEFAULT 0,
    is_downloadable INTEGER NOT NULL DEFAULT 0,
    is_for_sale INTEGER NOT NULL DEFAULT 0,
    price_display TEXT,
    external_links_json TEXT NOT NULL DEFAULT '[]',
    cartridge_art_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    cd_cover_art_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    og_image_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    web_export_path TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_published INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO games_new (
    id, slug, title, short_description, full_description, tags,
    is_browser_playable, is_downloadable, is_for_sale, price_display,
    external_links_json, cartridge_art_id, cd_cover_art_id, og_image_id,
    web_export_path, display_order, is_published, created_at, updated_at
)
SELECT
    g.new_id, o.slug, o.title, o.short_description, o.full_description, o.tags,
    o.is_browser_playable, o.is_downloadable, o.is_for_sale, o.price_display,
    o.external_links_json,
    (SELECT new_id FROM _media_id_map WHERE old_id = o.cartridge_art_id),
    (SELECT new_id FROM _media_id_map WHERE old_id = o.cd_cover_art_id),
    (SELECT new_id FROM _media_id_map WHERE old_id = o.og_image_id),
    o.web_export_path, o.display_order, o.is_published, o.created_at, o.updated_at
FROM games o JOIN _games_id_map g ON g.old_id = o.id;
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

-- game_screenshots -----------------------------------------------------
CREATE TABLE game_screenshots_new (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    media_id TEXT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0
);
INSERT INTO game_screenshots_new (id, game_id, media_id, display_order)
SELECT
    lower(hex(randomblob(12))),
    (SELECT new_id FROM _games_id_map WHERE old_id = o.game_id),
    (SELECT new_id FROM _media_id_map WHERE old_id = o.media_id),
    o.display_order
FROM game_screenshots o;
DROP TABLE game_screenshots;
ALTER TABLE game_screenshots_new RENAME TO game_screenshots;

-- members --------------------------------------------------------------
CREATE TABLE members_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    avatar_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    tags TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    links_json TEXT NOT NULL DEFAULT '[]',
    display_order INTEGER NOT NULL DEFAULT 0,
    is_published INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO members_new (
    id, name, avatar_id, tags, description, links_json,
    display_order, is_published, created_at, updated_at
)
SELECT
    lower(hex(randomblob(12))), o.name,
    (SELECT new_id FROM _media_id_map WHERE old_id = o.avatar_id),
    o.tags, o.description, o.links_json,
    o.display_order, o.is_published, o.created_at, o.updated_at
FROM members o;
DROP TABLE members;
ALTER TABLE members_new RENAME TO members;

-- devlog_posts ---------------------------------------------------------
CREATE TABLE devlog_posts_new (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_markdown TEXT NOT NULL DEFAULT '',
    game_id TEXT REFERENCES games(id) ON DELETE SET NULL,
    og_image_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    is_published INTEGER NOT NULL DEFAULT 0,
    published_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO devlog_posts_new (
    id, slug, title, content_markdown, game_id, og_image_id,
    is_published, published_at, created_at, updated_at
)
SELECT
    lower(hex(randomblob(12))), o.slug, o.title, o.content_markdown,
    (SELECT new_id FROM _games_id_map WHERE old_id = o.game_id),
    (SELECT new_id FROM _media_id_map WHERE old_id = o.og_image_id),
    o.is_published, o.published_at, o.created_at, o.updated_at
FROM devlog_posts o;
DROP TABLE devlog_posts;
ALTER TABLE devlog_posts_new RENAME TO devlog_posts;

-- awards ---------------------------------------------------------------
CREATE TABLE awards_new (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    issuer TEXT NOT NULL,
    date TEXT NOT NULL,
    picture_id TEXT REFERENCES media(id) ON DELETE SET NULL,
    game_id TEXT REFERENCES games(id) ON DELETE SET NULL,
    link TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO awards_new (id, title, issuer, date, picture_id, game_id, link, created_at)
SELECT
    lower(hex(randomblob(12))), o.title, o.issuer, o.date,
    (SELECT new_id FROM _media_id_map WHERE old_id = o.picture_id),
    (SELECT new_id FROM _games_id_map WHERE old_id = o.game_id),
    o.link, o.created_at
FROM awards o;
DROP TABLE awards;
ALTER TABLE awards_new RENAME TO awards;

-- contact_submissions --------------------------------------------------
CREATE TABLE contact_submissions_new (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    phone TEXT,
    email TEXT,
    message TEXT NOT NULL,
    wants_callback INTEGER NOT NULL DEFAULT 0,
    is_read INTEGER NOT NULL DEFAULT 0,
    ip_address TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO contact_submissions_new (
    id, subject, phone, email, message, wants_callback, is_read, ip_address, created_at
)
SELECT
    lower(hex(randomblob(12))), o.subject, o.phone, o.email, o.message,
    o.wants_callback, o.is_read, o.ip_address, o.created_at
FROM contact_submissions o;
DROP TABLE contact_submissions;
ALTER TABLE contact_submissions_new RENAME TO contact_submissions;

-- Cache-invalidation tags embed the game id ("game:12" -> "game:<cuid>").
UPDATE page_tags
SET tag = 'game:' || (SELECT new_id FROM _games_id_map WHERE 'game:' || old_id = page_tags.tag)
WHERE EXISTS (SELECT 1 FROM _games_id_map WHERE 'game:' || old_id = page_tags.tag);

UPDATE regen_jobs
SET tag = 'game:' || (SELECT new_id FROM _games_id_map WHERE 'game:' || old_id = regen_jobs.tag)
WHERE EXISTS (SELECT 1 FROM _games_id_map WHERE 'game:' || old_id = regen_jobs.tag);

DROP TABLE _media_id_map;
DROP TABLE _games_id_map;

CREATE INDEX idx_game_screenshots_game_id ON game_screenshots(game_id);
