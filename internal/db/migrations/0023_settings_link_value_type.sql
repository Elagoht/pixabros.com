-- Lets a setting be a link: a path on this site, or a whole address elsewhere.
--
-- The homepage's call-to-action is the one that needed it. Demanding a full
-- URL for a button that points at /games meant typing the domain in twice and
-- breaking the link the day the domain changed.
--
-- SQLite cannot alter a CHECK, so each table is rebuilt. The column list is
-- spelled out rather than using SELECT *, so a future column added to one of
-- these tables cannot silently reorder the copy.

CREATE TABLE site_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','uri_list','media','link'))
);
INSERT INTO site_settings_new (key, value, value_type)
    SELECT key, value, value_type FROM site_settings;
DROP TABLE site_settings;
ALTER TABLE site_settings_new RENAME TO site_settings;

CREATE TABLE homepage_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','uri_list','media','link'))
);
INSERT INTO homepage_settings_new (key, value, value_type)
    SELECT key, value, value_type FROM homepage_settings;
DROP TABLE homepage_settings;
ALTER TABLE homepage_settings_new RENAME TO homepage_settings;
