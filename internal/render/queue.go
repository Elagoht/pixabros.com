package render

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func EnqueueRegen(db *sql.DB, tag string) error {
	_, err := db.Exec(`INSERT INTO regen_jobs (tag) VALUES (?);`, tag)
	return err
}

type Worker struct {
	db           *sql.DB
	registry     *Registry
	store        *Store
	pollInterval time.Duration
}

func NewWorker(db *sql.DB, registry *Registry, store *Store, pollInterval time.Duration) *Worker {
	return &Worker{db: db, registry: registry, store: store, pollInterval: pollInterval}
}

// ProcessOnce drains every pending regen job once. It returns the number of
// jobs it processed (successfully or not), and any errors encountered while
// recording terminal job statuses (which are aggregated but do not stop processing).
func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	rows, err := w.db.Query(`SELECT id, tag FROM regen_jobs WHERE status = 'pending' ORDER BY id;`)
	if err != nil {
		return 0, err
	}
	type job struct {
		id  int64
		tag string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.tag); err != nil {
			rows.Close()
			return 0, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	var statusErrs []error
	for _, j := range jobs {
		// Allow context cancellation between jobs.
		if err := ctx.Err(); err != nil {
			return len(jobs), errors.Join(statusErrs...)
		}

		if _, err := w.db.Exec(`UPDATE regen_jobs SET status = 'processing' WHERE id = ?;`, j.id); err != nil {
			return 0, err
		}
		if err := w.processTag(j.tag); err != nil {
			if _, upErr := w.db.Exec(
				`UPDATE regen_jobs SET status = 'failed', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'), error = ? WHERE id = ?;`,
				err.Error(), j.id,
			); upErr != nil {
				statusErrs = append(statusErrs, fmt.Errorf("mark job %d failed: %w", j.id, upErr))
			}
			continue
		}
		if _, upErr := w.db.Exec(
			`UPDATE regen_jobs SET status = 'done', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
			j.id,
		); upErr != nil {
			statusErrs = append(statusErrs, fmt.Errorf("mark job %d done: %w", j.id, upErr))
		}
	}
	return len(jobs), errors.Join(statusErrs...)
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

// recoverStaleJobs resets any jobs stuck in 'processing' status back to
// 'pending', so they can be retried. This handles the case where a previous
// worker invocation crashed mid-processTag.
func (w *Worker) recoverStaleJobs() error {
	_, err := w.db.Exec(`UPDATE regen_jobs SET status = 'pending' WHERE status = 'processing';`)
	return err
}

// Run recovers any stale jobs, then polls for pending jobs every pollInterval
// until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	// Recover any jobs stuck in processing from a previous crash.
	w.recoverStaleJobs()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessOnce(ctx)
		}
	}
}
