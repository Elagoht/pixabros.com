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

func TestCreateAdmin_ShortPasswordFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("PIXABROS_DB_PATH", dbPath)

	if err := createAdmin("furkan", "short"); err == nil {
		t.Error("createAdmin() with a password under 8 characters should return an error")
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

func TestResetPassword(t *testing.T) {
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

	var adminID int64
	var oldHash string
	if err := conn.QueryRow(
		`SELECT id, password_hash FROM admins WHERE username = ?;`, "furkan",
	).Scan(&adminID, &oldHash); err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES (?, ?, ?);`,
		adminID, "some-token-hash", "2099-01-01T00:00:00Z",
	); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := resetPassword("furkan", "brand-new-password"); err != nil {
		t.Fatalf("resetPassword() error = %v", err)
	}

	var newHash string
	if err := conn.QueryRow(
		`SELECT password_hash FROM admins WHERE id = ?;`, adminID,
	).Scan(&newHash); err != nil {
		t.Fatalf("query updated admin: %v", err)
	}
	if newHash == oldHash {
		t.Error("resetPassword() did not change the stored password hash")
	}
	if newHash == "brand-new-password" {
		t.Error("resetPassword() stored the password in plaintext")
	}

	var sessionCount int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE admin_id = ?;`, adminID,
	).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Errorf("sessions remaining after reset = %d, want 0", sessionCount)
	}
}

func TestResetPassword_UnknownAdminFails(t *testing.T) {
	t.Setenv("PIXABROS_DB_PATH", filepath.Join(t.TempDir(), "test.db"))

	if err := resetPassword("nobody", "brand-new-password"); err == nil {
		t.Error("resetPassword() for an unknown admin should return an error")
	}
}

func TestResetPassword_ShortPasswordFails(t *testing.T) {
	t.Setenv("PIXABROS_DB_PATH", filepath.Join(t.TempDir(), "test.db"))

	if err := createAdmin("furkan", "s3cret-password"); err != nil {
		t.Fatalf("createAdmin() error = %v", err)
	}
	if err := resetPassword("furkan", "short"); err == nil {
		t.Error("resetPassword() with a password under 8 characters should return an error")
	}
}
