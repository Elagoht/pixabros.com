package site

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"pixabros/internal/db"
	"pixabros/internal/id"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

// fixedYear keeps rendered output stable so a golden file does not start
// failing when the calendar rolls over.
var fixedNow = func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) }

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

// newTestSite wires a Site against a temporary database and asset directory.
func newTestSite(t *testing.T, conn *sql.DB) *Site {
	t.Helper()
	bundle, err := Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	s, err := New(conn, bundle)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	s.renderer.now = fixedNow
	return s
}

func newTestStore(t *testing.T, conn *sql.DB) *render.Store {
	t.Helper()
	return render.NewStore(conn, storage.NewLocalDisk(t.TempDir(), "/rendered"))
}

func seedSiteSettings(t *testing.T, conn *sql.DB, values map[string]string) {
	t.Helper()
	for key, value := range values {
		if _, err := conn.Exec(
			`INSERT INTO site_settings (key, value, value_type) VALUES (?, ?, 'text')
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value;`,
			key, value,
		); err != nil {
			t.Fatalf("seed setting %s: %v", key, err)
		}
	}
}

func seedAward(t *testing.T, conn *sql.DB, title, issuer, date, link string, pictureID *string) string {
	t.Helper()
	awardID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO awards (id, title, issuer, date, link, picture_id)
		 VALUES (?, ?, ?, ?, ?, ?);`,
		awardID, title, issuer, date, link, pictureID,
	); err != nil {
		t.Fatalf("seed award %q: %v", title, err)
	}
	return awardID
}

func seedMedia(t *testing.T, conn *sql.DB, path, altText string) string {
	t.Helper()
	mediaID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height, alt_text) VALUES (?, ?, 320, 320, ?);`,
		mediaID, path, altText,
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	return mediaID
}

// The header and footer read from site_settings, so every page depends on it.
func TestChrome_FallsBackWhenSiteNameIsBlank(t *testing.T) {
	conn := setupTestDB(t)
	s := newTestSite(t, conn)

	chrome, err := s.chrome()
	if err != nil {
		t.Fatalf("chrome() error = %v", err)
	}
	if chrome.Name == "" {
		t.Error("site name is empty, which would render a header link with nothing to click")
	}
}

func TestChrome_ReadsSettings(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_name":       "Pixabros",
		"twitter_handle":  "PixaBrosStudio",
		"org_sameas_json": `["https://itch.io/pixabros","https://x.com/PixaBrosStudio"]`,
	})
	s := newTestSite(t, conn)

	chrome, err := s.chrome()
	if err != nil {
		t.Fatalf("chrome() error = %v", err)
	}
	if chrome.Name != "Pixabros" || chrome.Twitter != "PixaBrosStudio" {
		t.Errorf("chrome = %+v, want name Pixabros and handle PixaBrosStudio", chrome)
	}
	if len(chrome.Links) != 2 {
		t.Errorf("links = %v, want 2 entries", chrome.Links)
	}
}

// A malformed uri_list must not stop every page on the site from rendering.
func TestParseLinks_ToleratesRubbish(t *testing.T) {
	for _, raw := range []string{"", "not json", "{}", `["ok"]`} {
		if got := parseLinks(raw); raw == `["ok"]` && len(got) != 1 {
			t.Errorf("parseLinks(%q) = %v, want one entry", raw, got)
		}
	}
}

func TestFormatDate(t *testing.T) {
	cases := map[string]string{
		"2026-06-21": "21 June 2026",
		"2026-01-01": "1 January 2026",
		// Anything that is not a stored date comes back untouched rather than
		// silently becoming year one.
		"":         "",
		"sometime": "sometime",
	}
	for input, want := range cases {
		if got := formatDate(input); got != want {
			t.Errorf("formatDate(%q) = %q, want %q", input, got, want)
		}
	}
}
