package auth

import "testing"

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
