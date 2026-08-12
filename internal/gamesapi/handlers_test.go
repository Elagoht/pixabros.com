package gamesapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/games"
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

func TestList_ReturnsAllGames(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got []gameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Pixel Quest" {
		t.Errorf("List() = %+v, want one game titled Pixel Quest", got)
	}
}

func TestCreate_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	// is_browser_playable is deliberately sent and deliberately ignored: it
	// is derived from whether a build exists, not chosen by the caller.
	body, _ := json.Marshal(map[string]interface{}{
		"title":               "Pixel Quest",
		"short_description":   "A tiny adventure.",
		"is_browser_playable": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got gameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Slug != "pixel-quest" {
		t.Errorf("Slug = %q, want %q", got.Slug, "pixel-quest")
	}
	if got.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = true for a game with no build; the client must not be able to set it")
	}

	var jobCount int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = 'game:list';`).Scan(&jobCount); err != nil {
		t.Fatalf("query regen_jobs: %v", err)
	}
	if jobCount != 1 {
		t.Errorf("regen_jobs count for tag game:list = %d, want 1", jobCount)
	}
}

func TestCreate_MissingTitle(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]string{"title": "  "})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGet_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/1", nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGet_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdate_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]interface{}{
		"title":        "Pixel Quest: Remastered",
		"is_published": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/"+game.ID, bytes.NewReader(body))
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got gameResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Title != "Pixel Quest: Remastered" || got.Slug != "pixel-quest-remastered" {
		t.Errorf("Update() = %+v, want Title and Slug both to reflect the new title", got)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%s", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%s = %d, want 1", game.ID, jobCount)
	}
}

func TestGet_SuccessBySlug(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/pixel-quest", nil)
	req.SetPathValue("id", game.Slug)
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got gameResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.ID != game.ID {
		t.Errorf("ID = %s, want %s", got.ID, game.ID)
	}
}

func TestUpdate_SuccessBySlugAndMovesExtractedBuild(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})

	playDir := t.TempDir()
	oldBuildDir := filepath.Join(playDir, game.Slug)
	if err := os.MkdirAll(oldBuildDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldBuildDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, oldBuildDir); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, playDir)

	body, _ := json.Marshal(map[string]interface{}{"title": "Pixel Quest: Remastered"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/pixel-quest", bytes.NewReader(body))
	req.SetPathValue("id", game.Slug)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got gameResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Slug != "pixel-quest-remastered" {
		t.Fatalf("Slug = %q, want %q", got.Slug, "pixel-quest-remastered")
	}

	newBuildDir := filepath.Join(playDir, "pixel-quest-remastered")
	if _, err := os.Stat(filepath.Join(newBuildDir, "index.html")); err != nil {
		t.Errorf("os.Stat(%q) error = %v, want the build moved to the new slug's directory", newBuildDir, err)
	}
	if _, err := os.Stat(oldBuildDir); !os.IsNotExist(err) {
		t.Errorf("os.Stat(%q) error = %v, want the old slug's directory to be gone", oldBuildDir, err)
	}
	if got.WebExportPath != newBuildDir {
		t.Errorf("WebExportPath = %q, want %q", got.WebExportPath, newBuildDir)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]string{"title": "x"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa", bytes.NewReader(body))
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDelete_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/"+game.ID, nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, err := repo.FindByID(game.ID); err == nil {
		t.Error("game should be deleted")
	}
}

// TestDelete_RemovesExtractedBuildDirectory covers the disk side of deletion:
// /play/{slug}/ is an ungated file-server mount, so a deleted game whose
// extracted build survives stays publicly playable forever.
func TestDelete_RemovesExtractedBuildDirectory(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})

	playDir := t.TempDir()
	buildDir := filepath.Join(playDir, game.Slug)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, playDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/"+game.ID, nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Errorf("os.Stat(%q) error = %v, want the extracted build directory to be gone", buildDir, err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestReorder_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	first, _ := repo.Create(games.CreateInput{Title: "First", DisplayOrder: 0})
	second, _ := repo.Create(games.CreateInput{Title: "Second", DisplayOrder: 1})
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []string{second.ID, first.ID}})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/reorder", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Reorder(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	list, err := repo.List("", false)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Errorf("List() = %+v, want [second, first] in that order", list)
	}
}

func TestReorder_MissingIDs(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []string{}})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/reorder", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Reorder(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestReorder_UnknownIDNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []string{"aaaaaaaaaaaaaaaaaaaaaaaa"}})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/reorder", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Reorder(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestDeleteBuild_RemovesFilesAndClearsBrowserPlayable(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	playDir := t.TempDir()
	handlers := NewHandlers(repo, conn, playDir)

	game, err := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	buildDir := filepath.Join(playDir, game.Slug)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatalf("seed build dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("seed index.html: %v", err)
	}
	if err := repo.SetBuild(game.ID, buildDir); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/"+game.ID+"/build", nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.DeleteBuild(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Error("the extracted build is still on disk after DeleteBuild")
	}

	after, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if after.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = true after the build was deleted, want false")
	}
	if after.WebExportPath != "" {
		t.Errorf("WebExportPath = %q after delete, want it cleared", after.WebExportPath)
	}
}

func TestDeleteBuild_UnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	handlers := NewHandlers(games.NewRepo(conn), conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa/build", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.DeleteBuild(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestList_SortsByQueryParams(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	for _, title := range []string{"banana", "Apple", "cherry"} {
		if _, err := repo.Create(games.CreateInput{Title: title}); err != nil {
			t.Fatalf("Create(%q) error = %v", title, err)
		}
	}

	titlesFor := func(t *testing.T, query string) []string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/games"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var got []gameResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		titles := make([]string, 0, len(got))
		for _, g := range got {
			titles = append(titles, g.Title)
		}
		return titles
	}

	if got, want := titlesFor(t, "?sort=title&dir=asc"), []string{"Apple", "banana", "cherry"}; !slices.Equal(got, want) {
		t.Errorf("?sort=title&dir=asc = %v, want %v", got, want)
	}
	if got, want := titlesFor(t, "?sort=title&dir=desc"), []string{"cherry", "banana", "Apple"}; !slices.Equal(got, want) {
		t.Errorf("?sort=title&dir=desc = %v, want %v", got, want)
	}
	// No params at all must keep the manual display order.
	if got, want := titlesFor(t, ""), titlesFor(t, "?sort=display_order"); !slices.Equal(got, want) {
		t.Errorf("no params = %v, want the display_order order %v", got, want)
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	conn := setupTestDB(t)
	handlers := NewHandlers(games.NewRepo(conn), conn, t.TempDir())

	for _, query := range []string{"?sort=password_hash", "?sort=title&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/games"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}
