-- Makes page regeneration self-healing, so nobody has to watch it.
--
-- Before this, a job that failed was marked 'failed' and never looked at
-- again: the worker only ever selects 'pending'. A transient failure meant a
-- permanently stale page with no recovery, which is why the original plan
-- included an admin screen with a manual retry button. But the failures this
-- queue produces -- "no renderer registered for page key X", a panic in a
-- template -- are code bugs, not something the person editing content can fix
-- by pressing retry. So the queue retries itself instead, and gives up loudly
-- into the log rather than quietly into a screen.
--
-- attempts counts tries so far; next_attempt_at holds a backoff deadline the
-- worker respects. A pending job with no deadline runs immediately, which is
-- what every job enqueued before this migration should do.

ALTER TABLE regen_jobs ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE regen_jobs ADD COLUMN next_attempt_at TEXT;

-- The worker now filters on (status, next_attempt_at), so the old
-- status-only index no longer covers its query.
DROP INDEX IF EXISTS idx_regen_jobs_status;
CREATE INDEX idx_regen_jobs_claimable ON regen_jobs(status, next_attempt_at);

-- Enqueueing now skips a tag that is already waiting, which needs to find
-- pending rows by tag.
CREATE INDEX idx_regen_jobs_pending_tag ON regen_jobs(tag, status);
