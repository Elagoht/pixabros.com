package render

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MaxAttempts is how many times a tag is retried before the worker gives up
// on it. Beyond this the failure is almost certainly a code bug -- a missing
// renderer, a broken template -- which retrying will not fix, so it is logged
// rather than retried forever.
const MaxAttempts = 5

// retryBackoff is the wait before attempt n+1. It is deliberately short at the
// start: the common transient failure is a locked database during a burst of
// saves, which clears in milliseconds.
func retryBackoff(attempts int) time.Duration {
	switch attempts {
	case 1:
		return 2 * time.Second
	case 2:
		return 15 * time.Second
	case 3:
		return time.Minute
	default:
		return 5 * time.Minute
	}
}

// EnqueueRegen asks for a tag's pages to be re-rendered.
//
// A tag already waiting is not queued twice. Editing one game fifty times used
// to leave fifty identical 'game:list' rows, every one of which would
// re-render the same page once the public site exists. Collapsing them is safe
// because a job carries no payload beyond its tag: one pending 'game:list'
// re-renders whatever the list looks like when it runs.
func EnqueueRegen(db *sql.DB, tag string) error {
	_, err := db.Exec(
		`INSERT INTO regen_jobs (tag)
		 SELECT ?
		 WHERE NOT EXISTS (
		   SELECT 1 FROM regen_jobs WHERE tag = ? AND status = 'pending'
		 );`,
		tag, tag,
	)
	return err
}

type Worker struct {
	db           *sql.DB
	registry     *Registry
	store        *Store
	pollInterval time.Duration
	onError      func(error)
}

type WorkerOption func(*Worker)

// WithErrorLogger registers a callback invoked with any error encountered
// while polling or processing jobs. The default is a no-op.
func WithErrorLogger(onError func(error)) WorkerOption {
	return func(w *Worker) { w.onError = onError }
}

func NewWorker(db *sql.DB, registry *Registry, store *Store, pollInterval time.Duration, opts ...WorkerOption) *Worker {
	w := &Worker{db: db, registry: registry, store: store, pollInterval: pollInterval, onError: func(error) {}}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// ProcessOnce drains every claimable regen job once: pending, and either
// never attempted or past its backoff deadline. It returns the number of jobs
// it processed (successfully or not), and any errors encountered while
// recording job statuses (which are aggregated but do not stop processing).
func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	rows, err := w.db.Query(
		`SELECT id, tag, attempts FROM regen_jobs
		 WHERE status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
		 ORDER BY id;`,
		now.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	type job struct {
		id       int64
		tag      string
		attempts int
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.tag, &j.attempts); err != nil {
			rows.Close()
			return 0, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	var statusErrs []error
	processed := 0
	for _, j := range jobs {
		// Allow context cancellation between jobs.
		if err := ctx.Err(); err != nil {
			return processed, errors.Join(statusErrs...)
		}

		if _, err := w.db.Exec(`UPDATE regen_jobs SET status = 'processing' WHERE id = ?;`, j.id); err != nil {
			return processed, errors.Join(append(statusErrs, err)...)
		}
		if err := w.processTag(j.tag); err != nil {
			if upErr := w.recordFailure(j.id, j.tag, j.attempts, err, now); upErr != nil {
				statusErrs = append(statusErrs, upErr)
			}
			processed++
			continue
		}
		if _, upErr := w.db.Exec(
			`UPDATE regen_jobs SET status = 'done', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
			j.id,
		); upErr != nil {
			statusErrs = append(statusErrs, fmt.Errorf("mark job %d done: %w", j.id, upErr))
		}
		processed++
	}
	return processed, errors.Join(statusErrs...)
}

func (w *Worker) processTag(tag string) error {
	pageKeys, err := w.store.PageKeysForTag(tag)
	if err != nil {
		return err
	}
	for _, pageKey := range pageKeys {
		renderer, ok := w.registry.Resolve(pageKey)
		if !ok {
			return fmt.Errorf("no renderer registered for page key %q (tag %q)", pageKey, tag)
		}
		if _, err := w.store.RenderAndPersist(pageKey, renderer); err != nil {
			return fmt.Errorf("render %q: %w", pageKey, err)
		}
	}
	return nil
}

// recordFailure schedules a retry, or gives up once MaxAttempts is reached.
//
// Giving up is reported through onError rather than left sitting in a table
// for someone to notice: at that point the cause is a code bug, so it belongs
// in the operator's log, not in the content editor's way.
func (w *Worker) recordFailure(jobID int64, tag string, attempts int, cause error, now time.Time) error {
	attempts++

	if attempts >= MaxAttempts {
		w.onError(fmt.Errorf(
			"regen tag %q failed %d times, giving up: %w", tag, attempts, cause,
		))
		if _, err := w.db.Exec(
			`UPDATE regen_jobs
			 SET status = 'failed', attempts = ?, next_attempt_at = NULL,
			     processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'), error = ?
			 WHERE id = ?;`,
			attempts, cause.Error(), jobID,
		); err != nil {
			return fmt.Errorf("mark job %d failed: %w", jobID, err)
		}
		return nil
	}

	// Back to pending with a deadline, so the next poll leaves it alone until
	// the backoff has elapsed.
	retryAt := now.Add(retryBackoff(attempts))
	if _, err := w.db.Exec(
		`UPDATE regen_jobs
		 SET status = 'pending', attempts = ?, next_attempt_at = ?, error = ?
		 WHERE id = ?;`,
		attempts, retryAt.Format(time.RFC3339), cause.Error(), jobID,
	); err != nil {
		return fmt.Errorf("schedule retry for job %d: %w", jobID, err)
	}
	return nil
}

// PruneJobs deletes finished jobs that are old enough to be of no interest.
// Nothing removed them before, so the table grew without bound: a few hundred
// rows after light use, and one row per save forever after. Failed jobs are
// kept far longer than done ones, because they are the only record of
// something having gone wrong.
func (w *Worker) PruneJobs(doneOlderThan, failedOlderThan time.Duration, now time.Time) (int64, error) {
	res, err := w.db.Exec(
		`DELETE FROM regen_jobs
		 WHERE (status = 'done' AND processed_at IS NOT NULL AND processed_at < ?)
		    OR (status = 'failed' AND processed_at IS NOT NULL AND processed_at < ?);`,
		now.Add(-doneOlderThan).UTC().Format(time.RFC3339),
		now.Add(-failedOlderThan).UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// recoverStaleJobs resets any jobs stuck in 'processing' status back to
// 'pending', so they can be retried. This handles the case where a previous
// worker invocation crashed mid-processTag.
func (w *Worker) recoverStaleJobs() error {
	_, err := w.db.Exec(`UPDATE regen_jobs SET status = 'pending' WHERE status = 'processing';`)
	return err
}

// DoneRetention and FailedRetention are how long finished jobs are kept. A
// done job is only of interest for a moment; a failed one is evidence, so it
// outlives the log line that reported it.
const (
	DoneRetention   = 24 * time.Hour
	FailedRetention = 30 * 24 * time.Hour
)

// pruneInterval is how often finished jobs are swept up. Pruning is
// maintenance, not work anyone waits for, so it runs far less often than the
// poll that drains the queue.
const pruneInterval = time.Hour

// Run recovers any stale jobs, then polls for claimable jobs every
// pollInterval until ctx is cancelled, pruning finished ones as it goes. This
// is the whole of regeneration's management: it starts with the server, needs
// no schedule outside the process, and surfaces nothing to the admin UI.
// Errors go to the callback registered via WithErrorLogger, which is where a
// tag the queue has given up on ends up -- by then the cause is a code bug, so
// an operator needs to see it and nobody else can act on it.
func (w *Worker) Run(ctx context.Context) {
	// Recover any jobs stuck in processing from a previous crash.
	if err := w.recoverStaleJobs(); err != nil {
		w.onError(fmt.Errorf("recoverStaleJobs: %w", err))
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Finished jobs are cleaned up on the same goroutine as the work, so the
	// queue looks after itself with nothing scheduled outside the process.
	pruner := time.NewTicker(pruneInterval)
	defer pruner.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.ProcessOnce(ctx); err != nil {
				w.onError(fmt.Errorf("ProcessOnce: %w", err))
			}
		case <-pruner.C:
			if _, err := w.PruneJobs(DoneRetention, FailedRetention, time.Now()); err != nil {
				w.onError(fmt.Errorf("PruneJobs: %w", err))
			}
		}
	}
}
