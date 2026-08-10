package auth

import (
	"errors"
	"testing"
	"time"
)

func TestSessionStore_CreateValidateDelete(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "hash")

	store := NewSessionStore(conn)
	token, expiresAt, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if token == "" {
		t.Fatal("Create() returned an empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Error("Create() expiresAt should be in the future")
	}

	sess, err := store.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if sess.AdminID != adminID {
		t.Errorf("Validate() AdminID = %d, want %d", sess.AdminID, adminID)
	}

	if err := store.Delete(token); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Validate(token); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() after Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionStore_ValidateRejectsExpired(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "hash")

	store := NewSessionStore(conn)
	token, _, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`UPDATE sessions SET expires_at = ? WHERE admin_id = ?;`,
		time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), adminID,
	); err != nil {
		t.Fatalf("force-expire session: %v", err)
	}

	if _, err := store.Validate(token); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionStore_ValidateRejectsUnknownToken(t *testing.T) {
	conn := setupTestDB(t)
	store := NewSessionStore(conn)

	if _, err := store.Validate("does-not-exist"); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionStore_DeleteAllForAdmin(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "hash")

	store := NewSessionStore(conn)
	token1, _, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	token2, _, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.DeleteAllForAdmin(adminID); err != nil {
		t.Fatalf("DeleteAllForAdmin() error = %v", err)
	}

	if _, err := store.Validate(token1); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate(token1) error = %v, want ErrSessionNotFound", err)
	}
	if _, err := store.Validate(token2); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate(token2) error = %v, want ErrSessionNotFound", err)
	}
}
