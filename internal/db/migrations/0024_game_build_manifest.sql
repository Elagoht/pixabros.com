-- What a game's playable build is made of, so a visitor can be told the size
-- before being asked to download it and so a re-upload can be recognised.
--
-- build_version is derived from the contents of every file in the build, not
-- from a clock: re-uploading the same archive must not ask anyone to download
-- 90 MB they already hold. build_files_json follows external_links_json's
-- convention -- the list is only ever read whole, so a table of its own would
-- buy nothing.

ALTER TABLE games ADD COLUMN build_version TEXT NOT NULL DEFAULT '';
ALTER TABLE games ADD COLUMN build_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE games ADD COLUMN build_files_json TEXT NOT NULL DEFAULT '[]';
