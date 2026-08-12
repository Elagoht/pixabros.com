package games

import (
	"errors"
	"testing"
)

func TestRepo_AddListRemoveScreenshot(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100), ('media-102', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}

	first, err := repo.AddScreenshot(game.ID, "media-101", 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	if first.ID == "" {
		t.Fatal("AddScreenshot() returned a zero ID")
	}
	if _, err := repo.AddScreenshot(game.ID, "media-102", 1); err != nil {
		t.Fatalf("second AddScreenshot() error = %v", err)
	}

	list, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListScreenshots() returned %d, want 2", len(list))
	}
	if list[0].MediaID != "media-101" || list[1].MediaID != "media-102" {
		t.Errorf("ListScreenshots() = %+v, want media IDs [media-101, media-102] in order", list)
	}

	if err := repo.RemoveScreenshot(game.ID, first.ID); err != nil {
		t.Fatalf("RemoveScreenshot() error = %v", err)
	}
	remaining, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() after remove error = %v", err)
	}
	if len(remaining) != 1 || remaining[0].MediaID != "media-102" {
		t.Errorf("ListScreenshots() after remove = %+v, want just media ID media-102", remaining)
	}
}

func TestRepo_RemoveScreenshotIsScopedToItsGame(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	gameA, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() gameA error = %v", err)
	}
	gameB, err := repo.Create(CreateInput{Title: "Neon Drift"})
	if err != nil {
		t.Fatalf("Create() gameB error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	shot, err := repo.AddScreenshot(gameA.ID, "media-101", 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}

	err = repo.RemoveScreenshot(gameB.ID, shot.ID)
	if !errors.Is(err, ErrScreenshotNotFound) {
		t.Errorf("RemoveScreenshot(gameB, gameA's screenshot) error = %v, want ErrScreenshotNotFound", err)
	}

	list, err := repo.ListScreenshots(gameA.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != shot.ID {
		t.Errorf("ListScreenshots(gameA) = %+v, want game A to still own screenshot %s", list, shot.ID)
	}
}

func TestRepo_RemoveScreenshotUnknownIDNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.RemoveScreenshot(game.ID, "bbbbbbbbbbbbbbbbbbbbbbbb"); !errors.Is(err, ErrScreenshotNotFound) {
		t.Errorf("RemoveScreenshot() error = %v, want ErrScreenshotNotFound", err)
	}
}

func TestRepo_AddScreenshotUnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	if _, err := repo.AddScreenshot("aaaaaaaaaaaaaaaaaaaaaaaa", "media-101", 0); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("AddScreenshot() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_DeleteGameCascadesScreenshots(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	if _, err := repo.AddScreenshot(game.ID, "media-101", 0); err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}

	if err := repo.Delete(game.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	list, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() after game delete error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListScreenshots() after game delete = %+v, want empty (ON DELETE CASCADE)", list)
	}
}

func TestRepo_ReorderScreenshotsSetsDisplayOrderToIndex(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100), ('media-102', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	first, _ := repo.AddScreenshot(game.ID, "media-101", 0)
	second, _ := repo.AddScreenshot(game.ID, "media-102", 1)

	if err := repo.ReorderScreenshots(game.ID, []string{second.ID, first.ID}); err != nil {
		t.Fatalf("ReorderScreenshots() error = %v", err)
	}

	list, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() error = %v", err)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Errorf("ListScreenshots() = %+v, want [second, first] in that order", list)
	}
}

// A screenshot id belonging to a different game must not be reorderable
// through this game's URL, the same scoping RemoveScreenshot already
// enforces.
func TestRepo_ReorderScreenshotsRejectsIDFromAnotherGame(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	gameA, _ := repo.Create(CreateInput{Title: "Game A"})
	gameB, _ := repo.Create(CreateInput{Title: "Game B"})
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-101', 'shot1.webp', 100, 100), ('media-102', 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}
	ownScreenshot, _ := repo.AddScreenshot(gameA.ID, "media-101", 0)
	otherScreenshot, _ := repo.AddScreenshot(gameB.ID, "media-102", 0)

	err := repo.ReorderScreenshots(gameA.ID, []string{ownScreenshot.ID, otherScreenshot.ID})
	if !errors.Is(err, ErrScreenshotNotFound) {
		t.Fatalf("ReorderScreenshots() error = %v, want ErrScreenshotNotFound", err)
	}

	// Rolled back: gameA's own screenshot must be untouched too.
	list, findErr := repo.ListScreenshots(gameA.ID)
	if findErr != nil {
		t.Fatalf("ListScreenshots() error = %v", findErr)
	}
	if len(list) != 1 || list[0].DisplayOrder != 0 {
		t.Errorf("gameA's screenshots = %+v, want the original single screenshot at order 0", list)
	}
}
