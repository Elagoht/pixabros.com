package media

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func TestSweepOrphans_DeletesOnlyUnreferencedAndOld(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	repo := NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")

	referenced, err := repo.Create("referenced.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(referenced.Path, strings.NewReader("x"))

	tooNew, err := repo.Create("too-new.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(tooNew.Path, strings.NewReader("x"))

	orphan, err := repo.Create("orphan.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(orphan.Path, strings.NewReader("x"))
	if _, err := conn.Exec(`UPDATE media SET created_at = ? WHERE id = ?;`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339), orphan.ID); err != nil {
		t.Fatalf("backdate orphan media: %v", err)
	}

	lookup := func() (map[int64]bool, error) {
		return map[int64]bool{referenced.ID: true}, nil
	}

	deleted, err := SweepOrphans(repo, files, lookup, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("SweepOrphans() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	if _, err := repo.FindByID(orphan.ID); err == nil {
		t.Error("orphan media row should have been deleted")
	}
	if _, err := files.Get(orphan.Path); err == nil {
		t.Error("orphan media file should have been deleted")
	}

	if _, err := repo.FindByID(referenced.ID); err != nil {
		t.Error("referenced media row should not have been deleted")
	}
	if _, err := repo.FindByID(tooNew.ID); err != nil {
		t.Error("too-new unreferenced media row should not have been deleted yet")
	}
}
