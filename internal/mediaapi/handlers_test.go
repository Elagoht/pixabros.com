package mediaapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func setupMediaHandlers(t *testing.T) (*Handlers, *media.Repo) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	repo := media.NewRepo(conn)
	// Root = the bare data dir and baseURL = "" mirrors main.go exactly: the
	// stored key already carries the "media/" namespace.
	files := storage.NewLocalDisk(t.TempDir(), "")
	return NewHandlers(repo, files), repo
}

func TestGet_ReturnsMediaWithPublicURL(t *testing.T) {
	handlers, repo := setupMediaHandlers(t)
	saved, err := repo.Create("media/cartridge_art/2026-abc123.webp", 400, 560)
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got mediaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	want := mediaResponse{ID: saved.ID, URL: "/media/cartridge_art/2026-abc123.webp", Width: 400, Height: 560}
	if got != want {
		t.Errorf("Get() = %+v, want %+v", got, want)
	}
}

func TestGet_UnknownIDNotFound(t *testing.T) {
	handlers, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestGet_NonNumericIDRejected(t *testing.T) {
	handlers, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/not-a-number", nil)
	req.SetPathValue("id", "not-a-number")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
