package db

import (
	"path/filepath"
	"testing"
)

func TestMigrate_CreatesAllTablesAndIsIdempotent(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := Migrate(conn); err != nil {
		t.Fatalf("second Migrate() call must be a no-op, got error = %v", err)
	}

	wantTables := []string{
		"admins", "sessions", "media", "site_settings", "homepage_settings",
		"members", "games", "game_screenshots", "devlog_posts", "awards",
		"contact_submissions", "rendered_pages", "page_tags", "regen_jobs",
	}
	for _, table := range wantTables {
		var name string
		err := conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?;`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
