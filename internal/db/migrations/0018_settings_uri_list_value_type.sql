-- Adds 'uri_list' to the value_type constraint.
--
-- org_sameas_json holds JSON-LD's sameAs: a bare list of profile addresses.
-- It was declared as free text, which meant the admin hand-wrote JSON and
-- nothing checked it -- a bad entry would reach the public site as invalid
-- structured data. It is now its own kind, validated as a list of absolute
-- URLs, and the admin edits one field per address.
--
-- Note the cost of keeping value_type constrained at the database level:
-- every new settings kind needs a migration like this one, because SQLite
-- cannot alter a CHECK without rebuilding the table. That is deliberate --
-- the daily orphan-media sweep trusts value_type = 'media' to decide which
-- images are still in use, so it is worth having the database enforce the
-- set rather than only the Go registry.

CREATE TABLE site_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','uri_list','media'))
);
INSERT INTO site_settings_new (key, value, value_type)
SELECT key, value, value_type FROM site_settings;
DROP TABLE site_settings;
ALTER TABLE site_settings_new RENAME TO site_settings;

CREATE TABLE homepage_settings_new (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri','uri_list','media'))
);
INSERT INTO homepage_settings_new (key, value, value_type)
SELECT key, value, value_type FROM homepage_settings;
DROP TABLE homepage_settings;
ALTER TABLE homepage_settings_new RENAME TO homepage_settings;
