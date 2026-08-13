-- Gives a game a release date, a genre, and a kind.
--
-- All three default to something rather than being nullable, because every
-- existing row predates the columns and "not filled in yet" is a normal state
-- for a game, not a broken one. An empty date or genre simply does not render.
--
-- kind separates a jam entry from a production game. It defaults to
-- 'production' because that is what the studio's own releases are, and a jam
-- entry is the thing worth marking. The CHECK is what keeps the column to the
-- two values the site knows how to draw: a third value would reach the public
-- badge with nothing behind it.

ALTER TABLE games ADD COLUMN release_date TEXT NOT NULL DEFAULT '';
ALTER TABLE games ADD COLUMN genre TEXT NOT NULL DEFAULT '';
ALTER TABLE games ADD COLUMN kind TEXT NOT NULL DEFAULT 'production'
  CHECK (kind IN ('gamejam', 'production'));
