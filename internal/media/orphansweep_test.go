package media

import (
	"errors"
	"io"
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

	lookup := func() (map[string]bool, error) {
		return map[string]bool{referenced.ID: true}, nil
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

// failingStorage is a test double that fails to delete one specific path.
type failingStorage struct {
	delegate  storage.Storage
	failPath  string
	deleteErr error
}

func (s *failingStorage) Put(path string, r io.Reader) error {
	return s.delegate.Put(path, r)
}

func (s *failingStorage) Get(path string) (io.ReadCloser, error) {
	return s.delegate.Get(path)
}

func (s *failingStorage) Delete(path string) error {
	if path == s.failPath {
		return s.deleteErr
	}
	return s.delegate.Delete(path)
}

func (s *failingStorage) URL(path string) string {
	return s.delegate.URL(path)
}

func TestSweepOrphans_ContinuesOnSingleCandidateFailure(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	repo := NewRepo(conn)
	realFiles := storage.NewLocalDisk(t.TempDir(), "/media")
	files := &failingStorage{
		delegate:  realFiles,
		failPath:  "",
		deleteErr: errors.New("simulated delete failure"),
	}

	// Create two orphaned candidates, both old enough to delete
	failingOrphan, err := repo.Create("fail-orphan.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(failingOrphan.Path, strings.NewReader("x"))
	if _, err := conn.Exec(`UPDATE media SET created_at = ? WHERE id = ?;`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339), failingOrphan.ID); err != nil {
		t.Fatalf("backdate failing orphan: %v", err)
	}

	successOrphan, err := repo.Create("success-orphan.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(successOrphan.Path, strings.NewReader("x"))
	if _, err := conn.Exec(`UPDATE media SET created_at = ? WHERE id = ?;`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339), successOrphan.ID); err != nil {
		t.Fatalf("backdate success orphan: %v", err)
	}

	// Configure storage to fail only for the failing orphan's file
	files.failPath = failingOrphan.Path

	lookup := func() (map[string]bool, error) {
		return map[string]bool{}, nil // Nothing is referenced
	}

	deleted, err := SweepOrphans(repo, files, lookup, 24*time.Hour, time.Now())

	// Should have deleted 1 (the success candidate)
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Should have a non-nil error (the failing candidate's error)
	if err == nil {
		t.Error("SweepOrphans() should return non-nil error when a candidate fails")
	}

	// Failing candidate's row and file should still exist
	if _, err := repo.FindByID(failingOrphan.ID); err != nil {
		t.Error("failing orphan media row should still exist after failed delete")
	}
	if _, err := files.Get(failingOrphan.Path); err != nil {
		t.Error("failing orphan media file should still exist after failed delete")
	}

	// Success candidate's row and file should be gone
	if _, err := repo.FindByID(successOrphan.ID); err == nil {
		t.Error("success orphan media row should have been deleted")
	}
	if _, err := files.Get(successOrphan.Path); err == nil {
		t.Error("success orphan media file should have been deleted")
	}
}
