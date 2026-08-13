package contact

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

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

// The public form does not collect a name, so the column is nullable and most
// rows have none. Imported submissions do, and it has to survive the round
// trip -- including the NULL case, which would otherwise fail to scan.
func TestRepo_NameRoundTripsAndToleratesNull(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	withName := id.New()
	if _, err := conn.Exec(
		`INSERT INTO contact_submissions (id, name, subject, email, message, created_at)
		 VALUES (?, 'Deneme''nin Büyücüsü', 'Bir soru', 'oyuncu@example.com', 'Örnek bir mesaj.', '2026-06-21T04:03:55.000Z');`,
		withName,
	); err != nil {
		t.Fatalf("seed named submission: %v", err)
	}

	withoutName := id.New()
	if _, err := conn.Exec(
		`INSERT INTO contact_submissions (id, subject, message, created_at)
		 VALUES (?, 'From the form', 'No name here.', '2026-06-22T04:03:55.000Z');`,
		withoutName,
	); err != nil {
		t.Fatalf("seed anonymous submission: %v", err)
	}

	named, err := repo.FindByID(withName)
	if err != nil {
		t.Fatalf("FindByID(named) error = %v", err)
	}
	// The apostrophe is the point: it is both a Turkish name's own punctuation
	// and the character that would break the insert if it were not escaped.
	if named.Name != "Deneme'nin Büyücüsü" {
		t.Errorf("Name = %q, want %q", named.Name, "Deneme'nin Büyücüsü")
	}

	anonymous, err := repo.FindByID(withoutName)
	if err != nil {
		t.Fatalf("FindByID(anonymous) error = %v", err)
	}
	if anonymous.Name != "" {
		t.Errorf("Name = %q, want empty for a submission with no name", anonymous.Name)
	}

	// A timestamp without fractional seconds is what the import writes; it has
	// to parse the same as the column default's millisecond form.
	if named.CreatedAt.Format(time.RFC3339) != "2026-06-21T04:03:55Z" {
		t.Errorf("CreatedAt = %v, want 2026-06-21T04:03:55Z", named.CreatedAt)
	}

	// Sorting by the new column must be accepted, not rejected as unknown.
	if _, err := repo.List("name", false); err != nil {
		t.Errorf("List(sort=name) error = %v", err)
	}
}

func TestRepo_CreateStoresASubmission(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	created, err := repo.Create(CreateInput{
		Name:          "Someone",
		Subject:       "Hello",
		Email:         "someone@example.com",
		Message:       "A message long enough to be a real one, with more than a hundred characters in it so it passes validation.",
		WantsCallback: true,
		IPAddress:     "203.0.113.7",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != "Someone" || found.Subject != "Hello" || !found.WantsCallback {
		t.Errorf("stored submission = %+v, want the input back", found)
	}
	// A new submission is unread by definition -- that is the whole point of
	// the inbox.
	if found.IsRead {
		t.Error("a new submission arrived already marked read")
	}
}

// Phone and email are both optional, and an empty one must be stored as NULL
// rather than an empty string so the admin list can tell them apart.
func TestRepo_CreateLeavesMissingContactDetailsNull(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	created, err := repo.Create(CreateInput{
		Subject: "No way to reach me",
		Message: "A message long enough to be a real one, with more than a hundred characters in it so it passes validation.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var phone, email sql.NullString
	if err := conn.QueryRow(
		`SELECT phone, email FROM contact_submissions WHERE id = ?;`, created.ID,
	).Scan(&phone, &email); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if phone.Valid || email.Valid {
		t.Errorf("phone=%v email=%v, want both NULL", phone, email)
	}
}
