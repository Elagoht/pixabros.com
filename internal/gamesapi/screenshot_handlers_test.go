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
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

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
	handlers := NewHandlers(repo, conn, t.TempDir())

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
	handlers := NewHandlers(repo, conn, t.TempDir())

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

// TestRemoveScreenshot_OtherGamesScreenshotNotFound covers the cross-game
// case: game A's URL must not be able to delete game B's screenshot, because
// only game A's rendered page would then be invalidated.
func TestRemoveScreenshot_OtherGamesScreenshotNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	gameA, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	gameB, _ := repo.Create(games.CreateInput{Title: "Neon Drift"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	screenshot, err := repo.AddScreenshot(gameB.ID, 55, 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/games/%d/screenshots/%d", gameA.ID, screenshot.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", gameA.ID))
	req.SetPathValue("screenshotID", fmt.Sprintf("%d", screenshot.ID))
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
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", gameA.ID)).Scan(&jobCount)
	if jobCount != 0 {
		t.Errorf("regen_jobs count for game:%d = %d, want 0 (nothing changed)", gameA.ID, jobCount)
	}
}

func TestAddScreenshot_UnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(map[string]int{"media_id": 55, "display_order": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/999/screenshots", bytes.NewReader(body))
	req.SetPathValue("id", "999")
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
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100), (56, 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	if _, err := repo.AddScreenshot(game.ID, 56, 1); err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	if _, err := repo.AddScreenshot(game.ID, 55, 0); err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	handlers := NewHandlers(repo, conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/games/%d/screenshots", game.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
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
	if got[0].MediaID != 55 || got[0].DisplayOrder != 0 {
		t.Errorf("first screenshot = %+v, want MediaID=55 DisplayOrder=0", got[0])
	}
	if got[1].MediaID != 56 || got[1].DisplayOrder != 1 {
		t.Errorf("second screenshot = %+v, want MediaID=56 DisplayOrder=1", got[1])
	}
	if got[0].GameID != game.ID {
		t.Errorf("GameID = %d, want %d", got[0].GameID, game.ID)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/games/%d/screenshots", game.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
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

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/999/screenshots", nil)
	req.SetPathValue("id", "999")
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

func TestListScreenshots_NonNumericIDRejected(t *testing.T) {
	conn := setupTestDB(t)
	handlers := NewHandlers(games.NewRepo(conn), conn, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/abc/screenshots", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	handlers.ListScreenshots(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestReorderScreenshots_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100), (56, 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	first, _ := repo.AddScreenshot(game.ID, 55, 0)
	second, _ := repo.AddScreenshot(game.ID, 56, 1)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []int64{second.ID, first.ID}})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/games/%d/screenshots/reorder", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
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
		`INSERT INTO media (id, path, width, height) VALUES (55, 'shot1.webp', 100, 100), (56, 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	ownScreenshot, _ := repo.AddScreenshot(gameA.ID, 55, 0)
	otherScreenshot, _ := repo.AddScreenshot(gameB.ID, 56, 0)
	handlers := NewHandlers(repo, conn, t.TempDir())

	body, _ := json.Marshal(reorderRequest{IDs: []int64{ownScreenshot.ID, otherScreenshot.ID}})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/games/%d/screenshots/reorder", gameA.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", gameA.ID))
	rec := httptest.NewRecorder()
	handlers.ReorderScreenshots(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
