package contact

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/id"
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

// seed inserts a submission the way the public form will: nothing in this
// package creates them, so tests write the row directly.
func seed(t *testing.T, conn *sql.DB, subject, email, createdAt string) string {
	t.Helper()
	submissionID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO contact_submissions
			(id, subject, phone, email, message, wants_callback, ip_address, created_at)
		 VALUES (?, ?, '+900000000', ?, 'Hello there', 1, '203.0.113.7', ?);`,
		submissionID, subject, email, createdAt,
	); err != nil {
		t.Fatalf("seed submission: %v", err)
	}
	return submissionID
}

func TestRepo_FindByID(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	submissionID := seed(t, conn, "Collaboration", "a@example.com", "2026-08-01T10:00:00.000Z")

	got, err := repo.FindByID(submissionID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if got.Subject != "Collaboration" || got.Email != "a@example.com" {
		t.Errorf("FindByID() = %+v, want the seeded subject and email", got)
	}
	if got.Message != "Hello there" || !got.WantsCallback {
		t.Errorf("FindByID() = %+v, want the message and callback flag", got)
	}
	// The sender's address is kept for spam triage.
	if got.IPAddress != "203.0.113.7" {
		t.Errorf("IPAddress = %q, want it preserved", got.IPAddress)
	}
	// A submission arrives unread.
	if got.IsRead {
		t.Error("a new submission should be unread")
	}
}

func TestRepo_MissingSubmissionIsReported(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	missing := "aaaaaaaaaaaaaaaaaaaaaaaa"

	if _, err := repo.FindByID(missing); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("FindByID() error = %v, want ErrSubmissionNotFound", err)
	}
	if _, err := repo.SetRead(missing, true); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("SetRead() error = %v, want ErrSubmissionNotFound", err)
	}
	if err := repo.Delete(missing); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("Delete() error = %v, want ErrSubmissionNotFound", err)
	}
}

// Read state has to travel both ways: marking something unread again is how
// an inbox lets you come back to it.
func TestRepo_SetReadBothWays(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	submissionID := seed(t, conn, "Subject", "a@example.com", "2026-08-01T10:00:00.000Z")

	read, err := repo.SetRead(submissionID, true)
	if err != nil {
		t.Fatalf("SetRead(true) error = %v", err)
	}
	if !read.IsRead {
		t.Error("SetRead(true) did not mark the submission read")
	}

	unread, err := repo.SetRead(submissionID, false)
	if err != nil {
		t.Fatalf("SetRead(false) error = %v", err)
	}
	if unread.IsRead {
		t.Error("SetRead(false) did not mark the submission unread")
	}
}

// Marking a submission read must not disturb what the sender wrote.
func TestRepo_SetReadLeavesTheMessageAlone(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	submissionID := seed(t, conn, "Subject", "a@example.com", "2026-08-01T10:00:00.000Z")

	before, _ := repo.FindByID(submissionID)
	after, err := repo.SetRead(submissionID, true)
	if err != nil {
		t.Fatalf("SetRead() error = %v", err)
	}

	if after.Subject != before.Subject || after.Message != before.Message ||
		after.Email != before.Email || after.Phone != before.Phone ||
		after.IPAddress != before.IPAddress || !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("SetRead() changed the submission itself: %+v -> %+v", before, after)
	}
}

func TestRepo_UnreadCount(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	if count, _ := repo.UnreadCount(); count != 0 {
		t.Errorf("UnreadCount() on an empty inbox = %d, want 0", count)
	}

	first := seed(t, conn, "One", "a@example.com", "2026-08-01T10:00:00.000Z")
	seed(t, conn, "Two", "b@example.com", "2026-08-02T10:00:00.000Z")

	if count, _ := repo.UnreadCount(); count != 2 {
		t.Errorf("UnreadCount() = %d, want 2", count)
	}

	repo.SetRead(first, true)
	if count, _ := repo.UnreadCount(); count != 1 {
		t.Errorf("UnreadCount() after marking one read = %d, want 1", count)
	}
}

func TestRepo_Delete(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	submissionID := seed(t, conn, "Spam", "x@example.com", "2026-08-01T10:00:00.000Z")

	if err := repo.Delete(submissionID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(submissionID); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrSubmissionNotFound", err)
	}
}

func TestRepo_ListSorting(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	seed(t, conn, "middle", "b@example.com", "2026-08-02T10:00:00.000Z")
	seed(t, conn, "oldest", "c@example.com", "2026-08-01T10:00:00.000Z")
	seed(t, conn, "newest", "a@example.com", "2026-08-03T10:00:00.000Z")

	subjectsFor := func(t *testing.T, field string, descending bool) []string {
		t.Helper()
		list, err := repo.List(field, descending)
		if err != nil {
			t.Fatalf("List(%q, %v) error = %v", field, descending, err)
		}
		subjects := make([]string, 0, len(list))
		for _, s := range list {
			subjects = append(subjects, s.Subject)
		}
		return subjects
	}

	// An inbox is only useful newest-first.
	if got, want := subjectsFor(t, "", false), []string{"newest", "middle", "oldest"}; !slices.Equal(got, want) {
		t.Errorf("default order = %v, want %v", got, want)
	}
	if got, want := subjectsFor(t, "created_at", false), []string{"oldest", "middle", "newest"}; !slices.Equal(got, want) {
		t.Errorf("created_at asc = %v, want %v", got, want)
	}
	if got, want := subjectsFor(t, "email", false), []string{"newest", "middle", "oldest"}; !slices.Equal(got, want) {
		t.Errorf("email asc = %v, want %v", got, want)
	}

	if _, err := repo.List("subject; DROP TABLE contact_submissions", false); !errors.Is(err, ErrInvalidSort) {
		t.Errorf("List() with an injected field error = %v, want ErrInvalidSort", err)
	}
}
