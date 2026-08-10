package render

import (
	"context"
	"database/sql"
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
// jobs it processed (successfully or not).
func (w *Worker) ProcessOnce() (int, error) {
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

	for _, j := range jobs {
		if _, err := w.db.Exec(`UPDATE regen_jobs SET status = 'processing' WHERE id = ?;`, j.id); err != nil {
			return 0, err
		}
		if err := w.processTag(j.tag); err != nil {
			w.db.Exec(
				`UPDATE regen_jobs SET status = 'failed', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'), error = ? WHERE id = ?;`,
				err.Error(), j.id,
			)
			continue
		}
		w.db.Exec(
			`UPDATE regen_jobs SET status = 'done', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
			j.id,
		)
	}
	return len(jobs), nil
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

// Run polls for pending jobs every pollInterval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessOnce()
		}
	}
}
