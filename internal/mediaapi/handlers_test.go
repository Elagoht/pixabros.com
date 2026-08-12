package mediaapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func setupMediaHandlers(t *testing.T) (*Handlers, *media.Repo, *sql.DB) {
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
	return NewHandlers(repo, files, conn), repo, conn
}

func TestGet_ReturnsMediaWithPublicURL(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/cartridge_art/2026-abc123.webp", 400, 560)
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
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
	handlers, _, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
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

func TestList_AnnotatesUsageAndCountsOrphans(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)

	used, err := repo.Create("media/cartridge_art/used.webp", 400, 560)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := repo.Create("media/screenshot/loose.webp", 1280, 720); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id) VALUES ('g1', 'pixel', 'Pixel Quest', ?);`,
		used.ID,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got libraryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(got.Items))
	}
	// The count is reported separately because picking it out of the list by
	// eye is exactly the tedious part.
	if got.Orphaned != 1 {
		t.Errorf("orphaned = %d, want 1", got.Orphaned)
	}

	for _, item := range got.Items {
		if item.ID != used.ID {
			continue
		}
		if len(item.Usages) != 1 || item.Usages[0].Module != "games" ||
			item.Usages[0].Label != "Pixel Quest" {
			t.Errorf("usages = %+v, want one games usage labelled with the title", item.Usages)
		}
	}
}

// The UI iterates usages, so an unused image must carry an empty array rather
// than a JSON null.
func TestList_UnusedImageHasAnEmptyUsageArray(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	if _, err := repo.Create("media/screenshot/loose.webp", 1280, 720); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); !strings.Contains(body, `"usages":[]`) {
		t.Errorf("body = %s, want an empty usages array", body)
	}
}

func TestUpdate_SetsAltText(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/screenshot/a.webp", 1280, 720)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if saved.AltText != "" {
		t.Fatalf("AltText = %q, want empty on upload", saved.AltText)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{"alt_text":"Two players under a falling block"}`)))
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	after, _ := repo.FindByID(saved.ID)
	if after.AltText != "Two players under a falling block" {
		t.Errorf("AltText = %q, want the submitted text", after.AltText)
	}
}

// A pointer field is what lets alt text be cleared: a plain string could not
// tell "" apart from "the field was omitted".
func TestUpdate_CanClearAltTextButNotOmitIt(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, _ := repo.Create("media/screenshot/a.webp", 1280, 720)
	repo.SetAltText(saved.ID, "something")

	req := httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{"alt_text":""}`)))
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("clearing: status = %d, want %d", rec.Code, http.StatusOK)
	}
	after, _ := repo.FindByID(saved.ID)
	if after.AltText != "" {
		t.Errorf("AltText = %q, want it cleared", after.AltText)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", saved.ID)
	rec = httptest.NewRecorder()
	handlers.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("omitting: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDelete_RemovesAnUnusedImageAndItsFile(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/screenshot/loose.webp", 1280, 720)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, err := repo.FindByID(saved.ID); !errors.Is(err, media.ErrMediaNotFound) {
		t.Errorf("FindByID() after delete error = %v, want ErrMediaNotFound", err)
	}
}

// Deleting a referenced image would blank a game's artwork -- and for a
// screenshot, whose media_id cascades, remove the screenshot row outright. It
// is refused rather than quietly breaking a page.
func TestDelete_RefusesAnImageThatIsStillUsed(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)
	saved, _ := repo.Create("media/cartridge_art/used.webp", 400, 560)
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id) VALUES ('g1', 'pixel', 'Pixel Quest', ?);`,
		saved.ID,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "still_in_use" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "still_in_use")
	}
	if _, err := repo.FindByID(saved.ID); err != nil {
		t.Errorf("the refused delete removed the row anyway: %v", err)
	}
}

func TestDelete_UnknownImageIsNotFound(t *testing.T) {
	handlers, _, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
