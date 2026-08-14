package admincmd

import (
	"database/sql"
	"fmt"
	"time"

	"pixabros/internal/media"
	"pixabros/internal/mediarefs"
)

// reportStatus answers one question: is the background work actually running?
//
// Not by asking the process -- a process that is wedged will happily say it is
// fine -- but by looking for work that should have been done and was not. A
// render job that has sat pending for minutes means nothing is draining the
// queue. An orphaned image older than the grace period plus a sweep interval
// means nothing is reclaiming disk. Both are what the failure actually looks
// like from outside, which is the only view an operator has.
func reportStatus() error {
	conn, err := openDB()
	if err != nil {
		return err
	}
	defer conn.Close()

	now := time.Now().UTC()
	healthy := true

	fmt.Println("render queue")
	queueOK, err := reportQueue(conn, now)
	if err != nil {
		return err
	}
	healthy = healthy && queueOK

	fmt.Println()
	fmt.Println("media sweep")
	sweepOK, err := reportSweep(conn, now)
	if err != nil {
		return err
	}
	healthy = healthy && sweepOK

	fmt.Println()
	if healthy {
		fmt.Println("Both background jobs are keeping up.")
		return nil
	}
	// A non-zero exit so this can be the thing a monitor runs.
	return fmt.Errorf("something is not being processed; see above")
}

// stalePending is how long a job may sit before the queue is presumed stuck.
// The worker polls every two seconds, so a minute is a hundred missed turns
// rather than a slow one.
const stalePending = time.Minute

func reportQueue(conn *sql.DB, now time.Time) (bool, error) {
	var pending, processing, failed int
	if err := conn.QueryRow(`
		SELECT
			COALESCE(SUM(status = 'pending'), 0),
			COALESCE(SUM(status = 'processing'), 0),
			COALESCE(SUM(status = 'failed'), 0)
		FROM regen_jobs;`,
	).Scan(&pending, &processing, &failed); err != nil {
		return false, fmt.Errorf("read the queue: %w", err)
	}

	fmt.Printf("  %-22s %d pending, %d processing, %d failed\n",
		"jobs", pending, processing, failed)

	var oldest sql.NullString
	if err := conn.QueryRow(
		`SELECT MIN(created_at) FROM regen_jobs WHERE status = 'pending';`,
	).Scan(&oldest); err != nil {
		return false, fmt.Errorf("read the oldest job: %w", err)
	}

	if !oldest.Valid {
		fmt.Printf("  %-22s nothing waiting\n", "oldest waiting")
		return true, nil
	}

	since, err := time.Parse(time.RFC3339, oldest.String)
	if err != nil {
		// A timestamp that will not parse is itself worth reporting rather
		// than being swallowed into a clean bill of health.
		fmt.Printf("  %-22s %s (unreadable timestamp)\n", "oldest waiting", oldest.String)
		return false, nil
	}

	age := now.Sub(since.UTC())
	fmt.Printf("  %-22s %s old\n", "oldest waiting", age.Round(time.Second))
	if age > stalePending {
		fmt.Printf("  %-22s a job has waited longer than %s: the worker is not draining the queue\n",
			"PROBLEM", stalePending)
		return false, nil
	}
	return true, nil
}

// sweepSlack is how far past its own schedule the sweep is given before it is
// presumed stopped: one grace period, one interval, and an hour to spare.
const sweepSlack = media.OrphanGrace + 6*time.Hour + time.Hour

func reportSweep(conn *sql.DB, now time.Time) (bool, error) {
	referenced, err := mediarefs.ReferencedIDs(conn)
	if err != nil {
		return false, fmt.Errorf("read media references: %w", err)
	}

	rows, err := conn.Query(`SELECT id, created_at FROM media;`)
	if err != nil {
		return false, fmt.Errorf("read media: %w", err)
	}
	defer rows.Close()

	total, orphans, overdue := 0, 0, 0
	for rows.Next() {
		var id, createdAt string
		if err := rows.Scan(&id, &createdAt); err != nil {
			return false, fmt.Errorf("scan media: %w", err)
		}
		total++
		if referenced[id] {
			continue
		}
		orphans++

		created, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			continue
		}
		if now.Sub(created.UTC()) > sweepSlack {
			overdue++
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("read media: %w", err)
	}

	fmt.Printf("  %-22s %d total, %d unreferenced\n", "images", total, orphans)
	fmt.Printf("  %-22s %d past the point of collection\n", "overdue", overdue)

	if overdue > 0 {
		fmt.Printf("  %-22s images have gone uncollected for over %s: the sweep is not running\n",
			"PROBLEM", sweepSlack)
		return false, nil
	}
	return true, nil
}
