package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_EnablesForeignKeysAndWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	var fk int
	if err := conn.QueryRow(`PRAGMA foreign_keys;`).Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	var mode string
	if err := conn.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode pragma: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestOpen_CreatesMissingParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "sub", "test.db")

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected db file to exist at %q, stat error = %v", path, err)
	}
}

func TestForeignKeys_AreEnforced(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()
	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Inserting a session with a nonexistent admin_id must fail.
	_, err = conn.Exec(`INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES (?, ?, ?);`,
		999999, "deadbeef", "2099-01-01T00:00:00Z")
	if err == nil {
		t.Error("expected an error inserting a session with a nonexistent admin_id, got nil")
	}

	// Deleting an admin must cascade-delete their sessions (per the schema's ON DELETE CASCADE).
	res, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "fktest", "hash")
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	adminID, _ := res.LastInsertId()
	if _, err := conn.Exec(`INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES (?, ?, ?);`,
		adminID, "cafebabe", "2099-01-01T00:00:00Z"); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := conn.Exec(`DELETE FROM admins WHERE id = ?;`, adminID); err != nil {
		t.Fatalf("delete admin: %v", err)
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE admin_id = ?;`, adminID).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("sessions for deleted admin = %d, want 0 (ON DELETE CASCADE should have removed them)", count)
	}
}
