package awardsapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/awards"
	"pixabros/internal/db"
)

func setup(t *testing.T) (*Handlers, *awards.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	repo := awards.NewRepo(conn)
	return NewHandlers(repo, conn), repo, conn
}

func post(t *testing.T, h *Handlers, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/awards", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	return rec
}

func TestCreate_ValidatesRequiredFields(t *testing.T) {
	handlers, _, _ := setup(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"no title", map[string]any{"issuer": "IGF", "date": "2026-01-01"}},
		{"blank title", map[string]any{"title": "  ", "issuer": "IGF", "date": "2026-01-01"}},
		{"no issuer", map[string]any{"title": "Best", "date": "2026-01-01"}},
		{"blank issuer", map[string]any{"title": "Best", "issuer": " ", "date": "2026-01-01"}},
		{"no date", map[string]any{"title": "Best", "issuer": "IGF"}},
		// The column is TEXT and is ordered as a string, so a differently
		// shaped date would sort into the wrong place rather than fail.
		{"wrong date shape", map[string]any{"title": "Best", "issuer": "IGF", "date": "18/03/2026"}},
		{"partial date", map[string]any{"title": "Best", "issuer": "IGF", "date": "2026-3-1"}},
	}
	for _, tc := range cases {
		if rec := post(t, handlers, tc.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d, body = %s", tc.name, rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	}
}

func TestCreate_StoresAwardAndQueuesRegen(t *testing.T) {
	handlers, _, conn := setup(t)

	rec := post(t, handlers, map[string]any{
		"title": "Best Game", "issuer": "IGF", "date": "2026-03-18",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got awardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Title != "Best Game" || got.Date != "2026-03-18" {
		t.Errorf("response = %+v, want the submitted title and date", got)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, regenTag).Scan(&count); err != nil {
		t.Fatalf("count regen jobs: %v", err)
	}
	if count != 1 {
		t.Errorf("regen jobs = %d, want 1", count)
	}
}

func put(t *testing.T, h *Handlers, awardID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/awards/"+awardID, bytes.NewReader(raw))
	req.SetPathValue("id", awardID)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

func TestUpdate_RejectsMalformedReferences(t *testing.T) {
	handlers, repo, _ := setup(t)
	award, _ := repo.Create(awards.CreateInput{Title: "t", Issuer: "i", Date: "2026-01-01"})

	for _, field := range []string{"game_id", "picture_id"} {
		body := map[string]any{"title": "t", "issuer": "i", "date": "2026-01-01", field: "nope"}
		if rec := put(t, handlers, award.ID, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s malformed: status = %d, want %d", field, rec.Code, http.StatusBadRequest)
		}
	}
}

// A well-formed id for a game that does not exist is a caller mistake, so it
// must read as 400 rather than surfacing as a server error.
func TestUpdate_UnknownReferenceIsABadRequest(t *testing.T) {
	handlers, repo, _ := setup(t)
	award, _ := repo.Create(awards.CreateInput{Title: "t", Issuer: "i", Date: "2026-01-01"})

	body := map[string]any{
		"title": "t", "issuer": "i", "date": "2026-01-01",
		"game_id": "aaaaaaaaaaaaaaaaaaaaaaaa",
	}
	rec := put(t, handlers, award.ID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var body2 struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body2)
	if body2.Error.Code != "unknown_reference" {
		t.Errorf("error.code = %q, want %q", body2.Error.Code, "unknown_reference")
	}
}

func TestUpdate_UnknownAwardIsNotFound(t *testing.T) {
	handlers, _, _ := setup(t)

	body := map[string]any{"title": "t", "issuer": "i", "date": "2026-01-01"}
	if rec := put(t, handlers, "aaaaaaaaaaaaaaaaaaaaaaaa", body); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDelete_RemovesAward(t *testing.T) {
	handlers, repo, _ := setup(t)
	award, _ := repo.Create(awards.CreateInput{Title: "t", Issuer: "i", Date: "2026-01-01"})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/awards/"+award.ID, nil)
	req.SetPathValue("id", award.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if list, _ := repo.List("", false); len(list) != 0 {
		t.Errorf("awards remaining = %d, want 0", len(list))
	}
}

func TestList_DefaultsToNewestFirst(t *testing.T) {
	handlers, repo, _ := setup(t)
	for _, a := range []awards.CreateInput{
		{Title: "middle", Issuer: "b", Date: "2026-02-01"},
		{Title: "oldest", Issuer: "c", Date: "2025-01-01"},
		{Title: "newest", Issuer: "a", Date: "2026-12-01"},
	} {
		if _, err := repo.Create(a); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/awards", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	var got []awardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 3 || got[0].Title != "newest" || got[2].Title != "oldest" {
		t.Errorf("default order = %v, want newest first", []string{got[0].Title, got[1].Title, got[2].Title})
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, query := range []string{"?sort=password_hash", "?sort=title&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/awards"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestList_ReturnsAnEmptyArrayNotNull(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/awards", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("empty list body = %q, want %q", body, "[]\n")
	}
}
