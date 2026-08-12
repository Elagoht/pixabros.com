package mediarefs

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"pixabros/internal/db"
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

func seedMedia(t *testing.T, conn *sql.DB, ids ...string) {
	t.Helper()
	for _, mediaID := range ids {
		if _, err := conn.Exec(
			`INSERT INTO media (id, path, width, height) VALUES (?, ?, 100, 100);`,
			mediaID, mediaID+".webp",
		); err != nil {
			t.Fatalf("seed media %q: %v", mediaID, err)
		}
	}
}

// This is the test that matters most in this package. If someone adds a column
// referencing media and forgets to list it in `sources`, the orphan sweep will
// delete images the site is still using. Rather than trusting a comment, the
// schema is read back and every foreign key into media is checked against the
// queries above.
func TestSourcesCoverEveryMediaForeignKey(t *testing.T) {
	conn := setupTestDB(t)

	tables, err := conn.Query(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%';`,
	)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer tables.Close()

	var tableNames []string
	for tables.Next() {
		var name string
		if err := tables.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		tableNames = append(tableNames, name)
	}

	// Every query in `sources` mentions its table and column, so a reference is
	// "covered" when some query names both.
	covered := func(table, column string) bool {
		return slices.ContainsFunc(sources, func(src source) bool {
			return strings.Contains(src.query, table) && strings.Contains(src.query, column)
		})
	}

	found := 0
	for _, table := range tableNames {
		keys, err := conn.Query(fmt.Sprintf(`PRAGMA foreign_key_list(%q);`, table))
		if err != nil {
			t.Fatalf("foreign keys of %s: %v", table, err)
		}
		for keys.Next() {
			// PRAGMA foreign_key_list: id, seq, table, from, to, on_update,
			// on_delete, match
			var fkID, seq int
			var parent, from, to, onUpdate, onDelete, match sql.NullString
			if err := keys.Scan(&fkID, &seq, &parent, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				keys.Close()
				t.Fatalf("scan foreign key of %s: %v", table, err)
			}
			if parent.String != "media" {
				continue
			}
			found++
			if !covered(table, from.String) {
				t.Errorf("%s.%s references media but no query in sources reads it; "+
					"the orphan sweep would delete images this column still points at",
					table, from.String)
			}
		}
		keys.Close()
	}

	// If this drops to zero the test has stopped checking anything, which
	// would be worse than a failure.
	if found == 0 {
		t.Fatal("found no foreign keys into media; the schema check is not working")
	}
}

func TestLookup_ReportsWhereAnImageIsUsed(t *testing.T) {
	conn := setupTestDB(t)
	seedMedia(t, conn, "aaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbb")

	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id) VALUES ('g1', 'pixel', 'Pixel Quest', 'aaaaaaaaaaaaaaaaaaaaaaaa');`,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO members (id, name, avatar_id) VALUES ('m1', 'Furkan', 'bbbbbbbbbbbbbbbbbbbbbbbb');`,
	); err != nil {
		t.Fatalf("seed member: %v", err)
	}

	usages, err := Lookup(conn)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}

	game := usages["aaaaaaaaaaaaaaaaaaaaaaaa"]
	if len(game) != 1 || game[0].Module != "games" || game[0].Label != "Pixel Quest" {
		t.Errorf("cartridge art usage = %+v, want one games usage labelled with the title", game)
	}
	member := usages["bbbbbbbbbbbbbbbbbbbbbbbb"]
	if len(member) != 1 || member[0].Module != "members" || member[0].Label != "Furkan" {
		t.Errorf("avatar usage = %+v, want one members usage labelled with the name", member)
	}
}

// One image used in several places has to report all of them, or the library
// would let you delete something that is still on a page.
func TestLookup_ReportsEveryUsageOfTheSameImage(t *testing.T) {
	conn := setupTestDB(t)
	seedMedia(t, conn, "aaaaaaaaaaaaaaaaaaaaaaaa")

	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id, og_image_id)
		 VALUES ('g1', 'pixel', 'Pixel Quest', 'aaaaaaaaaaaaaaaaaaaaaaaa', 'aaaaaaaaaaaaaaaaaaaaaaaa');`,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO site_settings (key, value, value_type) VALUES ('org_logo', 'aaaaaaaaaaaaaaaaaaaaaaaa', 'media');`,
	); err != nil {
		t.Fatalf("seed setting: %v", err)
	}

	usages, err := Lookup(conn)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(usages["aaaaaaaaaaaaaaaaaaaaaaaa"]) != 3 {
		t.Errorf("usages = %+v, want three", usages["aaaaaaaaaaaaaaaaaaaaaaaa"])
	}
}

// A settings row holding a media id has no foreign key, so it can only be
// found by value_type. Missing it would make the sweep delete the site logo.
func TestLookup_FindsSettingsReferences(t *testing.T) {
	conn := setupTestDB(t)
	seedMedia(t, conn, "aaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbb")

	if _, err := conn.Exec(
		`INSERT INTO site_settings (key, value, value_type) VALUES ('org_logo', 'aaaaaaaaaaaaaaaaaaaaaaaa', 'media');`,
	); err != nil {
		t.Fatalf("seed site setting: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type) VALUES ('hero_logo', 'bbbbbbbbbbbbbbbbbbbbbbbb', 'media');`,
	); err != nil {
		t.Fatalf("seed homepage setting: %v", err)
	}

	referenced, err := ReferencedIDs(conn)
	if err != nil {
		t.Fatalf("ReferencedIDs() error = %v", err)
	}
	if !referenced["aaaaaaaaaaaaaaaaaaaaaaaa"] || !referenced["bbbbbbbbbbbbbbbbbbbbbbbb"] {
		t.Errorf("ReferencedIDs() = %v, want both settings images", referenced)
	}
}

// A text setting that happens to look like an id is not a media reference.
func TestLookup_IgnoresNonMediaSettings(t *testing.T) {
	conn := setupTestDB(t)
	seedMedia(t, conn, "aaaaaaaaaaaaaaaaaaaaaaaa")

	if _, err := conn.Exec(
		`INSERT INTO site_settings (key, value, value_type) VALUES ('site_name', 'aaaaaaaaaaaaaaaaaaaaaaaa', 'text');`,
	); err != nil {
		t.Fatalf("seed setting: %v", err)
	}

	referenced, err := ReferencedIDs(conn)
	if err != nil {
		t.Fatalf("ReferencedIDs() error = %v", err)
	}
	if referenced["aaaaaaaaaaaaaaaaaaaaaaaa"] {
		t.Error("a text setting was treated as a media reference")
	}
}

func TestReferencedIDs_EmptyDatabase(t *testing.T) {
	referenced, err := ReferencedIDs(setupTestDB(t))
	if err != nil {
		t.Fatalf("ReferencedIDs() error = %v", err)
	}
	if len(referenced) != 0 {
		t.Errorf("ReferencedIDs() = %v, want none", referenced)
	}
}
