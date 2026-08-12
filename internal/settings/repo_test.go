package settings

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
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

func siteGroup(t *testing.T) Group {
	t.Helper()
	group, err := LookupGroup("site")
	if err != nil {
		t.Fatalf("LookupGroup(\"site\") error = %v", err)
	}
	return group
}

func TestLookupGroup(t *testing.T) {
	for _, name := range GroupNames() {
		if _, err := LookupGroup(name); err != nil {
			t.Errorf("LookupGroup(%q) error = %v", name, err)
		}
	}
	if _, err := LookupGroup("nope"); !errors.Is(err, ErrUnknownGroup) {
		t.Errorf("LookupGroup(\"nope\") error = %v, want ErrUnknownGroup", err)
	}
}

// Both groups write to their own table, and the tags they invalidate are the
// ones the data-model spec names.
func TestGroupsAreDistinct(t *testing.T) {
	site, _ := LookupGroup("site")
	homepage, _ := LookupGroup("homepage")

	if site.Table == homepage.Table {
		t.Error("both groups write to the same table")
	}
	if site.RegenTag == homepage.RegenTag {
		t.Error("both groups invalidate the same tag")
	}
	if site.RegenTag != "site_settings" || homepage.RegenTag != "homepage" {
		t.Errorf("regen tags = %q, %q, want %q, %q",
			site.RegenTag, homepage.RegenTag, "site_settings", "homepage")
	}
}

func TestGroup_Define(t *testing.T) {
	group := siteGroup(t)

	definition, err := group.Define("org_logo")
	if err != nil {
		t.Fatalf("Define() error = %v", err)
	}
	if definition.Kind != KindMedia {
		t.Errorf("org_logo kind = %q, want %q", definition.Kind, KindMedia)
	}

	if _, err := group.Define("hero_slogan"); !errors.Is(err, ErrUnknownKey) {
		t.Error("a homepage key resolved inside the site group")
	}
	if _, err := group.Define("made_up"); !errors.Is(err, ErrUnknownKey) {
		t.Error("an undefined key resolved")
	}
}

// A key the registry defines but nobody has filled in reads as blank, so a
// caller never has to tell "unset" apart from "empty".
func TestRepo_ValuesReturnsEveryDefinedKey(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	group := siteGroup(t)

	values, err := repo.Values(group)
	if err != nil {
		t.Fatalf("Values() error = %v", err)
	}
	if len(values) != len(group.Definitions) {
		t.Errorf("Values() returned %d keys, want %d", len(values), len(group.Definitions))
	}
	for _, definition := range group.Definitions {
		if got, ok := values[definition.Key]; !ok || got != "" {
			t.Errorf("Values()[%q] = %q, present=%v, want an empty string", definition.Key, got, ok)
		}
	}
}

func TestRepo_ReplaceThenRead(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	group := siteGroup(t)

	if err := repo.Replace(group, map[string]string{
		"site_name":      "PixaBros",
		"twitter_handle": "@pixabros",
	}); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	values, _ := repo.Values(group)
	if values["site_name"] != "PixaBros" {
		t.Errorf("site_name = %q, want %q", values["site_name"], "PixaBros")
	}
	// Keys that were not part of the save keep their previous (blank) value.
	if values["org_logo"] != "" {
		t.Errorf("org_logo = %q, want it untouched", values["org_logo"])
	}

	// Saving again overwrites rather than inserting a duplicate.
	if err := repo.Replace(group, map[string]string{"site_name": "PixaBros Studio"}); err != nil {
		t.Fatalf("second Replace() error = %v", err)
	}
	values, _ = repo.Values(group)
	if values["site_name"] != "PixaBros Studio" {
		t.Errorf("site_name = %q, want the updated value", values["site_name"])
	}
}

func TestRepo_ReplaceRejectsAnUnknownKey(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	group := siteGroup(t)

	err := repo.Replace(group, map[string]string{"site_name": "ok", "site_nmae": "typo"})
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Replace() with a typo'd key error = %v, want ErrUnknownKey", err)
	}

	// The whole save is refused, so the valid half is not stored either.
	values, _ := repo.Values(group)
	if values["site_name"] != "" {
		t.Errorf("site_name = %q, want the rejected save to have stored nothing", values["site_name"])
	}
}

// value_type is written from the registry, not from the caller, which is what
// lets the orphan-media sweep find image references generically.
func TestRepo_MediaValues(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	group := siteGroup(t)

	if err := repo.Replace(group, map[string]string{
		"site_name": "PixaBros",
		"org_logo":  "0123456789abcdef01234567",
	}); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	ids, err := repo.MediaValues(group)
	if err != nil {
		t.Fatalf("MediaValues() error = %v", err)
	}
	if !slices.Equal(ids, []string{"0123456789abcdef01234567"}) {
		t.Errorf("MediaValues() = %v, want just the logo id", ids)
	}

	var storedType string
	conn.QueryRow(`SELECT value_type FROM site_settings WHERE key = 'org_logo';`).Scan(&storedType)
	if storedType != "media" {
		t.Errorf("org_logo value_type = %q, want %q", storedType, "media")
	}
}

// A blank media setting is not a reference, so it must not be reported as one.
func TestRepo_MediaValuesSkipsBlanks(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	group := siteGroup(t)

	repo.Replace(group, map[string]string{"org_logo": ""})

	ids, err := repo.MediaValues(group)
	if err != nil {
		t.Fatalf("MediaValues() error = %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("MediaValues() = %v, want none", ids)
	}
}

func TestRepo_MediaExists(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('0123456789abcdef01234567', 'l.webp', 512, 512);`,
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	if exists, _ := repo.MediaExists("0123456789abcdef01234567"); !exists {
		t.Error("MediaExists() = false for a row that is there")
	}
	if exists, _ := repo.MediaExists("aaaaaaaaaaaaaaaaaaaaaaaa"); exists {
		t.Error("MediaExists() = true for a row that is not there")
	}
}

// A key dropped from the registry may still be sitting in the table; it must
// not resurface in the editor, where nothing would read it.
func TestRepo_ValuesIgnoresKeysTheRegistryDropped(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	group := siteGroup(t)

	if _, err := conn.Exec(
		`INSERT INTO site_settings (key, value, value_type) VALUES ('legacy_key', 'old', 'text');`,
	); err != nil {
		t.Fatalf("seed legacy key: %v", err)
	}

	values, err := repo.Values(group)
	if err != nil {
		t.Fatalf("Values() error = %v", err)
	}
	if _, ok := values["legacy_key"]; ok {
		t.Error("Values() surfaced a key the registry no longer defines")
	}
}

// Every media setting must name an imaging target, or the admin UI would have
// nothing to upload through.
func TestMediaDefinitionsDeclareATarget(t *testing.T) {
	for _, name := range GroupNames() {
		group, err := LookupGroup(name)
		if err != nil {
			t.Fatalf("LookupGroup(%q) error = %v", name, err)
		}
		for _, definition := range group.Definitions {
			if definition.Kind == KindMedia && definition.Target == "" {
				t.Errorf("%s.%s is a media setting with no imaging target", name, definition.Key)
			}
			if definition.Kind != KindMedia && definition.Target != "" {
				t.Errorf("%s.%s declares a target but is not a media setting", name, definition.Key)
			}
		}
	}
}

// The database constrains value_type, so every kind the registry can produce
// has to be allowed by the CHECK -- otherwise saving that setting fails at
// write time rather than at review time.
func TestEveryRegistryKindIsStorable(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	seen := map[Kind]bool{}
	for _, name := range GroupNames() {
		group, err := LookupGroup(name)
		if err != nil {
			t.Fatalf("LookupGroup(%q) error = %v", name, err)
		}
		for _, definition := range group.Definitions {
			if seen[definition.Kind] {
				continue
			}
			seen[definition.Kind] = true
			if err := repo.Replace(group, map[string]string{definition.Key: ""}); err != nil {
				t.Errorf("storing a %q setting (%s.%s) failed: %v",
					definition.Kind, name, definition.Key, err)
			}
		}
	}

	// Guard against a kind being added to the registry but used nowhere, which
	// would leave it untested by the loop above.
	for _, kind := range []Kind{KindText, KindURI, KindURIList, KindMedia} {
		if !seen[kind] {
			t.Errorf("kind %q is defined but no setting uses it, so it is untested", kind)
		}
	}
}
