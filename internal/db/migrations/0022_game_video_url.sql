-- Gives a game a trailer.
--
-- Stored as the URL an admin pastes rather than a bare video id, because that
-- is what they copy out of YouTube's share dialog and what they will recognise
-- when they come back to edit it. The site reads the id out of it at render
-- time and frames only that, so the URL never reaches the page.
--
-- Empty is normal: most games have no trailer, and the player simply does not
-- render.

ALTER TABLE games ADD COLUMN video_url TEXT NOT NULL DEFAULT '';
