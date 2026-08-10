package render

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func TestEnqueueRegen_InsertsPendingJob(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	if err := EnqueueRegen(conn, "homepage"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	var status, tag string
	err = conn.QueryRow(`SELECT tag, status FROM regen_jobs;`).Scan(&tag, &status)
	if err != nil {
		t.Fatalf("query regen_jobs: %v", err)
	}
	if tag != "homepage" || status != "pending" {
		t.Errorf("job = (%q, %q), want (\"homepage\", \"pending\")", tag, status)
	}
}

func TestWorker_ProcessOnce_RendersDependentPages(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	renderCount := 0
	registry.Register("index.html", func(pageKey string) ([]byte, []string, error) {
		renderCount++
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})

	// Seed page_tags as if index.html had been rendered once before, so the
	// worker knows it depends on "homepage".
	if _, err := store.RenderAndPersist("index.html", registry.exact["index.html"]); err != nil {
		t.Fatalf("seed RenderAndPersist() error = %v", err)
	}
	renderCount = 0

	if err := EnqueueRegen(conn, "homepage"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	processed, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}
	if renderCount != 1 {
		t.Errorf("renderCount = %d, want 1", renderCount)
	}

	var status string
	if err := conn.QueryRow(`SELECT status FROM regen_jobs;`).Scan(&status); err != nil {
		t.Fatalf("query job status: %v", err)
	}
	if status != "done" {
		t.Errorf("status = %q, want %q", status, "done")
	}
}

func TestWorker_ProcessOnce_MarksJobFailedWhenNoRendererRegistered(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	// A page_tags row exists (as if rendered before) but no renderer is
	// registered anymore for it — this must fail the job, not panic.
	if _, err := conn.Exec(`INSERT INTO rendered_pages (page_key, etag) VALUES ('orphaned.html', 'x');`); err != nil {
		t.Fatalf("seed rendered_pages: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO page_tags (page_key, tag) VALUES ('orphaned.html', 'ghost-tag');`); err != nil {
		t.Fatalf("seed page_tags: %v", err)
	}

	if err := EnqueueRegen(conn, "ghost-tag"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	var status string
	var jobErr string
	if err := conn.QueryRow(`SELECT status, error FROM regen_jobs;`).Scan(&status, &jobErr); err != nil {
		t.Fatalf("query job: %v", err)
	}
	if status != "failed" {
		t.Errorf("status = %q, want %q", status, "failed")
	}
	if jobErr == "" {
		t.Error("expected a non-empty error message on the failed job")
	}
}

func TestWorker_Run_StopsOnContextCancel(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	worker := NewWorker(conn, NewRegistry(), NewStore(conn, files), 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestWorker_RecoverStaleJobs(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	// Register a renderer so the job can complete.
	registry.Register("index.html", func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})

	// Seed a page and its tag dependency.
	if _, err := store.RenderAndPersist("index.html", registry.exact["index.html"]); err != nil {
		t.Fatalf("seed RenderAndPersist() error = %v", err)
	}

	// Manually insert a job stuck in "processing" (as if a previous crash left it there).
	if _, err := conn.Exec(`INSERT INTO regen_jobs (tag, status) VALUES ('homepage', 'processing');`); err != nil {
		t.Fatalf("seed processing job: %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	if err := worker.recoverStaleJobs(); err != nil {
		t.Fatalf("recoverStaleJobs() error = %v", err)
	}

	var status string
	if err := conn.QueryRow(`SELECT status FROM regen_jobs WHERE tag = 'homepage';`).Scan(&status); err != nil {
		t.Fatalf("query job status: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want %q", status, "pending")
	}

	// Now process the recovered job and verify it completes successfully.
	processed, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}

	if err := conn.QueryRow(`SELECT status FROM regen_jobs WHERE tag = 'homepage';`).Scan(&status); err != nil {
		t.Fatalf("query job status: %v", err)
	}
	if status != "done" {
		t.Errorf("final status = %q, want %q", status, "done")
	}
}

func TestWorker_ProcessOnce_StopsOnContextCancellation(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	// Register renderers for multiple pages.
	registry.Register("page1.html", func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>page 1</h1>"), []string{"my-tag"}, nil
	})
	registry.Register("page2.html", func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>page 2</h1>"), []string{"my-tag"}, nil
	})
	registry.Register("page3.html", func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>page 3</h1>"), []string{"my-tag"}, nil
	})

	// Seed all pages so they exist in the database with tag dependencies.
	if _, err := store.RenderAndPersist("page1.html", registry.exact["page1.html"]); err != nil {
		t.Fatalf("seed page1: %v", err)
	}
	if _, err := store.RenderAndPersist("page2.html", registry.exact["page2.html"]); err != nil {
		t.Fatalf("seed page2: %v", err)
	}
	if _, err := store.RenderAndPersist("page3.html", registry.exact["page3.html"]); err != nil {
		t.Fatalf("seed page3: %v", err)
	}

	// Enqueue three jobs for the same tag.
	if err := EnqueueRegen(conn, "my-tag"); err != nil {
		t.Fatalf("EnqueueRegen first: %v", err)
	}
	if err := EnqueueRegen(conn, "my-tag"); err != nil {
		t.Fatalf("EnqueueRegen second: %v", err)
	}
	if err := EnqueueRegen(conn, "my-tag"); err != nil {
		t.Fatalf("EnqueueRegen third: %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)

	// Cancel context before calling ProcessOnce to guarantee 0 jobs are processed.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processed, err := worker.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	// With context already cancelled, no jobs should have been processed.
	// The return value should be exactly 0, not len(jobs).
	if processed != 0 {
		t.Errorf("processed = %d, want exactly 0 (context was pre-cancelled)", processed)
	}

	// Verify that all jobs are still in 'pending' status, not 'processing' or 'done'.
	// This proves the loop genuinely stopped early, not just returned with a wrong count.
	var pendingCount, processingCount, doneCount, failedCount int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE status = 'pending';`).Scan(&pendingCount); err != nil {
		t.Fatalf("query pending jobs: %v", err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE status = 'processing';`).Scan(&processingCount); err != nil {
		t.Fatalf("query processing jobs: %v", err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE status = 'done';`).Scan(&doneCount); err != nil {
		t.Fatalf("query done jobs: %v", err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE status = 'failed';`).Scan(&failedCount); err != nil {
		t.Fatalf("query failed jobs: %v", err)
	}

	if pendingCount != 3 {
		t.Errorf("pending jobs = %d, want 3", pendingCount)
	}
	if processingCount != 0 {
		t.Errorf("processing jobs = %d, want 0", processingCount)
	}
	if doneCount != 0 {
		t.Errorf("done jobs = %d, want 0", doneCount)
	}
	if failedCount != 0 {
		t.Errorf("failed jobs = %d, want 0", failedCount)
	}
}

// TestWorker_Run_ReportsErrorsToErrorLogger proves the injectable error
// callback actually fires: without it, a render pipeline failure produces no
// signal at all in production.
func TestWorker_Run_ReportsErrorsToErrorLogger(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)

	// Close the connection so every DB statement the worker issues fails.
	conn.Close()

	errCh := make(chan error, 16)
	worker := NewWorker(conn, NewRegistry(), store, time.Millisecond, WithErrorLogger(func(err error) {
		select {
		case errCh <- err:
		default:
		}
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	sawProcessOnce := false
	for !sawProcessOnce {
		select {
		case reported := <-errCh:
			if strings.Contains(reported.Error(), "ProcessOnce") {
				sawProcessOnce = true
			}
		case <-deadline:
			t.Fatal("no ProcessOnce error was reported to the error logger")
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

// TestWorker_ProcessOnce_ReportsPartialProgressWhenMarkProcessingFails uses a
// SQLite trigger to make exactly the mark-processing UPDATE fail on the second
// job of a two-job batch, and asserts the returned count still reflects the
// job that completed before the failure.
func TestWorker_ProcessOnce_ReportsPartialProgressWhenMarkProcessingFails(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()
	registry.Register("page1.html", func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>page 1</h1>"), []string{"good-tag"}, nil
	})
	if _, err := store.RenderAndPersist("page1.html", registry.exact["page1.html"]); err != nil {
		t.Fatalf("seed page1: %v", err)
	}

	if _, err := conn.Exec(`
		CREATE TRIGGER fail_mark_processing BEFORE UPDATE OF status ON regen_jobs
		WHEN NEW.status = 'processing' AND OLD.tag = 'boom-tag'
		BEGIN
			SELECT RAISE(ABORT, 'simulated failure marking job processing');
		END;`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if err := EnqueueRegen(conn, "good-tag"); err != nil {
		t.Fatalf("EnqueueRegen(good-tag): %v", err)
	}
	if err := EnqueueRegen(conn, "boom-tag"); err != nil {
		t.Fatalf("EnqueueRegen(boom-tag): %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	processed, err := worker.ProcessOnce(context.Background())
	if err == nil {
		t.Fatal("ProcessOnce() should return the mark-processing failure")
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1 (the job completed before the failure)", processed)
	}

	var status string
	if err := conn.QueryRow(`SELECT status FROM regen_jobs WHERE tag = 'good-tag';`).Scan(&status); err != nil {
		t.Fatalf("query good-tag job status: %v", err)
	}
	if status != "done" {
		t.Errorf("good-tag job status = %q, want %q", status, "done")
	}
}
