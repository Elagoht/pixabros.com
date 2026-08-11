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
		`INSERT INTO media (id, path, width, height) VALUES (101, 'shot1.webp', 100, 100), (102, 'shot2.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media rows error = %v", err)
	}

	first, err := repo.AddScreenshot(game.ID, 101, 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	if first.ID == 0 {
		t.Fatal("AddScreenshot() returned a zero ID")
	}
	if _, err := repo.AddScreenshot(game.ID, 102, 1); err != nil {
		t.Fatalf("second AddScreenshot() error = %v", err)
	}

	list, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListScreenshots() returned %d, want 2", len(list))
	}
	if list[0].MediaID != 101 || list[1].MediaID != 102 {
		t.Errorf("ListScreenshots() = %+v, want media IDs [101, 102] in order", list)
	}

	if err := repo.RemoveScreenshot(game.ID, first.ID); err != nil {
		t.Fatalf("RemoveScreenshot() error = %v", err)
	}
	remaining, err := repo.ListScreenshots(game.ID)
	if err != nil {
		t.Fatalf("ListScreenshots() after remove error = %v", err)
	}
	if len(remaining) != 1 || remaining[0].MediaID != 102 {
		t.Errorf("ListScreenshots() after remove = %+v, want just media ID 102", remaining)
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
		`INSERT INTO media (id, path, width, height) VALUES (101, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	shot, err := repo.AddScreenshot(gameA.ID, 101, 0)
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
		t.Errorf("ListScreenshots(gameA) = %+v, want game A to still own screenshot %d", list, shot.ID)
	}
}

func TestRepo_RemoveScreenshotUnknownIDNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.RemoveScreenshot(game.ID, 4242); !errors.Is(err, ErrScreenshotNotFound) {
		t.Errorf("RemoveScreenshot() error = %v, want ErrScreenshotNotFound", err)
	}
}

func TestRepo_AddScreenshotUnknownGameNotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (101, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	if _, err := repo.AddScreenshot(999, 101, 0); !errors.Is(err, ErrGameNotFound) {
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
		`INSERT INTO media (id, path, width, height) VALUES (101, 'shot1.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	if _, err := repo.AddScreenshot(game.ID, 101, 0); err != nil {
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
