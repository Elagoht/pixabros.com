package games

import (
	"fmt"
	"slices"
	"strconv"

	"database/sql"
	"errors"
	"path/filepath"
	"pixabros/internal/id"
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
	if game.ID == "" {
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

	if _, err := repo.FindByID("no-such-game-id-000000000"); !errors.Is(err, ErrGameNotFound) {
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
		t.Errorf("FindBySlug() ID = %s, want %s", found.ID, game.ID)
	}

	if _, err := repo.FindBySlug("does-not-exist"); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("FindBySlug() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_UpdateRegeneratesSlugFromTitle(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('0123456789abcdef01234567', 'cartridge.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media row error = %v", err)
	}

	cartridgeID := "0123456789abcdef01234567"
	updated, err := repo.Update(game.ID, UpdateInput{
		Title:          "Pixel Quest: Remastered",
		IsPublished:    true,
		CartridgeArtID: &cartridgeID,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Slug != "pixel-quest-remastered" {
		t.Errorf("Slug = %q, want %q", updated.Slug, "pixel-quest-remastered")
	}
	if updated.Title != "Pixel Quest: Remastered" {
		t.Errorf("Title = %q, want %q", updated.Title, "Pixel Quest: Remastered")
	}
	if updated.CartridgeArtID == nil || *updated.CartridgeArtID != cartridgeID {
		t.Errorf("CartridgeArtID = %v, want pointer to %s", updated.CartridgeArtID, cartridgeID)
	}

	if _, err := repo.Update("no-such-game-id-000000000", UpdateInput{Title: "x"}); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("Update() error = %v, want ErrGameNotFound", err)
	}
}

func TestRepo_UpdateKeepsSameSlugWhenTitleUnchanged(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := repo.Update(game.ID, UpdateInput{Title: "Pixel Quest", IsPublished: true})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Slug != "pixel-quest" {
		t.Errorf("Slug = %q, want it to stay %q when the title didn't change", updated.Slug, "pixel-quest")
	}
}

func TestRepo_UpdateSlugCollisionGetsSuffixed(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() first error = %v", err)
	}
	second, err := repo.Create(CreateInput{Title: "Dungrid Tactics"})
	if err != nil {
		t.Fatalf("Create() second error = %v", err)
	}

	updated, err := repo.Update(second.ID, UpdateInput{Title: first.Title})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Slug != "pixel-quest-2" {
		t.Errorf("Slug = %q, want %q", updated.Slug, "pixel-quest-2")
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

func TestRepo_SetBuildDerivesBrowserPlayable(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if game.IsBrowserPlayable {
		t.Error("a new game with no build should not be browser playable")
	}

	if err := repo.SetBuild(game.ID, "data/games/pixel-quest"); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	found, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.WebExportPath != "data/games/pixel-quest" {
		t.Errorf("WebExportPath = %q, want %q", found.WebExportPath, "data/games/pixel-quest")
	}
	if !found.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = false after a build was recorded, want true")
	}
}

// Removing the build is the only thing that turns browser play back off.
func TestRepo_SetBuildWithEmptyPathClearsBrowserPlayable(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, "data/games/pixel-quest"); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	if err := repo.SetBuild(game.ID, ""); err != nil {
		t.Fatalf("SetBuild(\"\") error = %v", err)
	}

	found, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.WebExportPath != "" {
		t.Errorf("WebExportPath = %q, want it cleared", found.WebExportPath)
	}
	if found.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = true after the build was removed, want false")
	}
}

// An ordinary edit must not disturb the build-derived flag: the admin form
// does not send it, and a full-replace update that zeroed it would take a
// game offline every time its description was corrected.
func TestRepo_UpdateLeavesBrowserPlayableAlone(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, "data/games/pixel-quest"); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	updated, err := repo.Update(game.ID, UpdateInput{Title: "Pixel Quest", ShortDescription: "edited"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !updated.IsBrowserPlayable {
		t.Error("Update() cleared IsBrowserPlayable; it must be left to SetBuild")
	}
	if updated.WebExportPath != "data/games/pixel-quest" {
		t.Errorf("Update() changed WebExportPath to %q", updated.WebExportPath)
	}
}

func TestRepo_ListOrdersByDisplayOrder(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	repo.Create(CreateInput{Title: "Third", DisplayOrder: 3})
	repo.Create(CreateInput{Title: "First", DisplayOrder: 1})
	repo.Create(CreateInput{Title: "Second", DisplayOrder: 2})

	list, err := repo.List("", false)
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

func TestRepo_ReorderSetsDisplayOrderToIndex(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, _ := repo.Create(CreateInput{Title: "First", DisplayOrder: 0})
	second, _ := repo.Create(CreateInput{Title: "Second", DisplayOrder: 1})
	third, _ := repo.Create(CreateInput{Title: "Third", DisplayOrder: 2})

	if err := repo.Reorder([]string{third.ID, first.ID, second.ID}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}

	list, err := repo.List("", false)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := []string{list[0].Title, list[1].Title, list[2].Title}
	want := []string{"Third", "First", "Second"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d].Title = %q, want %q (full order = %v)", i, got[i], want[i], got)
		}
	}
}

// An id that doesn't exist must roll back the whole reorder, not apply the
// ids that came before it in the list -- a caller retrying after this error
// should find the original order untouched.
func TestRepo_ReorderRollsBackOnUnknownID(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, _ := repo.Create(CreateInput{Title: "First", DisplayOrder: 0})
	second, _ := repo.Create(CreateInput{Title: "Second", DisplayOrder: 1})

	err := repo.Reorder([]string{second.ID, "no-such-game-id-000000000"})
	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Reorder() error = %v, want ErrGameNotFound", err)
	}

	reloadedFirst, findErr := repo.FindByID(first.ID)
	if findErr != nil {
		t.Fatalf("FindByID(first) error = %v", findErr)
	}
	if reloadedFirst.DisplayOrder != 0 {
		t.Errorf("first.DisplayOrder = %d, want 0 (unchanged -- the reorder should have rolled back)", reloadedFirst.DisplayOrder)
	}
}

// Ids are handed to the admin UI and appear in URLs, so they must be opaque
// and unguessable rather than a sequence anyone can count through.
func TestRepo_CreateAssignsOpaqueUnguessableIDs(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	seen := map[string]bool{}
	for i := range 5 {
		game, err := repo.Create(CreateInput{Title: fmt.Sprintf("Game %d", i)})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if !id.IsValid(game.ID) {
			t.Errorf("Create() id = %q, which is not a well-formed id", game.ID)
		}
		if _, err := strconv.Atoi(game.ID); err == nil {
			t.Errorf("Create() id = %q is a plain number; ids must not be enumerable", game.ID)
		}
		if seen[game.ID] {
			t.Errorf("Create() reused id %q", game.ID)
		}
		seen[game.ID] = true
	}
}

func TestRepo_AddScreenshotAssignsOpaqueIDs(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('media-1', 'shot.webp', 100, 100);`,
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	shot, err := repo.AddScreenshot(game.ID, "media-1", 0)
	if err != nil {
		t.Fatalf("AddScreenshot() error = %v", err)
	}
	if !id.IsValid(shot.ID) {
		t.Errorf("AddScreenshot() id = %q, which is not a well-formed id", shot.ID)
	}
}

func TestRepo_ListSorting(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	// Created out of alphabetical order, and with mixed case, so a naive
	// byte-wise sort would not match what a reader expects.
	for _, title := range []string{"banana", "Apple", "cherry"} {
		if _, err := repo.Create(CreateInput{Title: title}); err != nil {
			t.Fatalf("Create(%q) error = %v", title, err)
		}
	}

	titlesFor := func(t *testing.T, field string, descending bool) []string {
		t.Helper()
		list, err := repo.List(field, descending)
		if err != nil {
			t.Fatalf("List(%q, %v) error = %v", field, descending, err)
		}
		titles := make([]string, 0, len(list))
		for _, g := range list {
			titles = append(titles, g.Title)
		}
		return titles
	}

	t.Run("title ascending is case-insensitive", func(t *testing.T) {
		got := titlesFor(t, "title", false)
		want := []string{"Apple", "banana", "cherry"}
		if !slices.Equal(got, want) {
			t.Errorf("titles = %v, want %v", got, want)
		}
	})

	t.Run("title descending reverses it", func(t *testing.T) {
		got := titlesFor(t, "title", true)
		want := []string{"cherry", "banana", "Apple"}
		if !slices.Equal(got, want) {
			t.Errorf("titles = %v, want %v", got, want)
		}
	})

	t.Run("empty field falls back to the manual display order", func(t *testing.T) {
		got := titlesFor(t, "", false)
		want := titlesFor(t, "display_order", false)
		if !slices.Equal(got, want) {
			t.Errorf("default order = %v, want the display_order order %v", got, want)
		}
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		if _, err := repo.List("title; DROP TABLE games", false); !errors.Is(err, ErrInvalidSort) {
			t.Errorf("List() with an unknown field error = %v, want ErrInvalidSort", err)
		}
		if _, err := repo.List("password_hash", false); !errors.Is(err, ErrInvalidSort) {
			t.Errorf("List() with an off-whitelist column error = %v, want ErrInvalidSort", err)
		}
	})

	// Every game here shares display_order 0, so without the id tiebreaker
	// the order would be whatever SQLite felt like returning.
	t.Run("ties are broken stably", func(t *testing.T) {
		first := titlesFor(t, "display_order", false)
		for range 5 {
			if got := titlesFor(t, "display_order", false); !slices.Equal(got, first) {
				t.Fatalf("repeated List() returned %v then %v", first, got)
			}
		}
	})
}

// A game that predates the fields, or one an admin has not filled in yet, is a
// production game with no date and no genre -- not a broken row.
func TestRepo_CreateDefaultsAGameToAProduction(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	game, err := repo.Create(CreateInput{Title: "Untyped"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if game.Kind != KindProduction {
		t.Errorf("Kind = %q, want %q", game.Kind, KindProduction)
	}
	if game.Genre != "" || game.ReleaseDate != "" {
		t.Errorf("Genre = %q, ReleaseDate = %q, want both empty", game.Genre, game.ReleaseDate)
	}
}

func TestRepo_RoundTripsTheReleaseGenreAndKind(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	created, err := repo.Create(CreateInput{
		Title:       "Dungrid Tactics",
		Genre:       "Turn-based tactics",
		ReleaseDate: "2026-07-31",
		Kind:        KindGameJam,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Genre != "Turn-based tactics" || created.ReleaseDate != "2026-07-31" || created.Kind != KindGameJam {
		t.Fatalf("Create() stored %+v", created)
	}

	updated, err := repo.Update(created.ID, UpdateInput{
		Title:       "Dungrid Tactics",
		Genre:       "Tactics",
		ReleaseDate: "2026-08-01",
		Kind:        KindProduction,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Genre != "Tactics" || updated.ReleaseDate != "2026-08-01" || updated.Kind != KindProduction {
		t.Errorf("Update() stored %+v", updated)
	}
}

// The column is constrained so a value the site cannot draw never reaches the
// public badge. The API rejects one first; this is the layer underneath.
func TestRepo_RefusesAKindTheSiteCannotDraw(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	if _, err := repo.Create(CreateInput{Title: "Odd", Kind: "prototype"}); err == nil {
		t.Error("Create() accepted a kind outside the two the site knows")
	}
}

// An update that leaves the kind out must not silently demote a jam entry, so
// the caller is expected to send it back. What it must not do is store an
// empty string the templates would then have to guard against.
func TestNormaliseKind_FillsInTheDefault(t *testing.T) {
	if got := NormaliseKind(""); got != KindProduction {
		t.Errorf("NormaliseKind(\"\") = %q, want %q", got, KindProduction)
	}
	if got := NormaliseKind(KindGameJam); got != KindGameJam {
		t.Errorf("NormaliseKind(%q) = %q, want it unchanged", KindGameJam, got)
	}
}

func TestRepo_RoundTripsTheTrailer(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	created, err := repo.Create(CreateInput{
		Title: "Trailered", VideoURL: "https://youtu.be/9mjjowHX1-g",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.VideoURL != "https://youtu.be/9mjjowHX1-g" {
		t.Errorf("VideoURL = %q after create", created.VideoURL)
	}

	updated, err := repo.Update(created.ID, UpdateInput{Title: "Trailered", VideoURL: ""})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.VideoURL != "" {
		t.Errorf("VideoURL = %q, want the trailer removed", updated.VideoURL)
	}
}
