package stats

import (
	"database/sql"
	"path/filepath"
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

func seedGame(t *testing.T, conn *sql.DB, title string, published, playable, forSale int) {
	t.Helper()
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, is_published, is_browser_playable, is_for_sale)
		 VALUES (?, ?, ?, ?, ?, ?);`,
		id.New(), title, title, published, playable, forSale,
	); err != nil {
		t.Fatalf("seed game %q: %v", title, err)
	}
}

// An empty database must report zeroes, not fail. COALESCE around the SUMs is
// what makes that work: SUM over no rows is NULL, which will not scan into an
// int.
func TestRepo_GetOnAnEmptyDatabase(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	got, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != (Stats{}) {
		t.Errorf("Get() on an empty database = %+v, want every count zero", got)
	}
}

func TestRepo_GetCountsEachFlagSeparately(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	// Three games: one published-and-playable, one published-and-for-sale, one
	// draft. Every flag lands on a different combination so a query that mixed
	// two of them up could not still produce these numbers.
	seedGame(t, conn, "alpha", 1, 1, 0)
	seedGame(t, conn, "beta", 1, 0, 1)
	seedGame(t, conn, "gamma", 0, 0, 0)

	got, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	want := GameStats{Total: 3, Published: 2, Playable: 1, ForSale: 1}
	if got.Games != want {
		t.Errorf("Games = %+v, want %+v", got.Games, want)
	}
}

func TestRepo_GetCountsUnreadContactSubmissions(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	for _, read := range []int{0, 0, 1} {
		if _, err := conn.Exec(
			`INSERT INTO contact_submissions (id, subject, message, is_read)
			 VALUES (?, 'Hello', 'A message long enough to be real.', ?);`,
			id.New(), read,
		); err != nil {
			t.Fatalf("seed submission: %v", err)
		}
	}

	got, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	want := ContactStats{Total: 3, Unread: 2}
	if got.Contact != want {
		t.Errorf("Contact = %+v, want %+v", got.Contact, want)
	}
}

func TestRepo_GetCountsEveryModule(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	seedGame(t, conn, "solo", 1, 1, 0)
	if _, err := conn.Exec(
		`INSERT INTO devlog_posts (id, slug, title, content_markdown, is_published)
		 VALUES (?, 'first', 'First', 'Body', 1), (?, 'second', 'Second', 'Body', 0);`,
		id.New(), id.New(),
	); err != nil {
		t.Fatalf("seed devlog: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO awards (id, title, issuer, date)
		 VALUES (?, 'Best Game', 'Some Jury', '2026-01-01');`, id.New(),
	); err != nil {
		t.Fatalf("seed award: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO members (id, name, tags, description, links_json, display_order, is_published)
		 VALUES (?, 'Furkan', '', '', '[]', 0, 1), (?, 'Someone', '', '', '[]', 1, 1);`,
		id.New(), id.New(),
	); err != nil {
		t.Fatalf("seed members: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (?, 'media/a.webp', 10, 10);`,
		id.New(),
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	got, err := repo.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Devlog != (DevlogStats{Total: 2, Published: 1}) {
		t.Errorf("Devlog = %+v, want {Total:2 Published:1}", got.Devlog)
	}
	if got.Awards != 1 {
		t.Errorf("Awards = %d, want 1", got.Awards)
	}
	if got.Members != 2 {
		t.Errorf("Members = %d, want 2", got.Members)
	}
	if got.Media != 1 {
		t.Errorf("Media = %d, want 1", got.Media)
	}
}
