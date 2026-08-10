package auth

import (
	"errors"
	"testing"
)

func TestAdminRepo_FindByUsernameAndID(t *testing.T) {
	conn := setupTestDB(t)
	seedAdmin(t, conn, "furkan", "hash123")

	repo := NewAdminRepo(conn)

	byUsername, err := repo.FindByUsername("furkan")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if byUsername.PasswordHash != "hash123" {
		t.Errorf("PasswordHash = %q, want %q", byUsername.PasswordHash, "hash123")
	}

	byID, err := repo.FindByID(byUsername.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if byID.Username != "furkan" {
		t.Errorf("Username = %q, want %q", byID.Username, "furkan")
	}

	if _, err := repo.FindByUsername("nobody"); !errors.Is(err, ErrAdminNotFound) {
		t.Errorf("FindByUsername() error = %v, want ErrAdminNotFound", err)
	}
	if _, err := repo.FindByID(9999); !errors.Is(err, ErrAdminNotFound) {
		t.Errorf("FindByID() error = %v, want ErrAdminNotFound", err)
	}
}

func TestAdminRepo_UpdatePasswordHash(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "old-hash")
	repo := NewAdminRepo(conn)

	if err := repo.UpdatePasswordHash(adminID, "new-hash"); err != nil {
		t.Fatalf("UpdatePasswordHash() error = %v", err)
	}

	updated, err := repo.FindByID(adminID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want %q", updated.PasswordHash, "new-hash")
	}
}
