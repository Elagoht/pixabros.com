package gamesapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pixabros/internal/games"
)

func TestAddScreenshot_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]int{"media_id": 55, "display_order": 0})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/games/%d/screenshots", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.AddScreenshot(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got screenshotResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.MediaID != 55 || got.GameID != game.ID {
		t.Errorf("AddScreenshot() = %+v, want MediaID=55 GameID=%d", got, game.ID)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%d = %d, want 1", game.ID, jobCount)
	}
}

func TestAddScreenshot_MissingMediaID(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]int{"display_order": 0})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/games/%d/screenshots", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.AddScreenshot(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRemoveScreenshot_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	screenshot, _ := repo.AddScreenshot(game.ID, 55, 0)
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/games/%d/screenshots/%d", game.ID, screenshot.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	req.SetPathValue("screenshotID", fmt.Sprintf("%d", screenshot.ID))
	rec := httptest.NewRecorder()
	handlers.RemoveScreenshot(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	remaining, _ := repo.ListScreenshots(game.ID)
	if len(remaining) != 0 {
		t.Errorf("ListScreenshots() = %+v, want empty after removal", remaining)
	}
}
