-- The settings tables hold a few values that are media ids (org_logo,
-- default_og_image, hero_logo per the data-model spec), but value_type only
-- allowed 'text' and 'uri'. Storing a media id as 'text' would leave the
-- daily orphan-media sweep unable to tell which settings reference an image,
-- so it would delete artwork that is still in use.
--
-- SQLite cannot alter a CHECK constraint, so both tables are rebuilt. They
-- are keyed by `key` rather than an id and nothing references them, so there
-- is no foreign key to remap -- but db.Migrate still disables foreign keys
-- around the run and verifies integrity afterwards.

CREATE TABLE site_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','media'))
);
INSERT INTO site_settings_new (key, value, value_type)
SELECT key, value, value_type FROM site_settings;
DROP TABLE site_settings;
ALTER TABLE site_settings_new RENAME TO site_settings;

CREATE TABLE homepage_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','media'))
);
INSERT INTO homepage_settings_new (key, value, value_type)
SELECT key, value, value_type FROM homepage_settings;
DROP TABLE homepage_settings;
ALTER TABLE homepage_settings_new RENAME TO homepage_settings;
