-- Gives contact submissions a sender name.
--
-- The public form deliberately does not ask for one (the design spec settles
-- its fields as subject, phone, email, message and the callback checkbox), so
-- this column exists for submissions that arrived with a name from somewhere
-- else -- imported history, or a form that did collect one.
--
-- Nullable on purpose: a row without a name is normal, not incomplete, and
-- every submission already in the table predates the column.

ALTER TABLE contact_submissions ADD COLUMN name TEXT;
