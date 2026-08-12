package id

import (
	"strings"
	"testing"
)

func TestNew_ShapeAndUniqueness(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := range 1000 {
		got := New()
		if len(got) != Length {
			t.Fatalf("New() = %q, length %d, want %d", got, len(got), Length)
		}
		if !IsValid(got) {
			t.Fatalf("IsValid(New()) = false for %q", got)
		}
		if seen[got] {
			t.Fatalf("New() returned a duplicate id %q after %d draws", got, i)
		}
		seen[got] = true
	}
}

func TestNew_IsNotSequential(t *testing.T) {
	first, second := New(), New()
	if first == second {
		t.Fatal("New() returned the same id twice in a row")
	}
	// Two ids drawn back to back must not share a long common prefix, which
	// is what a timestamp- or counter-derived id would produce.
	shared := 0
	for shared < len(first) && shared < len(second) && first[shared] == second[shared] {
		shared++
	}
	if shared > Length/2 {
		t.Errorf("consecutive ids %q and %q share a %d-character prefix, want at most %d", first, second, shared, Length/2)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"generated id", New(), true},
		{"hex backfill from migration 0015", "a3f9c21b7e04d5a68b1c9f32", true},
		{"empty", "", false},
		{"too short", strings.Repeat("a", Length-1), false},
		{"too long", strings.Repeat("a", Length+1), false},
		{"uppercase", strings.Repeat("A", Length), false},
		{"numeric legacy id", "12", false},
		{"path traversal", strings.Repeat(".", Length), false},
		{"hyphenated uuid", "550e8400-e29b-41d4-a716", false},
	}
	for _, tt := range tests {
		if got := IsValid(tt.input); got != tt.want {
			t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
