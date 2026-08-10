# Games Module Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Games module's data access layer and admin REST API — CRUD for games, screenshot management, and wiring Plan B's archive-upload handler so a successful game-build upload actually updates `games.web_export_path` and triggers a regen. This is the first of several per-module backend plans (Games is first because Play, Landing's sales/portfolio sections, and Devlog/Awards' optional game links all depend on it).

**Architecture:** A `internal/games` repository package (hand-written SQL over the `games`/`game_screenshots` tables from Plan A), a `internal/gamesapi` package exposing that repository over `/api/admin/games*` JSON endpoints behind Plan A's `RequireSession` middleware, and a small amount of `internal/httpserver` wiring that finally mounts Plan B's `internal/gameupload` handler (built in Plan B but never mounted) with a real callback that updates the game record and enqueues a regen job via Plan B's `render.EnqueueRegen`.

**Tech Stack:** Go 1.22+ only — no new dependencies. Builds entirely on Plan A (`internal/db`, `internal/auth`, `internal/adminapi`, `internal/httpapi`) and Plan B (`internal/render`, `internal/gameupload`, `internal/storage`).

**Depends on:** `docs/superpowers/plans/2026-08-10-backend-core-data-model.md` (Plan A) and `docs/superpowers/plans/2026-08-10-content-rendering-pipelines.md` (Plan B) — both assumed implemented and passing. Modifies the post-Plan-C state of `internal/httpserver/router.go` (adds `Games`, `DB` to `Dependencies`) and `cmd/server/main.go`.

## Global Constraints

- Never use Go's `any` type alias anywhere — use `interface{}` or a concrete type instead (user's global CLAUDE.md rule).
- API error responses use `{"error": {"code": "...", "message": "..."}}` via `internal/httpapi.WriteError` (Plan A).
- Every admin endpoint in this plan is mounted behind `adminapi.RequireSession` (Plan A Task 9) — no unauthenticated write access to game data.
- A game's `slug` is generated once from its title at creation and is **immutable** thereafter (per the architecture spec) — `Update` never touches `slug`.
- Any create/update/delete/screenshot change that affects what a rendered page shows must enqueue a regen job via `render.EnqueueRegen` (Plan B) for the affected tag(s) — never mutate game data silently.
- Git commits in this repo: self-committed, one-sentence semantic messages, no co-author trailer.

## Scope

This plan covers the **backend only**: repository, REST API, and archive-upload wiring. Explicitly out of scope, for later plans:
- The Play page's public HTML/CSS rendering (skeuomorphic TV/console/cartridge/CD design from `2026-08-10-public-site-visual-design.md`) — a future "Play page rendering" plan that registers a `render.Renderer` consuming this plan's repo.
- The admin SPA's Games CRUD screens (list table, create/edit forms, image-upload UI, archive-upload UI, screenshot gallery manager) — a future "Games admin UI" plan built on Plan C's shell and this plan's API.
- Members, Devlog, Awards, Contact, Homepage/Site-Settings modules — separate plans, queued after this one.

---

## File Structure

```
internal/
  games/
    repo.go              # Game struct, Repo, Create/FindByID/FindBySlug/Update/Delete
    repo_test.go
    slug.go               # Slugify + uniqueSlug
    slug_test.go
    screenshots.go         # Screenshot struct, AddScreenshot/ListScreenshots/RemoveScreenshot
    screenshots_test.go
  gamesapi/
    handlers.go            # List/Create/Get/Update/Delete handlers, gameResponse
    handlers_test.go
    screenshot_handlers.go # AddScreenshot/RemoveScreenshot handlers
    screenshot_handlers_test.go
  httpserver/
    router.go               # (modify) mount games + screenshot + game-upload routes
    router_test.go           # (modify) end-to-end games CRUD assertions
cmd/
  server/
    main.go                  # (modify) construct games.Repo, wire into Dependencies
```

`internal/games` owns persistence, `internal/gamesapi` owns the HTTP contract — the same separation Plan A used for `internal/auth` vs `internal/adminapi`.

---

### Task 1: Games repository — CRUD and slug generation

**Files:**
- Create: `internal/games/repo.go`
- Create: `internal/games/repo_test.go`
- Create: `internal/games/slug.go`
- Create: `internal/games/slug_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate` (Plan A Task 2/3)
- Produces: `games.Game{ID int64, Slug, Title, ShortDescription, FullDescription, Tags string, IsBrowserPlayable, IsDownloadable, IsForSale bool, PriceDisplay, ExternalLinksJSON string, CartridgeArtID, CDCoverArtID, OGImageID *int64, WebExportPath string, DisplayOrder int, IsPublished bool, CreatedAt, UpdatedAt time.Time}`, `games.ErrGameNotFound`, `games.NewRepo(db *sql.DB) *Repo`, `(*Repo).Create(input CreateInput) (Game, error)`, `(*Repo).FindByID(id int64) (Game, error)`, `(*Repo).FindBySlug(slug string) (Game, error)`, `(*Repo).Update(id int64, input UpdateInput) (Game, error)`, `(*Repo).Delete(id int64) error`, `(*Repo).SetWebExportPath(id int64, path string) error`, `games.Slugify(title string) string`

- [ ] **Step 1: Write the failing tests for slug generation**

`internal/games/slug_test.go`:

```go
package games

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Pixel Quest", "pixel-quest"},
		{"  Leading and Trailing  ", "leading-and-trailing"},
		{"Weird!!! Chars??? 123", "weird-chars-123"},
		{"", "game"},
	}
	for _, c := range cases {
		if got := Slugify(c.title); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/games/... -v -run TestSlugify`
Expected: FAIL — package `games` does not exist yet.

- [ ] **Step 3: Implement `Slugify`**

`internal/games/slug.go`:

```go
package games

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(title string) string {
	lower := strings.ToLower(title)
	slug := slugInvalidChars.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "game"
	}
	return slug
}

func uniqueSlug(db *sql.DB, base string) (string, error) {
	candidate := base
	for suffix := 2; ; suffix++ {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM games WHERE slug = ?;`, candidate).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/games/... -v -run TestSlugify`
Expected: PASS

- [ ] **Step 5: Write the failing tests for the repository**

`internal/games/repo_test.go`:

```go
package games

import (
	"errors"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	return nil // replaced below — see note
}
```

Write the real helper directly (no placeholder):

`internal/games/repo_test.go`:

```go
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
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/games/... -v`
Expected: FAIL — `NewRepo`/`Game`/`CreateInput`/etc. undefined.

- [ ] **Step 7: Implement the repository**

`internal/games/repo.go`:

```go
package games

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrGameNotFound = errors.New("game not found")

type Game struct {
	ID                int64
	Slug              string
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsBrowserPlayable bool
	IsDownloadable    bool
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	CartridgeArtID    *int64
	CDCoverArtID      *int64
	OGImageID         *int64
	WebExportPath     string
	DisplayOrder      int
	IsPublished       bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateInput struct {
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsBrowserPlayable bool
	IsDownloadable    bool
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	DisplayOrder      int
	IsPublished       bool
}

type UpdateInput struct {
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsBrowserPlayable bool
	IsDownloadable    bool
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	CartridgeArtID    *int64
	CDCoverArtID      *int64
	OGImageID         *int64
	DisplayOrder      int
	IsPublished       bool
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const gameColumns = `id, slug, title, short_description, full_description, tags,
	is_browser_playable, is_downloadable, is_for_sale,
	price_display, external_links_json,
	cartridge_art_id, cd_cover_art_id, og_image_id,
	web_export_path, display_order, is_published, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanGame(row rowScanner) (Game, error) {
	var g Game
	var priceDisplay, webExportPath sql.NullString
	var cartridgeArtID, cdCoverArtID, ogImageID sql.NullInt64
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&g.ID, &g.Slug, &g.Title, &g.ShortDescription, &g.FullDescription, &g.Tags,
		&g.IsBrowserPlayable, &g.IsDownloadable, &g.IsForSale,
		&priceDisplay, &g.ExternalLinksJSON,
		&cartridgeArtID, &cdCoverArtID, &ogImageID,
		&webExportPath, &g.DisplayOrder, &g.IsPublished,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrGameNotFound
	}
	if err != nil {
		return Game{}, err
	}

	if priceDisplay.Valid {
		g.PriceDisplay = priceDisplay.String
	}
	if webExportPath.Valid {
		g.WebExportPath = webExportPath.String
	}
	if cartridgeArtID.Valid {
		id := cartridgeArtID.Int64
		g.CartridgeArtID = &id
	}
	if cdCoverArtID.Valid {
		id := cdCoverArtID.Int64
		g.CDCoverArtID = &id
	}
	if ogImageID.Valid {
		id := ogImageID.Int64
		g.OGImageID = &id
	}

	createdAt, err := parseTimestamp(createdAtStr)
	if err != nil {
		return Game{}, err
	}
	updatedAt, err := parseTimestamp(updatedAtStr)
	if err != nil {
		return Game{}, err
	}
	g.CreatedAt = createdAt
	g.UpdatedAt = updatedAt

	return g, nil
}

func parseTimestamp(s string) (time.Time, error) {
	normalized := s
	if i := strings.Index(s, "."); i != -1 {
		normalized = s[:i] + "Z"
	}
	return time.Parse(time.RFC3339, normalized)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func (r *Repo) Create(input CreateInput) (Game, error) {
	slug, err := uniqueSlug(r.db, Slugify(input.Title))
	if err != nil {
		return Game{}, err
	}
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}

	res, err := r.db.Exec(
		`INSERT INTO games (
			slug, title, short_description, full_description, tags,
			is_browser_playable, is_downloadable, is_for_sale,
			price_display, external_links_json, display_order, is_published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		slug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks, input.DisplayOrder, input.IsPublished,
	)
	if err != nil {
		return Game{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Game{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) FindByID(id int64) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE id = ?;`, id)
	return scanGame(row)
}

func (r *Repo) FindBySlug(slug string) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE slug = ?;`, slug)
	return scanGame(row)
}

func (r *Repo) Update(id int64, input UpdateInput) (Game, error) {
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}

	res, err := r.db.Exec(
		`UPDATE games SET
			title = ?, short_description = ?, full_description = ?, tags = ?,
			is_browser_playable = ?, is_downloadable = ?, is_for_sale = ?,
			price_display = ?, external_links_json = ?,
			cartridge_art_id = ?, cd_cover_art_id = ?, og_image_id = ?,
			display_order = ?, is_published = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks,
		nullableInt64(input.CartridgeArtID), nullableInt64(input.CDCoverArtID), nullableInt64(input.OGImageID),
		input.DisplayOrder, input.IsPublished, id,
	)
	if err != nil {
		return Game{}, err
	}
	if err := requireRowsAffected(res); err != nil {
		return Game{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM games WHERE id = ?;`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}

func (r *Repo) SetWebExportPath(id int64, path string) error {
	res, err := r.db.Exec(
		`UPDATE games SET web_export_path = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
		path, id,
	)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}

func requireRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGameNotFound
	}
	return nil
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/games/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/games/repo.go internal/games/repo_test.go internal/games/slug.go internal/games/slug_test.go
git commit -m "feat: add games repository with slug generation"
```

---

### Task 2: List and screenshot management

**Files:**
- Modify: `internal/games/repo.go` (add `List`)
- Modify: `internal/games/repo_test.go` (add `TestRepo_ListOrdersByDisplayOrder`)
- Create: `internal/games/screenshots.go`
- Create: `internal/games/screenshots_test.go`

**Interfaces:**
- Consumes: `Repo` (Task 1)
- Produces: `(*Repo).List() ([]Game, error)`, `games.Screenshot{ID, GameID, MediaID int64, DisplayOrder int}`, `(*Repo).AddScreenshot(gameID, mediaID int64, displayOrder int) (Screenshot, error)`, `(*Repo).ListScreenshots(gameID int64) ([]Screenshot, error)`, `(*Repo).RemoveScreenshot(screenshotID int64) error`

- [ ] **Step 1: Write the failing test for `List`**

Add to `internal/games/repo_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/games/... -v -run TestRepo_ListOrdersByDisplayOrder`
Expected: FAIL — `List` undefined.

- [ ] **Step 3: Implement `List`**

Add to `internal/games/repo.go`:

```go
func (r *Repo) List() ([]Game, error) {
	rows, err := r.db.Query(`SELECT ` + gameColumns + ` FROM games ORDER BY display_order ASC, id ASC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Game
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/games/... -v -run TestRepo_ListOrdersByDisplayOrder`
Expected: PASS

- [ ] **Step 5: Write the failing tests for screenshot management**

`internal/games/screenshots_test.go`:

```go
package games

import "testing"

func TestRepo_AddListRemoveScreenshot(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
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

	if err := repo.RemoveScreenshot(first.ID); err != nil {
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

func TestRepo_DeleteGameCascadesScreenshots(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	game, err := repo.Create(CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
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
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/games/... -v -run Screenshot`
Expected: FAIL — `AddScreenshot`/`ListScreenshots`/`RemoveScreenshot` undefined.

- [ ] **Step 7: Implement screenshot management**

`internal/games/screenshots.go`:

```go
package games

type Screenshot struct {
	ID           int64
	GameID       int64
	MediaID      int64
	DisplayOrder int
}

func (r *Repo) AddScreenshot(gameID, mediaID int64, displayOrder int) (Screenshot, error) {
	res, err := r.db.Exec(
		`INSERT INTO game_screenshots (game_id, media_id, display_order) VALUES (?, ?, ?);`,
		gameID, mediaID, displayOrder,
	)
	if err != nil {
		return Screenshot{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Screenshot{}, err
	}
	return Screenshot{ID: id, GameID: gameID, MediaID: mediaID, DisplayOrder: displayOrder}, nil
}

func (r *Repo) ListScreenshots(gameID int64) ([]Screenshot, error) {
	rows, err := r.db.Query(
		`SELECT id, game_id, media_id, display_order FROM game_screenshots
		 WHERE game_id = ? ORDER BY display_order ASC, id ASC;`,
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Screenshot
	for rows.Next() {
		var s Screenshot
		if err := rows.Scan(&s.ID, &s.GameID, &s.MediaID, &s.DisplayOrder); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *Repo) RemoveScreenshot(screenshotID int64) error {
	_, err := r.db.Exec(`DELETE FROM game_screenshots WHERE id = ?;`, screenshotID)
	return err
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/games/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/games
git commit -m "feat: add games list and screenshot management"
```

---

### Task 3: Admin REST handlers — List and Create

**Files:**
- Create: `internal/gamesapi/handlers.go`
- Create: `internal/gamesapi/handlers_test.go`

**Interfaces:**
- Consumes: `games.Repo`, `games.CreateInput` (Task 1/2), `render.EnqueueRegen` (Plan B Task 10), `httpapi.WriteJSON`/`WriteError` (Plan A Task 9)
- Produces: `gamesapi.NewHandlers(repo *games.Repo, db *sql.DB) *Handlers`, `(*Handlers).List`, `(*Handlers).Create` (both `http.HandlerFunc`-shaped)

- [ ] **Step 1: Write the failing tests**

`internal/gamesapi/handlers_test.go`:

```go
package gamesapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/games"
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

func TestList_ReturnsAllGames(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got []gameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Pixel Quest" {
		t.Errorf("List() = %+v, want one game titled Pixel Quest", got)
	}
}

func TestCreate_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]interface{}{
		"title":               "Pixel Quest",
		"short_description":   "A tiny adventure.",
		"is_browser_playable": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got gameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Slug != "pixel-quest" {
		t.Errorf("Slug = %q, want %q", got.Slug, "pixel-quest")
	}
	if !got.IsBrowserPlayable {
		t.Error("IsBrowserPlayable = false, want true")
	}

	var jobCount int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = 'game:list';`).Scan(&jobCount); err != nil {
		t.Fatalf("query regen_jobs: %v", err)
	}
	if jobCount != 1 {
		t.Errorf("regen_jobs count for tag game:list = %d, want 1", jobCount)
	}
}

func TestCreate_MissingTitle(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]string{"title": "  "})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gamesapi/... -v`
Expected: FAIL — package `gamesapi` does not exist yet.

- [ ] **Step 3: Implement the handlers**

`internal/gamesapi/handlers.go`:

```go
package gamesapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
	"pixabros/internal/render"
)

type Handlers struct {
	repo *games.Repo
	db   *sql.DB
}

func NewHandlers(repo *games.Repo, db *sql.DB) *Handlers {
	return &Handlers{repo: repo, db: db}
}

type gameResponse struct {
	ID                int64  `json:"id"`
	Slug              string `json:"slug"`
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	CartridgeArtID    *int64 `json:"cartridge_art_id"`
	CDCoverArtID      *int64 `json:"cd_cover_art_id"`
	OGImageID         *int64 `json:"og_image_id"`
	WebExportPath     string `json:"web_export_path"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func toGameResponse(g games.Game) gameResponse {
	return gameResponse{
		ID:                g.ID,
		Slug:              g.Slug,
		Title:             g.Title,
		ShortDescription:  g.ShortDescription,
		FullDescription:   g.FullDescription,
		Tags:              g.Tags,
		IsBrowserPlayable: g.IsBrowserPlayable,
		IsDownloadable:    g.IsDownloadable,
		IsForSale:         g.IsForSale,
		PriceDisplay:      g.PriceDisplay,
		ExternalLinksJSON: g.ExternalLinksJSON,
		CartridgeArtID:    g.CartridgeArtID,
		CDCoverArtID:      g.CDCoverArtID,
		OGImageID:         g.OGImageID,
		WebExportPath:     g.WebExportPath,
		DisplayOrder:      g.DisplayOrder,
		IsPublished:       g.IsPublished,
		CreatedAt:         g.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         g.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list games")
		return
	}
	responses := make([]gameResponse, 0, len(list))
	for _, g := range list {
		responses = append(responses, toGameResponse(g))
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}

type createRequest struct {
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return
	}

	game, err := h.repo.Create(games.CreateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		IsBrowserPlayable: req.IsBrowserPlayable,
		IsDownloadable:    req.IsDownloadable,
		IsForSale:         req.IsForSale,
		PriceDisplay:      req.PriceDisplay,
		ExternalLinksJSON: req.ExternalLinksJSON,
		DisplayOrder:      req.DisplayOrder,
		IsPublished:       req.IsPublished,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create game")
		return
	}

	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, toGameResponse(game))
}

func parseIDPathValue(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
```

`parseIDPathValue` is shared by `Get`/`Update`/`Delete` in Task 4.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gamesapi/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gamesapi/handlers.go internal/gamesapi/handlers_test.go
git commit -m "feat: add games list and create admin endpoints"
```

---

### Task 4: Admin REST handlers — Get, Update, Delete

**Files:**
- Modify: `internal/gamesapi/handlers.go`
- Modify: `internal/gamesapi/handlers_test.go`

**Interfaces:**
- Consumes: `games.Repo.FindByID`, `.Update`, `.Delete`, `games.ErrGameNotFound` (Task 1), `parseIDPathValue` (Task 3)
- Produces: `(*Handlers).Get`, `(*Handlers).Update`, `(*Handlers).Delete`

- [ ] **Step 1: Write the failing tests**

Append to `internal/gamesapi/handlers_test.go`:

```go
func TestGet_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/1", nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGet_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/games/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdate_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]interface{}{
		"title":        "Pixel Quest: Remastered",
		"is_published": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/"+fmt.Sprintf("%d", game.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got gameResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Title != "Pixel Quest: Remastered" || got.Slug != "pixel-quest" {
		t.Errorf("Update() = %+v, want Title changed and Slug unchanged", got)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%d = %d, want 1", game.ID, jobCount)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	body, _ := json.Marshal(map[string]string{"title": "x"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/games/999", bytes.NewReader(body))
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDelete_Success(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, _ := repo.Create(games.CreateInput{Title: "Pixel Quest"})
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/"+fmt.Sprintf("%d", game.ID), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", game.ID))
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, err := repo.FindByID(game.ID); err == nil {
		t.Error("game should be deleted")
	}
}

func TestDelete_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	handlers := NewHandlers(repo, conn)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/games/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
```

Add `"fmt"` to the test file's import block.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gamesapi/... -v -run "TestGet|TestUpdate|TestDelete"`
Expected: FAIL — `Get`/`Update`/`Delete` undefined.

- [ ] **Step 3: Implement the handlers**

Add `Get`, `Update`, and `Delete` to `internal/gamesapi/handlers.go`, right after the `parseIDPathValue` helper added in Task 3:

```go
func parseIDPathValue(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	game, err := h.repo.FindByID(id)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toGameResponse(game))
}

type updateRequest struct {
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	CartridgeArtID    *int64 `json:"cartridge_art_id"`
	CDCoverArtID      *int64 `json:"cd_cover_art_id"`
	OGImageID         *int64 `json:"og_image_id"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return
	}

	game, err := h.repo.Update(id, games.UpdateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		IsBrowserPlayable: req.IsBrowserPlayable,
		IsDownloadable:    req.IsDownloadable,
		IsForSale:         req.IsForSale,
		PriceDisplay:      req.PriceDisplay,
		ExternalLinksJSON: req.ExternalLinksJSON,
		CartridgeArtID:    req.CartridgeArtID,
		CDCoverArtID:      req.CDCoverArtID,
		OGImageID:         req.OGImageID,
		DisplayOrder:      req.DisplayOrder,
		IsPublished:       req.IsPublished,
	})
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update game")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%d", id)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, toGameResponse(game))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	err = h.repo.Delete(id)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete game")
		return
	}

	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

Add `"errors"` and `"fmt"` to `internal/gamesapi/handlers.go`'s import block (both are newly needed here: `errors.Is` for `games.ErrGameNotFound` checks, `fmt.Sprintf` for the `game:{id}` regen tag).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gamesapi/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gamesapi/handlers.go internal/gamesapi/handlers_test.go
git commit -m "feat: add games get, update, and delete admin endpoints"
```

---

### Task 5: Screenshot REST handlers

**Files:**
- Create: `internal/gamesapi/screenshot_handlers.go`
- Create: `internal/gamesapi/screenshot_handlers_test.go`

**Interfaces:**
- Consumes: `games.Repo.AddScreenshot`, `.RemoveScreenshot` (Task 2), `parseIDPathValue` (Task 3), `render.EnqueueRegen` (Plan B Task 10)
- Produces: `(*Handlers).AddScreenshot`, `(*Handlers).RemoveScreenshot`

- [ ] **Step 1: Write the failing tests**

`internal/gamesapi/screenshot_handlers_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gamesapi/... -v -run Screenshot`
Expected: FAIL — `AddScreenshot`/`RemoveScreenshot`/`screenshotResponse` undefined.

- [ ] **Step 3: Implement the handlers**

`internal/gamesapi/screenshot_handlers.go`:

```go
package gamesapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"pixabros/internal/httpapi"
	"pixabros/internal/render"
)

type screenshotResponse struct {
	ID           int64 `json:"id"`
	GameID       int64 `json:"game_id"`
	MediaID      int64 `json:"media_id"`
	DisplayOrder int   `json:"display_order"`
}

type addScreenshotRequest struct {
	MediaID      int64 `json:"media_id"`
	DisplayOrder int   `json:"display_order"`
}

func (h *Handlers) AddScreenshot(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var req addScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.MediaID == 0 {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "media_id is required")
		return
	}

	screenshot, err := h.repo.AddScreenshot(gameID, req.MediaID, req.DisplayOrder)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not add screenshot")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%d", gameID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, screenshotResponse{
		ID:           screenshot.ID,
		GameID:       screenshot.GameID,
		MediaID:      screenshot.MediaID,
		DisplayOrder: screenshot.DisplayOrder,
	})
}

func (h *Handlers) RemoveScreenshot(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}
	screenshotID, err := strconv.ParseInt(r.PathValue("screenshotID"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "screenshotID must be a number")
		return
	}

	if err := h.repo.RemoveScreenshot(screenshotID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not remove screenshot")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%d", gameID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gamesapi/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gamesapi/screenshot_handlers.go internal/gamesapi/screenshot_handlers_test.go
git commit -m "feat: add game screenshot admin endpoints"
```

---

### Task 6: Wire the game archive upload endpoint

Plan B built `internal/gameupload` (archive extraction + handler) but never mounted it in the router. This task mounts it with a real callback that updates the game record and triggers a regen — the piece Plan B explicitly deferred.

**Files:**
- Modify: `internal/httpserver/router.go`
- Modify: `internal/httpserver/router_test.go`

**Interfaces:**
- Consumes: `games.Repo.FindBySlug`, `.SetWebExportPath` (Task 1), `gameupload.NewHandler` (Plan B Task 8), `render.EnqueueRegen` (Plan B Task 10)
- Produces: `POST /api/admin/games/{slug}/upload` mounted and functional

- [ ] **Step 1: Write the failing test**

Add to `internal/httpserver/router_test.go` a new test function (the existing `TestRouter_LoginAndSingleOriginServing` is left untouched here — this is additive):

```go
func TestRouter_GameArchiveUpload(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	hash, err := auth.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash)

	gamesRepo := games.NewRepo(conn)
	game, err := gamesRepo.Create(games.CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("games.Create() error = %v", err)
	}

	playDir := t.TempDir()
	renderedFiles := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, renderedFiles)

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		DB:         conn,
		Games:      gamesRepo,
		AdminUIDir: t.TempDir(),
		PlayDir:    playDir,
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	loginResp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error = %v", err)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f, _ := zw.Create("index.html")
	f.Write([]byte("<html></html>"))
	zw.Close()

	var multipartBuf bytes.Buffer
	mw := multipart.NewWriter(&multipartBuf)
	part, _ := mw.CreateFormFile("file", "build.zip")
	part.Write(zipBuf.Bytes())
	mw.Close()

	uploadURL := fmt.Sprintf("%s/api/admin/games/%s/upload", srv.URL, game.Slug)
	uploadReq, _ := http.NewRequest(http.MethodPost, uploadURL, &multipartBuf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	for _, c := range loginResp.Cookies() {
		uploadReq.AddCookie(c)
	}

	uploadResp, err := srv.Client().Do(uploadReq)
	if err != nil {
		t.Fatalf("upload request error = %v", err)
	}
	if uploadResp.StatusCode != http.StatusNoContent {
		t.Fatalf("upload status = %d, want %d", uploadResp.StatusCode, http.StatusNoContent)
	}

	updated, err := gamesRepo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	wantPath := filepath.Join(playDir, game.Slug)
	if updated.WebExportPath != wantPath {
		t.Errorf("WebExportPath = %q, want %q", updated.WebExportPath, wantPath)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%d = %d, want 1", game.ID, jobCount)
	}
}
```

Add `"archive/zip"`, `"mime/multipart"`, `"fmt"`, `"pixabros/internal/games"` to `internal/httpserver/router_test.go`'s import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpserver/... -v -run TestRouter_GameArchiveUpload`
Expected: FAIL — `Dependencies` has no `Games` field, route not mounted, or `New` signature mismatch.

- [ ] **Step 3: Wire the route**

Modify `internal/httpserver/router.go`: add `DB *sql.DB` and `Games *games.Repo` to `Dependencies`, then mount the upload route:

```go
type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	DB         *sql.DB
	Games      *games.Repo
	AdminUIDir string
	PlayDir    string
	AssetsDir  string
}
```

Add this block to `New(deps Dependencies)`, after the existing `/api/admin/*` routes and before the static-file mounts:

```go
	onGameArchiveExtracted := func(slug string) error {
		game, err := deps.Games.FindBySlug(slug)
		if err != nil {
			return err
		}
		if err := deps.Games.SetWebExportPath(game.ID, filepath.Join(deps.PlayDir, slug)); err != nil {
			return err
		}
		return render.EnqueueRegen(deps.DB, fmt.Sprintf("game:%d", game.ID))
	}
	gameUploadHandler := gameupload.NewHandler(deps.PlayDir, onGameArchiveExtracted)
	mux.HandleFunc("POST /api/admin/games/{slug}/upload", adminapi.RequireSession(deps.Sessions, gameUploadHandler.Upload))
```

Add `"database/sql"`, `"fmt"`, `"path/filepath"`, `"pixabros/internal/games"`, `"pixabros/internal/gameupload"` to the import block.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpserver/... -v -run TestRouter_GameArchiveUpload`
Expected: PASS

- [ ] **Step 5: Run the full httpserver test suite**

Run: `go test ./internal/httpserver/... -v`
Expected: PASS (the pre-existing `TestRouter_LoginAndSingleOriginServing` must be updated to pass `DB: conn` and `Games: auth... ` — actually `Games: games.NewRepo(conn)` — in its `Dependencies` literal, since the struct gained two required-for-compilation fields; a zero-value `DB`/`Games` compiles fine in Go but leaves the new route non-functional for that test, which is acceptable since that test doesn't exercise game upload).

- [ ] **Step 6: Commit**

```bash
git add internal/httpserver
git commit -m "feat: wire game archive upload endpoint with web_export_path and regen"
```

---

### Task 7: Register games routes and update the server entrypoint

**Files:**
- Modify: `internal/httpserver/router.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `gamesapi.NewHandlers` (Task 3), `games.NewRepo` (Task 1)
- Produces: all `/api/admin/games*` routes mounted and reachable from a running server

- [ ] **Step 1: Mount the games and screenshot routes**

Modify `internal/httpserver/router.go`, adding this block right after the existing `whoami` route registration and before the game-archive-upload block from Task 6:

```go
	gamesHandlers := gamesapi.NewHandlers(deps.Games, deps.DB)
	mux.HandleFunc("GET /api/admin/games", adminapi.RequireSession(deps.Sessions, gamesHandlers.List))
	mux.HandleFunc("POST /api/admin/games", adminapi.RequireSession(deps.Sessions, gamesHandlers.Create))
	mux.HandleFunc("GET /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Get))
	mux.HandleFunc("PUT /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Delete))
	mux.HandleFunc("POST /api/admin/games/{id}/screenshots", adminapi.RequireSession(deps.Sessions, gamesHandlers.AddScreenshot))
	mux.HandleFunc("DELETE /api/admin/games/{id}/screenshots/{screenshotID}", adminapi.RequireSession(deps.Sessions, gamesHandlers.RemoveScreenshot))
```

Add `"pixabros/internal/gamesapi"` to the import block.

- [ ] **Step 2: Update the existing router test's `Dependencies` literals to compile**

In `internal/httpserver/router_test.go`, add `DB: conn,` and `Games: games.NewRepo(conn),` to the `Dependencies{...}` literal inside `TestRouter_LoginAndSingleOriginServing` (it already imports `"pixabros/internal/games"` after Task 6's changes).

- [ ] **Step 3: Run the full httpserver test suite**

Run: `go test ./internal/httpserver/... -v`
Expected: PASS

- [ ] **Step 4: Update the server entrypoint**

Modify `cmd/server/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/games"
	"pixabros/internal/httpserver"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	renderedFiles := storage.NewLocalDisk(cfg.DataDir+"/rendered-store", "/rendered")
	store := render.NewStore(conn, renderedFiles)
	registry := render.NewRegistry()

	worker := render.NewWorker(conn, registry, store, 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		DB:         conn,
		Games:      games.NewRepo(conn),
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		AssetsDir:  cfg.DataDir + "/assets",
	})

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
```

- [ ] **Step 5: Build and run the full backend test suite**

Run: `go build ./... && go test ./...`
Expected: builds cleanly, all packages PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/httpserver cmd/server
git commit -m "feat: mount games admin api routes and wire server entrypoint"
```

---

### Task 8: End-to-end verification

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: everything from Tasks 1–7

- [ ] **Step 1: Build and start the server**

```bash
go build ./...
mkdir -p data/admin-dist data/games data/rendered-store data/assets
go run ./cmd/admincli create-admin -username furkan -password "a-strong-password-1"
go run ./cmd/server &
sleep 1
curl -s -c /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"furkan","password":"a-strong-password-1"}' > /dev/null
```

- [ ] **Step 2: Create a game and verify the slug and defaults**

```bash
curl -s -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/games \
  -H 'Content-Type: application/json' \
  -d '{"title":"Pixel Quest","short_description":"A tiny adventure.","is_browser_playable":true}' | tee /tmp/pixabros-game.json
```

Expected: `200`-shaped JSON body with `"slug":"pixel-quest"`, `"is_browser_playable":true`, `"web_export_path":""`, `"cd_cover_art_id":null`.

- [ ] **Step 3: List, get, and update the game**

```bash
GAME_ID=$(python3 -c "import json;print(json.load(open('/tmp/pixabros-game.json'))['id'])")
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/$GAME_ID
curl -s -b /tmp/pixabros-cookies.txt -X PUT http://localhost:8080/api/admin/games/$GAME_ID \
  -H 'Content-Type: application/json' \
  -d '{"title":"Pixel Quest: Remastered","is_published":true,"cd_cover_art_id":1}'
```

Expected: list returns an array containing the game; get returns the same game; update returns `200` with `"title":"Pixel Quest: Remastered"`, `"slug":"pixel-quest"` (unchanged), `"cd_cover_art_id":1`.

- [ ] **Step 4: Add and remove a screenshot**

```bash
curl -s -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/games/$GAME_ID/screenshots \
  -H 'Content-Type: application/json' \
  -d '{"media_id":1,"display_order":0}' | tee /tmp/pixabros-screenshot.json
SCREENSHOT_ID=$(python3 -c "import json;print(json.load(open('/tmp/pixabros-screenshot.json'))['id'])")
curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/pixabros-cookies.txt -X DELETE \
  http://localhost:8080/api/admin/games/$GAME_ID/screenshots/$SCREENSHOT_ID
```

Expected: add returns `201` with the screenshot; delete returns `204`.

- [ ] **Step 5: Upload a game archive and verify `web_export_path`**

```bash
mkdir -p /tmp/pixabros-build && echo '<html></html>' > /tmp/pixabros-build/index.html
(cd /tmp/pixabros-build && zip -r /tmp/pixabros-build.zip index.html)
curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/pixabros-cookies.txt -X POST \
  -F "file=@/tmp/pixabros-build.zip" \
  http://localhost:8080/api/admin/games/pixel-quest/upload
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/$GAME_ID
curl -s http://localhost:8080/play/pixel-quest/index.html
```

Expected: upload returns `204`; the game's `web_export_path` is now `data/games/pixel-quest`; the last `curl` returns the uploaded `index.html` content, proving `/play/pixel-quest/*` serves the extracted build.

- [ ] **Step 6: Delete the game and confirm cleanup**

```bash
curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/pixabros-cookies.txt -X DELETE \
  http://localhost:8080/api/admin/games/$GAME_ID
curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/pixabros-cookies.txt \
  http://localhost:8080/api/admin/games/$GAME_ID
```

Expected: delete returns `204`; the follow-up get returns `404`.

- [ ] **Step 7: Stop the server and run the full test suite one last time**

```bash
kill %1
go test ./...
```

Expected: PASS for every package.

---

## Definition of Done

- `go build ./...` and `go test ./...` succeed across every package, including the new `internal/games` and `internal/gamesapi` packages and the modified `internal/httpserver`.
- A game's `slug` is generated once at creation from its title, is unique (auto-suffixed on collision), and is never altered by `Update`.
- Every create/update/delete/screenshot-change enqueues the correct `regen_jobs` row(s) (`game:{id}` and/or `game:list`) — verified both in unit tests and the end-to-end curl walkthrough.
- `POST /api/admin/games/{slug}/upload` (Plan B's handler, mounted for the first time in this plan) extracts the archive, updates `games.web_export_path`, enqueues a regen, and the game becomes servable at `/play/{slug}/*`.
- No `any` type appears anywhere in the Go source (`grep -rn '\bany\b' --include='*.go' .` returns nothing outside comments/strings).
