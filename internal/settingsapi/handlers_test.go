package settingsapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/settings"
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
	return NewHandlers(settings.NewRepo(conn), conn), conn
}

func get(t *testing.T, h *Handlers, group string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings/"+group, nil)
	req.SetPathValue("group", group)
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	return rec
}

func put(t *testing.T, h *Handlers, group, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodPut, "/api/admin/settings/"+group, bytes.NewReader([]byte(body)),
	)
	req.SetPathValue("group", group)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

// The UI builds its form from the server's registry rather than keeping a
// second copy of the key list that could drift.
func TestGet_ReturnsDefinitionsAlongsideValues(t *testing.T) {
	handlers, _ := setup(t)

	rec := get(t, handlers, "homepage")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got groupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Group != "homepage" || len(got.Definitions) == 0 {
		t.Fatalf("response = %+v, want the homepage definitions", got)
	}
	for _, definition := range got.Definitions {
		if _, ok := got.Values[definition.Key]; !ok {
			t.Errorf("values is missing the defined key %q", definition.Key)
		}
	}
}

func TestGet_UnknownGroupIsNotFound(t *testing.T) {
	handlers, _ := setup(t)

	if rec := get(t, handlers, "nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdate_StoresValuesAndQueuesRegen(t *testing.T) {
	handlers, conn := setup(t)

	rec := put(t, handlers, "site", `{"values":{"site_name":"PixaBros"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got groupResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Values["site_name"] != "PixaBros" {
		t.Errorf("site_name = %q, want %q", got.Values["site_name"], "PixaBros")
	}

	var count int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = 'site_settings';`).Scan(&count)
	if count != 1 {
		t.Errorf("regen jobs for site_settings = %d, want 1", count)
	}
}

// Each group invalidates only its own pages.
func TestUpdate_EachGroupQueuesItsOwnTag(t *testing.T) {
	handlers, conn := setup(t)

	put(t, handlers, "homepage", `{"values":{"hero_slogan":"Play"}}`)

	var homepageJobs, siteJobs int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = 'homepage';`).Scan(&homepageJobs)
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = 'site_settings';`).Scan(&siteJobs)
	if homepageJobs != 1 || siteJobs != 0 {
		t.Errorf("homepage jobs = %d, site jobs = %d, want 1 and 0", homepageJobs, siteJobs)
	}
}

// A mistyped key is the whole reason the registry exists: it must fail loudly
// rather than be stored where nothing will ever read it.
func TestUpdate_RejectsAnUnknownKey(t *testing.T) {
	handlers, _ := setup(t)

	rec := put(t, handlers, "site", `{"values":{"site_nmae":"typo"}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "unknown_key" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "unknown_key")
	}
}

// A key belonging to the other group is just as wrong as an invented one.
func TestUpdate_RejectsAKeyFromTheOtherGroup(t *testing.T) {
	handlers, _ := setup(t)

	if rec := put(t, handlers, "site", `{"values":{"hero_slogan":"Play"}}`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdate_ValidatesURIValues(t *testing.T) {
	handlers, _ := setup(t)

	for _, value := range []string{"example.com", "/relative", "not a url"} {
		body := `{"values":{"hero_cta_link":"` + value + `"}}`
		if rec := put(t, handlers, "homepage", body); rec.Code != http.StatusBadRequest {
			t.Errorf("hero_cta_link %q: status = %d, want %d", value, rec.Code, http.StatusBadRequest)
		}
	}

	if rec := put(t, handlers, "homepage", `{"values":{"hero_cta_link":"https://example.com/play"}}`); rec.Code != http.StatusOK {
		t.Errorf("a full URL was rejected: %s", rec.Body.String())
	}
}

// Blank means "not set", which every setting is allowed to be.
func TestUpdate_AllowsBlankValues(t *testing.T) {
	handlers, _ := setup(t)

	if rec := put(t, handlers, "homepage", `{"values":{"hero_cta_link":"","hero_logo":""}}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUpdate_ValidatesMediaValues(t *testing.T) {
	handlers, conn := setup(t)

	// Not id-shaped at all.
	if rec := put(t, handlers, "site", `{"values":{"org_logo":"nope"}}`); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed media id: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Well-formed but pointing at nothing. There is no foreign key on these
	// tables, so without this check the setting would silently render nothing.
	rec := put(t, handlers, "site", `{"values":{"org_logo":"aaaaaaaaaaaaaaaaaaaaaaaa"}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown media id: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "unknown_reference" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "unknown_reference")
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('0123456789abcdef01234567', 'l.webp', 512, 512);`,
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	if rec := put(t, handlers, "site", `{"values":{"org_logo":"0123456789abcdef01234567"}}`); rec.Code != http.StatusOK {
		t.Errorf("a real media id was rejected: %s", rec.Body.String())
	}
}

func TestUpdate_RequiresTheValuesField(t *testing.T) {
	handlers, _ := setup(t)

	if rec := put(t, handlers, "site", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdate_UnknownGroupIsNotFound(t *testing.T) {
	handlers, _ := setup(t)

	if rec := put(t, handlers, "nope", `{"values":{}}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
