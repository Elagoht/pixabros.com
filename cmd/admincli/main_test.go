package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateAdmin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("PIXABROS_DB_PATH", dbPath)

	if err := createAdmin("furkan", "s3cret-password"); err != nil {
		t.Fatalf("createAdmin() error = %v", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer conn.Close()

	var username, hash string
	err = conn.QueryRow(
		`SELECT username, password_hash FROM admins WHERE username = ?;`, "furkan",
	).Scan(&username, &hash)
	if err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if username != "furkan" {
		t.Errorf("username = %q, want %q", username, "furkan")
	}
	if hash == "s3cret-password" {
		t.Error("password_hash was stored in plaintext")
	}
}

func TestCreateAdmin_DuplicateUsernameFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("PIXABROS_DB_PATH", dbPath)

	if err := createAdmin("furkan", "s3cret-password"); err != nil {
		t.Fatalf("first createAdmin() error = %v", err)
	}
	if err := createAdmin("furkan", "another-password"); err == nil {
		t.Error("second createAdmin() with the same username should error")
	}
}
