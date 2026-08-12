package statsapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/id"
	"pixabros/internal/stats"
)

func setup(t *testing.T) (*Handlers, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return NewHandlers(stats.NewRepo(conn)), conn
}

func TestGet_ReturnsCountsAsJSON(t *testing.T) {
	handlers, conn := setup(t)

	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, is_published, is_browser_playable, is_for_sale)
		 VALUES (?, 'alpha', 'Alpha', 1, 1, 0), (?, 'beta', 'Beta', 0, 0, 1);`,
		id.New(), id.New(),
	); err != nil {
		t.Fatalf("seed games: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO contact_submissions (id, subject, message, is_read)
		 VALUES (?, 'Hi', 'A message.', 0);`, id.New(),
	); err != nil {
		t.Fatalf("seed submission: %v", err)
	}

	rec := httptest.NewRecorder()
	handlers.Get(rec, httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got stats.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}

	if got.Games.Total != 2 || got.Games.Published != 1 || got.Games.Playable != 1 || got.Games.ForSale != 1 {
		t.Errorf("Games = %+v, want {Total:2 Published:1 Playable:1 ForSale:1}", got.Games)
	}
	if got.Contact.Unread != 1 {
		t.Errorf("Contact.Unread = %d, want 1", got.Contact.Unread)
	}
}

// The dashboard renders on a fresh install too, so an empty database is a
// normal response and not an error.
func TestGet_EmptyDatabaseReturnsZeroes(t *testing.T) {
	handlers, _ := setup(t)

	rec := httptest.NewRecorder()
	handlers.Get(rec, httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got stats.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != (stats.Stats{}) {
		t.Errorf("stats = %+v, want every count zero", got)
	}
}

// The JSON keys are the frontend's contract; renaming a field silently would
// leave the dashboard showing zeroes rather than failing.
func TestGet_ResponseShapeIsStable(t *testing.T) {
	handlers, _ := setup(t)

	rec := httptest.NewRecorder()
	handlers.Get(rec, httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil))

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	for _, key := range []string{"games", "devlog", "awards", "members", "contact", "media"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("response is missing the %q key", key)
		}
	}

	var games map[string]int
	if err := json.Unmarshal(raw["games"], &games); err != nil {
		t.Fatalf("decode games: %v", err)
	}
	for _, key := range []string{"total", "published", "playable", "for_sale"} {
		if _, ok := games[key]; !ok {
			t.Errorf("games is missing the %q key", key)
		}
	}
}
