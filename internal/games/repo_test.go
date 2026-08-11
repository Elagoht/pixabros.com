package games

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
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

func TestRepo_CreateGeneratesUniqueSlug(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if first.Slug != "pixel-quest" {
		t.Errorf("Slug = %q, want %q", first.Slug, "pixel-quest")
	}

	second, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() second error = %v", err)
	}
	if second.Slug != "pixel-quest-2" {
		t.Errorf("Slug = %q, want %q", second.Slug, "pixel-quest-2")
	}
}

func TestRepo_CreateDefaultsAndFindByID(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest", IsPublished: true, DisplayOrder: 3})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if game.ID == 0 {
		t.Fatal("Create() returned a zero ID")
	}
	if game.ExternalLinksJSON != "[]" {
		t.Errorf("ExternalLinksJSON = %q, want %q", game.ExternalLinksJSON, "[]")
	}
	if game.WebExportPath != "" {
		t.Errorf("WebExportPath = %q, want empty (not yet uploaded)", game.WebExportPath)
	}
	if game.CDCoverArtID != nil {
		t.Errorf("CDCoverArtID = %v, want nil (not yet uploaded)", game.CDCoverArtID)
	}

	found, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Title != "Pixel Quest" || found.DisplayOrder != 3 || !found.IsPublished {
		t.Errorf("FindByID() = %+v, want Title=Pixel Quest DisplayOrder=3 IsPublished=true", found)
	}

	if _, err := repo.FindByID(999999); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("FindByID() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_FindBySlug(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindBySlug("pixel-quest")
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}
	if found.ID != game.ID {
		t.Errorf("FindBySlug() ID = %d, want %d", found.ID, game.ID)
	}

	if _, err := repo.FindBySlug("does-not-exist"); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("FindBySlug() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_UpdateNeverChangesSlug(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (42, 'cartridge.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	cartridgeID := int64(42)
	updated, err := repo.Update(game.ID, UpdateInput{
		Title:          "Pixel Quest: Remastered",
		IsPublished:    true,
		CartridgeArtID: &cartridgeID,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Slug != "pixel-quest" {
		t.Errorf("Slug changed to %q, want it to stay %q", updated.Slug, "pixel-quest")
	}
	if updated.Title != "Pixel Quest: Remastered" {
		t.Errorf("Title = %q, want %q", updated.Title, "Pixel Quest: Remastered")
	}
	if updated.CartridgeArtID == nil || *updated.CartridgeArtID != 42 {
		t.Errorf("CartridgeArtID = %v, want pointer to 42", updated.CartridgeArtID)
	}

	if _, err := repo.Update(999999, UpdateInput{Title: "x"}); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("Update() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_Delete(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(game.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(game.ID); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrGameNotFound", err)
	}
	if err := repo.Delete(game.ID); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("Delete() on already-deleted game error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_SetWebExportPath(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.SetWebExportPath(game.ID, "data/games/pixel-quest"); err != nil {
		t.Fatalf("SetWebExportPath() error = %v", err)
	}

	found, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.WebExportPath != "data/games/pixel-quest" {
		t.Errorf("WebExportPath = %q, want %q", found.WebExportPath, "data/games/pixel-quest")
	}
}

func TestRepo_ListOrdersByDisplayOrder(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	repo.Create(CreateInput{Title: "Third", DisplayOrder: 3})
	repo.Create(CreateInput{Title: "First", DisplayOrder: 1})
	repo.Create(CreateInput{Title: "Second", DisplayOrder: 2})

	list, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List() returned %d games, want 3", len(list))
	}
	got := []string{list[0].Title, list[1].Title, list[2].Title}
	want := []string{"First", "Second", "Third"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d].Title = %q, want %q (full order = %v)", i, got[i], want[i], got)
		}
	}
}
