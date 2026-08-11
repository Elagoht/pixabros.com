# Games Admin UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the admin SPA's Games module — a games list, a shared create/edit form, screenshot management with image previews, artwork upload widgets, and a game-archive upload widget — all wired to the real, already-shipped `/api/admin/games*` backend. Adds the three backend pieces the UI needs and nothing more: mounting the existing (never-yet-mounted) `mediaapi.UploadHandler`, a new `GET /api/admin/media/{id}` lookup endpoint, a new `GET /api/admin/games/{id}/screenshots` listing endpoint, and a public `/media/` static route so uploaded images can actually be displayed.

**Architecture:** The Go backend stays exactly as-is apart from Task 1's additive endpoints — no existing handler signature, response shape, or semantic changes. On the frontend, `admin-ui/src/api/client.ts` grows typed wire functions (snake_case field names, because these *are* the wire format) plus one new multipart helper pair; `admin-ui/src/games/queries.ts` wraps those in TanStack Query hooks that own cache invalidation and `sonner` toasts; and `admin-ui/src/games/*.tsx` holds the screens. `GameForm` is one Formik + Yup form shared by the create and edit pages. The edit page owns the three artwork media ids as local state (the form owns every other field) and always submits the complete merged state, because `PUT /api/admin/games/{id}` is a full replace, not a patch. Two new hand-authored shadcn-style primitives (`Textarea`, `Checkbox`) join the existing three. The route tree replaces exactly one placeholder route (`games`) with three real ones; every other module placeholder is untouched.

**Architecture Decisions:**

1. **The edit page owns the artwork ids, not `GameForm`.** The three artwork widgets must sit outside the form's own field list (they upload immediately, on file selection, without submitting the game), yet their ids must ride along on the next PUT. Threading them through `GameForm` would require a render-prop or a `children` callback for one narrow case. Instead `GameFormValues` mirrors the backend's *create* body field-for-field, `CreateGamePage` submits it verbatim, and `EditGamePage` spreads it into the PUT body next to the three ids it holds in `useState`. `GameForm` stays a plain `initialValues`/`onSubmit`/`submitLabel` component with zero knowledge of media.
2. **Artwork upload sets form state only; it never PUTs the game.** A file selection uploads bytes and gets a media id back — that is a media-library operation, not a game edit. Saving is always an explicit form submit, so a user who uploads and then navigates away has changed nothing about the game. The now-unreferenced media row is exactly what `internal/media/orphansweep.go` exists to reap; that is deliberately not this plan's concern.
3. **`/media/` is served publicly with plain `noDirListing`, not behind `RequireSession` and not with immutable cache headers.** Uploaded images are public site content (cartridge art, screenshots, OG images all appear on the public MPA), so gating them on an admin session would break the public site. And unlike `/assets/`, media keys are *not* content-hashed — a replaced or deleted image must be able to disappear, so the year-long `immutable` treatment used for bundle output would be wrong here.
4. **`storage.NewLocalDisk(cfg.DataDir, "")` for media.** `mediaapi`'s existing `randomMediaKey` already bakes the `"media/"` namespace into the key itself (`media/cartridge_art/2026-<hex>.webp`). Rooting the storage at the bare data directory with an empty base URL therefore yields both a correct on-disk path (`data/media/cartridge_art/…`) and a correct public URL (`/media/cartridge_art/…`) with no doubled segment — without touching the already-merged key helper.
5. **`Dependencies` gains a third field, `MediaDir`, beyond `Media`/`MediaFiles`.** `noDirListing` is directory-backed and `storage.Storage` exposes no root path, so the static route needs the directory as data. This is the minimal honest addition; the alternative (a bespoke `Storage.Get`-backed file handler) would duplicate `noDirListing`'s directory-listing refusal logic for no gain.
6. **A `requestMultipartVoid` alongside `requestMultipart`.** `POST /api/admin/media/upload` answers 201 + JSON while `POST /api/admin/games/{slug}/upload` answers 204 with no body. That is exactly the split the file already models with `requestJSON`/`requestVoid`, so the multipart side mirrors it rather than teaching one helper to guess.
7. **Mutations use explicit arrow-function `mutationFn` wrappers.** Some `@tanstack/react-query` v5 patch versions pass a second `mutationFnContext` argument to `mutationFn`. A wrapper like `(id: number) => apiDeleteGame(id)` makes the extra argument structurally unable to reach the API function, and tests additionally assert positionally (`mock.calls[0]?.[0]`) rather than on the whole argument list. This exact issue burned Plan C Tasks 7 and 8; it must not regress a third time.
8. **No `enableReinitialize` on `GameForm`.** A background refetch must never clobber a half-typed edit. The edit page renders the form only once the game has loaded, so initial values are always real.
9. **The artwork widget prefers its own upload result over the media query.** After a successful upload the widget holds the authoritative `{id, url}` for the value it just wrote into page state, so the thumbnail updates instantly and deterministically without depending on a `GET /api/admin/media/{id}` round-trip landing first.
10. **`window.confirm` for destructive actions.** Two confirm sites (list-row delete, edit-page delete) do not justify introducing a dialog primitive, a portal, and focus-trap tests into this plan.
11. **A native styled `<input type="checkbox">`, not Radix's checkbox.** The form needs four independent booleans with no indeterminate state, no grouping, and no custom keyboard behavior. A native control is already accessible, already form-associated, and needs no new dependency.

**Tech Stack:** Go 1.26 (`net/http` `ServeMux` method+wildcard patterns, `httptest`), modernc SQLite, `golang.org/x/image` + `go-webp` (via the existing `internal/imaging`); Vite 5, React 18, TypeScript 5 strict, `react-router-dom` 6, Tailwind CSS v3 (CSS-variable tokens, light default + `prefers-color-scheme: dark`), Formik + Yup, `@tanstack/react-query` v5, `sonner`, `lucide-react`, `classnames` + `tailwind-merge` via `cn()`, Vitest 2 + `@testing-library/react`.

**Depends on:** `docs/superpowers/plans/2026-08-10-backend-core-data-model.md` (Plan A), `docs/superpowers/plans/2026-08-10-content-rendering-pipelines.md` (Plan B), `docs/superpowers/plans/2026-08-10-games-module-backend.md` (Plan D), `docs/superpowers/plans/2026-08-10-admin-spa-shell.md` (Plan C). All four are merged and green. This plan consumes their code as-is: `games.Repo`, `gamesapi.Handlers` (constructed `NewHandlers(repo, db, playDir)`), `media.Repo`, `mediaapi.UploadHandler`, `imaging.Targets`, `storage.LocalDisk`, `internal/httpapi`, `adminapi.RequireSession`, the `noDirListing` router helper, `admin-ui/src/api/client.ts`'s `ApiResult`/`requestJSON`/`requestVoid`, `admin-ui/src/auth/queries.ts`'s hook conventions, and the `Shell`/`ProtectedRoute`/`Button`/`Input`/`Label` components.

## Global Constraints

- Never use Go's `any` type alias or TypeScript's `any` type anywhere — use `interface{}`/a concrete type in Go, `unknown` + type guards or precise generics in TypeScript.
- API error responses use `{"error":{"code":"...","message":"..."}}` via `internal/httpapi.WriteError`.
- Every admin endpoint is mounted behind `adminapi.RequireSession`.
- A game's `slug` is immutable — never send it in an Update request body.
- PUT is a full replace, not a patch — the form must always submit the complete current state.
- Git commits in this repo: self-committed, one-sentence semantic messages, no co-author trailer.
- Colors come only from the project's semantic tokens (`background`, `surface`, `text`, `muted`, `border`, `accent`, `accent-hover`, `accent-foreground`, `success`, `error`, `warning`). Never a hex literal, never Tailwind's default palette (`gray-500` and friends).
- `tsconfig.json` sets `noUnusedLocals` and `noUnusedParameters`, and `npm run build` runs `tsc -b` over `src/**` including test files — every import and every declared callback parameter must actually be used.
- Frontend tests mock only the network boundary (`vi.mock("../api/client")`) and assert on real rendered DOM. No snapshots, no mocking of our own components or hooks.
- Any assertion against a mocked API function's arguments is positional (`vi.mocked(fn).mock.calls[0]?.[0]`, `…?.[1]`) — never `toHaveBeenCalledWith(...)` on the full argument list, because some `@tanstack/react-query` patch versions append a `mutationFnContext` argument to `mutationFn`.
- Existing Go handler signatures, response shapes, status codes and error codes are frozen. Task 1 is purely additive.

## Scope

In scope: the three backend additions of Task 1, two new UI primitives, typed client functions, query hooks, and the four games screens/widgets plus route wiring, and an end-to-end verification pass.

Out of scope (stays `ModulePlaceholderPage`): Members, Devlog, Awards, Contact, Site Settings, and the standalone Media library screen. Also out of scope: screenshot reordering (add/remove only — `display_order` is assigned as `screenshots.length`), alt-text editing on media, media deletion from the admin UI, a games search/filter/pagination UI, and any public-site rendering change.

---

## File Structure

```
internal/
  mediaapi/
    handlers.go              # (create) Handlers + NewHandlers + Get -> {id,url,width,height}
    handlers_test.go         # (create) Get success / not-found / invalid-id
  gamesapi/
    screenshot_handlers.go   # (modify) add (*Handlers).ListScreenshots
    screenshot_handlers_test.go # (modify) add ListScreenshots tests
  httpserver/
    router.go                # (modify) Media/MediaFiles/MediaDir deps; mount media upload,
                             #          media get, screenshots list, and /media/ static
    router_test.go           # (modify) add TestRouter_MediaUploadAndServing
cmd/
  server/
    main.go                  # (modify) mkdir data/media, media repo + storage, new deps
admin-ui/
  src/
    components/ui/
      textarea.tsx           # (create)
      checkbox.tsx           # (create)
      ui.test.tsx             # (modify) add Textarea + Checkbox tests
    api/
      client.ts              # (modify) games/screenshots/media wire types + functions,
                             #          requestMultipart + requestMultipartVoid
      client.test.ts         # (modify) add tests for every new function
    games/
      queries.ts              # (create) query keys + 11 hooks
      queries.test.tsx        # (create)
      GameForm.tsx            # (create) shared Formik form
      GameForm.test.tsx       # (create)
      GamesListPage.tsx        # (create)
      GamesListPage.test.tsx  # (create)
      CreateGamePage.tsx       # (create)
      CreateGamePage.test.tsx # (create)
      EditGamePage.tsx         # (create) form + 3 artwork widgets + screenshots + archive + delete
      EditGamePage.test.tsx    # (create)
      ScreenshotManager.tsx    # (create)
      ScreenshotManager.test.tsx # (create)
      ArchiveUploadWidget.tsx    # (create)
      ArchiveUploadWidget.test.tsx # (create)
    App.tsx                  # (modify) replace the games placeholder route with three real routes
    App.test.tsx             # (modify) prove /games renders the real list page
```

`admin-ui/src/games/` owns the whole module (hooks + screens + widgets, tests colocated), mirroring how `src/auth/` owns auth. `src/api/client.ts` stays the single network boundary so every test has exactly one thing to mock.

---

### Task 1: Backend — mount media upload, add media lookup, add screenshot listing, serve `/media/`

**Files:**
- Create: `internal/mediaapi/handlers.go`
- Create: `internal/mediaapi/handlers_test.go`
- Modify: `internal/gamesapi/screenshot_handlers.go`
- Modify: `internal/gamesapi/screenshot_handlers_test.go`
- Modify: `internal/httpserver/router.go`
- Modify: `internal/httpserver/router_test.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `media.NewRepo`, `media.Repo.FindByID`, `media.ErrMediaNotFound`, `storage.Storage.URL`, `storage.NewLocalDisk`, `mediaapi.NewUploadHandler`, `games.Repo.FindByID`, `games.Repo.ListScreenshots`, `games.ErrGameNotFound`, `httpapi.WriteJSON`/`WriteError`, `adminapi.RequireSession`, `noDirListing`.
- Produces: `mediaapi.NewHandlers(repo *media.Repo, files storage.Storage) *Handlers` with `Get(w, r)`; `(*gamesapi.Handlers).ListScreenshots(w, r)`; routes `POST /api/admin/media/upload`, `GET /api/admin/media/{id}`, `GET /api/admin/games/{id}/screenshots`, and public `GET /media/…`; `httpserver.Dependencies` fields `Media *media.Repo`, `MediaFiles storage.Storage`, `MediaDir string`.

- [ ] **Step 1: Write the failing tests for the media lookup handler**

Create `internal/mediaapi/handlers_test.go`:

```go
package mediaapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func setupMediaHandlers(t *testing.T) (*Handlers, *media.Repo) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	repo := media.NewRepo(conn)
	// Root = the bare data dir and baseURL = "" mirrors main.go exactly: the
	// stored key already carries the "media/" namespace.
	files := storage.NewLocalDisk(t.TempDir(), "")
	return NewHandlers(repo, files), repo
}

func TestGet_ReturnsMediaWithPublicURL(t *testing.T) {
	handlers, repo := setupMediaHandlers(t)
	saved, err := repo.Create("media/cartridge_art/2026-abc123.webp", 400, 560)
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got mediaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	want := mediaResponse{ID: saved.ID, URL: "/media/cartridge_art/2026-abc123.webp", Width: 400, Height: 560}
	if got != want {
		t.Errorf("Get() = %+v, want %+v", got, want)
	}
}

func TestGet_UnknownIDNotFound(t *testing.T) {
	handlers, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

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

func TestGet_NonNumericIDRejected(t *testing.T) {
	handlers, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/not-a-number", nil)
	req.SetPathValue("id", "not-a-number")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mediaapi/... -run 'TestGet_' -v`
Expected: FAIL — `Handlers`, `NewHandlers`, and `mediaResponse` are undefined.

- [ ] **Step 3: Implement the media lookup handler**

Create `internal/mediaapi/handlers.go`:

```go
package mediaapi

import (
	"errors"
	"net/http"
	"strconv"

	"pixabros/internal/httpapi"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

// Handlers serves media metadata lookups. The upload path lives on
// UploadHandler in upload_handler.go; this type deliberately stays read-only
// so the two concerns can be mounted (and reasoned about) separately.
type Handlers struct {
	repo  *media.Repo
	files storage.Storage
}

func NewHandlers(repo *media.Repo, files storage.Storage) *Handlers {
	return &Handlers{repo: repo, files: files}
}

// mediaResponse matches uploadResponse's shape on purpose: the admin UI shows
// a thumbnail from an upload result and from a lookup with the same code.
type mediaResponse struct {
	ID     int64  `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	m, err := h.repo.FindByID(id)
	if errors.Is(err, media.ErrMediaNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load media")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, mediaResponse{
		ID:     m.ID,
		URL:    h.files.URL(m.Path),
		Width:  m.Width,
		Height: m.Height,
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mediaapi/... -v`
Expected: PASS (the three new tests plus the four pre-existing `TestUpload_*` tests).

- [ ] **Step 5: Commit**

```bash
git add internal/mediaapi
git commit -m "feat: add a media metadata lookup endpoint returning the public URL"
```

- [ ] **Step 6: Write the failing tests for screenshot listing**

Append to `internal/gamesapi/screenshot_handlers_test.go` (the file already imports `bytes`, `encoding/json`, `fmt`, `net/http`, `net/http/httptest`, `testing`, and `pixabros/internal/games`; add `"strings"` to its import block):

```go
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
```

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./internal/gamesapi/... -run TestListScreenshots -v`
Expected: FAIL — `handlers.ListScreenshots` undefined.

- [ ] **Step 8: Implement `ListScreenshots`**

Append to `internal/gamesapi/screenshot_handlers.go` (its import block already has `errors`, `net/http`, `pixabros/internal/games`, `pixabros/internal/httpapi` — no import changes needed):

```go
// ListScreenshots checks the game exists before listing, so a request for an
// unknown game is a 404 rather than an empty array that looks like "this game
// simply has no screenshots".
func (h *Handlers) ListScreenshots(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	if _, err := h.repo.FindByID(gameID); err != nil {
		if errors.Is(err, games.ErrGameNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}

	list, err := h.repo.ListScreenshots(gameID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list screenshots")
		return
	}

	responses := make([]screenshotResponse, 0, len(list))
	for _, s := range list {
		responses = append(responses, screenshotResponse{
			ID:           s.ID,
			GameID:       s.GameID,
			MediaID:      s.MediaID,
			DisplayOrder: s.DisplayOrder,
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./internal/gamesapi/... -v`
Expected: PASS (new `TestListScreenshots_*` plus every pre-existing `gamesapi` test).

- [ ] **Step 10: Commit**

```bash
git add internal/gamesapi
git commit -m "feat: add a screenshot listing endpoint scoped to one game"
```

- [ ] **Step 11: Write the failing router test for media upload, lookup and static serving**

Append to `internal/httpserver/router_test.go`, and add `"image"`, `"image/color"`, `"image/png"`, `"pixabros/internal/media"` to its import block (`bytes`, `encoding/json`, `fmt`, `io`, `mime/multipart`, `net/http`, `net/http/httptest`, `os`, `path/filepath`, `strings`, `testing`, `auth`, `db`, `games`, `render`, `storage` are already imported):

```go
func solidPNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 40, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

// TestRouter_MediaUploadAndServing proves the whole media path the admin UI
// depends on: an authenticated multipart upload stores a resized WebP, the
// metadata lookup answers with the same public URL, that URL really serves the
// stored bytes back over /media/, and a miss 404s instead of exposing a
// directory listing.
func TestRouter_MediaUploadAndServing(t *testing.T) {
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
	if _, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	dataDir := t.TempDir()
	mediaDir := filepath.Join(dataDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media: %v", err)
	}
	renderedFiles := storage.NewLocalDisk(t.TempDir(), "/rendered")

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      render.NewStore(conn, renderedFiles),
		Files:      renderedFiles,
		DB:         conn,
		Games:      games.NewRepo(conn),
		Media:      media.NewRepo(conn),
		MediaFiles: storage.NewLocalDisk(dataDir, ""),
		MediaDir:   mediaDir,
		AdminUIDir: t.TempDir(),
		PlayDir:    t.TempDir(),
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	loginResp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error = %v", err)
	}
	defer loginResp.Body.Close()
	cookies := loginResp.Cookies()

	var multipartBuf bytes.Buffer
	mw := multipart.NewWriter(&multipartBuf)
	part, err := mw.CreateFormFile("file", "art.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(solidPNGBytes(t, 800, 1120)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	mw.Close()

	uploadReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/admin/media/upload?target=cartridge_art", &multipartBuf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	for _, c := range cookies {
		uploadReq.AddCookie(c)
	}
	uploadResp, err := srv.Client().Do(uploadReq)
	if err != nil {
		t.Fatalf("upload request error = %v", err)
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(uploadResp.Body)
		t.Fatalf("upload status = %d, want %d, body = %s", uploadResp.StatusCode, http.StatusCreated, body)
	}

	var uploaded struct {
		ID     int64  `json:"id"`
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.NewDecoder(uploadResp.Body).Decode(&uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploaded.Width != 400 || uploaded.Height != 560 {
		t.Errorf("uploaded dimensions = %dx%d, want the cartridge_art target's 400x560", uploaded.Width, uploaded.Height)
	}
	if !strings.HasPrefix(uploaded.URL, "/media/cartridge_art/") {
		t.Fatalf("uploaded url = %q, want it to start with /media/cartridge_art/", uploaded.URL)
	}

	lookupReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/admin/media/%d", srv.URL, uploaded.ID), nil)
	for _, c := range cookies {
		lookupReq.AddCookie(c)
	}
	lookupResp, err := srv.Client().Do(lookupReq)
	if err != nil {
		t.Fatalf("lookup request error = %v", err)
	}
	defer lookupResp.Body.Close()
	if lookupResp.StatusCode != http.StatusOK {
		t.Fatalf("lookup status = %d, want %d", lookupResp.StatusCode, http.StatusOK)
	}
	var looked struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.NewDecoder(lookupResp.Body).Decode(&looked); err != nil {
		t.Fatalf("decode lookup response: %v", err)
	}
	if looked.URL != uploaded.URL || looked.Width != 400 || looked.Height != 560 {
		t.Errorf("lookup = %+v, want the same URL %q and 400x560", looked, uploaded.URL)
	}

	// The public URL must really serve the stored bytes: without this the admin
	// UI would render a thumbnail pointing at a 404.
	fileResp, err := srv.Client().Get(srv.URL + uploaded.URL)
	if err != nil {
		t.Fatalf("media file request error = %v", err)
	}
	defer fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusOK {
		t.Fatalf("media file status = %d, want %d", fileResp.StatusCode, http.StatusOK)
	}
	fileBytes, err := io.ReadAll(fileResp.Body)
	if err != nil {
		t.Fatalf("read media file: %v", err)
	}
	if len(fileBytes) < 12 || !bytes.HasPrefix(fileBytes, []byte("RIFF")) || !bytes.Equal(fileBytes[8:12], []byte("WEBP")) {
		t.Errorf("served %d bytes, want a WebP image (RIFF....WEBP header)", len(fileBytes))
	}

	missResp, err := srv.Client().Get(srv.URL + "/media/cartridge_art/does-not-exist.webp")
	if err != nil {
		t.Fatalf("missing media request error = %v", err)
	}
	defer missResp.Body.Close()
	if missResp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing media status = %d, want %d", missResp.StatusCode, http.StatusNotFound)
	}

	listResp, err := srv.Client().Get(srv.URL + "/media/cartridge_art/")
	if err != nil {
		t.Fatalf("media directory request error = %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusNotFound {
		t.Fatalf("media directory status = %d, want %d (no directory listings)", listResp.StatusCode, http.StatusNotFound)
	}

	anonResp, err := srv.Client().Get(srv.URL + fmt.Sprintf("/api/admin/media/%d", uploaded.ID))
	if err != nil {
		t.Fatalf("anonymous lookup request error = %v", err)
	}
	defer anonResp.Body.Close()
	if anonResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous lookup status = %d, want %d", anonResp.StatusCode, http.StatusUnauthorized)
	}
}
```

- [ ] **Step 12: Run test to verify it fails**

Run: `go test ./internal/httpserver/... -run TestRouter_MediaUploadAndServing -v`
Expected: FAIL to compile — `Dependencies` has no `Media`, `MediaFiles`, or `MediaDir` field.

- [ ] **Step 13: Wire the routes**

Modify `internal/httpserver/router.go`. Extend `Dependencies`:

```go
type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	DB         *sql.DB
	Games      *games.Repo
	Media      *media.Repo
	MediaFiles storage.Storage
	MediaDir   string
	AdminUIDir string
	PlayDir    string
	AssetsDir  string
}
```

Add `"pixabros/internal/media"` and `"pixabros/internal/mediaapi"` to the import block. Inside `New`, add the screenshot-listing route to the existing games block (immediately above the `POST .../screenshots` line):

```go
	mux.HandleFunc("GET /api/admin/games/{id}/screenshots", adminapi.RequireSession(deps.Sessions, gamesHandlers.ListScreenshots))
```

Then add this block directly after the games routes and before the `onGameArchiveExtracted` closure:

```go
	mediaUploadHandler := mediaapi.NewUploadHandler(deps.Media, deps.MediaFiles)
	mediaHandlers := mediaapi.NewHandlers(deps.Media, deps.MediaFiles)
	mux.HandleFunc("POST /api/admin/media/upload", adminapi.RequireSession(deps.Sessions, mediaUploadHandler.Upload))
	mux.HandleFunc("GET /api/admin/media/{id}", adminapi.RequireSession(deps.Sessions, mediaHandlers.Get))
```

And add the static mount alongside the existing ones, above the `/assets/` line:

```go
	// Uploaded media is public site content (cartridge art, screenshots, OG
	// images all appear on the public MPA), so it is served without a session.
	// It gets plain noDirListing rather than the immutable-cache treatment
	// /assets/ uses: media keys are not content-hashed, so a replaced or
	// deleted image has to be able to actually disappear.
	mux.Handle("/media/", http.StripPrefix("/media/", noDirListing(deps.MediaDir)))
```

- [ ] **Step 14: Run test to verify it passes**

Run: `go test ./internal/httpserver/... -run TestRouter_MediaUploadAndServing -v`
Expected: PASS.

- [ ] **Step 15: Run the full Go suite**

Run: `go build ./... && go test ./...`
Expected: PASS. The pre-existing `Dependencies{…}` literals in `router_test.go` still compile unchanged (Go struct literals with field names tolerate new fields); they leave `Media`/`MediaFiles`/`MediaDir` zero-valued, which is fine because none of those tests touch a media route.

- [ ] **Step 16: Update the server entrypoint**

Modify `cmd/server/main.go`. Add `"pixabros/internal/media"` to the import block. Add `cfg.DataDir + "/media"` to the `MkdirAll` loop:

```go
	for _, dir := range []string{
		cfg.DataDir + "/admin-dist",
		cfg.DataDir + "/games",
		cfg.DataDir + "/assets",
		cfg.DataDir + "/media",
		cfg.DataDir + "/rendered-store",
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
	}
```

Add the media storage and repo right after `renderedFiles`:

```go
	renderedFiles := storage.NewLocalDisk(cfg.DataDir+"/rendered-store", "/rendered")
	// mediaapi's storage keys already begin with "media/", so the root is the
	// bare data dir and the base URL is empty: that yields both the on-disk
	// path <data>/media/<target>/<name>.webp and the public URL
	// /media/<target>/<name>.webp with no duplicated segment.
	mediaFiles := storage.NewLocalDisk(cfg.DataDir, "")
	mediaRepo := media.NewRepo(conn)
```

And extend the `Dependencies` literal (everything else in this file — signal handling, worker, shutdown — stays untouched):

```go
	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		DB:         conn,
		Games:      games.NewRepo(conn),
		Media:      mediaRepo,
		MediaFiles: mediaFiles,
		MediaDir:   cfg.DataDir + "/media",
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		AssetsDir:  cfg.DataDir + "/assets",
	})
```

- [ ] **Step 17: Verify the whole backend builds and passes**

Run: `go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 18: Commit**

```bash
git add internal/httpserver cmd/server
git commit -m "feat: mount media upload, media lookup, screenshot listing and public /media serving"
```

---

### Task 2: Shared shadcn primitives — `Textarea` and `Checkbox`

**Files:**
- Create: `admin-ui/src/components/ui/textarea.tsx`
- Create: `admin-ui/src/components/ui/checkbox.tsx`
- Modify: `admin-ui/src/components/ui/ui.test.tsx`

**Interfaces:**
- Consumes: `cn` from `@/lib/utils`.
- Produces: `Textarea` (`React.TextareaHTMLAttributes<HTMLTextAreaElement>`, forwardRef to `HTMLTextAreaElement`), `Checkbox` (`Omit<React.InputHTMLAttributes<HTMLInputElement>, "type">`, forwardRef to `HTMLInputElement`).

- [ ] **Step 1: Write the failing tests**

Append to `admin-ui/src/components/ui/ui.test.tsx`, and extend its existing import lines with the two new components:

```tsx
import { Textarea } from "./textarea";
import { Checkbox } from "./checkbox";
```

```tsx
describe("Textarea", () => {
  it("associates a label with its textarea and reflects typed input", () => {
    render(
      <div>
        <Label htmlFor="short-description">Short description</Label>
        <Textarea id="short-description" onChange={() => undefined} />
      </div>,
    );
    const field = screen.getByLabelText("Short description");
    fireEvent.change(field, { target: { value: "A tiny adventure." } });
    expect(field).toHaveValue("A tiny adventure.");
  });

  it("uses the surface background and border tokens, never a raw palette color", () => {
    render(<Textarea aria-label="Notes" onChange={() => undefined} />);
    const className = screen.getByLabelText("Notes").className;
    expect(className).toContain("bg-surface");
    expect(className).toContain("border-border");
    expect(className).not.toMatch(/gray-\d/);
  });

  it("merges a caller-supplied className instead of dropping it", () => {
    render(<Textarea aria-label="Notes" className="min-h-[200px]" onChange={() => undefined} />);
    expect(screen.getByLabelText("Notes").className).toContain("min-h-[200px]");
  });
});

describe("Checkbox", () => {
  it("renders a checkbox role and toggles on click", () => {
    let checked = false;
    render(
      <div>
        <Checkbox id="is-published" checked={checked} onChange={(event) => (checked = event.target.checked)} />
        <Label htmlFor="is-published">Published</Label>
      </div>,
    );
    const box = screen.getByRole("checkbox", { name: "Published" });
    fireEvent.click(box);
    expect(checked).toBe(true);
  });

  it("reflects the checked prop", () => {
    render(
      <div>
        <Checkbox id="is-for-sale" checked onChange={() => undefined} />
        <Label htmlFor="is-for-sale">For sale</Label>
      </div>,
    );
    expect(screen.getByRole("checkbox", { name: "For sale" })).toBeChecked();
  });

  it("tints the native control with the accent token", () => {
    render(<Checkbox aria-label="Toggle" onChange={() => undefined} />);
    expect(screen.getByRole("checkbox", { name: "Toggle" }).className).toContain("accent-accent");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/components/ui/ui.test.tsx`
Expected: FAIL — cannot resolve `./textarea` or `./checkbox`.

- [ ] **Step 3: Implement `Textarea`**

Create `admin-ui/src/components/ui/textarea.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(({ className, ...props }, ref) => {
  return (
    <textarea
      className={cn(
        "flex min-h-[96px] w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      ref={ref}
      {...props}
    />
  );
});
Textarea.displayName = "Textarea";

export { Textarea };
```

- [ ] **Step 4: Implement `Checkbox`**

Create `admin-ui/src/components/ui/checkbox.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

// A native checkbox, deliberately: the games form needs four independent
// booleans with no indeterminate state, no grouping and no custom keyboard
// behaviour, all of which the platform control already gets right (including
// form association and screen-reader semantics). `accent-accent` is Tailwind's
// accent-color utility resolved against this project's own `accent` token --
// the naming collision is a coincidence, not an indirection.
export type CheckboxProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "type">;

const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(({ className, ...props }, ref) => {
  return (
    <input
      type="checkbox"
      className={cn(
        "h-4 w-4 shrink-0 cursor-pointer rounded border border-border bg-surface accent-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      ref={ref}
      {...props}
    />
  );
});
Checkbox.displayName = "Checkbox";

export { Checkbox };
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/components/ui/ui.test.tsx`
Expected: PASS (the two pre-existing describes plus the two new ones).

- [ ] **Step 6: Verify `accent-accent` really resolves against the CSS-variable token**

```bash
npm --prefix admin-ui run build
grep -o 'accent-color:[^;}]*' data/admin-dist/assets/*.css
```

Expected: at least one match, `accent-color:rgb(var(--accent)/1)`. If the grep finds nothing, the utility did not generate — in that case (and only then) replace `accent-accent` in `checkbox.tsx` with the arbitrary-value form `accent-[rgb(var(--accent))]`, update the class assertion in `ui.test.tsx` to match, and re-run both this build check and the test.

- [ ] **Step 7: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add admin-ui/src/components/ui
git commit -m "feat: add Textarea and Checkbox UI primitives styled with the project's color tokens"
```

---

### Task 3: Typed API client extensions

**Files:**
- Modify: `admin-ui/src/api/client.ts`
- Modify: `admin-ui/src/api/client.test.ts`

**Interfaces:**
- Consumes: the existing `ApiResult`, `isErrorBody`, `parseJSON`, `errorResult`, `requestJSON`, `requestVoid` (all unchanged).
- Produces: types `Game`, `CreateGameRequest`, `UpdateGameRequest`, `Screenshot`, `AddScreenshotRequest`, `MediaResponse`, `MediaTarget`; functions `listGames`, `getGame`, `createGame`, `updateGame`, `deleteGame`, `listScreenshots`, `addScreenshot`, `removeScreenshot`, `uploadMedia`, `getMedia`, `uploadGameArchive`; helpers `requestMultipart<T>`, `requestMultipartVoid`.

- [ ] **Step 1: Write the failing tests**

Append to `admin-ui/src/api/client.test.ts`. Replace its first two import lines with the full list, adding a `Mock` type import:

```ts
import { afterEach, describe, expect, it, vi, type Mock } from "vitest";
import {
  addScreenshot,
  changePassword,
  createGame,
  deleteGame,
  getGame,
  getMedia,
  listGames,
  listScreenshots,
  login,
  logout,
  removeScreenshot,
  updateGame,
  uploadGameArchive,
  uploadMedia,
  whoami,
  type CreateGameRequest,
  type Game,
  type UpdateGameRequest,
} from "./client";
```

Then append:

```ts
function stubFetchJSON(status: number, body: unknown): Mock {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function stubFetchNoContent(): Mock {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

const gameFixture: Game = {
  id: 7,
  slug: "pixel-quest",
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "A longer description.",
  tags: "platformer,retro",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  cartridge_art_id: null,
  cd_cover_art_id: null,
  og_image_id: null,
  web_export_path: "",
  display_order: 0,
  is_published: false,
  created_at: "2026-08-11T09:00:00Z",
  updated_at: "2026-08-11T09:00:00Z",
};

const createBodyFixture: CreateGameRequest = {
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "A longer description.",
  tags: "platformer,retro",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  display_order: 0,
  is_published: false,
};

describe("listGames", () => {
  it("returns the array of games", async () => {
    stubFetchJSON(200, [gameFixture]);
    const result = await listGames();
    expect(result).toEqual({ ok: true, data: [gameFixture] });
  });

  it("returns the error envelope when unauthorized", async () => {
    stubFetchJSON(401, { error: { code: "unauthorized", message: "not logged in" } });
    const result = await listGames();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("unauthorized");
      expect(result.status).toBe(401);
    }
  });
});

describe("getGame", () => {
  it("requests the game by id", async () => {
    const fetchMock = stubFetchJSON(200, gameFixture);
    const result = await getGame(7);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7");
    expect(result).toEqual({ ok: true, data: gameFixture });
  });

  it("returns not_found for an unknown game", async () => {
    stubFetchJSON(404, { error: { code: "not_found", message: "game not found" } });
    const result = await getGame(999);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("not_found");
    }
  });
});

describe("createGame", () => {
  it("POSTs the create body as JSON and returns the created game", async () => {
    const fetchMock = stubFetchJSON(201, gameFixture);
    const result = await createGame(createBodyFixture);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games");
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toEqual(createBodyFixture);
    expect(result).toEqual({ ok: true, data: gameFixture });
  });

  it("returns missing_fields when the title is empty", async () => {
    stubFetchJSON(400, { error: { code: "missing_fields", message: "title is required" } });
    const result = await createGame({ ...createBodyFixture, title: "" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("missing_fields");
    }
  });
});

describe("updateGame", () => {
  it("PUTs the complete replacement body, with no slug in it", async () => {
    const body: UpdateGameRequest = {
      ...createBodyFixture,
      cartridge_art_id: 3,
      cd_cover_art_id: null,
      og_image_id: null,
    };
    const fetchMock = stubFetchJSON(200, { ...gameFixture, cartridge_art_id: 3 });
    const result = await updateGame(7, body);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7");
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe("PUT");
    const sent = JSON.parse(String(init.body)) as Record<string, unknown>;
    expect(sent).toEqual(body);
    // slug is immutable server-side; sending it would be a lie about intent.
    expect("slug" in sent).toBe(false);
    expect(result.ok).toBe(true);
  });
});

describe("deleteGame", () => {
  it("resolves ok on 204", async () => {
    const fetchMock = stubFetchNoContent();
    const result = await deleteGame(7);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7");
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).method).toBe("DELETE");
    expect(result).toEqual({ ok: true, data: undefined });
  });
});

describe("listScreenshots", () => {
  it("returns the screenshots for one game", async () => {
    const fetchMock = stubFetchJSON(200, [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }]);
    const result = await listScreenshots(7);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7/screenshots");
    expect(result).toEqual({ ok: true, data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }] });
  });
});

describe("addScreenshot", () => {
  it("POSTs media_id and display_order", async () => {
    const fetchMock = stubFetchJSON(201, { id: 11, game_id: 7, media_id: 55, display_order: 0 });
    const result = await addScreenshot(7, { media_id: 55, display_order: 0 });
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7/screenshots");
    expect(JSON.parse(String((fetchMock.mock.calls[0]?.[1] as RequestInit).body))).toEqual({
      media_id: 55,
      display_order: 0,
    });
    expect(result.ok).toBe(true);
  });
});

describe("removeScreenshot", () => {
  it("DELETEs the screenshot scoped to its game", async () => {
    const fetchMock = stubFetchNoContent();
    const result = await removeScreenshot(7, 11);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/7/screenshots/11");
    expect(result).toEqual({ ok: true, data: undefined });
  });
});

describe("uploadMedia", () => {
  it("POSTs multipart form data with the file field and the target query parameter", async () => {
    const fetchMock = stubFetchJSON(201, {
      id: 9,
      url: "/media/cartridge_art/2026-abc.webp",
      width: 400,
      height: 560,
    });
    const file = new File(["png-bytes"], "art.png", { type: "image/png" });

    const result = await uploadMedia("cartridge_art", file);

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/media/upload?target=cartridge_art");
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(init.body instanceof FormData).toBe(true);
    expect((init.body as FormData).get("file")).toBe(file);
    // The browser must set its own multipart boundary: forcing
    // Content-Type: application/json here would make the body unparseable.
    expect(init.headers).toBeUndefined();
    expect(result).toEqual({
      ok: true,
      data: { id: 9, url: "/media/cartridge_art/2026-abc.webp", width: 400, height: 560 },
    });
  });

  it("returns invalid_image when the server rejects the upload", async () => {
    stubFetchJSON(400, { error: { code: "invalid_image", message: "could not decode or process the uploaded image" } });
    const result = await uploadMedia("screenshot", new File(["nope"], "art.txt", { type: "text/plain" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("invalid_image");
    }
  });
});

describe("getMedia", () => {
  it("returns the media metadata", async () => {
    const fetchMock = stubFetchJSON(200, { id: 9, url: "/media/screenshot/x.webp", width: 1280, height: 720 });
    const result = await getMedia(9);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/media/9");
    expect(result).toEqual({ ok: true, data: { id: 9, url: "/media/screenshot/x.webp", width: 1280, height: 720 } });
  });

  it("returns not_found for an unknown media id", async () => {
    stubFetchJSON(404, { error: { code: "not_found", message: "media not found" } });
    const result = await getMedia(999);
    expect(result.ok).toBe(false);
  });
});

describe("uploadGameArchive", () => {
  it("POSTs the archive to the slug's upload endpoint and resolves ok on 204", async () => {
    const fetchMock = stubFetchNoContent();
    const file = new File(["zip-bytes"], "build.zip", { type: "application/zip" });

    const result = await uploadGameArchive("pixel-quest", file);

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/admin/games/pixel-quest/upload");
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.body instanceof FormData).toBe(true);
    expect((init.body as FormData).get("file")).toBe(file);
    expect(init.headers).toBeUndefined();
    expect(result).toEqual({ ok: true, data: undefined });
  });

  it("returns invalid_archive when extraction fails", async () => {
    stubFetchJSON(400, { error: { code: "invalid_archive", message: "could not extract the uploaded archive" } });
    const result = await uploadGameArchive("pixel-quest", new File(["x"], "build.zip"));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("invalid_archive");
    }
  });

  it("returns file_too_large on a 413", async () => {
    stubFetchJSON(413, { error: { code: "file_too_large", message: "uploaded archive exceeds the maximum allowed size" } });
    const result = await uploadGameArchive("pixel-quest", new File(["x"], "build.zip"));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.status).toBe(413);
    }
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/api/client.test.ts`
Expected: FAIL — none of the new exports exist.

- [ ] **Step 3: Add the wire types and functions**

Append to `admin-ui/src/api/client.ts` (nothing above is modified; `requestJSON` and `requestVoid` are reused verbatim):

```ts
// The types below are wire formats, so their fields keep the backend's exact
// snake_case names -- camelCasing them here would only add a translation layer
// that every request and response would have to cross twice.

export interface Game {
  id: number;
  slug: string;
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  is_browser_playable: boolean;
  is_downloadable: boolean;
  is_for_sale: boolean;
  price_display: string;
  external_links_json: string;
  cartridge_art_id: number | null;
  cd_cover_art_id: number | null;
  og_image_id: number | null;
  web_export_path: string;
  display_order: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateGameRequest {
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  is_browser_playable: boolean;
  is_downloadable: boolean;
  is_for_sale: boolean;
  price_display: string;
  external_links_json: string;
  display_order: number;
  is_published: boolean;
}

// PUT /api/admin/games/{id} is a full replace: any field left out of the body
// is reset to its zero value server-side, and `slug` is immutable and never
// part of the body. Every field is therefore required here, on purpose.
export interface UpdateGameRequest extends CreateGameRequest {
  cartridge_art_id: number | null;
  cd_cover_art_id: number | null;
  og_image_id: number | null;
}

export interface Screenshot {
  id: number;
  game_id: number;
  media_id: number;
  display_order: number;
}

export interface AddScreenshotRequest {
  media_id: number;
  display_order: number;
}

export interface MediaResponse {
  id: number;
  url: string;
  width: number;
  height: number;
}

// The exact set of upload targets internal/imaging/targets.go accepts. Each
// target has its own fixed output dimensions, which the server owns -- the
// client only ever names the target.
export type MediaTarget =
  | "avatar"
  | "cd_cover_art"
  | "cartridge_art"
  | "og_image"
  | "screenshot"
  | "award_picture"
  | "org_logo";

// requestMultipart deliberately sets no headers at all: the browser has to
// generate the multipart boundary itself, and requestJSON's
// Content-Type: application/json would make the body unparseable server-side.
async function requestMultipart<T>(path: string, formData: FormData): Promise<ApiResult<T>> {
  const response = await fetch(path, {
    method: "POST",
    credentials: "include",
    body: formData,
  });
  const parsed = await parseJSON(response);
  if (!response.ok) {
    return errorResult(parsed, response.status, response.statusText);
  }
  return { ok: true, data: parsed as T };
}

// The game-archive upload answers 204 with no body, so it needs the same
// split requestJSON/requestVoid already models for JSON requests.
async function requestMultipartVoid(path: string, formData: FormData): Promise<ApiResult<void>> {
  const response = await fetch(path, {
    method: "POST",
    credentials: "include",
    body: formData,
  });
  if (response.status === 204) {
    return { ok: true, data: undefined };
  }
  const parsed = await parseJSON(response);
  return errorResult(parsed, response.status, response.statusText);
}

export function listGames(): Promise<ApiResult<Game[]>> {
  return requestJSON<Game[]>("/api/admin/games", { method: "GET" });
}

export function getGame(id: number): Promise<ApiResult<Game>> {
  return requestJSON<Game>(`/api/admin/games/${id}`, { method: "GET" });
}

export function createGame(body: CreateGameRequest): Promise<ApiResult<Game>> {
  return requestJSON<Game>("/api/admin/games", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function updateGame(id: number, body: UpdateGameRequest): Promise<ApiResult<Game>> {
  return requestJSON<Game>(`/api/admin/games/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function deleteGame(id: number): Promise<ApiResult<void>> {
  return requestVoid(`/api/admin/games/${id}`, { method: "DELETE" });
}

export function listScreenshots(gameId: number): Promise<ApiResult<Screenshot[]>> {
  return requestJSON<Screenshot[]>(`/api/admin/games/${gameId}/screenshots`, { method: "GET" });
}

export function addScreenshot(gameId: number, body: AddScreenshotRequest): Promise<ApiResult<Screenshot>> {
  return requestJSON<Screenshot>(`/api/admin/games/${gameId}/screenshots`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function removeScreenshot(gameId: number, screenshotId: number): Promise<ApiResult<void>> {
  return requestVoid(`/api/admin/games/${gameId}/screenshots/${screenshotId}`, { method: "DELETE" });
}

export function uploadMedia(target: MediaTarget, file: File): Promise<ApiResult<MediaResponse>> {
  const formData = new FormData();
  formData.append("file", file);
  return requestMultipart<MediaResponse>(
    `/api/admin/media/upload?target=${encodeURIComponent(target)}`,
    formData,
  );
}

export function getMedia(id: number): Promise<ApiResult<MediaResponse>> {
  return requestJSON<MediaResponse>(`/api/admin/media/${id}`, { method: "GET" });
}

export function uploadGameArchive(slug: string, file: File): Promise<ApiResult<void>> {
  const formData = new FormData();
  formData.append("file", file);
  return requestMultipartVoid(`/api/admin/games/${encodeURIComponent(slug)}/upload`, formData);
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/api/client.test.ts`
Expected: PASS (the four pre-existing describes plus the eleven new ones).

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/api
git commit -m "feat: add typed games, screenshot and media API client functions with a multipart helper"
```

---

### Task 4: TanStack Query hooks for games, screenshots and media

**Files:**
- Create: `admin-ui/src/games/queries.ts`
- Create: `admin-ui/src/games/queries.test.tsx`

**Interfaces:**
- Consumes: every function and type added in Task 3; `useQuery`/`useMutation`/`useQueryClient`; `toast`.
- Produces: `gamesQueryKey`, `gameQueryKey(id)`, `screenshotsQueryKey(gameId)`, `mediaQueryKey(id)`, `useGamesQuery`, `useGameQuery`, `useCreateGameMutation`, `useUpdateGameMutation`, `useDeleteGameMutation`, `useScreenshotsQuery`, `useAddScreenshotMutation`, `useRemoveScreenshotMutation`, `useUploadMediaMutation`, `useMediaQuery`, `useUploadGameArchiveMutation`.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/queries.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { toast } from "sonner";
import type { ReactNode } from "react";
import { createTestQueryClient } from "../testUtils";
import type { Game } from "../api/client";
import * as client from "../api/client";
import {
  mediaQueryKey,
  useAddScreenshotMutation,
  useCreateGameMutation,
  useDeleteGameMutation,
  useGameQuery,
  useGamesQuery,
  useMediaQuery,
  useRemoveScreenshotMutation,
  useScreenshotsQuery,
  useUpdateGameMutation,
  useUploadGameArchiveMutation,
  useUploadMediaMutation,
} from "./queries";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

afterEach(() => {
  vi.resetAllMocks();
});

function setup(): { queryClient: QueryClient; Wrapper: (props: { children: ReactNode }) => JSX.Element } {
  const queryClient = createTestQueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return { queryClient, Wrapper };
}

const gameFixture: Game = {
  id: 7,
  slug: "pixel-quest",
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "",
  tags: "",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  cartridge_art_id: null,
  cd_cover_art_id: null,
  og_image_id: null,
  web_export_path: "",
  display_order: 0,
  is_published: false,
  created_at: "2026-08-11T09:00:00Z",
  updated_at: "2026-08-11T09:00:00Z",
};

const createBody = {
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "",
  tags: "",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  display_order: 0,
  is_published: false,
};

describe("useGamesQuery", () => {
  it("returns the list of games", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [gameFixture] });
    const { Wrapper } = setup();

    const { result } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.data?.ok).toBe(true));
    expect(result.current.data?.ok === true && result.current.data.data).toEqual([gameFixture]);
  });
});

describe("useGameQuery", () => {
  it("fetches one game by id", async () => {
    vi.mocked(client.getGame).mockResolvedValue({ ok: true, data: gameFixture });
    const { Wrapper } = setup();

    const { result } = renderHook(() => useGameQuery(7), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.data?.ok).toBe(true));
    expect(vi.mocked(client.getGame).mock.calls[0]?.[0]).toBe(7);
  });

  it("stays disabled for a non-numeric id so no request is fired", async () => {
    const { Wrapper } = setup();

    const { result } = renderHook(() => useGameQuery(Number.NaN), { wrapper: Wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(client.getGame).not.toHaveBeenCalled();
  });
});

describe("useCreateGameMutation", () => {
  it("invalidates the games list and toasts on success", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [] });
    vi.mocked(client.createGame).mockResolvedValue({ ok: true, data: gameFixture });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });
    const { result: createResult } = renderHook(() => useCreateGameMutation(), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    createResult.current.mutate(createBody);

    await waitFor(() => expect(vi.mocked(client.listGames).mock.calls.length).toBe(2));
    expect(vi.mocked(client.createGame).mock.calls[0]?.[0]).toEqual(createBody);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Game created.");
  });

  it("toasts the server error and does not refetch the list on failure", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [] });
    vi.mocked(client.createGame).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "missing_fields", message: "title is required" },
    });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });
    const { result: createResult } = renderHook(() => useCreateGameMutation(), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    createResult.current.mutate({ ...createBody, title: "" });

    await waitFor(() => expect(createResult.current.data?.ok).toBe(false));
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("title is required");
    expect(vi.mocked(client.listGames).mock.calls.length).toBe(1);
  });

  it("toasts a network error when the request rejects", async () => {
    vi.mocked(client.createGame).mockRejectedValue(new Error("network down"));
    const { Wrapper } = setup();

    const { result } = renderHook(() => useCreateGameMutation(), { wrapper: Wrapper });
    result.current.mutate(createBody);

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith(NETWORK_ERROR_MESSAGE);
  });
});

describe("useUpdateGameMutation", () => {
  it("passes the id and the full replacement body, then refetches the list", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [gameFixture] });
    vi.mocked(client.updateGame).mockResolvedValue({ ok: true, data: gameFixture });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });
    const { result: updateResult } = renderHook(() => useUpdateGameMutation(), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    const body = { ...createBody, cartridge_art_id: 3, cd_cover_art_id: null, og_image_id: null };
    updateResult.current.mutate({ id: 7, body });

    await waitFor(() => expect(vi.mocked(client.listGames).mock.calls.length).toBe(2));
    // Asserted positionally, never as a whole argument list: some
    // @tanstack/react-query patch versions append a mutationFnContext argument.
    expect(vi.mocked(client.updateGame).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(client.updateGame).mock.calls[0]?.[1]).toEqual(body);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Game saved.");
  });
});

describe("useDeleteGameMutation", () => {
  it("deletes by id and refetches the list", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [gameFixture] });
    vi.mocked(client.deleteGame).mockResolvedValue({ ok: true, data: undefined });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });
    const { result: deleteResult } = renderHook(() => useDeleteGameMutation(), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    deleteResult.current.mutate(7);

    await waitFor(() => expect(vi.mocked(client.listGames).mock.calls.length).toBe(2));
    expect(vi.mocked(client.deleteGame).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Game deleted.");
  });
});

describe("screenshot hooks", () => {
  it("lists a game's screenshots", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }],
    });
    const { Wrapper } = setup();

    const { result } = renderHook(() => useScreenshotsQuery(7), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.data?.ok).toBe(true));
    expect(vi.mocked(client.listScreenshots).mock.calls[0]?.[0]).toBe(7);
  });

  it("refetches that game's screenshots after adding one", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });
    vi.mocked(client.addScreenshot).mockResolvedValue({
      ok: true,
      data: { id: 11, game_id: 7, media_id: 55, display_order: 0 },
    });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useScreenshotsQuery(7), { wrapper: Wrapper });
    const { result: addResult } = renderHook(() => useAddScreenshotMutation(7), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    addResult.current.mutate({ media_id: 55, display_order: 0 });

    await waitFor(() => expect(vi.mocked(client.listScreenshots).mock.calls.length).toBe(2));
    expect(vi.mocked(client.addScreenshot).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(client.addScreenshot).mock.calls[0]?.[1]).toEqual({ media_id: 55, display_order: 0 });
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Screenshot added.");
  });

  it("removes a screenshot scoped to its game and refetches", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }],
    });
    vi.mocked(client.removeScreenshot).mockResolvedValue({ ok: true, data: undefined });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useScreenshotsQuery(7), { wrapper: Wrapper });
    const { result: removeResult } = renderHook(() => useRemoveScreenshotMutation(7), { wrapper: Wrapper });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    removeResult.current.mutate(11);

    await waitFor(() => expect(vi.mocked(client.listScreenshots).mock.calls.length).toBe(2));
    expect(vi.mocked(client.removeScreenshot).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(client.removeScreenshot).mock.calls[0]?.[1]).toBe(11);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Screenshot removed.");
  });
});

describe("media hooks", () => {
  it("looks up media by id and stays idle for a null id", async () => {
    vi.mocked(client.getMedia).mockResolvedValue({
      ok: true,
      data: { id: 9, url: "/media/screenshot/x.webp", width: 1280, height: 720 },
    });
    const { Wrapper } = setup();

    const { result: idle } = renderHook(() => useMediaQuery(null), { wrapper: Wrapper });
    expect(idle.current.fetchStatus).toBe("idle");
    expect(client.getMedia).not.toHaveBeenCalled();

    const { result } = renderHook(() => useMediaQuery(9), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.data?.ok).toBe(true));
    expect(vi.mocked(client.getMedia).mock.calls[0]?.[0]).toBe(9);
  });

  it("primes the media cache with the upload result so a thumbnail can render immediately", async () => {
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: true,
      data: { id: 9, url: "/media/cartridge_art/new.webp", width: 400, height: 560 },
    });
    const { queryClient, Wrapper } = setup();

    const { result } = renderHook(() => useUploadMediaMutation(), { wrapper: Wrapper });
    const file = new File(["png"], "art.png", { type: "image/png" });
    result.current.mutate({ target: "cartridge_art", file });

    await waitFor(() => expect(result.current.data?.ok).toBe(true));
    expect(queryClient.getQueryData(mediaQueryKey(9))).toEqual({
      ok: true,
      data: { id: 9, url: "/media/cartridge_art/new.webp", width: 400, height: 560 },
    });
    expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[0]).toBe("cartridge_art");
    expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[1]).toBe(file);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Image uploaded.");
  });
});

describe("useUploadGameArchiveMutation", () => {
  it("binds the slug, refetches the games list and toasts on success", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [gameFixture] });
    vi.mocked(client.uploadGameArchive).mockResolvedValue({ ok: true, data: undefined });
    const { Wrapper } = setup();

    const { result: listResult } = renderHook(() => useGamesQuery(), { wrapper: Wrapper });
    const { result: uploadResult } = renderHook(() => useUploadGameArchiveMutation("pixel-quest"), {
      wrapper: Wrapper,
    });
    await waitFor(() => expect(listResult.current.data?.ok).toBe(true));

    const file = new File(["zip"], "build.zip", { type: "application/zip" });
    uploadResult.current.mutate(file);

    await waitFor(() => expect(vi.mocked(client.listGames).mock.calls.length).toBe(2));
    expect(vi.mocked(client.uploadGameArchive).mock.calls[0]?.[0]).toBe("pixel-quest");
    expect(vi.mocked(client.uploadGameArchive).mock.calls[0]?.[1]).toBe(file);
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Build uploaded.");
  });

  it("toasts the server error for a rejected archive", async () => {
    vi.mocked(client.uploadGameArchive).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "invalid_archive", message: "could not extract the uploaded archive" },
    });
    const { Wrapper } = setup();

    const { result } = renderHook(() => useUploadGameArchiveMutation("pixel-quest"), { wrapper: Wrapper });
    result.current.mutate(new File(["x"], "build.zip"));

    await waitFor(() => expect(result.current.data?.ok).toBe(false));
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("could not extract the uploaded archive");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/queries.test.tsx`
Expected: FAIL — `./queries` does not exist.

- [ ] **Step 3: Implement the hooks**

Create `admin-ui/src/games/queries.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  addScreenshot as apiAddScreenshot,
  createGame as apiCreateGame,
  deleteGame as apiDeleteGame,
  getGame as apiGetGame,
  getMedia as apiGetMedia,
  listGames as apiListGames,
  listScreenshots as apiListScreenshots,
  removeScreenshot as apiRemoveScreenshot,
  updateGame as apiUpdateGame,
  uploadGameArchive as apiUploadGameArchive,
  uploadMedia as apiUploadMedia,
  type AddScreenshotRequest,
  type ApiResult,
  type CreateGameRequest,
  type MediaResponse,
  type MediaTarget,
  type UpdateGameRequest,
} from "../api/client";

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

// gamesQueryKey is a prefix of both gameQueryKey and screenshotsQueryKey, so
// invalidating it also invalidates a single game's detail and screenshot
// queries. That is intentional: every mutation that changes the list changes
// the game it touched too.
export const gamesQueryKey = ["games"] as const;

export function gameQueryKey(id: number) {
  return ["games", id] as const;
}

export function screenshotsQueryKey(gameId: number) {
  return ["games", gameId, "screenshots"] as const;
}

export function mediaQueryKey(id: number) {
  return ["media", id] as const;
}

// Every mutationFn below is an explicit arrow wrapper rather than a bare
// reference to the API function: some @tanstack/react-query patch versions
// pass a second mutationFnContext argument to mutationFn, and a wrapper makes
// it structurally impossible for that to leak into a request.

export function useGamesQuery() {
  return useQuery({
    queryKey: gamesQueryKey,
    queryFn: () => apiListGames(),
  });
}

export function useGameQuery(id: number) {
  return useQuery({
    queryKey: gameQueryKey(id),
    queryFn: () => apiGetGame(id),
    // A route param like /games/abc parses to NaN; firing a request for it
    // would only produce a confusing 400.
    enabled: Number.isInteger(id) && id > 0,
  });
}

export function useCreateGameMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateGameRequest) => apiCreateGame(body),
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.invalidateQueries({ queryKey: gamesQueryKey });
        toast.success("Game created.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useUpdateGameMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (variables: { id: number; body: UpdateGameRequest }) =>
      apiUpdateGame(variables.id, variables.body),
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.invalidateQueries({ queryKey: gamesQueryKey });
        toast.success("Game saved.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useDeleteGameMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDeleteGame(id),
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.invalidateQueries({ queryKey: gamesQueryKey });
        toast.success("Game deleted.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useScreenshotsQuery(gameId: number) {
  return useQuery({
    queryKey: screenshotsQueryKey(gameId),
    queryFn: () => apiListScreenshots(gameId),
    enabled: Number.isInteger(gameId) && gameId > 0,
  });
}

export function useAddScreenshotMutation(gameId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AddScreenshotRequest) => apiAddScreenshot(gameId, body),
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.invalidateQueries({ queryKey: screenshotsQueryKey(gameId) });
        toast.success("Screenshot added.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useRemoveScreenshotMutation(gameId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (screenshotId: number) => apiRemoveScreenshot(gameId, screenshotId),
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.invalidateQueries({ queryKey: screenshotsQueryKey(gameId) });
        toast.success("Screenshot removed.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useMediaQuery(id: number | null) {
  return useQuery({
    queryKey: mediaQueryKey(id ?? 0),
    queryFn: () => apiGetMedia(id ?? 0),
    enabled: id !== null && Number.isInteger(id) && id > 0,
  });
}

export function useUploadMediaMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (variables: { target: MediaTarget; file: File }) =>
      apiUploadMedia(variables.target, variables.file),
    onSuccess: (result) => {
      if (result.ok) {
        // Priming the cache means a thumbnail rendered from a list row's
        // media_id (a screenshot, say) appears immediately instead of waiting
        // for its own GET /api/admin/media/{id} to land.
        const uploaded = result.data;
        queryClient.setQueryData(
          mediaQueryKey(uploaded.id),
          (): ApiResult<MediaResponse> => ({ ok: true, data: uploaded }),
        );
        toast.success("Image uploaded.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}

export function useUploadGameArchiveMutation(slug: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => apiUploadGameArchive(slug, file),
    onSuccess: (result) => {
      if (result.ok) {
        // A successful extraction sets web_export_path server-side, so the
        // cached game is now stale.
        queryClient.invalidateQueries({ queryKey: gamesQueryKey });
        toast.success("Build uploaded.");
      } else {
        toast.error(result.error.message);
      }
    },
    onError: () => {
      toast.error(NETWORK_ERROR_MESSAGE);
    },
  });
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/queries.test.tsx`
Expected: PASS.

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/games
git commit -m "feat: add games, screenshot and media query hooks with cache invalidation and toasts"
```

---

### Task 5: Shared `GameForm` component

**Files:**
- Create: `admin-ui/src/games/GameForm.tsx`
- Create: `admin-ui/src/games/GameForm.test.tsx`

**Interfaces:**
- Consumes: `useFormik`, `Yup`, `Button`, `Input`, `Label`, `Textarea` (Task 2), `Checkbox` (Task 2), `cn`.
- Produces: default export `GameForm`; named exports `GameFormValues` (field-for-field identical to `CreateGameRequest`), `GameFormProps`.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/GameForm.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import GameForm, { type GameFormValues } from "./GameForm";

const emptyValues: GameFormValues = {
  title: "",
  short_description: "",
  full_description: "",
  tags: "",
  is_browser_playable: false,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  display_order: 0,
  is_published: false,
};

describe("GameForm", () => {
  it("renders every field with the given initial values", () => {
    render(
      <GameForm
        initialValues={{
          ...emptyValues,
          title: "Pixel Quest",
          short_description: "A tiny adventure.",
          full_description: "The long version.",
          tags: "platformer,retro",
          price_display: "$4.99",
          display_order: 3,
          is_browser_playable: true,
          is_downloadable: true,
          is_for_sale: true,
          is_published: true,
        }}
        submitLabel="Save game"
        onSubmit={() => undefined}
      />,
    );

    expect(screen.getByLabelText("Title")).toHaveValue("Pixel Quest");
    expect(screen.getByLabelText("Short description")).toHaveValue("A tiny adventure.");
    expect(screen.getByLabelText("Full description")).toHaveValue("The long version.");
    expect(screen.getByLabelText("Tags")).toHaveValue("platformer,retro");
    expect(screen.getByLabelText("Price display")).toHaveValue("$4.99");
    expect(screen.getByLabelText("External links JSON")).toHaveValue("[]");
    expect(screen.getByLabelText("Display order")).toHaveValue(3);
    expect(screen.getByRole("checkbox", { name: "Browser playable" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Downloadable" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "For sale" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Published" })).toBeChecked();
    expect(screen.getByRole("button", { name: "Save game" })).toBeInTheDocument();
  });

  it("submits every field, with booleans and the number coerced to real types", async () => {
    const onSubmit = vi.fn();
    render(<GameForm initialValues={emptyValues} submitLabel="Create game" onSubmit={onSubmit} />);

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.change(screen.getByLabelText("Short description"), { target: { value: "A tiny adventure." } });
    fireEvent.change(screen.getByLabelText("Full description"), { target: { value: "The long version." } });
    fireEvent.change(screen.getByLabelText("Tags"), { target: { value: "platformer,retro" } });
    fireEvent.change(screen.getByLabelText("Price display"), { target: { value: "$4.99" } });
    fireEvent.change(screen.getByLabelText("External links JSON"), {
      target: { value: '[{"label":"itch.io","url":"https://example.com"}]' },
    });
    fireEvent.change(screen.getByLabelText("Display order"), { target: { value: "5" } });
    fireEvent.click(screen.getByRole("checkbox", { name: "Browser playable" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Published" }));
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0]?.[0]).toEqual({
      title: "Pixel Quest",
      short_description: "A tiny adventure.",
      full_description: "The long version.",
      tags: "platformer,retro",
      is_browser_playable: true,
      is_downloadable: false,
      is_for_sale: false,
      price_display: "$4.99",
      external_links_json: '[{"label":"itch.io","url":"https://example.com"}]',
      display_order: 5,
      is_published: true,
    });
  });

  it("blocks submission and shows an error when the title is blank", async () => {
    const onSubmit = vi.fn();
    render(<GameForm initialValues={emptyValues} submitLabel="Create game" onSubmit={onSubmit} />);

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "   " } });
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(screen.getByText("Title is required.")).toBeInTheDocument());
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("rejects malformed external links JSON before any request is made", async () => {
    const onSubmit = vi.fn();
    render(<GameForm initialValues={emptyValues} submitLabel="Create game" onSubmit={onSubmit} />);

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.change(screen.getByLabelText("External links JSON"), { target: { value: "{not json" } });
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(screen.getByText(/must be valid JSON/)).toBeInTheDocument());
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("accepts an empty external links field as meaning no links", async () => {
    const onSubmit = vi.fn();
    render(
      <GameForm
        initialValues={{ ...emptyValues, external_links_json: "" }}
        submitLabel="Create game"
        onSubmit={onSubmit}
      />,
    );

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect((onSubmit.mock.calls[0]?.[0] as GameFormValues).external_links_json).toBe("");
  });

  it("rejects a negative display order", async () => {
    const onSubmit = vi.fn();
    render(<GameForm initialValues={emptyValues} submitLabel="Create game" onSubmit={onSubmit} />);

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.change(screen.getByLabelText("Display order"), { target: { value: "-2" } });
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(screen.getByText("Display order cannot be negative.")).toBeInTheDocument());
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("disables the submit button while a submission is in flight", async () => {
    let resolveSubmit: (() => void) | null = null;
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve;
        }),
    );
    render(
      <GameForm
        initialValues={{ ...emptyValues, title: "Pixel Quest" }}
        submitLabel="Save game"
        onSubmit={onSubmit}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Save game" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled());
    resolveSubmit?.();
    await waitFor(() => expect(screen.getByRole("button", { name: "Save game" })).toBeEnabled());
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/GameForm.test.tsx`
Expected: FAIL — `./GameForm` does not exist.

- [ ] **Step 3: Implement `GameForm`**

Create `admin-ui/src/games/GameForm.tsx`:

```tsx
import { useFormik } from "formik";
import * as Yup from "yup";
import { Button } from "../components/ui/button";
import { Checkbox } from "../components/ui/checkbox";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { cn } from "../lib/utils";

// GameFormValues mirrors the backend's create body field-for-field, so the
// create page submits it verbatim and the edit page spreads it into the
// full-replace PUT body next to the three artwork ids that page owns itself.
// `slug` is deliberately absent: it is immutable server-side.
export interface GameFormValues {
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  is_browser_playable: boolean;
  is_downloadable: boolean;
  is_for_sale: boolean;
  price_display: string;
  external_links_json: string;
  display_order: number;
  is_published: boolean;
}

// Formik's own change handler parses a type="number" input with parseFloat and
// falls back to "" when the field is emptied, so the internal state type
// allows "" and the submit handler normalizes it away before any caller sees
// it. Validation rejects "" anyway; the fallback only keeps the types total.
type GameFormState = Omit<GameFormValues, "display_order"> & { display_order: number | "" };

export interface GameFormProps {
  initialValues: GameFormValues;
  submitLabel: string;
  onSubmit: (values: GameFormValues) => void | Promise<void>;
}

const gameFormSchema = Yup.object({
  title: Yup.string().trim().required("Title is required."),
  short_description: Yup.string(),
  full_description: Yup.string(),
  tags: Yup.string(),
  price_display: Yup.string(),
  external_links_json: Yup.string().test(
    "valid-json",
    'External links must be valid JSON — for example [] or [{"label":"itch.io","url":"https://example.com"}].',
    (value) => {
      if (value === undefined || value.trim() === "") {
        return true;
      }
      try {
        JSON.parse(value);
        return true;
      } catch {
        return false;
      }
    },
  ),
  display_order: Yup.number()
    .typeError("Display order must be a whole number.")
    .integer("Display order must be a whole number.")
    .min(0, "Display order cannot be negative.")
    .required("Display order is required."),
  is_browser_playable: Yup.boolean(),
  is_downloadable: Yup.boolean(),
  is_for_sale: Yup.boolean(),
  is_published: Yup.boolean(),
});

export default function GameForm({ initialValues, submitLabel, onSubmit }: GameFormProps) {
  // No enableReinitialize on purpose: a background refetch must never clobber
  // an edit in progress. Callers mount this form only once real values exist.
  const formik = useFormik<GameFormState>({
    initialValues,
    validationSchema: gameFormSchema,
    onSubmit: async (values) => {
      await onSubmit({
        ...values,
        display_order: values.display_order === "" ? 0 : values.display_order,
      });
    },
  });

  return (
    <form onSubmit={formik.handleSubmit} className="space-y-5 rounded-lg border border-border bg-surface p-6" noValidate>
      <div className="space-y-1.5">
        <Label htmlFor="game-title">Title</Label>
        <Input
          id="game-title"
          name="title"
          value={formik.values.title}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        {formik.touched.title && formik.errors.title ? (
          <p className="text-sm text-error">{formik.errors.title}</p>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="game-short-description">Short description</Label>
        <Textarea
          id="game-short-description"
          name="short_description"
          rows={2}
          value={formik.values.short_description}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        {formik.touched.short_description && formik.errors.short_description ? (
          <p className="text-sm text-error">{formik.errors.short_description}</p>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="game-full-description">Full description</Label>
        <Textarea
          id="game-full-description"
          name="full_description"
          rows={6}
          value={formik.values.full_description}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        {formik.touched.full_description && formik.errors.full_description ? (
          <p className="text-sm text-error">{formik.errors.full_description}</p>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="game-tags">Tags</Label>
        <Input
          id="game-tags"
          name="tags"
          placeholder="platformer,retro"
          value={formik.values.tags}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        <p className="text-xs text-muted">Comma-separated.</p>
        {formik.touched.tags && formik.errors.tags ? (
          <p className="text-sm text-error">{formik.errors.tags}</p>
        ) : null}
      </div>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium text-text">Availability</legend>
        <div className="flex items-center gap-2">
          <Checkbox
            id="game-is-browser-playable"
            name="is_browser_playable"
            checked={formik.values.is_browser_playable}
            onChange={formik.handleChange}
          />
          <Label htmlFor="game-is-browser-playable">Browser playable</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="game-is-downloadable"
            name="is_downloadable"
            checked={formik.values.is_downloadable}
            onChange={formik.handleChange}
          />
          <Label htmlFor="game-is-downloadable">Downloadable</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="game-is-for-sale"
            name="is_for_sale"
            checked={formik.values.is_for_sale}
            onChange={formik.handleChange}
          />
          <Label htmlFor="game-is-for-sale">For sale</Label>
        </div>
      </fieldset>

      <div className="space-y-1.5">
        <Label htmlFor="game-price-display">Price display</Label>
        <Input
          id="game-price-display"
          name="price_display"
          placeholder="$4.99 or Free"
          value={formik.values.price_display}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        {formik.touched.price_display && formik.errors.price_display ? (
          <p className="text-sm text-error">{formik.errors.price_display}</p>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="game-external-links-json">External links JSON</Label>
        <Textarea
          id="game-external-links-json"
          name="external_links_json"
          rows={3}
          placeholder='[{"label":"itch.io","url":"https://example.com"}]'
          value={formik.values.external_links_json}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        {formik.touched.external_links_json && formik.errors.external_links_json ? (
          <p className="text-sm text-error">{formik.errors.external_links_json}</p>
        ) : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="game-display-order">Display order</Label>
        <Input
          id="game-display-order"
          name="display_order"
          type="number"
          min={0}
          step={1}
          className="max-w-[10rem]"
          value={formik.values.display_order}
          onChange={formik.handleChange}
          onBlur={formik.handleBlur}
        />
        <p className="text-xs text-muted">Lower numbers appear first.</p>
        {formik.touched.display_order && formik.errors.display_order ? (
          <p className="text-sm text-error">{formik.errors.display_order}</p>
        ) : null}
      </div>

      <div className="flex items-center gap-2">
        <Checkbox
          id="game-is-published"
          name="is_published"
          checked={formik.values.is_published}
          onChange={formik.handleChange}
        />
        <Label htmlFor="game-is-published">Published</Label>
      </div>

      <Button type="submit" disabled={formik.isSubmitting} className={cn(formik.isSubmitting && "opacity-70")}>
        {formik.isSubmitting ? "Saving…" : submitLabel}
      </Button>
    </form>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/GameForm.test.tsx`
Expected: PASS.

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/games/GameForm.tsx admin-ui/src/games/GameForm.test.tsx
git commit -m "feat: add a shared Formik game form covering every create and update field"
```

---

### Task 6: Games list page

**Files:**
- Create: `admin-ui/src/games/GamesListPage.tsx`
- Create: `admin-ui/src/games/GamesListPage.test.tsx`

**Interfaces:**
- Consumes: `useGamesQuery`, `useDeleteGameMutation` (Task 4), `Game` type (Task 3), `Button`, `cn`, `Link` from `react-router-dom`, `Plus` from `lucide-react`.
- Produces: default export `GamesListPage`.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/GamesListPage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import GamesListPage from "./GamesListPage";
import { renderWithQueryClient } from "../testUtils";
import type { Game } from "../api/client";
import * as client from "../api/client";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.resetAllMocks();
});

const baseGame: Game = {
  id: 7,
  slug: "pixel-quest",
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "",
  tags: "",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  cartridge_art_id: null,
  cd_cover_art_id: null,
  og_image_id: null,
  web_export_path: "",
  display_order: 0,
  is_published: true,
  created_at: "2026-08-11T09:00:00Z",
  updated_at: "2026-08-11T09:00:00Z",
};

function renderGamesListPage() {
  return renderWithQueryClient(
    <Routes>
      <Route path="/games" element={<GamesListPage />} />
      <Route path="/games/new" element={<p>New Game Screen</p>} />
      <Route path="/games/:id" element={<p>Edit Game Screen</p>} />
    </Routes>,
    { route: "/games" },
  );
}

describe("GamesListPage", () => {
  it("renders a row per game with its slug and status badges", async () => {
    vi.mocked(client.listGames).mockResolvedValue({
      ok: true,
      data: [
        baseGame,
        {
          ...baseGame,
          id: 8,
          slug: "neon-drift",
          title: "Neon Drift",
          is_published: false,
          is_browser_playable: false,
          is_downloadable: true,
          is_for_sale: true,
        },
      ],
    });

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("link", { name: "Pixel Quest" })).toBeInTheDocument());
    expect(screen.getByText("pixel-quest")).toBeInTheDocument();
    expect(screen.getByText("neon-drift")).toBeInTheDocument();

    const rows = screen.getAllByRole("row");
    // One header row plus two game rows.
    expect(rows).toHaveLength(3);
    expect(rows[1]).toHaveTextContent("Published");
    expect(rows[1]).toHaveTextContent("Browser");
    expect(rows[2]).toHaveTextContent("Draft");
    expect(rows[2]).toHaveTextContent("Download");
    expect(rows[2]).toHaveTextContent("For sale");
  });

  it("shows a dash instead of badges for a game with no distribution flags", async () => {
    vi.mocked(client.listGames).mockResolvedValue({
      ok: true,
      data: [{ ...baseGame, is_browser_playable: false, is_downloadable: false, is_for_sale: false }],
    });

    renderGamesListPage();

    await waitFor(() => expect(screen.getByText("—")).toBeInTheDocument());
    expect(screen.queryByText("Browser")).not.toBeInTheDocument();
  });

  it("links each title to that game's edit route and offers a new-game link", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [baseGame] });

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("link", { name: "Pixel Quest" })).toHaveAttribute("href", "/games/7"));
    fireEvent.click(screen.getByRole("link", { name: "New game" }));
    await waitFor(() => expect(screen.getByText("New Game Screen")).toBeInTheDocument());
  });

  it("shows an empty state when there are no games", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [] });

    renderGamesListPage();

    await waitFor(() => expect(screen.getByText("No games yet.")).toBeInTheDocument());
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows the server error message when the list fails", async () => {
    vi.mocked(client.listGames).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("not logged in"));
  });

  it("shows a network error, not an empty list, when the request itself rejects", async () => {
    // A rejected request never reached the server; rendering "No games yet."
    // here would be a lie about the state of the world.
    vi.mocked(client.listGames).mockRejectedValue(new Error("network down"));

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Network error — please try again."));
    expect(screen.queryByText("No games yet.")).not.toBeInTheDocument();
  });

  it("deletes a game after the confirmation is accepted", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [baseGame] });
    vi.mocked(client.deleteGame).mockResolvedValue({ ok: true, data: undefined });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => expect(vi.mocked(client.deleteGame).mock.calls[0]?.[0]).toBe(7));
    expect(confirmSpy).toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("does not delete when the confirmation is dismissed", async () => {
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [baseGame] });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    renderGamesListPage();

    await waitFor(() => expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));

    expect(client.deleteGame).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/GamesListPage.test.tsx`
Expected: FAIL — `./GamesListPage` does not exist.

- [ ] **Step 3: Implement the page**

Create `admin-ui/src/games/GamesListPage.tsx`:

```tsx
import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Plus } from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import type { Game } from "../api/client";
import { useDeleteGameMutation, useGamesQuery } from "./queries";

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

// A plain semantic table styled with Tailwind: one list screen does not need a
// table primitive, and a real <table> keeps row/column semantics for free.
function Badge({ children, tone }: { children: ReactNode; tone: "accent" | "muted" | "success" }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium",
        tone === "success" && "border-success/40 bg-success/10 text-success",
        tone === "accent" && "border-accent/40 bg-accent/10 text-accent",
        tone === "muted" && "border-border bg-background text-muted",
      )}
    >
      {children}
    </span>
  );
}

function DistributionBadges({ game }: { game: Game }) {
  const flags: string[] = [];
  if (game.is_browser_playable) {
    flags.push("Browser");
  }
  if (game.is_downloadable) {
    flags.push("Download");
  }
  if (game.is_for_sale) {
    flags.push("For sale");
  }
  if (flags.length === 0) {
    return <span className="text-xs text-muted">—</span>;
  }
  return (
    <span className="flex flex-wrap gap-1">
      {flags.map((flag) => (
        <Badge key={flag} tone="accent">
          {flag}
        </Badge>
      ))}
    </span>
  );
}

export default function GamesListPage() {
  const gamesQuery = useGamesQuery();
  const deleteMutation = useDeleteGameMutation();

  const handleDelete = (game: Game) => {
    // window.confirm rather than a dialog primitive: two confirm sites in this
    // whole module do not justify a portal and a focus trap.
    const confirmed = window.confirm(
      `Delete "${game.title}"? Its screenshots and uploaded build are deleted too. This cannot be undone.`,
    );
    if (confirmed) {
      deleteMutation.mutate(game.id);
    }
  };

  return (
    <section>
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold text-text">Games</h1>
        <Button asChild size="sm">
          <Link to="/games/new">
            <Plus className="mr-2 h-4 w-4" />
            New game
          </Link>
        </Button>
      </div>

      {gamesQuery.isPending ? <p className="text-sm text-muted">Loading games…</p> : null}

      {gamesQuery.isError ? (
        <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {NETWORK_ERROR_MESSAGE}
        </p>
      ) : null}

      {gamesQuery.data && !gamesQuery.data.ok ? (
        <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {gamesQuery.data.error.message}
        </p>
      ) : null}

      {gamesQuery.data?.ok && gamesQuery.data.data.length === 0 ? (
        <p className="text-sm text-muted">No games yet.</p>
      ) : null}

      {gamesQuery.data?.ok && gamesQuery.data.data.length > 0 ? (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full border-collapse text-left text-sm">
            <thead className="bg-surface text-xs uppercase tracking-wide text-muted">
              <tr>
                <th scope="col" className="px-4 py-3 font-medium">
                  Title
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  Slug
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  Status
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  Distribution
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  Order
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {gamesQuery.data.data.map((game) => (
                <tr key={game.id} className="border-t border-border">
                  <td className="px-4 py-3">
                    <Link to={`/games/${game.id}`} className="font-medium text-accent hover:text-accent-hover">
                      {game.title}
                    </Link>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted">{game.slug}</td>
                  <td className="px-4 py-3">
                    {game.is_published ? <Badge tone="success">Published</Badge> : <Badge tone="muted">Draft</Badge>}
                  </td>
                  <td className="px-4 py-3">
                    <DistributionBadges game={game} />
                  </td>
                  <td className="px-4 py-3 text-muted">{game.display_order}</td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      disabled={deleteMutation.isPending}
                      onClick={() => handleDelete(game)}
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/GamesListPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/games/GamesListPage.tsx admin-ui/src/games/GamesListPage.test.tsx
git commit -m "feat: add the games list page with status badges and a guarded delete action"
```

---

### Task 7: Create game page

**Files:**
- Create: `admin-ui/src/games/CreateGamePage.tsx`
- Create: `admin-ui/src/games/CreateGamePage.test.tsx`

**Interfaces:**
- Consumes: `GameForm`, `GameFormValues` (Task 5), `useCreateGameMutation` (Task 4), `useNavigate`, `Link`, `ArrowLeft` from `lucide-react`.
- Produces: default export `CreateGamePage`.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/CreateGamePage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import CreateGamePage from "./CreateGamePage";
import { renderWithQueryClient } from "../testUtils";
import type { Game } from "../api/client";
import * as client from "../api/client";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.resetAllMocks();
});

const createdGame: Game = {
  id: 7,
  slug: "pixel-quest",
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "",
  tags: "",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  cartridge_art_id: null,
  cd_cover_art_id: null,
  og_image_id: null,
  web_export_path: "",
  display_order: 0,
  is_published: false,
  created_at: "2026-08-11T09:00:00Z",
  updated_at: "2026-08-11T09:00:00Z",
};

function renderCreateGamePage() {
  return renderWithQueryClient(
    <Routes>
      <Route path="/games" element={<p>Games List Screen</p>} />
      <Route path="/games/new" element={<CreateGamePage />} />
      <Route path="/games/:id" element={<p>Edit Game Screen</p>} />
    </Routes>,
    { route: "/games/new" },
  );
}

describe("CreateGamePage", () => {
  it("creates the game and navigates to the new game's edit route", async () => {
    vi.mocked(client.createGame).mockResolvedValue({ ok: true, data: createdGame });

    renderCreateGamePage();

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.change(screen.getByLabelText("Short description"), { target: { value: "A tiny adventure." } });
    fireEvent.click(screen.getByRole("checkbox", { name: "Browser playable" }));
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() =>
      expect(vi.mocked(client.createGame).mock.calls[0]?.[0]).toEqual({
        title: "Pixel Quest",
        short_description: "A tiny adventure.",
        full_description: "",
        tags: "",
        is_browser_playable: true,
        is_downloadable: false,
        is_for_sale: false,
        price_display: "",
        external_links_json: "[]",
        display_order: 0,
        is_published: false,
      }),
    );
    // The real id from the response, not a guess.
    await waitFor(() => expect(screen.getByText("Edit Game Screen")).toBeInTheDocument());
  });

  it("has no slug field, because the slug is derived server-side and immutable", () => {
    renderCreateGamePage();

    expect(screen.queryByLabelText("Slug")).not.toBeInTheDocument();
  });

  it("offers a link back to the games list", async () => {
    renderCreateGamePage();

    fireEvent.click(screen.getByRole("link", { name: "Back to games" }));

    await waitFor(() => expect(screen.getByText("Games List Screen")).toBeInTheDocument());
  });

  it("stays on the page and shows the server error when creation fails", async () => {
    vi.mocked(client.createGame).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "missing_fields", message: "title is required" },
    });

    renderCreateGamePage();

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest" } });
    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("title is required"));
    expect(screen.queryByText("Edit Game Screen")).not.toBeInTheDocument();
  });

  it("never calls the API when the title is missing", async () => {
    renderCreateGamePage();

    fireEvent.click(screen.getByRole("button", { name: "Create game" }));

    await waitFor(() => expect(screen.getByText("Title is required.")).toBeInTheDocument());
    expect(client.createGame).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/CreateGamePage.test.tsx`
Expected: FAIL — `./CreateGamePage` does not exist.

- [ ] **Step 3: Implement the page**

Create `admin-ui/src/games/CreateGamePage.tsx`:

```tsx
import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import GameForm, { type GameFormValues } from "./GameForm";
import { useCreateGameMutation } from "./queries";

// The slug is derived server-side from the title and is immutable afterwards,
// so there is no slug field here at all.
const emptyGame: GameFormValues = {
  title: "",
  short_description: "",
  full_description: "",
  tags: "",
  is_browser_playable: false,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  display_order: 0,
  is_published: false,
};

export default function CreateGamePage() {
  const navigate = useNavigate();
  const createMutation = useCreateGameMutation();

  const serverError = createMutation.data && !createMutation.data.ok ? createMutation.data.error.message : null;

  return (
    <section className="mx-auto max-w-3xl">
      <Link to="/games" className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-text">
        <ArrowLeft className="h-4 w-4" />
        Back to games
      </Link>
      <h1 className="mb-2 text-2xl font-semibold text-text">New game</h1>
      <p className="mb-6 text-sm text-muted">
        The slug is generated from the title on save and cannot be changed afterwards. Artwork, screenshots and the
        playable build are attached after the game exists.
      </p>

      {/* GameFormValues is field-for-field the backend's create body, so the
          form's values go straight to the API with no translation. */}
      <GameForm
        initialValues={emptyGame}
        submitLabel="Create game"
        onSubmit={async (values) => {
          const result = await createMutation.mutateAsync(values);
          if (result.ok) {
            navigate(`/games/${result.data.id}`, { replace: true });
          }
        }}
      />

      {serverError ? (
        <p role="alert" className="mt-4 rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {serverError}
        </p>
      ) : null}
    </section>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/CreateGamePage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add admin-ui/src/games/CreateGamePage.tsx admin-ui/src/games/CreateGamePage.test.tsx
git commit -m "feat: add the create game page redirecting to the new game's edit screen"
```

---

### Task 8: Edit game page with artwork upload widgets

**Files:**
- Create: `admin-ui/src/games/EditGamePage.tsx`
- Create: `admin-ui/src/games/EditGamePage.test.tsx`

**Interfaces:**
- Consumes: `useGameQuery`, `useUpdateGameMutation`, `useDeleteGameMutation`, `useMediaQuery`, `useUploadMediaMutation` (Task 4), `GameForm`/`GameFormValues` (Task 5), `MediaTarget` (Task 3), `Button`, `Label`, `useParams`, `useNavigate`, `Link`, `ArrowLeft`.
- Produces: default export `EditGamePage`. Later tasks graft two sections onto it: `ScreenshotManager` (Task 9) and `ArchiveUploadWidget` (Task 10).

> **Sequencing note:** the final page renders `ScreenshotManager` and `ArchiveUploadWidget`, which do not exist yet. Step 3 below builds the page **without** them, so Task 8's tests pass standalone; Task 9 Step 5 and Task 10 Step 5 insert those two sections and add their own integration assertions to this same test file. Do not import either component in this task.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/EditGamePage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import EditGamePage from "./EditGamePage";
import { renderWithQueryClient } from "../testUtils";
import type { Game } from "../api/client";
import * as client from "../api/client";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.resetAllMocks();
});

const savedGame: Game = {
  id: 7,
  slug: "pixel-quest",
  title: "Pixel Quest",
  short_description: "A tiny adventure.",
  full_description: "The long version.",
  tags: "platformer,retro",
  is_browser_playable: true,
  is_downloadable: false,
  is_for_sale: false,
  price_display: "",
  external_links_json: "[]",
  cartridge_art_id: 3,
  cd_cover_art_id: null,
  og_image_id: null,
  web_export_path: "",
  display_order: 2,
  is_published: true,
  created_at: "2026-08-11T09:00:00Z",
  updated_at: "2026-08-11T09:00:00Z",
};

// mockHappyPath is shared with the sections Tasks 9 and 10 append to this file,
// which is why it also stubs listScreenshots.
function mockHappyPath(game: Game = savedGame): void {
  vi.mocked(client.getGame).mockResolvedValue({ ok: true, data: game });
  vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });
  vi.mocked(client.getMedia).mockImplementation(async (id: number) => ({
    ok: true,
    data: { id, url: `/media/cartridge_art/old-${id}.webp`, width: 400, height: 560 },
  }));
}

function renderEditGamePage() {
  return renderWithQueryClient(
    <Routes>
      <Route path="/games" element={<p>Games List Screen</p>} />
      <Route path="/games/:id" element={<EditGamePage />} />
    </Routes>,
    { route: "/games/7" },
  );
}

describe("EditGamePage", () => {
  it("prefills the form from the loaded game and shows the immutable slug", async () => {
    mockHappyPath();

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Title")).toHaveValue("Pixel Quest"));
    expect(screen.getByLabelText("Short description")).toHaveValue("A tiny adventure.");
    expect(screen.getByLabelText("Full description")).toHaveValue("The long version.");
    expect(screen.getByLabelText("Tags")).toHaveValue("platformer,retro");
    expect(screen.getByLabelText("Display order")).toHaveValue(2);
    expect(screen.getByRole("checkbox", { name: "Published" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Browser playable" })).toBeChecked();
    expect(screen.getByText("pixel-quest")).toBeInTheDocument();
    // The slug is displayed as text, never as an editable field.
    expect(screen.queryByLabelText("Slug")).not.toBeInTheDocument();
    expect(vi.mocked(client.getGame).mock.calls[0]?.[0]).toBe(7);
  });

  it("rejects a non-numeric route id without firing a request", async () => {
    renderWithQueryClient(
      <Routes>
        <Route path="/games/:id" element={<EditGamePage />} />
      </Routes>,
      { route: "/games/not-a-number" },
    );

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Invalid game id."));
    expect(client.getGame).not.toHaveBeenCalled();
  });

  it("submits the complete current state, including untouched fields and artwork ids", async () => {
    mockHappyPath();
    vi.mocked(client.updateGame).mockResolvedValue({ ok: true, data: savedGame });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Title")).toHaveValue("Pixel Quest"));
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Pixel Quest: Remastered" } });
    fireEvent.click(screen.getByRole("button", { name: "Save game" }));

    await waitFor(() => expect(vi.mocked(client.updateGame).mock.calls.length).toBe(1));
    expect(vi.mocked(client.updateGame).mock.calls[0]?.[0]).toBe(7);
    // PUT is a full replace: every field the server knows about has to be in
    // this body, or it is silently reset to its zero value.
    expect(vi.mocked(client.updateGame).mock.calls[0]?.[1]).toEqual({
      title: "Pixel Quest: Remastered",
      short_description: "A tiny adventure.",
      full_description: "The long version.",
      tags: "platformer,retro",
      is_browser_playable: true,
      is_downloadable: false,
      is_for_sale: false,
      price_display: "",
      external_links_json: "[]",
      display_order: 2,
      is_published: true,
      cartridge_art_id: 3,
      cd_cover_art_id: null,
      og_image_id: null,
    });
  });

  it("never sends a slug in the update body", async () => {
    mockHappyPath();
    vi.mocked(client.updateGame).mockResolvedValue({ ok: true, data: savedGame });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Title")).toHaveValue("Pixel Quest"));
    fireEvent.click(screen.getByRole("button", { name: "Save game" }));

    await waitFor(() => expect(vi.mocked(client.updateGame).mock.calls.length).toBe(1));
    const body = vi.mocked(client.updateGame).mock.calls[0]?.[1];
    expect(body && "slug" in body).toBe(false);
  });

  it("shows the current cartridge art thumbnail from the stored media id", async () => {
    mockHappyPath();

    renderEditGamePage();

    await waitFor(() =>
      expect(screen.getByAltText("Cartridge art preview")).toHaveAttribute(
        "src",
        "/media/cartridge_art/old-3.webp",
      ),
    );
    // Unset artwork fields do not look up media at all.
    expect(vi.mocked(client.getMedia).mock.calls.length).toBe(1);
    expect(vi.mocked(client.getMedia).mock.calls[0]?.[0]).toBe(3);
  });

  it("swaps the thumbnail as soon as an image is uploaded, with no form submit", async () => {
    mockHappyPath();
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: true,
      data: { id: 9, url: "/media/cartridge_art/new.webp", width: 400, height: 560 },
    });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByAltText("Cartridge art preview")).toBeInTheDocument());
    const file = new File(["png-bytes"], "art.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("Cartridge art"), { target: { files: [file] } });

    await waitFor(() =>
      expect(screen.getByAltText("Cartridge art preview")).toHaveAttribute("src", "/media/cartridge_art/new.webp"),
    );
    expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[0]).toBe("cartridge_art");
    expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[1]).toBe(file);
    // Uploading an image is not saving the game.
    expect(client.updateGame).not.toHaveBeenCalled();
  });

  it("includes a freshly uploaded artwork id in the next save", async () => {
    mockHappyPath();
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: true,
      data: { id: 9, url: "/media/cartridge_art/new.webp", width: 400, height: 560 },
    });
    vi.mocked(client.updateGame).mockResolvedValue({ ok: true, data: { ...savedGame, cartridge_art_id: 9 } });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Cartridge art")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Cartridge art"), {
      target: { files: [new File(["png"], "art.png", { type: "image/png" })] },
    });
    await waitFor(() =>
      expect(screen.getByAltText("Cartridge art preview")).toHaveAttribute("src", "/media/cartridge_art/new.webp"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Save game" }));

    await waitFor(() => expect(vi.mocked(client.updateGame).mock.calls.length).toBe(1));
    const body = vi.mocked(client.updateGame).mock.calls[0]?.[1];
    expect(body?.cartridge_art_id).toBe(9);
    expect(body?.cd_cover_art_id).toBeNull();
    expect(body?.og_image_id).toBeNull();
  });

  it("uploads the CD cover and OG image against their own targets", async () => {
    mockHappyPath();
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: true,
      data: { id: 10, url: "/media/cd_cover_art/new.webp", width: 600, height: 600 },
    });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("CD cover art")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("CD cover art"), {
      target: { files: [new File(["png"], "cover.png", { type: "image/png" })] },
    });
    await waitFor(() => expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[0]).toBe("cd_cover_art"));

    fireEvent.change(screen.getByLabelText("OG image"), {
      target: { files: [new File(["png"], "og.png", { type: "image/png" })] },
    });
    await waitFor(() => expect(vi.mocked(client.uploadMedia).mock.calls[1]?.[0]).toBe("og_image"));
  });

  it("shows 'No image yet.' for artwork slots that are unset", async () => {
    mockHappyPath({ ...savedGame, cartridge_art_id: null });

    renderEditGamePage();

    // All three artwork slots are empty in this fixture.
    await waitFor(() => expect(screen.getAllByText("No image yet.")).toHaveLength(3));
    expect(client.getMedia).not.toHaveBeenCalled();
  });

  it("shows the upload error inline when an image is rejected", async () => {
    mockHappyPath();
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "invalid_image", message: "could not decode or process the uploaded image" },
    });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Cartridge art")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Cartridge art"), {
      target: { files: [new File(["nope"], "art.txt", { type: "text/plain" })] },
    });

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("could not decode or process the uploaded image"),
    );
  });

  it("deletes the game after confirmation and returns to the list", async () => {
    mockHappyPath();
    vi.mocked(client.deleteGame).mockResolvedValue({ ok: true, data: undefined });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);

    renderEditGamePage();

    await waitFor(() => expect(screen.getByRole("button", { name: "Delete game" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Delete game" }));

    await waitFor(() => expect(vi.mocked(client.deleteGame).mock.calls[0]?.[0]).toBe(7));
    await waitFor(() => expect(screen.getByText("Games List Screen")).toBeInTheDocument());
    confirmSpy.mockRestore();
  });

  it("does not delete when the confirmation is dismissed", async () => {
    mockHappyPath();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    renderEditGamePage();

    await waitFor(() => expect(screen.getByRole("button", { name: "Delete game" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Delete game" }));

    expect(client.deleteGame).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("shows the server error when the game cannot be loaded", async () => {
    vi.mocked(client.getGame).mockResolvedValue({
      ok: false,
      status: 404,
      error: { code: "not_found", message: "game not found" },
    });
    vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("game not found"));
    expect(screen.queryByLabelText("Title")).not.toBeInTheDocument();
  });

  it("shows the server error when a save is rejected", async () => {
    mockHappyPath();
    vi.mocked(client.updateGame).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "missing_fields", message: "title is required" },
    });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByLabelText("Title")).toHaveValue("Pixel Quest"));
    fireEvent.click(screen.getByRole("button", { name: "Save game" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("title is required"));
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/EditGamePage.test.tsx`
Expected: FAIL — `./EditGamePage` does not exist.

- [ ] **Step 3: Implement the page**

Create `admin-ui/src/games/EditGamePage.tsx`:

```tsx
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "../components/ui/button";
import { Label } from "../components/ui/label";
import type { MediaTarget } from "../api/client";
import GameForm, { type GameFormValues } from "./GameForm";
import {
  useDeleteGameMutation,
  useGameQuery,
  useMediaQuery,
  useUpdateGameMutation,
  useUploadMediaMutation,
} from "./queries";

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

interface ImageUploadFieldProps {
  label: string;
  inputId: string;
  target: MediaTarget;
  mediaId: number | null;
  onUploaded: (mediaId: number) => void;
}

// ImageUploadField uploads bytes the moment a file is picked and reports the
// new media id upward. It deliberately does not save the game: the id becomes
// part of the next form submit, so a user who uploads and walks away has
// changed nothing about the game (the unreferenced media row is what
// internal/media/orphansweep.go exists to reap).
function ImageUploadField({ label, inputId, target, mediaId, onUploaded }: ImageUploadFieldProps) {
  const mediaQuery = useMediaQuery(mediaId);
  const uploadMutation = useUploadMediaMutation();

  // A successful upload is authoritative for the value we just wrote upward,
  // so the thumbnail swaps instantly instead of waiting on a media lookup.
  const uploaded = uploadMutation.data?.ok ? uploadMutation.data.data : null;
  const fetched = mediaQuery.data?.ok ? mediaQuery.data.data : null;
  const preview = uploaded ?? fetched;
  const uploadError = uploadMutation.data && !uploadMutation.data.ok ? uploadMutation.data.error.message : null;

  return (
    <div className="space-y-2 rounded-md border border-border bg-background p-4">
      <Label htmlFor={inputId}>{label}</Label>
      {preview ? (
        <img
          src={preview.url}
          alt={`${label} preview`}
          width={96}
          className="block rounded border border-border bg-surface"
        />
      ) : (
        <p className="text-xs text-muted">No image yet.</p>
      )}
      <input
        id={inputId}
        type="file"
        accept="image/*"
        className="block w-full text-sm text-muted file:mr-3 file:rounded-md file:border-0 file:bg-accent file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-accent-foreground hover:file:bg-accent-hover"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (!file) {
            return;
          }
          // Clearing the input lets the same file be re-picked after a failed
          // upload; setting a file input's value to "" is the one assignment
          // the platform allows.
          event.target.value = "";
          // mutate (not mutateAsync) so a network failure cannot become an
          // unhandled rejection inside this event handler -- the hook's own
          // onError already toasts it.
          uploadMutation.mutate(
            { target, file },
            {
              onSuccess: (result) => {
                if (result.ok) {
                  onUploaded(result.data.id);
                }
              },
            },
          );
        }}
      />
      {uploadMutation.isPending ? <p className="text-xs text-muted">Uploading…</p> : null}
      {uploadError ? (
        <p role="alert" className="text-sm text-error">
          {uploadError}
        </p>
      ) : null}
    </div>
  );
}

export default function EditGamePage() {
  const params = useParams();
  const gameId = Number(params.id);
  const validId = Number.isInteger(gameId) && gameId > 0;

  const navigate = useNavigate();
  const gameQuery = useGameQuery(validId ? gameId : Number.NaN);
  const updateMutation = useUpdateGameMutation();
  const deleteMutation = useDeleteGameMutation();

  // The three artwork ids live here rather than in GameForm: the widgets that
  // set them upload immediately and sit outside the form's field list, but
  // their values still have to ride along on the next full-replace PUT.
  const [cartridgeArtId, setCartridgeArtId] = useState<number | null>(null);
  const [cdCoverArtId, setCDCoverArtId] = useState<number | null>(null);
  const [ogImageId, setOGImageId] = useState<number | null>(null);

  const game = gameQuery.data?.ok ? gameQuery.data.data : null;

  // Every dependency here is a primitive, so a background refetch that returns
  // unchanged data does not re-run this effect and cannot clobber an uploaded
  // but not yet saved artwork id.
  useEffect(() => {
    if (!game) {
      return;
    }
    setCartridgeArtId(game.cartridge_art_id);
    setCDCoverArtId(game.cd_cover_art_id);
    setOGImageId(game.og_image_id);
  }, [game?.id, game?.cartridge_art_id, game?.cd_cover_art_id, game?.og_image_id]);

  if (!validId) {
    return (
      <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
        Invalid game id.
      </p>
    );
  }

  const serverError =
    (gameQuery.data && !gameQuery.data.ok ? gameQuery.data.error.message : null) ??
    (updateMutation.data && !updateMutation.data.ok ? updateMutation.data.error.message : null);

  const handleSubmit = async (values: GameFormValues) => {
    // PUT is a full replace, not a patch: the complete current state -- form
    // fields plus the three artwork ids this page owns -- goes out every time,
    // and the immutable slug is never part of it.
    await updateMutation.mutateAsync({
      id: gameId,
      body: {
        ...values,
        cartridge_art_id: cartridgeArtId,
        cd_cover_art_id: cdCoverArtId,
        og_image_id: ogImageId,
      },
    });
  };

  const handleDelete = () => {
    const confirmed = window.confirm(
      "Delete this game? Its screenshots and uploaded build are deleted too. This cannot be undone.",
    );
    if (!confirmed) {
      return;
    }
    deleteMutation.mutate(gameId, {
      onSuccess: (result) => {
        if (result.ok) {
          navigate("/games", { replace: true });
        }
      },
    });
  };

  return (
    <section className="mx-auto max-w-3xl space-y-8">
      <div>
        <Link to="/games" className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-text">
          <ArrowLeft className="h-4 w-4" />
          Back to games
        </Link>
        <h1 className="text-2xl font-semibold text-text">{game ? game.title : "Game"}</h1>
        {game ? (
          <p className="mt-1 text-sm text-muted">
            Slug <span className="font-mono text-xs text-text">{game.slug}</span> — generated on creation and
            permanent.
          </p>
        ) : null}
      </div>

      {gameQuery.isPending ? <p className="text-sm text-muted">Loading game…</p> : null}
      {gameQuery.isError ? (
        <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {NETWORK_ERROR_MESSAGE}
        </p>
      ) : null}
      {serverError ? (
        <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {serverError}
        </p>
      ) : null}

      {game ? (
        <>
          <div className="space-y-3">
            <h2 className="text-lg font-semibold text-text">Artwork</h2>
            <p className="text-sm text-muted">
              An uploaded image is attached to this game when you save the form below.
            </p>
            <div className="grid gap-4 sm:grid-cols-3">
              <ImageUploadField
                label="Cartridge art"
                inputId="cartridge-art-input"
                target="cartridge_art"
                mediaId={cartridgeArtId}
                onUploaded={setCartridgeArtId}
              />
              <ImageUploadField
                label="CD cover art"
                inputId="cd-cover-art-input"
                target="cd_cover_art"
                mediaId={cdCoverArtId}
                onUploaded={setCDCoverArtId}
              />
              <ImageUploadField
                label="OG image"
                inputId="og-image-input"
                target="og_image"
                mediaId={ogImageId}
                onUploaded={setOGImageId}
              />
            </div>
          </div>

          {/* No enableReinitialize inside GameForm, and this page mounts it only
              once real values exist, so a background refetch never clobbers an
              edit in progress. */}
          <GameForm
            initialValues={{
              title: game.title,
              short_description: game.short_description,
              full_description: game.full_description,
              tags: game.tags,
              is_browser_playable: game.is_browser_playable,
              is_downloadable: game.is_downloadable,
              is_for_sale: game.is_for_sale,
              price_display: game.price_display,
              external_links_json: game.external_links_json,
              display_order: game.display_order,
              is_published: game.is_published,
            }}
            submitLabel="Save game"
            onSubmit={handleSubmit}
          />

          <div className="rounded-lg border border-error/40 bg-error/5 p-6">
            <h2 className="text-lg font-semibold text-text">Danger zone</h2>
            <p className="mb-4 mt-1 text-sm text-muted">
              Deleting this game also removes its screenshots and its uploaded build from disk.
            </p>
            <Button type="button" variant="destructive" disabled={deleteMutation.isPending} onClick={handleDelete}>
              Delete game
            </Button>
          </div>
        </>
      ) : null}
    </section>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/EditGamePage.test.tsx`
Expected: PASS.

Two notes if something misbehaves here:
- `fireEvent.change(input, { target: { files: [file] } })` works because `dom-testing-library` special-cases the `files` key with `Object.defineProperty` rather than a plain assignment — do not replace it with a manual `defineProperty` dance.
- If `react-hooks/exhaustive-deps` warns about the `game?.…` dependency list, keep the primitive dependencies. That list is the mechanism that stops a refetch from clobbering an unsaved upload, and the accompanying comment documents it; the rule is a warning, not an error, in this repo's `plugin:react-hooks/recommended` setup.

- [ ] **Step 5: Lint and typecheck**

Run: `npm --prefix admin-ui run lint && npm --prefix admin-ui run build`
Expected: no errors.

- [ ] **Step 6: Run the whole games suite so far**

Run: `npm --prefix admin-ui run test -- src/games`
Expected: PASS — `queries`, `GameForm`, `GamesListPage`, `CreateGamePage`, `EditGamePage`.

- [ ] **Step 7: Commit**

```bash
git add admin-ui/src/games/EditGamePage.tsx admin-ui/src/games/EditGamePage.test.tsx
git commit -m "feat: add the edit game page with artwork upload widgets and full-replace saves"
```

---

### Task 9: Screenshot manager section

**Files:**
- Create: `admin-ui/src/games/ScreenshotManager.tsx`
- Create: `admin-ui/src/games/ScreenshotManager.test.tsx`
- Modify: `admin-ui/src/games/EditGamePage.tsx`
- Modify: `admin-ui/src/games/EditGamePage.test.tsx`

**Interfaces:**
- Consumes: `useScreenshotsQuery`, `useAddScreenshotMutation`, `useRemoveScreenshotMutation`, `useUploadMediaMutation`, `useMediaQuery` (Task 4), `Button`, `Label`.
- Produces: default export `ScreenshotManager` with props `{ gameId: number }`.

- [ ] **Step 1: Write the failing tests**

Create `admin-ui/src/games/ScreenshotManager.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import ScreenshotManager from "./ScreenshotManager";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.resetAllMocks();
});

function mockMediaLookups(): void {
  vi.mocked(client.getMedia).mockImplementation(async (id: number) => ({
    ok: true,
    data: { id, url: `/media/screenshot/${id}.webp`, width: 1280, height: 720 },
  }));
}

describe("ScreenshotManager", () => {
  it("renders a thumbnail per screenshot in display order", async () => {
    mockMediaLookups();
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [
        { id: 11, game_id: 7, media_id: 55, display_order: 0 },
        { id: 12, game_id: 7, media_id: 56, display_order: 1 },
      ],
    });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByAltText("Screenshot 1")).toHaveAttribute("src", "/media/screenshot/55.webp"));
    expect(screen.getByAltText("Screenshot 2")).toHaveAttribute("src", "/media/screenshot/56.webp");
    expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
    expect(vi.mocked(client.listScreenshots).mock.calls[0]?.[0]).toBe(7);
  });

  it("shows an empty state when there are no screenshots", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByText("No screenshots yet.")).toBeInTheDocument());
  });

  it("uploads the file and then attaches it with the next display order", async () => {
    mockMediaLookups();
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }],
    });
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: true,
      data: { id: 56, url: "/media/screenshot/56.webp", width: 1280, height: 720 },
    });
    vi.mocked(client.addScreenshot).mockResolvedValue({
      ok: true,
      data: { id: 12, game_id: 7, media_id: 56, display_order: 1 },
    });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByAltText("Screenshot 1")).toBeInTheDocument());
    const file = new File(["png"], "shot.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("Add screenshot"), { target: { files: [file] } });

    await waitFor(() => expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[0]).toBe("screenshot"));
    expect(vi.mocked(client.uploadMedia).mock.calls[0]?.[1]).toBe(file);
    await waitFor(() => expect(vi.mocked(client.addScreenshot).mock.calls.length).toBe(1));
    expect(vi.mocked(client.addScreenshot).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(client.addScreenshot).mock.calls[0]?.[1]).toEqual({ media_id: 56, display_order: 1 });
  });

  it("never attaches a screenshot when the upload is rejected", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });
    vi.mocked(client.uploadMedia).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "invalid_image", message: "could not decode or process the uploaded image" },
    });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByLabelText("Add screenshot")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Add screenshot"), {
      target: { files: [new File(["nope"], "shot.txt", { type: "text/plain" })] },
    });

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("could not decode or process the uploaded image"),
    );
    expect(client.addScreenshot).not.toHaveBeenCalled();
  });

  it("removes a screenshot scoped to its game", async () => {
    mockMediaLookups();
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }],
    });
    vi.mocked(client.removeScreenshot).mockResolvedValue({ ok: true, data: undefined });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(vi.mocked(client.removeScreenshot).mock.calls.length).toBe(1));
    expect(vi.mocked(client.removeScreenshot).mock.calls[0]?.[0]).toBe(7);
    expect(vi.mocked(client.removeScreenshot).mock.calls[0]?.[1]).toBe(11);
  });

  it("shows the server error when the list cannot be loaded", async () => {
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: false,
      status: 404,
      error: { code: "not_found", message: "game not found" },
    });

    renderWithQueryClient(<ScreenshotManager gameId={7} />);

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("game not found"));
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/ScreenshotManager.test.tsx`
Expected: FAIL — `./ScreenshotManager` does not exist.

- [ ] **Step 3: Implement `ScreenshotManager`**

Create `admin-ui/src/games/ScreenshotManager.tsx`:

```tsx
import { Button } from "../components/ui/button";
import { Label } from "../components/ui/label";
import {
  useAddScreenshotMutation,
  useMediaQuery,
  useRemoveScreenshotMutation,
  useScreenshotsQuery,
  useUploadMediaMutation,
} from "./queries";

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

function ScreenshotThumbnail({ mediaId, position }: { mediaId: number; position: number }) {
  const mediaQuery = useMediaQuery(mediaId);
  const media = mediaQuery.data?.ok ? mediaQuery.data.data : null;

  if (!media) {
    return <div className="h-20 w-32 rounded border border-border bg-background" />;
  }
  return (
    <img
      src={media.url}
      alt={`Screenshot ${position}`}
      width={128}
      className="block rounded border border-border bg-background"
    />
  );
}

export interface ScreenshotManagerProps {
  gameId: number;
}

export default function ScreenshotManager({ gameId }: ScreenshotManagerProps) {
  const screenshotsQuery = useScreenshotsQuery(gameId);
  const uploadMutation = useUploadMediaMutation();
  const addMutation = useAddScreenshotMutation(gameId);
  const removeMutation = useRemoveScreenshotMutation(gameId);

  const screenshots = screenshotsQuery.data?.ok ? screenshotsQuery.data.data : [];
  const listError = screenshotsQuery.data && !screenshotsQuery.data.ok ? screenshotsQuery.data.error.message : null;
  const uploadError = uploadMutation.data && !uploadMutation.data.ok ? uploadMutation.data.error.message : null;
  const addError = addMutation.data && !addMutation.data.ok ? addMutation.data.error.message : null;
  const removeError = removeMutation.data && !removeMutation.data.ok ? removeMutation.data.error.message : null;
  const error = listError ?? uploadError ?? addError ?? removeError;
  const busy = uploadMutation.isPending || addMutation.isPending;

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-text">Screenshots</h2>

      {screenshotsQuery.isPending ? <p className="text-sm text-muted">Loading screenshots…</p> : null}
      {screenshotsQuery.isError ? (
        <p role="alert" className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error">
          {NETWORK_ERROR_MESSAGE}
        </p>
      ) : null}

      {screenshotsQuery.data?.ok && screenshots.length === 0 ? (
        <p className="text-sm text-muted">No screenshots yet.</p>
      ) : null}

      {screenshots.length > 0 ? (
        <ul className="grid gap-4 sm:grid-cols-2">
          {screenshots.map((screenshot, index) => (
            <li
              key={screenshot.id}
              className="flex items-center gap-3 rounded-md border border-border bg-surface p-3"
            >
              <ScreenshotThumbnail mediaId={screenshot.media_id} position={index + 1} />
              <div className="flex-1">
                <p className="text-xs text-muted">Order {screenshot.display_order}</p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={removeMutation.isPending}
                onClick={() => removeMutation.mutate(screenshot.id)}
              >
                Remove
              </Button>
            </li>
          ))}
        </ul>
      ) : null}

      <div className="space-y-2">
        <Label htmlFor="add-screenshot-input">Add screenshot</Label>
        <input
          id="add-screenshot-input"
          type="file"
          accept="image/*"
          disabled={busy}
          className="block w-full text-sm text-muted file:mr-3 file:rounded-md file:border-0 file:bg-accent file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-accent-foreground hover:file:bg-accent-hover disabled:opacity-50"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (!file) {
              return;
            }
            event.target.value = "";
            // Two steps, chained through mutate's per-call callback: bytes
            // become a media row first, then that row is attached to this game.
            // A failed upload must never produce a screenshot row.
            const nextDisplayOrder = screenshots.length;
            uploadMutation.mutate(
              { target: "screenshot", file },
              {
                onSuccess: (result) => {
                  if (!result.ok) {
                    return;
                  }
                  addMutation.mutate({ media_id: result.data.id, display_order: nextDisplayOrder });
                },
              },
            );
          }}
        />
        {busy ? <p className="text-xs text-muted">Adding screenshot…</p> : null}
        {error ? (
          <p role="alert" className="text-sm text-error">
            {error}
          </p>
        ) : null}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/ScreenshotManager.test.tsx`
Expected: PASS.

- [ ] **Step 5: Render `ScreenshotManager` inside the edit page**

In `admin-ui/src/games/EditGamePage.tsx`, add the import:

```tsx
import ScreenshotManager from "./ScreenshotManager";
```

and insert this section between the `<GameForm … />` element and the `Danger zone` block:

```tsx
          <div className="rounded-lg border border-border bg-surface p-6">
            <ScreenshotManager gameId={gameId} />
          </div>
```

- [ ] **Step 6: Add the edit-page integration test**

Append to `admin-ui/src/games/EditGamePage.test.tsx`:

```tsx
describe("EditGamePage screenshots section", () => {
  it("renders the game's screenshots inside the edit page", async () => {
    vi.mocked(client.getGame).mockResolvedValue({ ok: true, data: savedGame });
    vi.mocked(client.listScreenshots).mockResolvedValue({
      ok: true,
      data: [{ id: 11, game_id: 7, media_id: 55, display_order: 0 }],
    });
    vi.mocked(client.getMedia).mockImplementation(async (id: number) => ({
      ok: true,
      data: { id, url: `/media/screenshot/${id}.webp`, width: 1280, height: 720 },
    }));

    renderEditGamePage();

    await waitFor(() => expect(screen.getByRole("heading", { name: "Screenshots" })).toBeInTheDocument());
    expect(screen.getByAltText("Screenshot 1")).toHaveAttribute("src", "/media/screenshot/55.webp");
    expect(vi.mocked(client.listScreenshots).mock.calls[0]?.[0]).toBe(7);
  });
});
```

- [ ] **Step 7: Run the games suite**

Run: `npm --prefix admin-ui run test -- src/games`
Expected: PASS (queries, GameForm, GamesListPage, CreateGamePage, EditGamePage, ScreenshotManager).

- [ ] **Step 8: Lint, typecheck and commit**

```bash
npm --prefix admin-ui run lint && npm --prefix admin-ui run build
git add admin-ui/src/games
git commit -m "feat: add screenshot management with thumbnails, upload-then-attach and removal"
```

---

### Task 10: Archive upload widget and route wiring

**Files:**
- Create: `admin-ui/src/games/ArchiveUploadWidget.tsx`
- Create: `admin-ui/src/games/ArchiveUploadWidget.test.tsx`
- Modify: `admin-ui/src/games/EditGamePage.tsx`
- Modify: `admin-ui/src/games/EditGamePage.test.tsx`
- Modify: `admin-ui/src/App.tsx`
- Modify: `admin-ui/src/App.test.tsx`

**Interfaces:**
- Consumes: `useUploadGameArchiveMutation` (Task 4), `Label`, `GamesListPage` (Task 6), `CreateGamePage` (Task 7), `EditGamePage` (Task 8).
- Produces: default export `ArchiveUploadWidget` with props `{ slug: string; webExportPath: string }`; routes `games`, `games/new`, `games/:id`.

- [ ] **Step 1: Write the failing widget tests**

Create `admin-ui/src/games/ArchiveUploadWidget.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import ArchiveUploadWidget from "./ArchiveUploadWidget";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";

vi.mock("../api/client");
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.resetAllMocks();
});

describe("ArchiveUploadWidget", () => {
  it("says there is no build yet and offers no play link when web_export_path is empty", () => {
    renderWithQueryClient(<ArchiveUploadWidget slug="pixel-quest" webExportPath="" />);

    expect(screen.getByText("No build uploaded yet.")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Play" })).not.toBeInTheDocument();
  });

  it("shows the current build path and a play link when a build exists", () => {
    renderWithQueryClient(
      <ArchiveUploadWidget slug="pixel-quest" webExportPath="/srv/pixabros/data/games/pixel-quest" />,
    );

    expect(screen.getByText("/srv/pixabros/data/games/pixel-quest")).toBeInTheDocument();
    // /play/ lives outside the SPA's basename, so this is a plain anchor.
    expect(screen.getByRole("link", { name: "Play" })).toHaveAttribute("href", "/play/pixel-quest/");
  });

  it("uploads the selected archive against the game's slug", async () => {
    vi.mocked(client.uploadGameArchive).mockResolvedValue({ ok: true, data: undefined });

    renderWithQueryClient(<ArchiveUploadWidget slug="pixel-quest" webExportPath="" />);

    const file = new File(["zip-bytes"], "build.zip", { type: "application/zip" });
    fireEvent.change(screen.getByLabelText("Upload build archive"), { target: { files: [file] } });

    await waitFor(() => expect(vi.mocked(client.uploadGameArchive).mock.calls[0]?.[0]).toBe("pixel-quest"));
    expect(vi.mocked(client.uploadGameArchive).mock.calls[0]?.[1]).toBe(file);
    await waitFor(() => expect(screen.getByText("Build uploaded.")).toBeInTheDocument());
  });

  it("shows the server error when the archive is rejected", async () => {
    vi.mocked(client.uploadGameArchive).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "invalid_archive", message: "could not extract the uploaded archive" },
    });

    renderWithQueryClient(<ArchiveUploadWidget slug="pixel-quest" webExportPath="" />);

    fireEvent.change(screen.getByLabelText("Upload build archive"), {
      target: { files: [new File(["x"], "build.zip", { type: "application/zip" })] },
    });

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("could not extract the uploaded archive"));
  });

  it("accepts only the archive extensions the backend can extract", () => {
    renderWithQueryClient(<ArchiveUploadWidget slug="pixel-quest" webExportPath="" />);

    expect(screen.getByLabelText("Upload build archive")).toHaveAttribute("accept", ".zip,.tar,.tar.gz,.tgz");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix admin-ui run test -- src/games/ArchiveUploadWidget.test.tsx`
Expected: FAIL — `./ArchiveUploadWidget` does not exist.

- [ ] **Step 3: Implement `ArchiveUploadWidget`**

Create `admin-ui/src/games/ArchiveUploadWidget.tsx`:

```tsx
import { Label } from "../components/ui/label";
import { useUploadGameArchiveMutation } from "./queries";

const NETWORK_ERROR_MESSAGE = "Network error — please try again.";

export interface ArchiveUploadWidgetProps {
  slug: string;
  webExportPath: string;
}

export default function ArchiveUploadWidget({ slug, webExportPath }: ArchiveUploadWidgetProps) {
  const uploadMutation = useUploadGameArchiveMutation(slug);

  const uploadError = uploadMutation.data && !uploadMutation.data.ok ? uploadMutation.data.error.message : null;
  const succeeded = uploadMutation.data?.ok === true;

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-text">Playable build</h2>

      {webExportPath ? (
        <p className="text-sm text-muted">
          Current build: <span className="font-mono text-xs text-text">{webExportPath}</span>{" "}
          {/* /play/ is served by the Go server outside the SPA's basename, so a
              react-router Link would resolve to the wrong URL here. */}
          <a
            href={`/play/${slug}/`}
            target="_blank"
            rel="noreferrer"
            className="ml-1 font-medium text-accent hover:text-accent-hover"
          >
            Play
          </a>
        </p>
      ) : (
        <p className="text-sm text-muted">No build uploaded yet.</p>
      )}

      <div className="space-y-2">
        <Label htmlFor="archive-upload-input">Upload build archive</Label>
        <input
          id="archive-upload-input"
          type="file"
          accept=".zip,.tar,.tar.gz,.tgz"
          disabled={uploadMutation.isPending}
          className="block w-full text-sm text-muted file:mr-3 file:rounded-md file:border-0 file:bg-accent file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-accent-foreground hover:file:bg-accent-hover disabled:opacity-50"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (!file) {
              return;
            }
            event.target.value = "";
            uploadMutation.mutate(file);
          }}
        />
        <p className="text-xs text-muted">
          A .zip, .tar, .tar.gz or .tgz archive with index.html at its root. Replaces the current build.
        </p>
        {uploadMutation.isPending ? <p className="text-xs text-muted">Uploading build…</p> : null}
        {succeeded ? <p className="text-sm text-success">Build uploaded.</p> : null}
        {uploadMutation.isError ? (
          <p role="alert" className="text-sm text-error">
            {NETWORK_ERROR_MESSAGE}
          </p>
        ) : null}
        {uploadError ? (
          <p role="alert" className="text-sm text-error">
            {uploadError}
          </p>
        ) : null}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix admin-ui run test -- src/games/ArchiveUploadWidget.test.tsx`
Expected: PASS.

- [ ] **Step 5: Render `ArchiveUploadWidget` inside the edit page**

In `admin-ui/src/games/EditGamePage.tsx`, add the import:

```tsx
import ArchiveUploadWidget from "./ArchiveUploadWidget";
```

and insert this section between the screenshots section and the `Danger zone` block:

```tsx
          <div className="rounded-lg border border-border bg-surface p-6">
            <ArchiveUploadWidget slug={game.slug} webExportPath={game.web_export_path} />
          </div>
```

- [ ] **Step 6: Add the edit-page integration test for the build section**

Append to `admin-ui/src/games/EditGamePage.test.tsx`:

```tsx
describe("EditGamePage build section", () => {
  it("renders the build section with the game's slug bound to the upload", async () => {
    mockHappyPath({ ...savedGame, web_export_path: "/srv/pixabros/data/games/pixel-quest" });
    vi.mocked(client.uploadGameArchive).mockResolvedValue({ ok: true, data: undefined });

    renderEditGamePage();

    await waitFor(() => expect(screen.getByRole("heading", { name: "Playable build" })).toBeInTheDocument());
    expect(screen.getByRole("link", { name: "Play" })).toHaveAttribute("href", "/play/pixel-quest/");

    fireEvent.change(screen.getByLabelText("Upload build archive"), {
      target: { files: [new File(["zip"], "build.zip", { type: "application/zip" })] },
    });

    await waitFor(() => expect(vi.mocked(client.uploadGameArchive).mock.calls[0]?.[0]).toBe("pixel-quest"));
  });
});
```

- [ ] **Step 7: Write the failing route-tree test**

Append to `admin-ui/src/App.test.tsx`:

```tsx
describe("App games routes", () => {
  it("renders the real games list page at /games", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    vi.mocked(client.listGames).mockResolvedValue({ ok: true, data: [] });
    window.history.pushState({}, "", "/I-am-a-pixabro/games");

    renderApp();

    // The placeholder page also renders a "Games" heading, so the real page is
    // identified by content only it has.
    await waitFor(() => expect(screen.getByRole("link", { name: "New game" })).toBeInTheDocument());
    expect(screen.getByText("No games yet.")).toBeInTheDocument();
    expect(screen.queryByText("This module's screens land here in a later plan.")).not.toBeInTheDocument();
  });

  it("renders the create page at /games/new", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    window.history.pushState({}, "", "/I-am-a-pixabro/games/new");

    renderApp();

    // A static segment outranks the :id wildcard in react-router's route
    // ranking, so /games/new is never treated as a game with id "new".
    await waitFor(() => expect(screen.getByRole("heading", { name: "New game" })).toBeInTheDocument());
    expect(client.getGame).not.toHaveBeenCalled();
  });

  it("renders the edit page at /games/:id", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    vi.mocked(client.getGame).mockResolvedValue({
      ok: true,
      data: {
        id: 7,
        slug: "pixel-quest",
        title: "Pixel Quest",
        short_description: "",
        full_description: "",
        tags: "",
        is_browser_playable: false,
        is_downloadable: false,
        is_for_sale: false,
        price_display: "",
        external_links_json: "[]",
        cartridge_art_id: null,
        cd_cover_art_id: null,
        og_image_id: null,
        web_export_path: "",
        display_order: 0,
        is_published: false,
        created_at: "2026-08-11T09:00:00Z",
        updated_at: "2026-08-11T09:00:00Z",
      },
    });
    vi.mocked(client.listScreenshots).mockResolvedValue({ ok: true, data: [] });
    window.history.pushState({}, "", "/I-am-a-pixabro/games/7");

    renderApp();

    await waitFor(() => expect(screen.getByRole("heading", { name: "Pixel Quest" })).toBeInTheDocument());
    expect(vi.mocked(client.getGame).mock.calls[0]?.[0]).toBe(7);
  });

  it("leaves the other module placeholders untouched", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    window.history.pushState({}, "", "/I-am-a-pixabro/members");

    renderApp();

    await waitFor(() => expect(screen.getByRole("heading", { name: "Members" })).toBeInTheDocument());
    expect(screen.getByText("This module's screens land here in a later plan.")).toBeInTheDocument();
  });
});
```

- [ ] **Step 8: Run test to verify it fails**

Run: `npm --prefix admin-ui run test -- src/App.test.tsx`
Expected: FAIL — `/games` still renders `ModulePlaceholderPage`, so there is no "New game" link and no `/games/new` or `/games/7` route.

- [ ] **Step 9: Wire the routes**

Modify `admin-ui/src/App.tsx`: add the three imports and replace the single `games` placeholder route with three real routes. Every other route, including `change-password` and the six remaining module placeholders, stays exactly as it is.

```tsx
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import Shell from "./pages/Shell";
import DashboardPage from "./pages/DashboardPage";
import ChangePasswordPage from "./pages/ChangePasswordPage";
import ModulePlaceholderPage from "./pages/ModulePlaceholderPage";
import GamesListPage from "./games/GamesListPage";
import CreateGamePage from "./games/CreateGamePage";
import EditGamePage from "./games/EditGamePage";

export default function App() {
  return (
    <BrowserRouter basename="/I-am-a-pixabro">
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Shell />
            </ProtectedRoute>
          }
        >
          <Route index element={<DashboardPage />} />
          <Route path="change-password" element={<ChangePasswordPage />} />
          <Route path="games" element={<GamesListPage />} />
          <Route path="games/new" element={<CreateGamePage />} />
          <Route path="games/:id" element={<EditGamePage />} />
          <Route path="members" element={<ModulePlaceholderPage title="Members" />} />
          <Route path="devlog" element={<ModulePlaceholderPage title="Devlog" />} />
          <Route path="awards" element={<ModulePlaceholderPage title="Awards" />} />
          <Route path="contact" element={<ModulePlaceholderPage title="Contact" />} />
          <Route path="site-settings" element={<ModulePlaceholderPage title="Site Settings" />} />
          <Route path="media" element={<ModulePlaceholderPage title="Media" />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
```

- [ ] **Step 10: Run the whole frontend suite**

Run: `npm --prefix admin-ui run test`
Expected: PASS — every pre-existing test plus the new games tests. The Shell's existing "Games" sidebar link now lands on the real page with no change to `Shell.tsx`.

- [ ] **Step 11: Lint, typecheck and commit**

```bash
npm --prefix admin-ui run lint && npm --prefix admin-ui run build
git add admin-ui/src
git commit -m "feat: add the game archive upload widget and wire the real games routes"
```

---

### Task 11: End-to-end verification

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: everything from Tasks 1–10.

- [ ] **Step 1: Build both halves and start the server clean**

```bash
cd /Users/furkan/Desktop/pixabros.com
go vet ./... && go test ./...
make admin-build
rm -rf /tmp/pixabros-e2e && mkdir -p /tmp/pixabros-e2e
cp -R data/admin-dist /tmp/pixabros-e2e/admin-dist
PIXABROS_DATA_DIR=/tmp/pixabros-e2e PIXABROS_DB_PATH=/tmp/pixabros-e2e/pixabros.db \
  go run ./cmd/admincli create-admin -username furkan -password "a-strong-password-1"
PIXABROS_DATA_DIR=/tmp/pixabros-e2e PIXABROS_DB_PATH=/tmp/pixabros-e2e/pixabros.db \
  go run ./cmd/server &
sleep 2
ls /tmp/pixabros-e2e
```

Expected: `go test ./...` all PASS; `make admin-build` writes `data/admin-dist/index.html` plus `data/admin-dist/assets/*`; `admincli` prints `admin created: furkan`; the server logs `listening on :8080`; the `ls` shows `admin-dist  assets  games  media  pixabros.db  rendered-store` — `media/` proves Task 1's new `MkdirAll` entry ran.

- [ ] **Step 2: Log in through the real UI**

Open `http://localhost:8080/I-am-a-pixabro/` in a browser, sign in as `furkan` / `a-strong-password-1`, then click **Games** in the sidebar.

Expected: the dashboard appears with "Signed in as furkan"; `/I-am-a-pixabro/games` shows the heading **Games**, the **New game** button, and **No games yet.** — not the old placeholder text "This module's screens land here in a later plan."

- [ ] **Step 3: Create a game through the UI and confirm the slug**

In the UI: **New game** → Title `Pixel Quest`, Short description `A tiny adventure.`, tick **Browser playable**, Display order `1`, tick **Published** → **Create game**.

Expected: a "Game created." toast; the browser lands on `/I-am-a-pixabro/games/7` (or whatever id was assigned); the heading reads **Pixel Quest** with `Slug pixel-quest — generated on creation and permanent.`; going back to **Games** shows one row: title `Pixel Quest`, slug `pixel-quest`, a green **Published** badge, a **Browser** badge, order `1`.

Confirm the same from the API:

```bash
curl -s -c /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"furkan","password":"a-strong-password-1"}' > /dev/null
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games
```

Expected: a one-element array with `"slug":"pixel-quest"`, `"is_browser_playable":true`, `"is_published":true`, `"display_order":1`, `"cartridge_art_id":null`, `"web_export_path":""`.

- [ ] **Step 4: Upload cartridge art and confirm the thumbnail appears without saving**

Create a test image, then use the UI's **Cartridge art** file input:

```bash
python3 - <<'PY'
import struct, zlib
def chunk(t, d):
    return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)
w = h = 512
raw = b"".join(b"\x00" + bytes([200, 40, 220]) * w for _ in range(h))
png = (b"\x89PNG\r\n\x1a\n"
       + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
       + chunk(b"IDAT", zlib.compress(raw))
       + chunk(b"IEND", b""))
open("/tmp/pixabros-art.png", "wb").write(png)
print("wrote /tmp/pixabros-art.png")
PY
```

In the UI, pick `/tmp/pixabros-art.png` for **Cartridge art**.

Expected: an "Image uploaded." toast; the "No image yet." text is replaced by a magenta thumbnail immediately, with **no** page reload and **no** form submit. Then confirm nothing was saved to the game yet and that the file is really on disk and really served:

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7 | python3 -m json.tool | grep cartridge_art_id
find /tmp/pixabros-e2e/media -name '*.webp'
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/media/1
curl -s -o /dev/null -w '%{http_code} %{content_type} %{size_download}\n' \
  "http://localhost:8080$(curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/media/1 | python3 -c 'import json,sys;print(json.load(sys.stdin)["url"])')"
```

Expected: `"cartridge_art_id": null` (an upload is not a save — this is the point); `find` shows exactly one file under `/tmp/pixabros-e2e/media/cartridge_art/`; the lookup returns `{"id":1,"url":"/media/cartridge_art/2026-….webp","width":400,"height":560}` (the `cartridge_art` target's fixed size, applied server-side); the final curl prints `200 image/webp` and a non-zero byte count.

- [ ] **Step 5: Save the game and prove the full-replace PUT does not blank untouched fields**

In the UI, on the same edit page, change only the Title to `Pixel Quest: Remastered` and press **Save game**.

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7 | python3 -m json.tool
```

Expected: a "Game saved." toast, and the JSON shows **all** of: `"title":"Pixel Quest: Remastered"`, `"slug":"pixel-quest"` (unchanged — the form never sends a slug), `"short_description":"A tiny adventure."` (untouched but preserved), `"is_browser_playable":true`, `"is_published":true`, `"display_order":1`, and `"cartridge_art_id":1` (the id from Step 4, attached by this save). If any of the untouched fields came back empty or `false`, the form is not resubmitting complete state and Task 8's `handleSubmit` is wrong.

- [ ] **Step 6: Add and remove a screenshot**

In the UI, under **Screenshots**, pick `/tmp/pixabros-art.png` for **Add screenshot**.

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7/screenshots
```

Expected: a thumbnail appears in the screenshots grid with `Order 0`; the API returns `[{"id":1,"game_id":7,"media_id":2,"display_order":0}]`. Add a second screenshot and confirm it gets `"display_order":1`. Then press **Remove** on the first one.

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7/screenshots
```

Expected: a "Screenshot removed." toast, the thumbnail disappears from the grid without a reload, and the API now returns only the second screenshot. Removing the last one leaves `[]` (an empty array, never `null`) and the UI shows **No screenshots yet.**

- [ ] **Step 7: Upload a build archive and confirm `/play/{slug}/` serves it**

```bash
rm -rf /tmp/pixabros-build && mkdir -p /tmp/pixabros-build
printf '<!doctype html><title>Pixel Quest</title><h1>Pixel Quest runs</h1>' > /tmp/pixabros-build/index.html
(cd /tmp/pixabros-build && zip -q -r /tmp/pixabros-build.zip .)
unzip -l /tmp/pixabros-build.zip
```

Expected: the listing shows `index.html` at the archive root (the extractor requires it there).

In the UI, under **Playable build**, pick `/tmp/pixabros-build.zip`.

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7 | grep -o '"web_export_path":"[^"]*"'
ls /tmp/pixabros-e2e/games/pixel-quest
curl -s http://localhost:8080/play/pixel-quest/
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/play/pixel-quest/
```

Expected: a "Build uploaded." toast and `Build uploaded.` inline text; `"web_export_path":"/tmp/pixabros-e2e/games/pixel-quest"`; `ls` shows `index.html`; the page body is `<!doctype html><title>Pixel Quest</title><h1>Pixel Quest runs</h1>` with status `200`. Reload the edit page and confirm the **Play** link now appears and opens that page.

- [ ] **Step 8: Confirm the archive guard rails**

```bash
printf 'not an archive' > /tmp/pixabros-bad.zip
curl -s -b /tmp/pixabros-cookies.txt -X POST -F file=@/tmp/pixabros-bad.zip \
  http://localhost:8080/api/admin/games/pixel-quest/upload
curl -s -b /tmp/pixabros-cookies.txt -X POST -F file=@/tmp/pixabros-build.zip \
  http://localhost:8080/api/admin/games/no-such-game/upload
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/admin/media/1
```

Expected: `{"error":{"code":"invalid_archive",…}}`; `{"error":{"code":"not_found",…}}`; and `401` for the cookie-less media lookup (every admin endpoint stays behind `RequireSession`). Confirm the UI surfaces the first case too by uploading `/tmp/pixabros-bad.zip` through the widget — the inline error must read "could not extract the uploaded archive", and `/play/pixel-quest/` must still serve the previous good build.

- [ ] **Step 9: Confirm `/media/` refuses directory listings**

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/media/
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/media/cartridge_art/
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/media/cartridge_art/nope.webp
```

Expected: `404` for all three — uploaded media is publicly readable by exact path only, never enumerable.

- [ ] **Step 10: Delete the game and confirm it disappears everywhere**

In the UI, on the edit page, press **Delete game** and accept the confirmation.

Expected: a "Game deleted." toast and a redirect to `/I-am-a-pixabro/games`, which now shows **No games yet.**

```bash
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games
curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/play/pixel-quest/
ls /tmp/pixabros-e2e/games
curl -s -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/games/7/screenshots
```

Expected: `[]`; `404`; `404` for `/play/pixel-quest/`; `ls` shows an empty `games` directory (the handler removed the extracted build from disk); and `{"error":{"code":"not_found",…}}` for the screenshot listing of the deleted game.

- [ ] **Step 11: Stop the server and clean up**

```bash
kill %1
wait %1 2>/dev/null
rm -rf /tmp/pixabros-e2e /tmp/pixabros-build /tmp/pixabros-build.zip /tmp/pixabros-bad.zip \
  /tmp/pixabros-art.png /tmp/pixabros-cookies.txt
git status --short
```

Expected: the server exits on SIGTERM after its graceful-shutdown path (the render worker joins before the DB closes, so no "database is closed" log); `git status --short` is clean — every task committed its own work and this verification task created no repository files.

- [ ] **Step 12: Final full-suite gate**

```bash
go vet ./... && go test ./...
npm --prefix admin-ui run lint
npm --prefix admin-ui run build
npm --prefix admin-ui run test
```

Expected: all four PASS with no warnings introduced by this plan.

---

## Consistency Notes Across Tasks

- **Wire types flow forward unchanged:** `gameResponse` (Go, Task 1's untouched neighbour) ↔ `Game` (TS, Task 3) ↔ `useGamesQuery`/`useGameQuery` payloads (Task 4) ↔ `GameFormValues` + the three artwork ids (Tasks 5/8). `screenshotResponse` ↔ `Screenshot`. `uploadResponse`/`mediaResponse` ↔ `MediaResponse`.
- **`GameFormValues` is exactly `CreateGameRequest`'s field set**, so `CreateGamePage` passes it straight to `createGame`, and `EditGamePage` spreads it into `UpdateGameRequest` by adding only `cartridge_art_id`, `cd_cover_art_id`, `og_image_id`. If a field is ever added to the backend's create body, both pages fail to typecheck until it is added to `GameFormValues` — which is the intended pressure.
- **`MediaTarget` is the single source of target names** used by Task 8 (`cartridge_art`, `cd_cover_art`, `og_image`) and Task 9 (`screenshot`); no dimension number appears anywhere in the frontend.
- **Query key prefixes:** `["games"]` ⊃ `["games", id]` ⊃ `["games", id, "screenshots"]`, so game mutations invalidate detail and screenshot data as a side effect, while screenshot mutations invalidate only their own narrower key.
- **`gameId` is a `number` everywhere** — parsed once in `EditGamePage` from `useParams`, validated there, and passed as a number to `useGameQuery`, `ScreenshotManager`, `useUpdateGameMutation` and `useDeleteGameMutation`. `slug` is a `string` passed only to `ArchiveUploadWidget` and only ever used to build a URL.

### Critical Files for Implementation
- /Users/furkan/Desktop/pixabros.com/internal/httpserver/router.go
- /Users/furkan/Desktop/pixabros.com/admin-ui/src/api/client.ts
- /Users/furkan/Desktop/pixabros.com/admin-ui/src/games/queries.ts
- /Users/furkan/Desktop/pixabros.com/admin-ui/src/games/EditGamePage.tsx
- /Users/furkan/Desktop/pixabros.com/cmd/server/main.go
