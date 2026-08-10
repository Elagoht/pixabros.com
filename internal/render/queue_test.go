package render

import (
	"context"
	"path/filepath"
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
	processed, err := worker.ProcessOnce()
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
	if _, err := worker.ProcessOnce(); err != nil {
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
