package members

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"pixabros/internal/db"
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
	repo := NewRepo(setupTestDB(t))

	created, err := repo.Create(CreateInput{Name: "Furkan", Tags: "code, design"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !id.IsValid(created.ID) {
		t.Errorf("Create() id = %q, which is not a well-formed id", created.ID)
	}
	// Defaults matter: a new member must not appear on the public site until
	// it is deliberately published.
	if created.IsPublished {
		t.Error("a new member should start unpublished")
	}
	if created.LinksJSON != "[]" {
		t.Errorf("LinksJSON = %q, want %q for a member with no links", created.LinksJSON, "[]")
	}
	if created.AvatarID != nil {
		t.Errorf("AvatarID = %v, want nil for a new member", created.AvatarID)
	}

	found, err := repo.FindByID(created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != "Furkan" {
		t.Errorf("Name = %q, want %q", found.Name, "Furkan")
	}

	avatar := "0123456789abcdef01234567"
	if _, err := setupMedia(t, repo, avatar); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	updated, err := repo.Update(created.ID, UpdateInput{
		Name:        "Furkan B",
		AvatarID:    &avatar,
		IsPublished: true,
		LinksJSON:   `[{"label":"GitHub","url":"https://example.dev"}]`,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Furkan B" || !updated.IsPublished {
		t.Errorf("Update() = %+v, want the new name and published", updated)
	}
	if updated.AvatarID == nil || *updated.AvatarID != avatar {
		t.Errorf("AvatarID = %v, want %q", updated.AvatarID, avatar)
	}

	if err := repo.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(created.ID); !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrMemberNotFound", err)
	}
}

// setupMedia inserts a media row so avatar_id has something real to point at:
// foreign keys are enforced, so an arbitrary id would be rejected.
func setupMedia(t *testing.T, repo *Repo, mediaID string) (sql.Result, error) {
	t.Helper()
	return repo.db.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (?, 'avatar.webp', 400, 400);`, mediaID,
	)
}

func TestRepo_MissingMemberIsReported(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	missing := "aaaaaaaaaaaaaaaaaaaaaaaa"

	if _, err := repo.FindByID(missing); !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("FindByID() error = %v, want ErrMemberNotFound", err)
	}
	if _, err := repo.Update(missing, UpdateInput{Name: "x"}); !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("Update() error = %v, want ErrMemberNotFound", err)
	}
	if err := repo.Delete(missing); !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("Delete() error = %v, want ErrMemberNotFound", err)
	}
}

func TestRepo_ListSorting(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	for _, name := range []string{"berk", "Ada", "cem"} {
		if _, err := repo.Create(CreateInput{Name: name}); err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
	}

	namesFor := func(t *testing.T, field string, descending bool) []string {
		t.Helper()
		list, err := repo.List(field, descending)
		if err != nil {
			t.Fatalf("List(%q, %v) error = %v", field, descending, err)
		}
		names := make([]string, 0, len(list))
		for _, m := range list {
			names = append(names, m.Name)
		}
		return names
	}

	if got, want := namesFor(t, "name", false), []string{"Ada", "berk", "cem"}; !slices.Equal(got, want) {
		t.Errorf("name asc = %v, want %v (case-insensitive)", got, want)
	}
	if got, want := namesFor(t, "name", true), []string{"cem", "berk", "Ada"}; !slices.Equal(got, want) {
		t.Errorf("name desc = %v, want %v", got, want)
	}
	if got, want := namesFor(t, "", false), namesFor(t, "display_order", false); !slices.Equal(got, want) {
		t.Errorf("default order = %v, want the display_order order %v", got, want)
	}

	if _, err := repo.List("name; DROP TABLE members", false); !errors.Is(err, ErrInvalidSort) {
		t.Errorf("List() with an injected field error = %v, want ErrInvalidSort", err)
	}
}

func TestRepo_Reorder(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	first, _ := repo.Create(CreateInput{Name: "first"})
	second, _ := repo.Create(CreateInput{Name: "second"})
	third, _ := repo.Create(CreateInput{Name: "third"})

	if err := repo.Reorder([]string{third.ID, first.ID, second.ID}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}

	list, err := repo.List("", false)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := make([]string, 0, len(list))
	for _, m := range list {
		got = append(got, m.Name)
	}
	if want := []string{"third", "first", "second"}; !slices.Equal(got, want) {
		t.Errorf("order after Reorder() = %v, want %v", got, want)
	}
}

// A reorder naming an id that does not exist is a caller bug, and applying
// half of it would leave the list in an order nobody asked for.
func TestRepo_ReorderIsAllOrNothing(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	first, _ := repo.Create(CreateInput{Name: "first"})
	second, _ := repo.Create(CreateInput{Name: "second"})

	before, _ := repo.List("", false)

	err := repo.Reorder([]string{second.ID, first.ID, "aaaaaaaaaaaaaaaaaaaaaaaa"})
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("Reorder() with an unknown id error = %v, want ErrMemberNotFound", err)
	}

	after, _ := repo.List("", false)
	for i := range before {
		if before[i].ID != after[i].ID || before[i].DisplayOrder != after[i].DisplayOrder {
			t.Fatalf("a failed Reorder() changed the list: %+v -> %+v", before, after)
		}
	}
}
