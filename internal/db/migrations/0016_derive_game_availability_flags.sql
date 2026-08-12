-- Both availability flags stopped being editable and became derived.
--
-- is_browser_playable now means exactly "a playable build exists", and is
-- written only by games.Repo.SetBuild -- on upload, and cleared again when
-- the build is removed. Existing rows are re-derived from web_export_path so
-- the column agrees with reality rather than with whatever the old form last
-- submitted.
--
-- is_downloadable is forced off and stays off: direct downloads are never
-- offered, so nothing writes this column any more. It is kept rather than
-- dropped so the decision stays visible in the schema, and so re-introducing
-- downloads later does not need a data migration.

UPDATE games
SET is_browser_playable = (web_export_path IS NOT NULL AND web_export_path != '');

UPDATE games SET is_downloadable = 0;
