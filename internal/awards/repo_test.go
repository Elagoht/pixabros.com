package awards

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/dbutil"
	"pixabros/internal/id"
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

func TestRepo_CreateFindUpdateDelete(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	created, err := repo.Create(CreateInput{
		Title: "Best Game", Issuer: "IGF", Date: "2026-03-18",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !id.IsValid(created.ID) {
		t.Errorf("Create() id = %q, which is not a well-formed id", created.ID)
	}
	if created.PictureID != nil || created.GameID != nil {
		t.Error("a new award should have no picture or game attached")
	}
	// An award with no link must read as absent, not as the empty string.
	if created.Link != "" {
		t.Errorf("Link = %q, want empty", created.Link)
	}

	// Seed the rows the foreign keys point at.
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES ('0123456789abcdef01234567', 'p.webp', 320, 320);`,
	); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title) VALUES ('abcdef0123456789abcdef01', 'g', 'G');`,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	picture := "0123456789abcdef01234567"
	game := "abcdef0123456789abcdef01"
	updated, err := repo.Update(created.ID, UpdateInput{
		Title: "Best Game Ever", Issuer: "IGF", Date: "2026-03-19",
		Link: "https://igf.example", PictureID: &picture, GameID: &game,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "Best Game Ever" || updated.Date != "2026-03-19" {
		t.Errorf("Update() = %+v, want the new title and date", updated)
	}
	if updated.PictureID == nil || *updated.PictureID != picture {
		t.Errorf("PictureID = %v, want %q", updated.PictureID, picture)
	}
	if updated.GameID == nil || *updated.GameID != game {
		t.Errorf("GameID = %v, want %q", updated.GameID, game)
	}

	if err := repo.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(created.ID); !errors.Is(err, ErrAwardNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrAwardNotFound", err)
	}
}

func TestRepo_MissingAwardIsReported(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	missing := "aaaaaaaaaaaaaaaaaaaaaaaa"

	if _, err := repo.FindByID(missing); !errors.Is(err, ErrAwardNotFound) {
		t.Errorf("FindByID() error = %v, want ErrAwardNotFound", err)
	}
	if _, err := repo.Update(missing, UpdateInput{Title: "t", Issuer: "i", Date: "2026-01-01"}); !errors.Is(err, ErrAwardNotFound) {
		t.Errorf("Update() error = %v, want ErrAwardNotFound", err)
	}
	if err := repo.Delete(missing); !errors.Is(err, ErrAwardNotFound) {
		t.Errorf("Delete() error = %v, want ErrAwardNotFound", err)
	}
}

// Pointing an award at a game that does not exist has to be refused, or the
// public timeline would render an award linked to nothing.
func TestRepo_UnknownGameIsRefused(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	award, err := repo.Create(CreateInput{Title: "t", Issuer: "i", Date: "2026-01-01"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	missing := "aaaaaaaaaaaaaaaaaaaaaaaa"
	_, err = repo.Update(award.ID, UpdateInput{
		Title: "t", Issuer: "i", Date: "2026-01-01", GameID: &missing,
	})
	if !dbutil.IsForeignKeyViolation(err) {
		t.Errorf("Update() with an unknown game error = %v, want a foreign key violation", err)
	}
}

// Deleting a game must leave its awards in place, just unlinked.
func TestRepo_DeletingAGameClearsTheLink(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title) VALUES ('abcdef0123456789abcdef01', 'g', 'G');`,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}
	award, _ := repo.Create(CreateInput{Title: "t", Issuer: "i", Date: "2026-01-01"})
	game := "abcdef0123456789abcdef01"
	if _, err := repo.Update(award.ID, UpdateInput{
		Title: "t", Issuer: "i", Date: "2026-01-01", GameID: &game,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if _, err := conn.Exec(`DELETE FROM games WHERE id = ?;`, game); err != nil {
		t.Fatalf("delete game: %v", err)
	}

	after, err := repo.FindByID(award.ID)
	if err != nil {
		t.Fatalf("FindByID() after the game was deleted error = %v", err)
	}
	if after.GameID != nil {
		t.Errorf("GameID = %v after its game was deleted, want nil", after.GameID)
	}
}

func TestRepo_ListSorting(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	for _, a := range []CreateInput{
		{Title: "middle", Issuer: "b", Date: "2026-02-01"},
		{Title: "oldest", Issuer: "c", Date: "2025-01-01"},
		{Title: "newest", Issuer: "a", Date: "2026-12-01"},
	} {
		if _, err := repo.Create(a); err != nil {
			t.Fatalf("Create(%q) error = %v", a.Title, err)
		}
	}

	titlesFor := func(t *testing.T, field string, descending bool) []string {
		t.Helper()
		list, err := repo.List(field, descending)
		if err != nil {
			t.Fatalf("List(%q, %v) error = %v", field, descending, err)
		}
		titles := make([]string, 0, len(list))
		for _, a := range list {
			titles = append(titles, a.Title)
		}
		return titles
	}

	// Awards are a timeline, so with no sort chosen the newest comes first.
	if got, want := titlesFor(t, "", false), []string{"newest", "middle", "oldest"}; !slices.Equal(got, want) {
		t.Errorf("default order = %v, want %v (newest first)", got, want)
	}
	if got, want := titlesFor(t, "date", false), []string{"oldest", "middle", "newest"}; !slices.Equal(got, want) {
		t.Errorf("date asc = %v, want %v", got, want)
	}
	if got, want := titlesFor(t, "issuer", false), []string{"newest", "middle", "oldest"}; !slices.Equal(got, want) {
		t.Errorf("issuer asc = %v, want %v", got, want)
	}

	if _, err := repo.List("title; DROP TABLE awards", false); !errors.Is(err, ErrInvalidSort) {
		t.Errorf("List() with an injected field error = %v, want ErrInvalidSort", err)
	}
}
