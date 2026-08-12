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

// Migrate disables foreign keys while it runs (see the comment on Migrate).
// If it ever failed to turn them back on, cascades and constraint checks
// would be silently dead for the rest of the process's life.
func TestMigrate_LeavesForeignKeysEnforced(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO game_screenshots (id, game_id, media_id, display_order) VALUES ('a', 'no-such-game', 'no-such-media', 0);`,
	); err == nil {
		t.Error("inserting a screenshot for a non-existent game succeeded; foreign keys are not being enforced")
	}
}

func TestMigrate_ContentTablesUseTextIDs(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// admins and sessions deliberately keep integer ids: they never appear in
	// a URL, so there is nothing to enumerate.
	textIDTables := []string{
		"media", "games", "game_screenshots", "members",
		"devlog_posts", "awards", "contact_submissions",
	}
	for _, table := range textIDTables {
		var idType string
		err := conn.QueryRow(
			`SELECT type FROM pragma_table_info(?) WHERE name = 'id';`, table,
		).Scan(&idType)
		if err != nil {
			t.Errorf("%s: could not read id column type: %v", table, err)
			continue
		}
		if idType != "TEXT" {
			t.Errorf("%s.id type = %q, want TEXT", table, idType)
		}
	}

	for _, table := range []string{"admins", "sessions"} {
		var idType string
		if err := conn.QueryRow(
			`SELECT type FROM pragma_table_info(?) WHERE name = 'id';`, table,
		).Scan(&idType); err != nil {
			t.Errorf("%s: could not read id column type: %v", table, err)
			continue
		}
		if idType != "INTEGER" {
			t.Errorf("%s.id type = %q, want INTEGER (internal ids stay numeric)", table, idType)
		}
	}
}

// Deleting a game must still take its screenshots with it after the re-key.
func TestMigrate_ScreenshotCascadeSurvivesRekey(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO media (id, path, width, height) VALUES ('m1', 'p.webp', 1, 1);`); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO games (id, slug, title) VALUES ('g1', 'g', 'G');`); err != nil {
		t.Fatalf("seed game: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO game_screenshots (id, game_id, media_id, display_order) VALUES ('s1', 'g1', 'm1', 0);`); err != nil {
		t.Fatalf("seed screenshot: %v", err)
	}

	if _, err := conn.Exec(`DELETE FROM games WHERE id = 'g1';`); err != nil {
		t.Fatalf("delete game: %v", err)
	}

	var remaining int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM game_screenshots;`).Scan(&remaining); err != nil {
		t.Fatalf("count screenshots: %v", err)
	}
	if remaining != 0 {
		t.Errorf("screenshots remaining after deleting their game = %d, want 0", remaining)
	}
}
