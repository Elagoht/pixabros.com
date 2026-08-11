package gamesapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	handlers := NewHandlers(repo, conn)

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
	handlers := NewHandlers(repo, conn)

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
	if !got.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = false, want true")
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
	handlers := NewHandlers(repo, conn)

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
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/1", nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGet_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/999", nil)
	req.SetPathValue("id", "999")
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
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]interface{}{
		"title":        "Pixel Quest: Remastered",
		"is_published": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/"+fmt.Sprintf("%d", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got gameResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Title != "Pixel Quest: Remastered" || got.Slug != "pixel-quest" {
		t.Errorf("Update() = %+v, want Title changed and Slug unchanged", got)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%d = %d, want 1", game.ID, jobCount)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]string{"title": "x"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/999", bytes.NewReader(body))
	req.SetPathValue("id", "999")
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
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/"+fmt.Sprintf("%d", game.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, err := repo.FindByID(game.ID); err == nil {
		t.Error("game should be deleted")
	}
}

func TestDelete_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
