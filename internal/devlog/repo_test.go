package devlog

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

// pinToday fixes the stamped publication date so assertions are not a race
// against midnight.
func pinToday(t *testing.T, date string) {
	t.Helper()
	original := today
	today = func() string { return date }
	t.Cleanup(func() { today = original })
}

func TestRepo_CreateGeneratesASlug(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	post, err := repo.Create(CreateInput{Title: "Kartuş Sistemi Geldi"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !id.IsValid(post.ID) {
		t.Errorf("Create() id = %q, which is not a well-formed id", post.ID)
	}
	if post.Slug != "kartus-sistemi-geldi" {
		t.Errorf("Slug = %q, want %q", post.Slug, "kartus-sistemi-geldi")
	}
	if post.IsPublished {
		t.Error("a new post should start unpublished")
	}
	// Nothing to publish yet, so no date should have been invented.
	if post.PublishedAt != "" {
		t.Errorf("PublishedAt = %q, want empty for a draft", post.PublishedAt)
	}
}

func TestRepo_CreateGivesCollidingTitlesDistinctSlugs(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	first, _ := repo.Create(CreateInput{Title: "Devlog"})
	second, _ := repo.Create(CreateInput{Title: "Devlog"})

	if first.Slug == second.Slug {
		t.Fatalf("both posts got the slug %q", first.Slug)
	}
	if second.Slug != "devlog-2" {
		t.Errorf("second slug = %q, want %q", second.Slug, "devlog-2")
	}
}

func TestRepo_PublishedAtIsStampedOnFirstPublish(t *testing.T) {
	pinToday(t, "2026-08-12")
	repo := NewRepo(setupTestDB(t))

	post, _ := repo.Create(CreateInput{Title: "Draft"})
	if post.PublishedAt != "" {
		t.Fatalf("PublishedAt = %q, want empty", post.PublishedAt)
	}

	published, err := repo.Update(post.ID, UpdateInput{Title: "Draft", IsPublished: true})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if published.PublishedAt != "2026-08-12" {
		t.Errorf("PublishedAt = %q, want today's date", published.PublishedAt)
	}
}

// The date is editable after the fact, which is what makes back-dating an
// imported post possible.
func TestRepo_PublishedAtCanBeOverridden(t *testing.T) {
	pinToday(t, "2026-08-12")
	repo := NewRepo(setupTestDB(t))

	post, _ := repo.Create(CreateInput{Title: "Draft"})
	updated, err := repo.Update(post.ID, UpdateInput{
		Title: "Draft", IsPublished: true, PublishedAt: "2024-01-05",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.PublishedAt != "2024-01-05" {
		t.Errorf("PublishedAt = %q, want the supplied date", updated.PublishedAt)
	}
}

// Re-saving a published post must not re-date it, and unpublishing must not
// throw the date away -- re-publishing should not silently move the post to
// today.
func TestRepo_PublishedAtSurvivesLaterSaves(t *testing.T) {
	pinToday(t, "2026-08-12")
	repo := NewRepo(setupTestDB(t))

	post, _ := repo.Create(CreateInput{Title: "Draft"})
	published, _ := repo.Update(post.ID, UpdateInput{Title: "Draft", IsPublished: true})

	pinToday(t, "2027-01-01")

	resaved, _ := repo.Update(post.ID, UpdateInput{Title: "Draft edited", IsPublished: true})
	if resaved.PublishedAt != published.PublishedAt {
		t.Errorf("PublishedAt changed on re-save: %q -> %q", published.PublishedAt, resaved.PublishedAt)
	}

	unpublished, _ := repo.Update(post.ID, UpdateInput{Title: "Draft edited", IsPublished: false})
	if unpublished.PublishedAt != published.PublishedAt {
		t.Errorf("unpublishing cleared PublishedAt: %q", unpublished.PublishedAt)
	}

	republished, _ := repo.Update(post.ID, UpdateInput{Title: "Draft edited", IsPublished: true})
	if republished.PublishedAt != published.PublishedAt {
		t.Errorf("re-publishing re-dated the post: %q -> %q", published.PublishedAt, republished.PublishedAt)
	}
}

func TestRepo_SlugFollowsTheTitleWhileUnpublished(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	post, _ := repo.Create(CreateInput{Title: "First Draft"})
	renamed, err := repo.Update(post.ID, UpdateInput{Title: "Second Draft"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if renamed.Slug != "second-draft" {
		t.Errorf("Slug = %q, want it to follow the title while unpublished", renamed.Slug)
	}
}

// Once a post is public its slug is a URL people may have linked to, so
// renaming the post must not move it.
func TestRepo_SlugIsFrozenOncePublished(t *testing.T) {
	pinToday(t, "2026-08-12")
	repo := NewRepo(setupTestDB(t))

	post, _ := repo.Create(CreateInput{Title: "Launch Day"})
	published, _ := repo.Update(post.ID, UpdateInput{Title: "Launch Day", IsPublished: true})

	renamed, err := repo.Update(post.ID, UpdateInput{
		Title: "Launch Day (updated)", IsPublished: true,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if renamed.Slug != published.Slug {
		t.Errorf("Slug moved after publication: %q -> %q", published.Slug, renamed.Slug)
	}
	if renamed.Title != "Launch Day (updated)" {
		t.Errorf("Title = %q, want the rename to still apply", renamed.Title)
	}
}

func TestRepo_FindBySlug(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	post, _ := repo.Create(CreateInput{Title: "Hello World"})

	found, err := repo.FindBySlug("hello-world")
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}
	if found.ID != post.ID {
		t.Errorf("FindBySlug() id = %q, want %q", found.ID, post.ID)
	}
	if _, err := repo.FindBySlug("nope"); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("FindBySlug() error = %v, want ErrPostNotFound", err)
	}
}

func TestRepo_MissingPostIsReported(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	missing := "aaaaaaaaaaaaaaaaaaaaaaaa"

	if _, err := repo.FindByID(missing); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("FindByID() error = %v, want ErrPostNotFound", err)
	}
	if _, err := repo.Update(missing, UpdateInput{Title: "x"}); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("Update() error = %v, want ErrPostNotFound", err)
	}
	if err := repo.Delete(missing); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("Delete() error = %v, want ErrPostNotFound", err)
	}
}

func TestRepo_ListSorting(t *testing.T) {
	pinToday(t, "2026-08-12")
	repo := NewRepo(setupTestDB(t))

	older, _ := repo.Create(CreateInput{Title: "older"})
	repo.Update(older.ID, UpdateInput{Title: "older", IsPublished: true, PublishedAt: "2025-01-01"})

	newer, _ := repo.Create(CreateInput{Title: "newer"})
	repo.Update(newer.ID, UpdateInput{Title: "newer", IsPublished: true, PublishedAt: "2026-06-01"})

	// A draft has no published_at at all.
	repo.Create(CreateInput{Title: "draft"})

	titlesFor := func(t *testing.T, field string, descending bool) []string {
		t.Helper()
		list, err := repo.List(field, descending)
		if err != nil {
			t.Fatalf("List(%q, %v) error = %v", field, descending, err)
		}
		titles := make([]string, 0, len(list))
		for _, p := range list {
			titles = append(titles, p.Title)
		}
		return titles
	}

	// The draft was created today, so falling back to its creation date puts
	// it above both published posts rather than sinking it to the bottom.
	if got, want := titlesFor(t, "", false), []string{"draft", "newer", "older"}; !slices.Equal(got, want) {
		t.Errorf("default order = %v, want %v", got, want)
	}
	if got, want := titlesFor(t, "title", false), []string{"draft", "newer", "older"}; !slices.Equal(got, want) {
		t.Errorf("title asc = %v, want %v", got, want)
	}

	if _, err := repo.List("title; DROP TABLE devlog_posts", false); !errors.Is(err, ErrInvalidSort) {
		t.Errorf("List() with an injected field error = %v, want ErrInvalidSort", err)
	}
}
