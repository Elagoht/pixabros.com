package media

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestRepo_CreateFindDelete(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	m, err := repo.Create("media/2026/abc.webp", 400, 400)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if m.ID == 0 {
		t.Fatal("Create() returned a zero ID")
	}
	if m.Format != "webp" {
		t.Errorf("Format = %q, want %q", m.Format, "webp")
	}

	found, err := repo.FindByID(m.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Path != "media/2026/abc.webp" {
		t.Errorf("Path = %q, want %q", found.Path, "media/2026/abc.webp")
	}

	if err := repo.Delete(m.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(m.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrMediaNotFound", err)
	}
}

func TestRepo_AllIDs(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, _ := repo.Create("a.webp", 10, 10)
	second, _ := repo.Create("b.webp", 10, 10)

	ids, err := repo.AllIDs()
	if err != nil {
		t.Fatalf("AllIDs() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("AllIDs() returned %d ids, want 2", len(ids))
	}
	seen := map[int64]bool{ids[0]: true, ids[1]: true}
	if !seen[first.ID] || !seen[second.ID] {
		t.Errorf("AllIDs() = %v, want to contain %d and %d", ids, first.ID, second.ID)
	}
}
