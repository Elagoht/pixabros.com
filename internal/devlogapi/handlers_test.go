package devlogapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/devlog"
)

func setup(t *testing.T) (*Handlers, *devlog.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	repo := devlog.NewRepo(conn)
	return NewHandlers(repo, conn), repo, conn
}

func post(t *testing.T, h *Handlers, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/devlog", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	return rec
}

func put(t *testing.T, h *Handlers, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/devlog/"+key, bytes.NewReader(raw))
	req.SetPathValue("id", key)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

func TestCreate_RequiresATitle(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, body := range []map[string]any{{}, {"title": "   "}} {
		if rec := post(t, handlers, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: status = %d, want %d", body, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestCreate_RejectsAMalformedDate(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, date := range []string{"12/08/2026", "2026-8-1", "yesterday"} {
		body := map[string]any{"title": "Post", "published_at": date}
		if rec := post(t, handlers, body); rec.Code != http.StatusBadRequest {
			t.Errorf("date %q: status = %d, want %d", date, rec.Code, http.StatusBadRequest)
		}
	}
}

// An empty date is not an error: the repo stamps one on first publish.
func TestCreate_AllowsAnEmptyDate(t *testing.T) {
	handlers, _, _ := setup(t)

	rec := post(t, handlers, map[string]any{"title": "Post", "published_at": ""})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestCreate_QueuesBothThePostAndTheIndex(t *testing.T) {
	handlers, _, conn := setup(t)

	rec := post(t, handlers, map[string]any{"title": "Kartuş Sistemi"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got postResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Slug != "kartus-sistemi" {
		t.Errorf("Slug = %q, want %q", got.Slug, "kartus-sistemi")
	}

	// A post has a page of its own as well as appearing on the index, so both
	// tags have to be invalidated.
	for _, tag := range []string{regenTagFor(got.ID), listRegenTag} {
		var count int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, tag).Scan(&count); err != nil {
			t.Fatalf("count regen jobs: %v", err)
		}
		if count != 1 {
			t.Errorf("regen jobs for %q = %d, want 1", tag, count)
		}
	}
}

// The admin addresses posts by id, but a link built from a public URL uses the
// slug, so both have to resolve.
func TestGet_ResolvesByIDAndBySlug(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Hello World"})

	for _, key := range []string{created.ID, created.Slug} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog/"+key, nil)
		req.SetPathValue("id", key)
		rec := httptest.NewRecorder()
		handlers.Get(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("key %q: status = %d, want %d", key, rec.Code, http.StatusOK)
		}
		var got postResponse
		json.Unmarshal(rec.Body.Bytes(), &got)
		if got.ID != created.ID {
			t.Errorf("key %q resolved to %q, want %q", key, got.ID, created.ID)
		}
	}
}

func TestGet_UnknownPostIsNotFound(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog/nope", nil)
	req.SetPathValue("id", "nope")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdate_RejectsMalformedReferences(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	for _, field := range []string{"game_id", "og_image_id"} {
		body := map[string]any{"title": "Post", field: "nope"}
		if rec := put(t, handlers, created.ID, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", field, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestUpdate_UnknownReferenceIsABadRequest(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	body := map[string]any{"title": "Post", "game_id": "aaaaaaaaaaaaaaaaaaaaaaaa"}
	rec := put(t, handlers, created.ID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "unknown_reference" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "unknown_reference")
	}
}

func TestDelete_RemovesPost(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/devlog/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if list, _ := repo.List("", false); len(list) != 0 {
		t.Errorf("posts remaining = %d, want 0", len(list))
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, query := range []string{"?sort=password_hash", "?sort=title&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestList_ReturnsAnEmptyArrayNotNull(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("empty list body = %q, want %q", body, "[]\n")
	}
}
