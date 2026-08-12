package gamesapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pixabros/internal/games"
)

func TestAddScreenshot_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]any{"media_id": "media-55", "display_order": 0})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/games/%s/screenshots", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.AddScreenshot(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got screenshotResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.MediaID != "media-55" || got.GameID != game.ID {
		t.Errorf("AddScreenshot() = %+v, want MediaID=media-55 GameID=%s", got, game.ID)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%s", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%s = %d, want 1", game.ID, jobCount)
	}
}

func TestAddScreenshot_MissingMediaID(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]int{"display_order": 0})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/games/%s/screenshots", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", game.ID)
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
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	screenshot, _ := repo.AddScreenshot(game.ID, "media-55", 0)
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/games/%s/screenshots/%s", game.ID, screenshot.ID), nil)
	req.SetPathValue("id", game.ID)
	req.SetPathValue("screenshotID", screenshot.ID)
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

// TestRemoveScreenshot_OtherGamesScreenshotNotFound covers the cross-game
// case: game A's URL must not be able to delete game B's screenshot, because
// only game A's rendered page would then be invalidated.
func TestRemoveScreenshot_OtherGamesScreenshotNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	gameA, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	gameB, _ := repo.Create(games.CreateInput{Title: "Neon Drift"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	screenshot, err := repo.AddScreenshot(gameB.ID, "media-55", 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/games/%s/screenshots/%s", gameA.ID, screenshot.ID), nil)
	req.SetPathValue("id", gameA.ID)
	req.SetPathValue("screenshotID", screenshot.ID)
	rec := httptest.NewRecorder()
	handlers.RemoveScreenshot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	remaining, _ := repo.ListScreenshots(gameB.ID)
	if len(remaining) != 1 {
		t.Errorf("ListScreenshots(gameB) = %+v, want game B's screenshot untouched", remaining)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%s", gameA.ID)).Scan(&jobCount)
	if jobCount != 0 {
		t.Errorf("regen_jobs count for game:%s = %d, want 0 (nothing changed)", gameA.ID, jobCount)
	}
}

func TestAddScreenshot_UnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]any{"media_id": "media-55", "display_order": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa/screenshots", bytes.NewReader(body))
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.AddScreenshot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestListScreenshots_ReturnsScreenshotsInDisplayOrder(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100), ('media-56', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	if _, err := repo.AddScreenshot(game.ID, "media-56", 1); err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	if _, err := repo.AddScreenshot(game.ID, "media-55", 0); err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/games/%s/screenshots", game.ID), nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.ListScreenshots(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got []screenshotResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListScreenshots()) = %d, want 2", len(got))
	}
	if got[0].MediaID != "media-55" || got[0].DisplayOrder != 0 {
		t.Errorf("first screenshot = %+v, want MediaID=55 DisplayOrder=0", got[0])
	}
	if got[1].MediaID != "media-56" || got[1].DisplayOrder != 1 {
		t.Errorf("second screenshot = %+v, want MediaID=56 DisplayOrder=1", got[1])
	}
	if got[0].GameID != game.ID {
		t.Errorf("GameID = %s, want %s", got[0].GameID, game.ID)
	}
}

// A game with no screenshots must answer with [] rather than null: the admin
// UI maps over the response directly, and null would force every caller to
// defend against it.
func TestListScreenshots_EmptyListIsAnArrayNotNull(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/games/%s/screenshots", game.ID), nil)
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.ListScreenshots(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %q, want %q", body, "[]")
	}
}

func TestListScreenshots_UnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/aaaaaaaaaaaaaaaaaaaaaaaa/screenshots", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.ListScreenshots(rec, req)

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

// Game ids are opaque strings now, so there is no "must be a number" shape
// to reject up front: anything that is not a real id is simply not found.
func TestListScreenshots_MalformedIDIsNotFound(t *testing.T) {
	conn := setupTestDB(t)
	handlers := NewHandlers(games.NewRepo(conn), conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/abc/screenshots", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	handlers.ListScreenshots(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestReorderScreenshots_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100), ('media-56', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	first, _ := repo.AddScreenshot(game.ID, "media-55", 0)
	second, _ := repo.AddScreenshot(game.ID, "media-56", 1)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []string{second.ID, first.ID}})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/games/%s/screenshots/reorder", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", game.ID)
	rec := httptest.NewRecorder()
	handlers.ReorderScreenshots(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	list, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() error = %v", err)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Errorf("ListScreenshots() = %+v, want [second, first] in that order", list)
	}
}

func TestReorderScreenshots_IDFromAnotherGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	gameA, _ := repo.Create(games.CreateInput{Title: "Game A"})
	gameB, _ := repo.Create(games.CreateInput{Title: "Game B"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-55', 'shot1.webp', 100, 100), ('media-56', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	ownScreenshot, _ := repo.AddScreenshot(gameA.ID, "media-55", 0)
	otherScreenshot, _ := repo.AddScreenshot(gameB.ID, "media-56", 0)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []string{ownScreenshot.ID, otherScreenshot.ID}})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/games/%s/screenshots/reorder", gameA.ID), bytes.NewReader(body))
	req.SetPathValue("id", gameA.ID)
	rec := httptest.NewRecorder()
	handlers.ReorderScreenshots(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
