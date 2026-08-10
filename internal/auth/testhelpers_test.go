package auth

import (
	"database/sql"
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

func seedAdmin(t *testing.T, conn *sql.DB, username, passwordHash string) int64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, username, passwordHash)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("get inserted admin id: %v", err)
	}
	return id
}
