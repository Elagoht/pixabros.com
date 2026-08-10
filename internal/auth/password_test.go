package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Error("HashPassword() returned the plaintext password")
	}
	if !VerifyPassword(hash, "correct-horse-battery-staple") {
		t.Error("VerifyPassword() = false for the correct password, want true")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("VerifyPassword() = true for a wrong password, want false")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("a", 7)); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("ValidatePassword(7 chars) error = %v, want ErrPasswordTooShort", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 73)); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("ValidatePassword(73 chars) error = %v, want ErrPasswordTooLong", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 8)); err != nil {
		t.Errorf("ValidatePassword(8 chars) error = %v, want nil", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 72)); err != nil {
		t.Errorf("ValidatePassword(72 chars) error = %v, want nil", err)
	}
}
