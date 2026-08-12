# Public Site Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the public site exist at all. Today `/*` is wired to `render.ServePages`, but no renderer has ever been registered, nothing writes public CSS, and nothing triggers a first render — so every public URL 404s and the regen queue has no work to do. This plan builds the shell every future page sits on (template layer, design tokens, asset pipeline, page-key reconciliation, styled 404) and proves it end-to-end by shipping one real page: `/awards`.

**Architecture:** A new `internal/site` package owns everything public-facing: an `embed.FS` of `html/template` files and CSS, a `Renderers` type that registers each page key against the existing `render.Registry`, and a `Reconciler` that keeps the set of rendered pages in step with the database. `cmd/server/main.go` gains three lines: build the asset bundle, register the renderers, run the reconciler. No existing package changes shape except `internal/render`'s serve handler, which learns to serve a styled 404, and `internal/httpserver`, which mounts the built assets.

**Tech Stack:** Go 1.26 standard library only — `html/template`, `embed`, `crypto/sha256`. **No new dependencies.** Hand-written plain CSS with `:root` custom properties (Tailwind is admin-only).

**Depends on:** Plan A (`internal/db`, `internal/httpserver`), Plan B (`internal/render`, `internal/storage`), and the per-module backend plans — `internal/games`, `internal/members`, `internal/awards`, `internal/devlog`, `internal/settings` are all implemented and are consumed read-only here.

**Source of truth for design:** `docs/superpowers/specs/2026-08-10-public-site-visual-design.md` (per-page visual decisions) and `docs/superpowers/specs/2026-08-10-frontend-design-and-stack.md` (palette, typography). Re-read both before starting.

---

## Global Constraints

- Never use Go's `any` type alias — use `interface{}` or a concrete type (user's global CLAUDE.md rule).
- **The public site is English only.** No i18n layer, no translation map, no language switcher. Write English copy directly into the templates. Turkish exists solely in the admin panel.
- Every public page is rendered ahead of time and served from `rendered_pages`. A request never touches a content table.
- Renderers are pure reads. Nothing under `internal/site` writes to a content table or enqueues a regen job.
- Tag strings must match the producers exactly — see the table below. A typo means a page silently never refreshes.
- Templates are `html/template`, never `text/template`, so interpolated content is escaped. Devlog markdown is the one place raw HTML is emitted, and it is out of scope here.
- Git commits: self-committed, one-sentence semantic messages, no co-author trailer.

## The tag vocabulary (already produced by the admin API — do not invent new ones)

| Tag | Enqueued by | Meaning |
|---|---|---|
| `game:list` | `gamesapi` | any change to the set or ordering of games |
| `game:{id}` | `gamesapi`, screenshot handlers, upload wiring | one game changed |
| `devlog:list` | `devlogapi` | the post index changed |
| `devlog:{id}` | `devlogapi` | one post changed |
| `award:list` | `awardsapi` | any award changed (there is no per-award tag) |
| `member:list` | `membersapi` | any member changed |
| `site_settings` | `settingsapi` | header/footer/site-wide settings |
| `homepage` | `settingsapi` | homepage-specific settings |

Awards and members have list tags only, which matches the design: awards live on one listing page, members are a section of the landing page.

## URL scheme

```
/                  index.html        Landing
/games             games             Play page (CRT TV, cartridge grid, CD boxes)
/games/{slug}      games/{slug}      Game detail
/devlog            devlog            Post index
/devlog/{slug}     devlog/{slug}     Post
/awards            awards            Awards timeline
/contact           contact           Contact form
```

`page_key` is the request path minus its leading slash (`/` maps to `index.html`) — see `render.ServePages`. The names above are therefore both the URL and the storage key.

**`/play` is not available.** `/play/{slug}/*` serves extracted game builds and Go's `ServeMux` redirects `/play` to `/play/`, so the visual spec's "Play Sayfası" lives at `/games`. Reserved prefixes a public page may never use: `api`, `I-am-a-pixabro`, `play`, `media`, `assets`.

## Scope

**In scope:** the template layer, design-token CSS, the asset pipeline, page-key reconciliation, the styled 404, and the `/awards` page complete.

**Explicitly out of scope**, each a follow-up plan:
- Landing (`/`) — hero, Steam-style portfolio carousel with thumbnail hover-swap, for-sale grid, members section.
- Play page (`/games`) — skeuomorphic CRT TV, NES cartridge grid, 3D-flip CD boxes, `Press Start 2P`.
- Game detail (`/games/{slug}`).
- Devlog (`/devlog`, `/devlog/{slug}`) — needs markdown→HTML rendering and sanitising.
- Contact (`/contact`) — **also needs a backend that does not exist yet**: there is no `POST /api/contact`, and `contact.Repo` has no `Create`. The spec requires min-100-character messages, `wants_callback` implying phone-or-email, a honeypot field, and a per-IP rate limit. That is a plan of its own, not a page.

### Why `/awards` proves the foundation

It is the cheapest page that exercises every part of the pipeline: it reads a content table, it depends on a list tag *and* `site_settings`, it has images (award badges) and dates, and its design is fully specified and simple (a vertical timeline). Building it first means the expensive pages are built on a foundation that has already been shown to work, rather than debugging the pipeline and a carousel at the same time.

---

## Deliberate deviations from the architecture spec

1. **No HTML/CSS minification.** The spec asks for minified output. Minifying would mean either a new dependency or a hand-rolled minifier, and a naive one breaks on `<pre>`, on `url()` values and on strings containing `/*`. The site is behind Cloudflare with compression on, where minification buys a few percent over gzip. Content hashing and immutable caching — the parts that actually matter — are implemented. Revisit if a page ever gets large enough to matter.
2. **Regen already deviates**, as recorded: no admin screen and no manual retry; the queue retries with backoff and gives up into the operator's log. Nothing in this plan reintroduces a UI for it.

---

## File Structure

```
internal/site/
  site.go              # Renderers struct, New(), Register(registry)
  templates.go         # embed.FS, template parsing, shared FuncMap
  assets.go            # Bundle: hash + publish embedded CSS, expose URLs
  assets_test.go
  pages.go             # DesiredPages(db) -> []string  (every page key that should exist)
  pages_test.go
  reconcile.go         # Reconciler: render missing pages, drop deleted ones
  reconcile_test.go
  awards.go            # the /awards renderer
  awards_test.go
  notfound.go          # the static 404 body
  testdata/
    awards.golden.html

  templates/
    layout.html        # <!doctype>, <head>, header, footer, block "main"
    awards.html
    404.html
  assets/
    site.css           # :root tokens + base typography + layout + components
    fonts/             # self-hosted woff2 (Inter, Press Start 2P)

internal/render/
  serve.go             # CHANGED: styled 404 instead of http.NotFound

internal/httpserver/
  router.go            # CHANGED: serve built assets

cmd/server/
  main.go              # CHANGED: build assets, register renderers, reconcile
```

---

### Task 1: Design tokens and the base stylesheet

- [ ] Create `internal/site/assets/site.css` opening with the palette from the stack spec as custom properties, verbatim — these are the same raw values the admin's Tailwind config uses:

```css
:root {
  --color-bg: #0F1115;
  --color-surface: #171A21;
  --color-text: #F1F1F3;
  --color-text-muted: #9AA0AC;
  --color-border: #2A2E37;
  --color-accent: #E879F9;
  --color-accent-strong: #C026D3;
  --color-success: #34D399;
  --color-error: #F87171;
  --color-warning: #FBBF24;
}
```

- [ ] Add a modern reset, base typography (Inter), the shared page frame (header, nav, footer, content container), and link/focus styles using `--color-accent` for the focus ring.
- [ ] The site is dark-only. Do **not** add a `prefers-color-scheme` light branch — the design language is a dark studio site, and a half-built light theme is worse than none.
- [ ] Self-host fonts under `assets/fonts/` and declare `@font-face` with `font-display: swap`. Do not link Google Fonts: it is a third-party request on every page load and the site already controls its own asset caching. `Press Start 2P` is loaded but referenced by nothing yet — the Play page is the only place it is ever allowed.
- [ ] **Verify:** every custom property referenced anywhere in the file is also defined in `:root`. A typo'd `var(--color-acccent)` silently renders as nothing.

### Task 2: The asset bundle — content hashing and publishing

The router already serves `<DataDir>/assets/` with `Cache-Control: immutable` and that directory also holds hand-placed files (the admin logo). Generated files therefore go in their own subdirectory so pruning can never touch anything a human put there.

- [ ] Create `internal/site/assets.go`:

```go
//go:embed assets
var assetFS embed.FS

// Bundle publishes the embedded assets under content-hashed names and remembers
// where they landed, so a template can ask for "site.css" and get the URL that
// is safe to cache forever.
type Bundle struct {
    urls map[string]string
}

// Build writes every embedded asset to <dir>/build/<name>.<hash><ext> and
// removes any file in that directory it did not just write. Pruning is scoped
// to build/ because the parent directory also holds hand-placed files.
func Build(dir string) (*Bundle, error)

// URL returns the public path for a logical asset name, e.g.
// "site.css" -> "/assets/build/site.a1b2c3d4.css".
func (b *Bundle) URL(name string) string
```

- [ ] Hash with `sha256`, take the first 8 hex characters. Collisions at this length are irrelevant for a handful of files, and short names keep the HTML readable.
- [ ] Fonts are published the same way, and `site.css` must reference them **through the bundle**, not by a literal path — which means the CSS needs its font URLs rewritten at publish time, or the fonts must keep stable unhashed names. **Choose the second:** publish fonts under their plain names in `build/fonts/`, and give only `site.css` a content hash. A font file's contents never change without its name changing, so it does not need the hash to bust caches; the stylesheet does.
- [ ] **Tests:** `Build` produces a hashed name; the same input twice produces the same name; changed input produces a different one; a stale file from a previous build is removed; a file in the parent `assets/` directory is left alone.

### Task 3: The template layer

- [ ] Create `internal/site/templates.go` with `//go:embed templates` and parse once at startup into a `*template.Template`. Parsing per render would turn a template typo into a runtime 500 on a page that used to work.
- [ ] `layout.html` defines the frame and a `{{block "main" .}}` the page templates fill. It receives a struct carrying, for every page:

```go
// pageData is what every template can rely on: the site chrome plus whatever
// the page itself adds.
type pageData struct {
    Title       string  // <title>, already page-specific
    Description string  // meta description
    Path        string  // for marking the current nav item
    CSS         string  // bundle URL
    Site        SiteChrome // name, tagline, social links from site_settings
    Data        interface{} // the page's own view model
}
```

- [ ] Nav links are a package-level slice so `/awards` and every later page mark themselves current from one place.
- [ ] Shared `FuncMap`: `formatDate` (English, e.g. `2 June 2026`), `mediaURL` (media id → `/media/...`), `hasImage`. Keep it small — logic belongs in the renderer, not the template.
- [ ] **Verify:** rendering the layout with an empty `Data` must not panic. A `nil` map access inside a template is a runtime error, not a compile error.

### Task 4: Page-key reconciliation — the missing first render

This is the gap that makes the whole site 404. `render.Worker` resolves a tag to the pages that already declared it in `page_tags`; a page key that has never been rendered has no rows, so no job can ever produce it. Something has to render a page for the first time, and delete it when its content is gone.

- [ ] `internal/site/pages.go`:

```go
// DesiredPages is every page key the site should currently have, static pages
// plus one per published game and devlog post. It is the single definition of
// "what the site consists of" -- the reconciler renders anything here that is
// missing and removes anything rendered that is no longer here.
func DesiredPages(db *sql.DB) ([]string, error)
```

Static keys: `index.html`, `games`, `devlog`, `awards`, `contact`. Dynamic: `games/{slug}` per game, `devlog/{slug}` per published post.

- [ ] `internal/site/reconcile.go`:

```go
// Reconcile renders every desired page that has no rendered_pages row, and
// forgets every rendered page that is no longer desired -- a deleted game must
// stop being served, not linger until someone notices.
func (r *Reconciler) Reconcile() (rendered int, removed int, err error)
```

- [ ] Removal deletes the `rendered_pages` row (its `page_tags` go with it by cascade). It does **not** delete the file from the rendered store: files are keyed by content hash and may be shared, and the store has no reference counting. Note this as known, bounded garbage; if it ever matters, it is a sweeper like `internal/media`'s, not an inline delete.
- [ ] Call `Reconcile` at startup **and** after the worker processes a batch, because content changes always enqueue a tag: a new game enqueues `game:list`, which is the signal that a new `games/{slug}` may be needed. Wire it through a callback on the worker rather than a second polling loop.
- [ ] A page that fails to render must not abort the run — log it and keep going, or one broken page takes the whole site down.
- [ ] **Tests:** a missing page gets rendered; an already-rendered page is not re-rendered; deleting a game removes its page; a renderer error is reported without stopping the others.

### Task 5: The 404 page

- [ ] `render.ServePages` currently calls `http.NotFound`, which returns bare text with no styling. Give the handler an optional not-found body (`[]byte`) and status-404 it with `Content-Type: text/html`.
- [ ] Render `404.html` once at startup through the same layout, so it carries the real header, footer and stylesheet. Per the architecture spec it stays outside the regen pipeline — it has no content dependencies beyond the chrome.
- [ ] Keep the fallback: if no body was supplied, behave exactly as today. That keeps `ServePages`' existing tests meaningful.
- [ ] **Test:** an unknown path returns 404 with `text/html` and the site header in the body.

### Task 6: The `/awards` page

- [ ] `internal/site/awards.go` registers `awards` and returns tags `award:list` and `site_settings`.
- [ ] View model per the visual spec: newest first (`date DESC`), each entry a badge image (`picture_id`, 320×320), title, issuer, date, and an optional outbound link. The vertical timeline line is CSS, not markup — no spacer elements.
- [ ] An award with no picture must render without a hole in the layout; an award with a `game_id` may link to that game, but the game detail page does not exist yet, so **do not** emit that link in this plan.
- [ ] Empty state: with no awards, the page still renders with its heading and a short line of English copy. A page that renders an empty timeline is fine; a page that fails to render is not.
- [ ] **Golden-file test** per the spec's test strategy: seed a fixed set of awards, render, compare against `testdata/awards.golden.html`. Include one award without a picture and one without a link so the branches are covered.

### Task 7: Wiring

- [ ] `cmd/server/main.go`: build the bundle from `cfg.DataDir + "/assets"`, construct `site.New(db, bundle)`, call `Register(registry)` **before** the worker starts (the registry is populated only at startup, by convention), run the reconciler once, then pass the 404 body to the router.
- [ ] Fail fast on bundle or template errors — those are startup misconfigurations, not runtime conditions, and a server that boots without CSS is worse than one that refuses to boot.
- [ ] `internal/httpserver/router.go`: no new mount is needed — `/assets/` already covers `build/`. Confirm `serveImmutableAssets` sets `immutable` for the nested path and that directory listing stays disabled.
- [ ] **Verify by hand, against a copy of the real database:**
  - `/awards` returns 200 with the real awards.
  - A second request with `If-None-Match` returns 304.
  - The CSS URL returns 200 with `Cache-Control: immutable`.
  - An unknown path returns a styled 404.
  - Editing an award in the admin panel changes the page within one poll interval, with no manual step.
  - `regen_jobs` shows the `award:list` job going to `done`.

### Task 8: End-to-end verification

- [ ] `gofmt -l .` clean, `go vet ./...` clean, `go test ./...` green.
- [ ] Run against a **copy** of `data/pixabros.db`, never the original, and confirm row counts are unchanged afterwards.
- [ ] Check the rendered HTML in a browser at both a narrow and a wide viewport. The site is dark-only, so also confirm it looks right in a browser set to light mode — the page must not inherit a white background.
- [ ] Confirm no request leaves the origin: no Google Fonts, no CDN, no analytics.

---

## Definition of Done

- `/awards` and a styled 404 are served from pre-rendered HTML with working ETag/304.
- Editing content in the admin panel updates the public page automatically, with no admin action beyond saving.
- A newly created game or devlog post gets a page key without anyone running anything, and a deleted one stops being served.
- CSS is content-hashed, served `immutable`, and self-hosted along with its fonts.
- The design tokens in `site.css` match the stack spec exactly, in one place.
- Every new package has tests; the awards render is covered by a golden file.
- No new Go dependency was added.

## What this unlocks

With the foundation in place, each remaining page is a renderer plus a template plus CSS, with no infrastructure work: Landing, Play, game detail, devlog (plus markdown rendering), and contact (plus its missing backend). Landing should come next — it is the site root and exercises the most content at once.
