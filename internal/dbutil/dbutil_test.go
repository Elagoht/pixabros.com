package dbutil

import (
	"errors"
	"testing"
	"time"
)

func TestNullableString(t *testing.T) {
	if got := NullableString(""); got != nil {
		t.Errorf("NullableString(\"\") = %v, want nil", got)
	}
	if got := NullableString("x"); got != "x" {
		t.Errorf("NullableString(%q) = %v, want %q", "x", got, "x")
	}
}

func TestNullableID(t *testing.T) {
	if got := NullableID(nil); got != nil {
		t.Errorf("NullableID(nil) = %v, want nil", got)
	}
	id := "abc"
	if got := NullableID(&id); got != "abc" {
		t.Errorf("NullableID(&%q) = %v, want %q", id, got, id)
	}
}

type fakeResult struct {
	rows int64
	err  error
}

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error) { return f.rows, f.err }

func TestRequireRows(t *testing.T) {
	notFound := errors.New("not found")

	if err := RequireRows(fakeResult{rows: 1}, notFound); err != nil {
		t.Errorf("RequireRows(1 row) = %v, want nil", err)
	}
	if err := RequireRows(fakeResult{rows: 0}, notFound); !errors.Is(err, notFound) {
		t.Errorf("RequireRows(0 rows) = %v, want the notFound error", err)
	}

	boom := errors.New("driver failure")
	if err := RequireRows(fakeResult{err: boom}, notFound); !errors.Is(err, boom) {
		t.Errorf("RequireRows(driver error) = %v, want it propagated", err)
	}
}

func TestParseTimestamp(t *testing.T) {
	// The format SQLite's strftime('%Y-%m-%dT%H:%M:%fZ') actually produces.
	got, err := ParseTimestamp("2026-08-12T09:15:04.123Z")
	if err != nil {
		t.Fatalf("ParseTimestamp() error = %v", err)
	}
	want := time.Date(2026, 8, 12, 9, 15, 4, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseTimestamp() = %v, want %v", got, want)
	}

	if _, err := ParseTimestamp("2026-08-12T09:15:04Z"); err != nil {
		t.Errorf("ParseTimestamp() without a fraction error = %v", err)
	}
	if _, err := ParseTimestamp("not a time"); err == nil {
		t.Error("ParseTimestamp(\"not a time\") should fail")
	}
}
