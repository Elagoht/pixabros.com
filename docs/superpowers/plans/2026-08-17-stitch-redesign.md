# Stitch Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-skin the public site with the approved 8-Bit Legacy (Google Stitch) design system while the CMS data flow, page behaviour, and SEO stay exactly as they are.

**Architecture:** The site is a Go server-rendered MPA: renderers in `internal/site/*.go` build view models, Go templates in `internal/site/templates/` lay them out, one hand-written stylesheet `internal/site/assets/site.css` paints everything, and a `Bundle` (internal/site/assets.go) publishes assets under content-hashed URLs. The redesign rewrites the stylesheet and template markup; it touches two renderers (`landing.go` for home previews, `devlog.go` for filter counts) and adds one small page script (`devlog-filter.js`). SEO (`seo.go`, `schema.go`, title/meta pipeline in `templates.go` `render()`) is untouched.

**Tech Stack:** Go 1.26, `html/template`, plain CSS (no Tailwind on the public site), embedded WOFF2 fonts, vanilla JS page scripts, SQLite via `modernc.org/sqlite`, `go test` + vitest (`make test`).

**Spec:** `docs/superpowers/specs/2026-08-17-stitch-redesign-design.md` — read it before starting; this plan argues from it. Visual references: `STITCH/*/code.html` and `STITCH/8_bit_legacy/DESIGN.md`.

## Global Constraints

(From the spec — every task inherits these.)

- **Dark only.** No `prefers-color-scheme: light` block survives. `color-scheme: dark`.
- **No new runtime dependencies.** No Tailwind, no icon font, no Google Fonts CDN. Icons stay inline SVG.
- **No login UI.** No `LOGIN` button anywhere on the public site.
- **Header menu stays on the right**; the osd retract behaviour and `CHxx` ghost stay (do not touch `osd.js`).
- **SEO frozen.** `internal/site/seo.go`, `internal/site/schema.go`, and the `<head>` SEO tags (title, description, canonical, OG, Twitter, JSON-LD) are not edited. The only allowed `<head>` change: the two `theme-color` metas collapse to one `<meta name="theme-color" content="#101412" />`.
- **JS behaviour contracts frozen.** These attributes/classes must keep existing in templates: `data-osd`, `.osd--retracted`, `.osd-ghost`, `data-console*` (`data-console-drop`, `-stage`, `-screen`, `-idle`, `-start`, `-controls`, `-crt`, `-fullscreen`, `-bay`, `-cartridge`, `-led`, `-reset`, `-eject`, `-title`), `data-play-url`, `data-play-title`, `data-case-open`, `data-case-close`, `data-carousel-*`, `data-carousel-target`, `data-zoom-src`, `data-zoom-alt`, `data-zoom-dialog`, `data-zoom-image`, `data-zoom-close`, `data-contact-status`, `data-contact-submit`, `data-offline-game`. Classes `arcade.js`/`cases.js`/`carousel.js`/`osd.js` query may also be restyled but not renamed — check each script before renaming any class it touches.
- **Contact form contract frozen:** field names `name`, `subject`, `phone`, `email`, `message`, `wants_callback`, the honeypot field name, `method="post" action="/api/contact"`, `required`/`minlength` attributes.
- **Images only from the CMS** (`/media/...`). Never ship a Stitch placeholder URL.
- **Content copy rules:** CMS-managed copy comes from settings/repos. Hard-coded English chrome may move to the Stitch voice ("Game Vault", "The Library", "Press start"). Existing test-asserted strings that must survive verbatim: `No games published yet.`, `Nothing written up yet.`, `No awards to show just yet.` (check the exact strings in `games_test.go:167`, `devlog_test.go:79`, `awards_test.go:124` before editing copy near them).
- **Reduced motion:** every new decorative animation sits inside `@media (prefers-reduced-motion: no-preference)`.
- **Commits:** conventional prefixes (`feat:`, `fix:`, `docs:`, `test:`, `style:`) matching repo history, one commit per task, with `Co-Authored-By: Claude <noreply@anthropic.com>` footer.
- **Run tests with:** `go test ./internal/site/...` for fast loops, `go test ./...` and `make test` before each commit.

---

### Task 1: Capture the SEO baseline

Before any edit, snapshot the served `<head>` of every public page so Task 11 can prove SEO is unchanged.

**Files:**
- Create: `/tmp/pixabros-seo-baseline/` (outside the repo; throwaway)

- [ ] **Step 1: Build and run the server against a scratch data dir**

```bash
make build
export PIXABROS_DATA_DIR=$(mktemp -d)
export PIXABROS_DB_PATH="$PIXABROS_DATA_DIR/test.db"
./pixabros &
echo $! > /tmp/pixabros-seo-baseline.pid
sleep 1
curl -sf http://localhost:8080/ >/dev/null || { echo "server did not come up"; exit 1; }
```

- [ ] **Step 2: Save each page's head**

```bash
mkdir -p /tmp/pixabros-seo-baseline
for p in / /games /devlog /awards /contact /contact/sent /offline /no-such-page; do
  curl -s "http://localhost:8080$p" \
    | sed -n '/<head>/,/<\/head>/p' \
    | grep -Ev 'theme-color|stylesheet' \
    > "/tmp/pixabros-seo-baseline/$(echo $p | tr '/' '_').head"
done
kill $(cat /tmp/pixabros-seo-baseline.pid)
```

(The grep strips `theme-color` — the one intentional change — and the stylesheet link, whose URL is content-hashed and changes with every CSS edit.)

- [ ] **Step 3: Verify the baseline is non-empty**

```bash
wc -l /tmp/pixabros-seo-baseline/*.head
```

Expected: every file has a `<title>` line. No commit — this produces no repo changes.

---

### Task 2: Swap the typefaces

Replace Archivo + VT323 with Space Grotesk + Courier Prime. Public Sans stays.

**Files:**
- Create: `internal/site/assets/fonts/space-grotesk.woff2`, `internal/site/assets/fonts/courier-prime.woff2`, `internal/site/assets/fonts/courier-prime-700.woff2`, plus OFL licence files under `internal/site/assets/fonts/licences/`
- Delete: `internal/site/assets/fonts/archivo.woff2`, `internal/site/assets/fonts/vt323.woff2` (and their licence files, if named for those faces)
- Modify: `internal/site/assets/site.css:70-76` (font tokens), `site.css:126-149` (@font-face), `internal/site/shell.go:31-41` (shellAssets), `internal/site/assets_test.go:85-110`, `internal/site/shell_test.go:42`, `internal/site/assets/sw.test.js:41`

**Interfaces:**
- Produces: font token values used by every later CSS task: `--font-display: "Space Grotesk", ...`, `--font-sans: "Public Sans", ...`, `--font-osd: "Courier Prime", ...`. Asset names `fonts/space-grotesk.woff2`, `fonts/courier-prime.woff2`, `fonts/courier-prime-700.woff2`, `fonts/public-sans.woff2`.

- [ ] **Step 1: Write the failing test changes**

In `internal/site/assets_test.go`, update the face list in `TestBuild_PublishesFontsUnderStableNames` (line 96):

```go
for _, face := range []string{"space-grotesk", "public-sans", "courier-prime", "courier-prime-700"} {
```

In `internal/site/shell_test.go:42` update the expected font list to:

```go
"fonts/space-grotesk.woff2", "fonts/public-sans.woff2",
"fonts/courier-prime.woff2", "fonts/courier-prime-700.woff2",
```

In `internal/site/assets/sw.test.js:41` change the classified path to `"assets/build/fonts/space-grotesk.woff2"` (the test asserts the worker classifies a font URL as `asset`; any font name works, it just must be one that exists).

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/site/ -run 'TestBuild_PublishesFonts|TestShell' && npx vitest run internal/site/assets/sw.test.js
```

Expected: FAIL — the new font files do not exist yet.

- [ ] **Step 3: Download the font files**

All three are SIL Open Font License; licence text must ship beside them.

```bash
cd internal/site/assets/fonts
UA="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

# Space Grotesk is variable: one file covers 300-700.
curl -s -A "$UA" "https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300..700&display=swap" | grep -A6 "latin;" | grep -o "https://[^)]*woff2" | tail -1 | xargs curl -s -o space-grotesk.woff2

# Courier Prime has no variable cut: 400 and 700 are separate files.
curl -s -A "$UA" "https://fonts.googleapis.com/css2?family=Courier+Prime:wght@400&display=swap" | grep -A6 "latin;" | grep -o "https://[^)]*woff2" | tail -1 | xargs curl -s -o courier-prime.woff2
curl -s -A "$UA" "https://fonts.googleapis.com/css2?family=Courier+Prime:wght@700&display=swap" | grep -A6 "latin;" | grep -o "https://[^)]*woff2" | tail -1 | xargs curl -s -o courier-prime-700.woff2

curl -s -o licences/space-grotesk-OFL.txt "https://raw.githubusercontent.com/google/fonts/main/ofl/spacegrotesk/OFL.txt"
curl -s -o licences/courier-prime-OFL.txt "https://raw.githubusercontent.com/google/fonts/main/ofl/courierprime/OFL.txt"

rm archivo.woff2 vt323.woff2
# Remove the archivo/vt323 licence files too if present:
ls licences/ && rm -f licences/archivo* licences/vt323*
```

Verify each woff2 starts with `wOF2`: `file *.woff2` or `head -c4 space-grotesk.woff2`. If a download produced HTML instead of a font (Google returning different CSS than expected), open the css2 URL manually, pick the **latin** subset woff2 URL, and `curl` it — do not commit a broken file.

- [ ] **Step 4: Point the stylesheet at them**

In `site.css`, replace the token block at lines 74-76:

```css
  --font-display: "Space Grotesk", ui-sans-serif, system-ui, sans-serif;
  --font-sans: "Public Sans", ui-sans-serif, system-ui, sans-serif;
  --font-osd: "Courier Prime", ui-monospace, SFMono-Regular, monospace;
```

and replace the three @font-face blocks at lines 126-149 with:

```css
@font-face {
  font-family: "Space Grotesk";
  src: url("/assets/build/fonts/space-grotesk.woff2") format("woff2");
  font-weight: 300 700;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: "Public Sans";
  src: url("/assets/build/fonts/public-sans.woff2") format("woff2");
  font-weight: 100 900;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: "Courier Prime";
  src: url("/assets/build/fonts/courier-prime.woff2") format("woff2");
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: "Courier Prime";
  src: url("/assets/build/fonts/courier-prime-700.woff2") format("woff2");
  font-weight: 700;
  font-style: normal;
  font-display: swap;
}
```

Update the comment above the tokens (lines 70-73) to name the new faces: Space Grotesk for display, Public Sans for body, Courier Prime for terminal labels.

- [ ] **Step 5: Update the offline shell**

In `internal/site/shell.go`, replace the three font lines in `shellAssets` (lines 35-37) with:

```go
	"fonts/space-grotesk.woff2",
	"fonts/public-sans.woff2",
	"fonts/courier-prime.woff2",
	"fonts/courier-prime-700.woff2",
```

- [ ] **Step 6: Run all suites**

```bash
go test ./internal/site/... && npx vitest run
```

Expected: PASS (including `sw.test.js` and `offline.test.js`).

- [ ] **Step 7: Commit**

```bash
git add internal/site/assets/fonts internal/site/assets/site.css internal/site/shell.go internal/site/assets_test.go internal/site/shell_test.go internal/site/assets/sw.test.js
git commit -m "feat: set type in Space Grotesk and Courier Prime

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Design tokens, base styles, header and footer

Retune every token to the 8-Bit Legacy palette, go dark-only, and restyle the osd banner, footer, buttons, and focus ring. The old component styles further down `site.css` keep working — every old token name still exists, only its value changes.

**Files:**
- Modify: `internal/site/assets/site.css` (blocks `:root` lines 27-88, light block 90-124, base/typography/focus/buttons sections), `internal/site/templates/layout.html:41-42`, `internal/site/manifest.go:24`
- Test: `internal/site/manifest_test.go`, new test in `internal/site/assets_test.go`

**Interfaces:**
- Produces the token vocabulary every later task styles against (names are final — later tasks must not invent spellings):

```css
--color-bg: #101412;            /* page */
--color-surface: #1a1a1a;       /* console gray: cards, panels */
--color-surface-low: #191c1a;
--color-surface-panel: #1d211e; /* inset panels */
--color-border: #323632;        /* structural borders (surface-container-highest) */
--color-border-strong: #544338; /* cartridge/retro borders (outline-variant) */
--color-text: #e0e3de;
--color-text-muted: #dac2b3;
--color-accent: #f29f05;        /* pixel amber: primary actions */
--color-accent-strong: #ffb781; /* soft orange: links, hovers */
--color-accent-ink: #101412;    /* ink on an amber surface */
--color-input-bg: #0b0f0c;
--color-lamp: #e5312a;          /* the power lamp stays red */
--color-success: #22c55e;       /* crt green */
--color-warning: #f29f05;
--color-error: #ffb4ab;
--phosphor / --phosphor-bright / --phosphor-ink / --on-dark / --on-dark-muted: keep names, set to #ffb781 / #ffdcc4 / #101412 / #e0e3de / #dac2b3
--room-light: rgb(242 159 5 / 0.06);
--room-grille: rgb(224 227 222 / 0.02);
--step: 4px 4px 0 #000;  --step-sm: 2px 2px 0 #000;  --step-lg: 8px 8px 0 #000;
--space-page: 1.5rem; --header-height: 3.25rem; --width-content: 68rem;
--radius: 0;            /* square everywhere; chips may hardcode 2px */
```

- [ ] **Step 1: Write the failing tests**

Add to `internal/site/assets_test.go`:

```go
// The site is dark-only by design: the lit room was retired with the Stitch
// redesign. A light block sneaking back in would ship half a theme.
func TestStylesheet_IsDarkOnly(t *testing.T) {
	css, err := assetFS.ReadFile("assets/site.css")
	if err != nil {
		t.Fatalf("read site.css: %v", err)
	}
	if strings.Contains(string(css), "(prefers-color-scheme: light)") {
		t.Error("site.css still styles a light room; the redesign is dark-only")
	}
}
```

In `internal/site/manifest_test.go`, find the test that pins `themeColor` to the stylesheet's `--color-bg` (the file comment at `manifest.go:19` says a test holds the two together). Update both sides: `manifest.go:24` becomes `const themeColor = "#101412"` and the test's expected `--color-bg` becomes `#101412`.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/site/ -run 'TestStylesheet_IsDarkOnly|TestManifest'
```

Expected: FAIL (light block still present, colours still old).

- [ ] **Step 3: Rewrite the token blocks**

Replace `:root` (site.css lines 27-88) and delete the whole `@media (prefers-color-scheme: light)` block (lines 90-124). The new `:root`, including the file-top comment rewrite ("the page is a games console: dark plastic, amber accents, hard edges"):

```css
:root {
  /* The console. */
  --color-bg: #101412;
  --color-surface: #1a1a1a;
  --color-surface-low: #191c1a;
  --color-surface-panel: #1d211e;
  --color-border: #323632;
  --color-border-strong: #544338;

  --color-text: #e0e3de;
  --color-text-muted: #dac2b3;
  --color-accent-ink: #101412;
  --color-input-bg: #0b0f0c;

  --color-accent: #f29f05;
  --color-accent-strong: #ffb781;
  --color-lamp: #e5312a;

  --color-success: #22c55e;
  --color-warning: #f29f05;
  --color-error: #ffb4ab;

  /* What the tube and its furniture glow with. */
  --phosphor: #ffb781;
  --phosphor-bright: #ffdcc4;
  --phosphor-ink: #101412;
  --on-dark: #e0e3de;
  --on-dark-muted: #dac2b3;

  --room-light: rgb(242 159 5 / 0.06);
  --room-grille: rgb(224 227 222 / 0.02);

  --font-display: "Space Grotesk", ui-sans-serif, system-ui, sans-serif;
  --font-sans: "Public Sans", ui-sans-serif, system-ui, sans-serif;
  --font-osd: "Courier Prime", ui-monospace, SFMono-Regular, monospace;

  color-scheme: dark;

  --step-sm: 2px 2px 0 #000;
  --step: 4px 4px 0 #000;
  --step-lg: 8px 8px 0 #000;

  --space-page: 1.5rem;
  --header-height: 3.25rem;
  --width-content: 68rem;
  --radius: 0;
}
```

- [ ] **Step 4: Collapse the theme-color metas**

In `internal/site/templates/layout.html` replace lines 41-42 with:

```html
    <meta name="theme-color" content="#101412" />
```

Update the comment above it (lines 37-40) to say the site has one room, the dark one.

- [ ] **Step 5: Restyle base, headings, focus, buttons, osd, footer**

Rewrite these sections of `site.css` (keep the section comment banners, keep every class name — only rules change):

- **Headings** (~line 209): `font-family: var(--font-display); font-weight: 700; letter-spacing: -0.02em; text-transform: uppercase;` — headings shout in the Stitch manner. `h1` additionally `font-size: clamp(2rem, 5vw, 3rem); line-height: 1.1;`.
- **Focus** (~line 256): `:focus-visible { outline: 3px solid var(--color-accent); outline-offset: 2px; }`.
- **Buttons** (~line 840): the `.button` class becomes a console button — square (`border-radius: 0`), `border: 2px solid var(--color-border-strong); box-shadow: var(--step-sm); font-family: var(--font-osd); text-transform: uppercase; letter-spacing: 0.05em; background: var(--color-surface); color: var(--color-text);`. Hover swaps bg/text (`background: var(--color-accent); color: var(--color-accent-ink);`). Active: `transform: translate(2px, 2px); box-shadow: none;`. Add a filled variant later tasks link with:

```css
.button--primary {
  background: var(--color-accent);
  color: var(--color-accent-ink);
  font-weight: 700;
}
.button--primary:hover {
  background: var(--color-accent-strong);
}
```

(If the stylesheet already has a `--primary`/`--ghost` variant, fold these values into it instead of adding a second definition.)
- **`.osd`** (~line 274): keep sticky/retract mechanics and all class names. New skin: `background: var(--color-bg); border-bottom: 4px solid var(--color-border); box-shadow: var(--step);`. `.osd__inner` keeps its flex `space-between`.
- **`.osd__set`**: lamp (`--color-lamp`, unchanged shape) + `.osd__name` in `var(--font-display)`, `font-weight: 700`, `color: var(--color-accent)`, `text-transform: uppercase`, `letter-spacing: -0.02em`.
- **`.osd__channels`**: keep right alignment. `.osd__channel` labels in `var(--font-osd)`, `text-transform: uppercase`, `color: var(--on-dark-muted)`; hover/`focus-visible` → `color: var(--color-accent-strong)`. `.osd__number` stays, in `--color-accent` at reduced size.
- **Active channel** (`.osd__channel[aria-current="page"]`): drop the old selected-block `::before`; use `color: var(--color-accent); border-bottom: 4px solid var(--color-accent); padding-bottom: 2px;` — the Stitch underline.
- **Narrow screens** (~line 435): the numbers hide and the labels wrap onto a second row — `flex-wrap: wrap; row-gap: 4px;` on `.osd__channels`, no hamburger, no new JS.
- **`.osd-ghost`**: same fixed top-right placement, `var(--font-osd)`, `color: var(--color-accent)`, `border: 2px solid var(--color-border-strong); padding: 2px 6px; background: var(--color-bg);` — a small mono chip.
- **`.site-footer`**: `background: var(--color-surface-low)` is wrong at this stage if the token list above doesn't define it — it does (`#191c1a`); use `--color-surface-low` with `border-top: 4px solid var(--color-border)`. Copyright line in `var(--font-osd)`, `color: var(--color-accent)`, `text-transform: uppercase`. `.site-social__link`: square bordered chip — `border: 2px solid var(--color-border-strong); background: var(--color-surface); padding: 8px;` hover: `background: var(--color-accent); color: var(--color-accent-ink);` (the svg uses `currentColor`).

Do not touch any other section yet — old components inherit the new tokens and will look roughly right until their own tasks.

- [ ] **Step 6: Run the tests**

```bash
go test ./internal/site/... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Look at it**

```bash
make run
```

Walk `/` and `/games`: banner and footer read as Stitch, page bodies are token-shifted but structurally old. That is the expected intermediate state.

- [ ] **Step 8: Commit**

```bash
git add internal/site/assets/site.css internal/site/templates/layout.html internal/site/manifest.go internal/site/manifest_test.go internal/site/assets_test.go
git commit -m "feat: 8-Bit Legacy tokens, dark-only room, console chrome

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Retro utility classes

The shared vocabulary later tasks compose: step shadows, scanlines, chamfer, pixel dividers, recessed inputs. These are additions; nothing existing changes.

**Files:**
- Modify: `internal/site/assets/site.css` (new section after the Buttons section)

**Interfaces:**
- Produces (class names are final): `.step` / `.step-sm` / `.step-lg` (box-shadow only), `.scanlines` (overlay), `.chamfer` (clip-path), `.pixel-rule` (dashed divider), `.inset-field` (recessed input/textarea/select skin).

- [ ] **Step 1: Add the utilities**

```css
/* Retro utilities ---------------------------------------------------------- */

/* Depth is a solid offset block, never a blur. */
.step { box-shadow: var(--step); }
.step-sm { box-shadow: var(--step-sm); }
.step-lg { box-shadow: var(--step-lg); }

/* The tube's raster: a 4px-period stripe, painted as an overlay layer on top
   of whatever it covers. Pointer-events none always: it is decoration. */
.scanlines {
  background-image: repeating-linear-gradient(
    to bottom,
    transparent 0 2px,
    rgb(0 0 0 / 0.22) 2px 4px
  );
}

/* A cartridge's clipped top-right corner. */
.chamfer { clip-path: polygon(0 0, calc(100% - 16px) 0, 100% 16px, 100% 100%, 0 100%); }

/* A pixel-dashed divider: 4 on, 4 off. */
.pixel-rule {
  border: 0;
  border-top: 4px dashed var(--color-border);
}

/* A field is a hole in the panel: darker than its card, hard-edged, amber
   when it has focus. */
.inset-field {
  background: var(--color-input-bg);
  border: 2px solid var(--color-border);
  border-bottom-color: var(--color-border-strong);
  border-right-color: var(--color-border-strong);
  color: var(--color-text);
  font-family: var(--font-osd);
  padding: 10px 12px;
}
.inset-field::placeholder { color: var(--on-dark-muted); }
.inset-field:focus {
  outline: none;
  border-color: var(--color-accent);
}
```

- [ ] **Step 2: Verify the bundle still builds and tests pass**

```bash
go test ./internal/site/ -run TestBuild
```

Expected: PASS (the CSS still minifies).

- [ ] **Step 3: Commit**

```bash
git add internal/site/assets/site.css
git commit -m "feat: retro utility classes (step shadows, scanlines, chamfer)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: Landing renderer — System Log and Achievements data

The home page gains the latest two devlog posts and latest two awards. TDD against the real renderer.

**Files:**
- Modify: `internal/site/landing.go`, `internal/site/templates/landing.html`
- Test: `internal/site/landing_test.go`

**Interfaces:**
- Consumes: `s.publishedPosts()` (devlog.go:165), `s.awards.List("date", true)` (awards.go:38), `excerpt(source, limit)` (devlog.go:241), `lookupImage`, `yearOf` (awards.go:107), existing seed helpers `seedPost` (devlog_test.go:11), `seedAward` (site_test.go:66).
- Produces: `landingPage` fields `LogPosts []landingPostView`, `Achievements []landingAwardView`, `HasLog bool`, `HasAchievements bool`; tag list grows `devlog:list` + `award:list`. Template partial `{{template "landing-log" .}}` markup (unstyled this task; Task 6 styles it).

- [ ] **Step 1: Write the failing tests**

Append to `internal/site/landing_test.go`:

```go
func seedLandingPost(t *testing.T, conn *sql.DB, title, slug, publishedAt string, published bool) string {
	t.Helper()
	return seedPost(t, conn, title, slug, "Body of "+title, publishedAt, published, nil)
}

func TestRenderLanding_DeclaresDevlogAndAwardTags(t *testing.T) {
	conn := setupTestDB(t)
	_, tags := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{"devlog:list", "award:list"} {
		found := false
		for _, tag := range tags {
			if tag == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the landing page never declared the %q tag", want)
		}
	}
}

func TestRenderLanding_ShowsLatestTwoPostsAndAwards(t *testing.T) {
	conn := setupTestDB(t)
	seedLandingPost(t, conn, "Newest", "newest", "2026-08-01", true)
	seedLandingPost(t, conn, "Second", "second", "2026-07-01", true)
	seedLandingPost(t, conn, "Third", "third", "2026-06-01", true)
	seedLandingPost(t, conn, "Draft", "draft", "2026-08-02", false)
	seedAward(t, conn, "Best Game", "IndieCade", "2026-05-01", "", nil)
	seedAward(t, conn, "Best Audio", "PixelFest", "2025-05-01", "", nil)

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{"Newest", "Second", "Best Game", "Best Audio", "IndieCade"} {
		if !strings.Contains(html, want) {
			t.Errorf("landing page is missing %q", want)
		}
	}
	if strings.Contains(html, "Third") {
		t.Error("a third post leaked past the two-post budget")
	}
	if strings.Contains(html, ">Draft<") {
		t.Error("an unpublished post leaked onto the landing page")
	}
}

func TestRenderLanding_HidesLogAndAchievementsWhenEmpty(t *testing.T) {
	conn := setupTestDB(t)
	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if strings.Contains(html, "System log") {
		t.Error("the system log heading rendered with no posts")
	}
	if strings.Contains(html, "Achievements") {
		t.Error("the achievements heading rendered with no awards")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/site/ -run TestRenderLanding
```

Expected: the three new tests FAIL (`devlog:list` missing, titles missing); others pass.

- [ ] **Step 3: Extend the renderer**

In `internal/site/landing.go`:

Add view models beside `memberView`:

```go
// landingPostView is one entry of the home page's System Log: the two most
// recent posts, newest first.
type landingPostView struct {
	Title string
	Slug  string
	Date  string
	// Blurb is the markdown collapsed to plain text; posts have no summary
	// field and the card has no room for more than a taste.
	Blurb string
}

// landingAwardView is one entry of the home page's Achievements pair.
type landingAwardView struct {
	Title  string
	Issuer string
	Year   string
}
```

Extend `landingPage`:

```go
	LogPosts       []landingPostView
	Achievements   []landingAwardView
	HasLog         bool
	HasAchievements bool
```

In `renderLanding`, after `memberList` is fetched, add:

```go
	posts, err := s.publishedPosts()
	if err != nil {
		return nil, nil, err
	}
	logPosts := make([]landingPostView, 0, 2)
	for _, post := range posts {
		if len(logPosts) == 2 {
			break
		}
		logPosts = append(logPosts, landingPostView{
			Title: post.Title,
			Slug:  post.Slug,
			Date:  post.PublishedAt,
			Blurb: excerpt(post.ContentMarkdown, 120),
		})
	}

	awardList, err := s.awards.List("date", true)
	if err != nil {
		return nil, nil, fmt.Errorf("list awards: %w", err)
	}
	achievements := make([]landingAwardView, 0, 2)
	for _, award := range awardList {
		if len(achievements) == 2 {
			break
		}
		achievements = append(achievements, landingAwardView{
			Title:  award.Title,
			Issuer: award.Issuer,
			Year:   yearOf(award.Date),
		})
	}
```

Wire them into the `page := landingPage{...}` literal (`LogPosts: logPosts, Achievements: achievements`) and after the existing `Has*` lines:

```go
	page.HasLog = len(page.LogPosts) > 0
	page.HasAchievements = len(page.Achievements) > 0
```

Update the render function's doc comment (lines 99-102) to name the two new tags. Update the return at line 176:

```go
	return html, []string{homepageTag, gameListTag, memberListTag, siteSettingsTag, devlogListTag, awardsListTag}, nil
```

(`devlogListTag` and `awardsListTag` are already declared in `devlog.go:20` and `awards.go:89`.)

- [ ] **Step 4: Render them in the template**

In `internal/site/templates/landing.html`, insert between the sales section and the members section:

```html
  {{if or .Data.HasLog .Data.HasAchievements}}
    <div class="home-split">
      {{if .Data.HasLog}}
        <section class="section" aria-labelledby="log-heading">
          <h2 class="section__title" id="log-heading">System log</h2>
          <ul class="log-list">
            {{range .Data.LogPosts}}
              <li>
                <a class="log-entry" href="/devlog/{{.Slug}}">
                  {{if .Date}}<span class="log-entry__date">{{.Date}}</span>{{end}}
                  <span class="log-entry__title">{{.Title}}</span>
                  {{if .Blurb}}<span class="log-entry__blurb">{{.Blurb}}</span>{{end}}
                </a>
              </li>
            {{end}}
          </ul>
        </section>
      {{end}}
      {{if .Data.HasAchievements}}
        <section class="section" aria-labelledby="achievements-heading">
          <h2 class="section__title" id="achievements-heading">Achievements</h2>
          <ul class="medals">
            {{range .Data.Achievements}}
              <li class="medal">
                {{if .Year}}<span class="medal__year">{{.Year}}</span>{{end}}
                <span class="medal__title">{{.Title}}</span>
                <span class="medal__issuer">{{.Issuer}}</span>
              </li>
            {{end}}
          </ul>
        </section>
      {{end}}
    </div>
  {{end}}
```

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/site/ -run TestRenderLanding && go test ./internal/site/...
```

Expected: PASS, including the tag test from `TestRenderLanding_DeclaresEveryTagItDependsOn` — **that test (landing_test.go:46) must be updated** to add `"devlog:list": false, "award:list": false` to its `want` map, since it rejects unexpected tags.

- [ ] **Step 6: Commit**

```bash
git add internal/site/landing.go internal/site/templates/landing.html internal/site/landing_test.go
git commit -m "feat: home page system log and achievements from the CMS

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: Landing page Stitch layout

Hero, carousel cards, the new split section, sales and members — the whole home page in the new language.

**Files:**
- Modify: `internal/site/templates/landing.html` (hero), `internal/site/assets/site.css` (Hero, Portfolio carousel, Games on sale, Members sections + new `.home-split`, `.log-*`, `.medal*` styles), `internal/site/landing.go:186-206` (buildHero comment only)

**Interfaces:**
- Consumes: hero fields (`Hero.Logo`, `Hero.Slogan`, `Hero.Description`, `Hero.CTAText`, `Hero.CTALink`), slide/sale/member views unchanged, Task 4 utilities, Task 3 tokens.
- Produces: hero markup shape later CSS depends on: `.hero` grid, `.hero__status`, `.hero__copy`, `.hero__actions`, `.hero__brand`.

- [ ] **Step 1: Rewrite the hero markup**

Replace the hero section of `landing.html` (lines 2-22) with:

```html
  <section class="hero">
    {{if .Data.Hero.Logo.URL}}
      <div class="hero__brand step-lg">
        {{/* The stored mark is 512 square; the stylesheet draws it at its own
             width. Pixelated, because it is a pixel-art mark and a smooth
             scale would blur what the studio drew. */}}
        <img
          class="hero__logo"
          src="{{.Data.Hero.Logo.URL}}"
          alt="{{.Data.Hero.Logo.Alt}}"
          width="512"
          height="512"
        />
      </div>
    {{end}}
    <div class="hero__copy">
      <p class="hero__status" aria-hidden="true">
        <span class="hero__lamp"></span>
        System online
      </p>
      <h1>{{if .Data.Hero.Slogan}}{{.Data.Hero.Slogan}}{{else}}{{.Site.Name}}{{end}}</h1>
      {{if .Data.Hero.Description}}
        <p class="hero__description">{{.Data.Hero.Description}}</p>
      {{end}}
      <div class="hero__actions">
        {{if .Data.Hero.CTAText}}
          <a class="button button--primary" href="{{.Data.Hero.CTALink}}">{{.Data.Hero.CTAText}}</a>
        {{end}}
        <a class="button" href="/games">View catalog</a>
      </div>
    </div>
  </section>
```

In `landing.go`, replace the comment above `buildHero` (lines 186-191) — the old one explains why there is no featured image; the new design puts the CMS logo beside the copy, so say that:

```go
// buildHero assembles the site's opening statement. The logo from the
// homepage settings stands beside the copy, the way the Stitch hero pairs
// its artwork with the title; a studio that has not set one gets a text-only
// hero rather than an arbitrary picture.
```

- [ ] **Step 2: Style the hero and the split section**

Rewrite the Hero section of `site.css` and add the new classes:

```css
.hero {
  position: relative;
  display: grid;
  gap: var(--space-page);
  align-items: center;
  padding: var(--space-page);
  background: var(--color-surface);
  border: 4px solid var(--color-border-strong);
  box-shadow: var(--step);
  overflow: hidden;
}
.hero::after {
  content: "";
  position: absolute;
  inset: 0;
  background-image: repeating-linear-gradient(to bottom, transparent 0 2px, rgb(0 0 0 / 0.16) 2px 4px);
  pointer-events: none;
}
@media (min-width: 48rem) {
  .hero { grid-template-columns: 1fr auto; }
}
.hero__copy { position: relative; z-index: 1; display: grid; gap: 12px; justify-items: start; max-width: 34rem; }
.hero__status {
  display: inline-flex; align-items: center; gap: 8px;
  font-family: var(--font-osd); font-size: 12px; font-weight: 700;
  letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-success);
  background: var(--color-input-bg); border: 2px solid var(--color-border);
  padding: 4px 10px;
}
.hero__lamp { width: 8px; height: 8px; background: var(--color-success); }
.hero__description { color: var(--color-text-muted); font-size: 18px; line-height: 1.6; }
.hero__actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 8px; }
.hero__brand { position: relative; z-index: 1; background: var(--color-bg); border: 4px solid var(--color-border-strong); padding: 16px; }
.hero__logo { display: block; width: min(280px, 60vw); height: auto; image-rendering: pixelated; }

.home-split { display: grid; gap: var(--space-page); }
@media (min-width: 48rem) { .home-split { grid-template-columns: 1fr 1fr; } }
.log-list { display: grid; gap: 12px; list-style: none; padding: 0; margin: 0; }
.log-entry {
  display: grid; gap: 4px; padding: 12px;
  background: var(--color-surface); border: 4px solid var(--color-border-strong);
  color: inherit; text-decoration: none;
}
.log-entry:hover, .log-entry:focus-visible { border-color: var(--color-accent); }
.log-entry__date { font-family: var(--font-osd); font-size: 12px; text-transform: uppercase; color: var(--on-dark-muted); }
.log-entry__title { font-weight: 600; }
.log-entry:hover .log-entry__title { color: var(--color-accent-strong); }
.log-entry__blurb { color: var(--color-text-muted); font-size: 14px; line-height: 1.5; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.medals { display: grid; gap: 12px; grid-template-columns: 1fr 1fr; list-style: none; padding: 0; margin: 0; }
.medal {
  display: grid; gap: 4px; justify-items: center; text-align: center;
  min-height: 150px; align-content: center; padding: 16px;
  background: var(--color-surface); border: 4px solid var(--color-border-strong);
}
.medal__year { font-family: var(--font-osd); font-size: 12px; font-weight: 700; color: var(--color-accent); text-transform: uppercase; }
.medal__title { font-family: var(--font-display); font-weight: 600; text-transform: uppercase; }
.medal__issuer { font-family: var(--font-osd); font-size: 12px; color: var(--on-dark-muted); text-transform: uppercase; }
```

- [ ] **Step 3: Restyle the carousel slides, sale cards and members**

In the existing sections (keep every class name the scripts and `:has()` rules use):

- `.slide`: background `var(--color-surface)`, `border: 4px solid var(--color-border-strong)`, `box-shadow: var(--step)`. On `.slide:has(.slide__link:focus-visible), .slide:hover`: `border-color: var(--color-accent)`.
- `.slide__title`, `.sale-card__title`, `.member__name`: `font-family: var(--font-display); text-transform: uppercase;`.
- `.slide__image--cover, .slide__image--shot`: `image-rendering: pixelated;` — the artwork is pixel art.
- `.tag`: square chip — `border-radius: 2px` (the one sanctioned radius), `background: var(--color-surface-panel); border: 1px solid var(--color-border); font-family: var(--font-osd); font-size: 11px; text-transform: uppercase;`.
- `.jam-badge`: `background: var(--color-accent); color: var(--color-accent-ink); font-family: var(--font-osd); font-weight: 700; padding: 1px 6px;` — now a solid amber block.
- `.sale-card`: `.chamfer` shape (add `clip-path` directly), `border: 4px solid var(--color-border-strong)`, `box-shadow: var(--step-lg)`; hover lifts `-2px` and grows the shadow (keep the existing transition, it is already motion-safe guarded — check and keep its reduced-motion handling).
- `.member`: `background: var(--color-surface); border: 4px solid var(--color-border-strong);` with `.member__avatar` square, bordered `2px solid var(--color-border-strong)`, `image-rendering: pixelated`.
- `.section__title`: `font-family: var(--font-display); text-transform: uppercase;` followed by the existing rule line — recolour it to `var(--color-border)`.
- `.carousel__arrow`: console button (Task 3 button skin), square, `var(--font-osd)` label styling is irrelevant (svg) — just borders/shadow/active-press.

- [ ] **Step 4: Run all suites and look**

```bash
go test ./internal/site/... && make run
```

Expected: PASS. Home page reads as the Stitch reference (`STITCH/home_pixabros/screen.png`): status chip, amber display heading, amber primary button + ghost catalog button, pixelated logo in a framed box, bordered carousel cards, log/achievements split, retro sale/member cards. Existing tests asserting hero copy (`TestRenderLanding_UsesHomepageCopy`, `HidesTheCTAWithoutALink`) must still pass — they assert CMS strings only.

- [ ] **Step 5: Commit**

```bash
git add internal/site/templates/landing.html internal/site/assets/site.css internal/site/landing.go
git commit -m "feat: Stitch hero, cards and split sections on the home page

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 7: Games index — Game Vault

Restyle the console partial as the Stitch TV, the cartridge shelf, and turn the shelf into the Library grid.

**Files:**
- Modify: `internal/site/templates/arcade.html` (head copy only), `internal/site/templates/partials.html` (console partial: slot-bar decoration only), `internal/site/assets/site.css` (Console, Cartridges, CD cases, The open case sections)

**Interfaces:**
- Consumes: existing `arcadePage` view unchanged; all `data-console*`/`data-case*`/`data-play-*` hooks frozen.
- Produces: `.crt__slot` decorative bar above the screen (new class); `.case` grid gets Library proportions via CSS only.

- [ ] **Step 1: Adjust the page copy**

`arcade.html` lines 2-8: keep the `page-header` structure; set the `<h1>` to `Game vault` and the lead to `Slot a cartridge in to play right here in the browser, or open a case from the library below.` Change the shelf section `<h2>` (line 60) from `The shelf` to `The library`, its subtitle to `Every game we have made. Open a case to read about one.` Do not touch the empty-state string `No games published yet.` (asserted by `games_test.go:167`).

- [ ] **Step 2: Add the slot bar to the console partial**

In `partials.html`, inside `{{define "console"}}` immediately before `<div class="crt">` (line 6), add:

```html
  {{/* The mouth of the cartridge slot, drawn on the machine's face plate. */}}
  <div class="crt__slot" aria-hidden="true">
    <span class="crt__slot-label">Insert cartridge below</span>
  </div>
```

And after the `crt__controls` div (so it sits under the screen on the face plate), add the Stitch detail row:

```html
    <div class="crt__plate" aria-hidden="true">
      <span class="crt__led crt__led--on"></span>
      <span class="crt__plate-label">Power</span>
      <span class="crt__plate-channel">CH-03</span>
    </div>
```

Nothing else in the partial changes — every `data-console*` attribute stays byte-for-byte.

- [ ] **Step 3: Restyle the machine**

In `site.css`'s console section:

- `.console`: `background: var(--color-surface-panel); border: 4px solid var(--color-border); box-shadow: var(--step-lg); padding: 16px;` (it already has padding — keep the layout, change the skin).
- `.crt__slot`: `height: 48px; background: var(--color-input-bg); border-top: 4px solid var(--color-border); border-bottom: 4px solid var(--color-border); display: flex; align-items: center; justify-content: center; margin-bottom: 16px; position: relative; overflow: hidden;` plus a bottom lip `::after { content: ""; position: absolute; left: 0; right: 0; bottom: 0; height: 8px; background: #000; }`. `.crt__slot-label`: `font-family: var(--font-osd); font-size: 12px; font-weight: 700; letter-spacing: 0.12em; text-transform: uppercase; color: var(--on-dark-muted);`.
- `.crt`: `border: 4px solid var(--color-border-strong);` keep existing screen internals; `.crt__screen` keeps its aspect and children. If the current `.crt` rounds the tube (`border-radius`), set it to `0`.
- `.crt__plate`: `display: flex; align-items: center; gap: 8px; margin-top: 8px;`. `.crt__led--on`: `width: 10px; height: 10px; background: var(--color-success); box-shadow: 0 0 8px var(--color-success);`. `.crt__plate-label`, `.crt__plate-channel`: `font-family: var(--font-osd); font-size: 12px; text-transform: uppercase; color: var(--on-dark-muted);` with `.crt__plate-channel { margin-left: auto; letter-spacing: 0.12em; }`.
- `.nes`: the control deck — `background: var(--color-surface); border-top: 4px solid var(--color-border);`. `.nes__brand-mark`: `font-family: var(--font-display); font-weight: 700; text-transform: uppercase; color: var(--color-accent);`. `.nes__brand-sub`, `.nes__now`, `.crt__message`, `.crt__standby-note`: `var(--font-osd)`, uppercase where they already are. `.nes__button`: console button skin (Task 3 pattern). `.crt__start`: amber block button — `background: var(--color-accent); color: var(--color-accent-ink); font-family: var(--font-osd); text-transform: uppercase; border: 2px solid var(--color-border-strong); box-shadow: var(--step-sm); padding: 10px 20px;` with the Task 3 active press.

- [ ] **Step 4: Restyle the shelf and the library**

- `.cartridges` / `.cartridge`: keep dimensions and drag anatomy (grip, label, notch) — reskin only: `background: var(--color-surface); border: 2px solid var(--color-border);` hover/`.is-active` (check `arcade.js` for the class it toggles — use exactly that one): `border-color: var(--color-accent); transform: translateY(-4px);` with the transform only under `prefers-reduced-motion: no-preference`. `.cartridge__label img { image-rendering: pixelated; }`.
- `.cases`: `display: grid; gap: 24px; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); list-style: none; padding: 0; margin: 0;`
- `.case`: the Library box — `position: relative; aspect-ratio: 3 / 4; background: var(--color-surface); border: 4px solid var(--color-border-strong); box-shadow: var(--step-lg); overflow: hidden; cursor: pointer; padding: 0;` plus the gradient overlay: `::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, var(--color-bg) 0%, transparent 55%); pointer-events: none; }`. `.case__art`: `width: 100%; height: 100%; object-fit: cover; opacity: 0.85; image-rendering: pixelated; display: block;` — hover `opacity: 1`. `.case__front-title`: `position: absolute; left: 12px; right: 12px; bottom: 12px; z-index: 1; text-align: left; font-family: var(--font-display); font-weight: 700; text-transform: uppercase; color: var(--color-accent-strong);` — it must remain legible over the gradient.
- The open `.jewel` dialog: keep the 3D open/close mechanism (its `transform`/perspective rules and `cases.js` hooks), change only surfaces: `background: var(--color-surface); border: 4px solid var(--color-border-strong); box-shadow: var(--step-lg);`. `.jewel__title` → display font, uppercase. `.jewel__price` → `color: var(--color-success); font-family: var(--font-osd);`. `.jewel__more` → console button.

- [ ] **Step 5: Run suites and look**

```bash
go test ./internal/site/... && make run
```

Expected: PASS — `games_test.go` asserts content and hooks, all of which are untouched. `/games` shows the slot bar, the TV with bezel and plate, the cartridge shelf, and the 3/4 library boxes against `STITCH/games_pixabros/screen.png`.

- [ ] **Step 6: Commit**

```bash
git add internal/site/templates/arcade.html internal/site/templates/partials.html internal/site/assets/site.css
git commit -m "feat: Game Vault console, cartridge shelf and library grid

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 8: Game detail page

**Files:**
- Modify: `internal/site/templates/game.html` (minor: fact panel structure stays), `internal/site/assets/site.css` (Game detail, Stores, screenshots, offline control sections)

**Interfaces:**
- Consumes: `gamePage` view unchanged; `data-offline-game` div untouched (offline.js builds its own DOM — style what it builds by its existing class names; read `internal/site/assets/offline.js` for the class names it generates before writing CSS).

- [ ] **Step 1: Restyle the page**

- `.crumbs`: `font-family: var(--font-osd); font-size: 13px; text-transform: uppercase; color: var(--on-dark-muted);` links `color: var(--color-accent-strong);` — the terminal breadcrumb.
- `.page-header` on this page: `h1` already display/uppercase from Task 3; add `text-shadow: 4px 4px 0 #000;` for the Stitch drop block.
- `.game-facts dt`: `font-family: var(--font-osd); font-size: 12px; text-transform: uppercase; color: var(--on-dark-muted);` — `dd` in amber (`var(--color-accent)`).
- `.game-price`: `color: var(--color-success); font-family: var(--font-osd); font-size: 18px;`.
- `.store-link`: console button row — `border: 2px solid var(--color-border-strong); background: var(--color-surface); padding: 10px 14px;` hover invert to amber; keeps `brand-icon` svg at `currentColor`.
- `.shot`: `border: 2px solid var(--color-border);` hover `border-color: var(--color-accent);` — zoom affordance unchanged.
- Offline control: whatever `offline.js` renders (buttons/progress), give its progress bar the block style — if it draws a fill element, `background: var(--color-success);` on `var(--color-input-bg)` with a `2px solid var(--color-border)` track; verify actual class names in the script first and restyle those, never rename them.
- `.prose`: keep widths; recolour links to `var(--color-accent-strong)` with `text-decoration: underline`; `prose img` and `prose table` borders to `var(--color-border)`; headings inherit Task 3 display styles automatically.

- [ ] **Step 2: Run suites and look**

```bash
go test ./internal/site/... && make run
```

Expected: PASS (`games_test.go` game-page tests assert hooks/copy, untouched). Open a playable game's page: console shows the slotted cartridge, press-start flow works, offline widget styled.

- [ ] **Step 3: Commit**

```bash
git add internal/site/assets/site.css internal/site/templates/game.html
git commit -m "feat: retro game detail page

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 9: Devlog index — terminal, featured post, filter sidebar

Renderer data first (TDD), then the template and a small client-side filter script.

**Files:**
- Modify: `internal/site/devlog.go`, `internal/site/templates/devlog.html`, `internal/site/assets/site.css` (new Devlog section), new `internal/site/assets/devlog-filter.js`
- Test: `internal/site/devlog_test.go`

**Interfaces:**
- Consumes: `postSummary` (add `GameSlug`, `Year` fields), `seedPost`, `seedGame`, `gameNames()`.
- Produces: `devlogIndexPage` fields `Featured postSummary` (first post), `Rest []postSummary`, `Directories []directoryView` (`{Title, Slug, Count}`), `Archive []archiveView` (`{Year, Count}`); template rows carry `data-game="{{.GameSlug}}" data-year="{{.Year}}"`; sidebar links carry `data-filter-game` / `data-filter-year`; script registered as `devlog-filter.js` and listed in `renderDevlogIndex`'s `Scripts`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/site/devlog_test.go`:

```go
func seedPostFor(t *testing.T, conn *sql.DB, title, slug, publishedAt string, published bool, gameID *string) string {
	t.Helper()
	return seedPost(t, conn, title, slug, "Body", publishedAt, published, gameID)
}

func renderDevlogIndexPage(t *testing.T, s *Site) string {
	t.Helper()
	html, _, err := s.renderDevlogIndex(PageDevlog)
	if err != nil {
		t.Fatalf("renderDevlogIndex() error = %v", err)
	}
	return string(html)
}

func TestRenderDevlogIndex_BuildsDirectoriesAndArchive(t *testing.T) {
	conn := setupTestDB(t)
	grimID := seedGame(t, conn, "Grimoire", "grimoire", true, false, "")
	seedPostFor(t, conn, "First", "first", "2026-05-01", true, &grimID)
	seedPostFor(t, conn, "Second", "second", "2025-05-01", true, nil)
	seedPostFor(t, conn, "Third", "third", "2025-04-01", true, &grimID)

	html := renderDevlogIndexPage(t, newTestSite(t, conn))

	for _, want := range []string{
		`data-filter-game=grimoire`, "Grimoire", "[2]",
		`data-filter-year=2025`, "[2]", `data-filter-year=2026`, "[1]",
		`data-game=grimoire`, `data-year=2025`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("devlog index is missing %q", want)
		}
	}
	if !strings.Contains(html, "devlog-filter.js") {
		t.Error("the filter script is not on the page")
	}
}

func TestRenderDevlogIndex_NoSidebarWithoutTwoBuckets(t *testing.T) {
	conn := setupTestDB(t)
	seedPostFor(t, conn, "Only One", "only-one", "2026-05-01", true, nil)

	html := renderDevlogIndexPage(t, newTestSite(t, conn))

	if strings.Contains(html, "Directories") {
		t.Error("directories rendered with a single unfiled post")
	}
}
```

(The exact quoting in `want` must match the minified output — run once, read the failure output, and adjust attribute-order quoting to what the minifier emits; assert the substance, not the serialisation.)

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/site/ -run TestRenderDevlogIndex
```

Expected: FAIL.

- [ ] **Step 3: Extend the renderer**

In `devlog.go`, extend `postSummary`:

```go
type postSummary struct {
	Title    string
	Slug     string
	Date     string
	GameName string
	// GameSlug and Year drive the sidebar's client-side filter; a post with
	// neither can only be found by scrolling, which is what it falls back to.
	GameSlug string
	Year     string
	Image    imageView
}
```

Add bucket views and change the page struct:

```go
type directoryView struct {
	Title string
	Slug  string
	Count int
}

type archiveView struct {
	Year  string
	Count int
}

type devlogIndexPage struct {
	Featured   postSummary
	Rest       []postSummary
	Directories []directoryView
	Archive    []archiveView
}
```

In `renderDevlogIndex`, while building summaries also record `GameSlug` (from `gameNames`) and `Year` (`yearOf(post.PublishedAt)`). Then assemble the page:

```go
	directories := make([]directoryView, 0)
	dirCounts := map[string]int{}
	dirTitles := map[string]string{}
	archive := make([]archiveView, 0)
	yearCounts := map[string]int{}
	for _, summary := range summaries {
		if summary.GameSlug != "" {
			if _, seen := dirTitles[summary.GameSlug]; !seen {
				directories = append(directories, directoryView{Title: summary.GameName, Slug: summary.GameSlug})
				dirTitles[summary.GameSlug] = summary.GameName
			}
			dirCounts[summary.GameSlug]++
		}
		if summary.Year != "" && yearCounts[summary.Year] == 0 {
			archive = append(archive, archiveView{Year: summary.Year})
		}
		if summary.Year != "" {
			yearCounts[summary.Year]++
		}
	}
	for i := range directories {
		directories[i].Count = dirCounts[directories[i].Slug]
	}
	for i := range archive {
		archive[i].Count = yearCounts[archive[i].Year]
	}

	var featured postSummary
	var rest []postSummary
	if len(summaries) > 0 {
		featured = summaries[0]
		rest = summaries[1:]
	}
	// A sidebar with one bucket is a list calling itself a filter.
	if len(directories) < 2 {
		directories = nil
	}
	if len(archive) < 2 {
		archive = nil
	}
```

Data becomes `devlogIndexPage{Featured: featured, Rest: rest, Directories: directories, Archive: archive}`, and `Scripts` gains `s.renderer.bundle.URL("devlog-filter.js")`. Update the doc comment's tag note (unchanged tags — `devlog:list`, `game:list`, `site_settings` already declared — no new tags needed since filtering is client-side).

- [ ] **Step 4: Write the template**

Replace `devlog.html`'s `main` with:

```html
{{define "main"}}
  <div class="page-header">
    <h1><span class="page-header__cursor" aria-hidden="true">&gt;</span> Devlog terminal</h1>
    <p class="page-header__lead">
      System status: online. Notes from the workshop, newest first.
    </p>
  </div>

  {{if .Data.Featured.Slug}}
    <div class="devlog-layout">
      <div class="devlog-feed">
        <a class="post-feature" href="/devlog/{{.Data.Featured.Slug}}"
           data-game="{{.Data.Featured.GameSlug}}" data-year="{{.Data.Featured.Year}}">
          {{if .Data.Featured.Image.URL}}
            <span class="post-feature__media">
              <img src="{{.Data.Featured.Image.URL}}" alt="" loading="lazy" />
            </span>
          {{end}}
          <span class="post-feature__body">
            {{if .Data.Featured.Date}}<span class="post-row__meta-line">{{formatDate .Data.Featured.Date}}</span>{{end}}
            {{if .Data.Featured.GameName}}<span class="post-row__meta-line">{{.Data.Featured.GameName}}</span>{{end}}
            <span class="post-feature__title">{{.Data.Featured.Title}}</span>
            <span class="post-feature__read">Read protocol</span>
          </span>
        </a>

        {{if .Data.Rest}}
          <ul class="post-list">
            {{range .Data.Rest}}
              <li>
                <a class="post-row" href="/devlog/{{.Slug}}"
                   data-game="{{.GameSlug}}" data-year="{{.Year}}">
                  {{if .Image.URL}}
                    <img class="post-row__thumb" src="{{.Image.URL}}" alt="" loading="lazy" />
                  {{else}}
                    <span class="post-row__thumb post-row__thumb--empty" aria-hidden="true"></span>
                  {{end}}
                  <span class="post-row__text">
                    <span class="post-row__meta-line">
                      {{if .Date}}<span>{{formatDate .Date}}</span>{{end}}
                      {{if .GameName}}<span class="post-row__game">{{.GameName}}</span>{{end}}
                    </span>
                    <span class="post-row__title">{{.Title}}</span>
                  </span>
                </a>
              </li>
            {{end}}
          </ul>
        {{end}}

        <p class="empty-note" data-filter-empty hidden>Nothing in this directory.</p>
      </div>

      {{if or .Data.Directories .Data.Archive}}
        <aside class="devlog-side" data-devlog-filters>
          {{if .Data.Directories}}
            <nav class="side-panel" aria-label="Filter by game">
              <h2 class="side-panel__title">Directories</h2>
              <ul class="side-list">
                <li><button class="side-list__item is-active" type="button" data-filter-game="">&#9656; All posts</button></li>
                {{range .Data.Directories}}
                  <li>
                    <button class="side-list__item" type="button" data-filter-game="{{.Slug}}">
                      {{.Title}} <span class="side-list__count">[{{.Count}}]</span>
                    </button>
                  </li>
                {{end}}
              </ul>
            </nav>
          {{end}}
          {{if .Data.Archive}}
            <nav class="side-panel" aria-label="Filter by year">
              <h2 class="side-panel__title">Archives</h2>
              <ul class="side-archive">
                {{range .Data.Archive}}
                  <li><button class="side-archive__item" type="button" data-filter-year="{{.Year}}">{{.Year}} ({{.Count}})</button></li>
                {{end}}
              </ul>
            </nav>
          {{end}}
        </aside>
      {{end}}
    </div>
  {{else}}
    <p class="empty-note">Nothing written up yet.</p>
  {{end}}
{{end}}
```

(`Nothing written up yet.` is test-asserted — keep it verbatim. Existing tests asserting `post-row` classes and titles keep passing; run them and fix any markup-coupled assertion deliberately.)

- [ ] **Step 5: Write the filter script**

Create `internal/site/assets/devlog-filter.js`:

```js
// The devlog sidebar filter. The page is statically rendered, so filtering is
// purely presentational: rows are hidden, never re-fetched. Without this
// script the sidebar buttons do nothing and every post stays listed.
(() => {
  const root = document.querySelector("[data-devlog-filters]");
  const rows = Array.from(document.querySelectorAll("[data-game], [data-year]"));
  const empty = document.querySelector("[data-filter-empty]");
  if (!root || rows.length === 0) return;

  let game = "";
  let year = "";

  const matches = (row) =>
    (!game || row.dataset.game === game) && (!year || row.dataset.year === year);

  const apply = () => {
    let shown = 0;
    for (const row of rows) {
      const hit = matches(row);
      row.hidden = !hit;
      if (hit) shown++;
    }
    if (empty) empty.hidden = shown > 0;
  };

  const setActive = (clicked) => {
    for (const button of root.querySelectorAll("[data-filter-game]")) {
      button.classList.toggle("is-active", button === clicked);
    }
  };

  for (const button of root.querySelectorAll("[data-filter-game]")) {
    button.addEventListener("click", () => {
      game = button.dataset.filterGame || "";
      setActive(button);
      apply();
    });
  }
  for (const button of root.querySelectorAll("[data-filter-year]")) {
    button.addEventListener("click", () => {
      year = year === button.dataset.filterYear ? "" : button.dataset.filterYear;
      button.classList.toggle("is-active", year !== "");
      apply();
    });
  }
})();
```

- [ ] **Step 6: Style the page**

New `site.css` section (replace the old devlog list styles that this markup obsoletes — the old `.post-row__link` rules go, `.post-row` classes are restyled):

```css
/* Devlog ------------------------------------------------------------------- */
.devlog-layout { display: grid; gap: var(--space-page); }
@media (min-width: 56rem) { .devlog-layout { grid-template-columns: minmax(0, 2fr) minmax(220px, 1fr); align-items: start; } }
.page-header__cursor { color: var(--color-accent); }

.post-feature {
  display: grid; gap: 0; text-decoration: none; color: inherit;
  background: var(--color-surface-low); border: 4px solid var(--color-border);
  box-shadow: var(--step-lg);
}
.post-feature:hover, .post-feature:focus-visible { border-color: var(--color-accent-strong); }
.post-feature__media img { display: block; width: 100%; height: auto; max-height: 260px; object-fit: cover; filter: grayscale(1); image-rendering: pixelated; }
.post-feature:hover .post-feature__media img { filter: none; }
.post-feature__body { display: grid; gap: 6px; padding: 16px; }
.post-feature__title { font-family: var(--font-display); font-weight: 600; font-size: 24px; line-height: 1.2; text-transform: uppercase; }
.post-feature__read { font-family: var(--font-osd); font-size: 14px; text-transform: uppercase; color: var(--color-accent); border-bottom: 2px solid var(--color-accent); width: max-content; padding-bottom: 2px; }

.post-list { display: grid; gap: 12px; list-style: none; padding: 0; margin: 0; }
.post-row {
  display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 0;
  text-decoration: none; color: inherit;
  background: var(--color-surface-low); border: 2px solid var(--color-border); box-shadow: var(--step);
}
.post-row:hover, .post-row:focus-visible { border-color: var(--color-accent-strong); }
.post-row__thumb { width: 100%; height: 100%; object-fit: cover; filter: grayscale(1); }
.post-row__thumb--empty { background: var(--color-surface-panel); }
.post-row:hover .post-row__thumb { filter: none; }
.post-row__text { display: grid; gap: 4px; padding: 12px; align-content: center; }
.post-row__meta-line { display: flex; gap: 12px; font-family: var(--font-osd); font-size: 12px; text-transform: uppercase; color: var(--on-dark-muted); }
.post-row__title { font-weight: 600; }
.post-row:hover .post-row__title { color: var(--color-accent-strong); }

.devlog-side { display: grid; gap: 16px; }
.side-panel { background: var(--color-surface-low); border: 2px solid var(--color-border); box-shadow: var(--step); }
.side-panel__title { font-family: var(--font-osd); font-size: 14px; text-transform: uppercase; letter-spacing: 0.1em; background: var(--color-border); color: var(--on-dark); padding: 8px 12px; }
.side-list { list-style: none; margin: 0; padding: 8px; display: grid; }
.side-list__item { all: unset; display: flex; gap: 8px; width: 100%; box-sizing: border-box; padding: 8px; cursor: pointer; font-family: var(--font-osd); font-size: 14px; color: var(--on-dark-muted); }
.side-list__item:hover { color: var(--color-accent); background: var(--color-surface-panel); }
.side-list__item.is-active { color: var(--color-accent); }
.side-list__item.is-active::before { content: "\203A"; }
.side-list__count { margin-left: auto; color: var(--color-border); }
.side-archive { list-style: none; margin: 0; padding: 12px; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.side-archive__item { all: unset; box-sizing: border-box; text-align: center; border: 2px solid var(--color-border); padding: 8px; cursor: pointer; font-family: var(--font-osd); font-size: 12px; text-transform: uppercase; color: var(--on-dark-muted); }
.side-archive__item:hover, .side-archive__item.is-active { border-color: var(--color-accent); color: var(--color-accent); }
```

- [ ] **Step 7: Run suites, look, no-JS check**

```bash
go test ./internal/site/... && make run
```

Expected: PASS. With JS: sidebar filters hide rows, empty note appears when a combination matches nothing. With JS blocked (`curl` the page or disable JS in the browser): full list, sidebar buttons inert. Compare against `STITCH/devlog_pixabros_dual_theme/screen.png`.

- [ ] **Step 8: Commit**

```bash
git add internal/site/devlog.go internal/site/templates/devlog.html internal/site/assets/devlog-filter.js internal/site/assets/site.css internal/site/devlog_test.go
git commit -m "feat: devlog terminal with filterable directories and archives

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 10: Devlog post, awards, contact and the small pages

**Files:**
- Modify: `internal/site/templates/awards.html` (card markup order), `internal/site/templates/contact.html` (two-column layout), `internal/site/assets/site.css` (Awards, Post/prose, Form, error page sections), templates `contact-sent.html`, `404.html`, `offline.html` (copy unchanged, wrapped in existing classes)

**Interfaces:**
- Consumes: Task 3/4 tokens and utilities; `SiteChrome.Email` and `.Site.LinkViews` for the contact sidebar; form contract frozen.

- [ ] **Step 1: Awards — Stitch card markup**

In `awards.html`, reorder each `li.award` to the Stitch card shape (keep the zoom button, `data-zoom-*` hooks, and all field usage; picture area gets `chamfer` styling via CSS):

```html
          <li class="award chamfer">
            {{/* ...the existing picture button or placeholder, unchanged... */}}
            <div class="award__body">
              {{if .Year}}<p class="award__year">{{.Year}}</p>{{end}}
              <h2 class="award__title">{{.Title}}</h2>
              <p class="award__issuer">{{.Issuer}}</p>
              {{if .Date}}<p class="award__date">{{formatDate .Date}}</p>{{end}}
              {{if .Link}}<a class="award__link" href="{{.Link}}" rel="noopener">Read more</a>{{end}}
            </div>
          </li>
```

(Only the `chamfer` class is added and the year stays first — the current order already matches; keep whatever minimal diff achieves the Stitch card look.)

CSS: `.awards` grid `repeat(auto-fill, minmax(260px, 1fr))`; `.award`: `background: var(--color-surface-low); border: 4px solid var(--color-border); box-shadow: var(--step);` hover grows the shadow and lifts 2px (reduced-motion guarded); `.award__picture`: `background: var(--color-surface-panel); border-bottom: 4px solid var(--color-border); height: 190px; display: flex; align-items: center; justify-content: center; padding: 16px;` images `object-fit: contain; height: 100%; filter: drop-shadow(4px 4px 0 #000); image-rendering: pixelated;`. `.award__year`: mono amber chip `background: var(--color-surface-panel); border: 2px solid var(--color-border); display: inline-block; padding: 2px 8px;`. `.award__title`: display font uppercase. `.award__issuer`: mono muted uppercase with a `border-top: 2px solid var(--color-border); padding-top: 8px; margin-top: auto;` pinned to the card foot (make `.award__body` a flex column with `flex: 1`). Page header gains the amber left rule: `.page-header { border-left: 4px solid var(--color-accent); padding-left: 20px; }` scoped to `awards.html`'s own page class if one exists — otherwise a `.awards-page .page-header` wrapper class added to the template's top element.

- [ ] **Step 2: Contact — Comm Link layout**

Rewrite `contact.html`'s `main` structure (field names, `required`, `minlength`, honeypot, status/submit hooks all byte-identical — only wrappers, labels' voice, and classes move):

```html
{{define "main"}}
  <div class="comm">
    <div class="comm__side">
      <div class="comm-card step">
        <h1>Comm link</h1>
        <p class="comm-card__lead">
          Transmission initiated. Questions, press or collaborations -- write
          to us and we will read it.
        </p>
        {{if .Site.Email}}
          <p class="comm-card__row"><span class="comm-card__key">Email</span> {{.Site.Email}}</p>
        {{end}}
      </div>
      {{if .Site.LinkViews}}
        <div class="comm-card comm-card--nodes step">
          <h2>Network nodes</h2>
          <ul class="comm-nodes">
            {{range .Site.LinkViews}}
              <li><a class="comm-node" href="{{.URL}}" rel="me noopener">{{.Label}}</a></li>
            {{end}}
          </ul>
        </div>
      {{end}}
    </div>

    <form class="form form-card form-card--signal step-lg" method="post" action="/api/contact">
      {{/* every existing label/input block from the current contact.html,
             unchanged in name/autocomplete/required/minlength, with spans
             rewritten to the Stitch voice:
             "Your name" -> "Player name", "Email" -> "Player email",
             "Phone" -> "Player phone", "Subject" -> "Quest objective",
             "Message" -> "Message data"; inputs gain class "inset-field"
             alongside field__input; the honeypot block and
             data-contact-status / data-contact-submit hooks stay as they are */}}
      <button class="button button--primary" type="submit" data-contact-submit>Press start</button>
    </form>
  </div>
{{end}}
```

Copy the existing field blocks verbatim from the current `contact.html` (lines 16-92) into the marked slot — do not retype them; move them. CSS: `.comm { display: grid; gap: var(--space-page); } @media (min-width: 56rem) { .comm { grid-template-columns: minmax(240px, 1fr) minmax(0, 2fr); align-items: start; } }`; `.comm-card { background: var(--color-surface); border: 4px solid var(--color-border-strong); padding: 20px; display: grid; gap: 12px; }`; `.comm-card__lead { border-left: 4px solid var(--color-accent); padding-left: 12px; color: var(--color-text-muted); }`; `.comm-card__key, .comm-nodes, .comm-node`: mono uppercase; `.comm-node`: bordered chip like `.site-social__link`; the signal form header: `.form-card--signal { border: 4px solid var(--color-border-strong); background: var(--color-surface); padding: 24px; }` with a form-top bar `<h2>Signal input</h2>` — add that `<h2 class="signal-title"><span class="signal-cursor" aria-hidden="true">_</span> Signal input</h2>` as the first child inside the form, styled `font-family: var(--font-display); text-transform: uppercase; border-bottom: 4px solid var(--color-border); padding-bottom: 12px;` and `.signal-cursor { color: var(--color-accent); animation: signal-blink 1.1s steps(2) infinite; }` with

```css
@media (prefers-reduced-motion: no-preference) {
  @keyframes signal-blink { 50% { opacity: 0; } }
}
@media (prefers-reduced-motion: reduce) {
  .signal-cursor { animation: none; }
}
```

`contact-sent.html`: keep copy; add classes `comm-card step` to its `form-card--centred` wrapper so it picks up the panel skin.

- [ ] **Step 3: Devlog post + 404/offline**

- `devlog-post.html`: no structural change — style `.post__meta` mono/uppercase, `.post__hero` bordered `4px solid var(--color-border-strong)` with `image-rendering: pixelated`, `.prose` inherits Task 8 styling.
- `404.html` / `offline.html`: copy unchanged; restyle `.error-page__code` to `font-family: var(--font-display); font-weight: 700; font-size: clamp(4rem, 12vw, 7rem); color: var(--color-accent); text-shadow: 6px 6px 0 #000;` and the page bodies pick up the token shift automatically.

- [ ] **Step 4: Run suites and look**

```bash
go test ./internal/site/... && go test ./... && make test && make run
```

Expected: all PASS. `/awards` reads as chamfered trophy cards; `/contact` is Comm Link; submitting the form still lands on `/contact/sent` (test by hand once with `make run`).

- [ ] **Step 5: Commit**

```bash
git add internal/site/templates internal/site/assets/site.css
git commit -m "feat: awards gallery, comm link contact and retro small pages

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 11: Sweep, dead CSS removal and full verification

**Files:**
- Modify: `internal/site/assets/site.css` (remove sections no template references)

- [ ] **Step 1: Find dead rules**

For every class defined in `site.css`, check it is referenced by a template, a page script (`arcade.js`, `cases.js`, `carousel.js`, `osd.js`, `contact.js`, `lightbox.js`, `offline.js`, `devlog-filter.js`), or another CSS rule:

```bash
grep -o '^\.[a-z][a-z0-9_-]*' internal/site/assets/site.css | sort -u | while read -r cls; do
  name=${cls#.}
  used=$(grep -rlF "$name" internal/site/templates internal/site/assets \
        --include='*.html' --include='*.js' | grep -v 'site.css$')
  if [ -z "$used" ] && ! grep -v '^\.' internal/site/assets/site.css | grep -qF "$name"; then
    echo "possibly dead: $name"
  fi
done
```

(Heuristic — verify each hit by eye before deleting. Classes toggled by scripts (`osd--retracted`, `is-active`, `lightbox--open`, console state classes) are referenced from JS and must stay.)

Delete confirmed-dead sections (expected: old carousel-internals the restyle replaced, old `.post-row__link`, any hero rule the new markup dropped).

- [ ] **Step 2: Full suites**

```bash
go test ./... && make test
```

Expected: PASS.

- [ ] **Step 3: SEO head diff**

Re-run Task 1's capture into a second directory and diff:

```bash
export PIXABROS_DATA_DIR=$(mktemp -d)
export PIXABROS_DB_PATH="$PIXABROS_DATA_DIR/test.db"
./pixabros &
echo $! > /tmp/pixabros-seo-after.pid
sleep 1
mkdir -p /tmp/pixabros-seo-after
for p in / /games /devlog /awards /contact /contact/sent /offline /no-such-page; do
  curl -s "http://localhost:8080$p" \
    | sed -n '/<head>/,/<\/head>/p' \
    | grep -Ev 'theme-color|stylesheet' \
    > "/tmp/pixabros-seo-after/$(echo $p | tr '/' '_').head"
done
kill $(cat /tmp/pixabros-seo-after.pid)
diff -r /tmp/pixabros-seo-baseline /tmp/pixabros-seo-after
```

Expected: **no differences at all.** Any diff is a spec violation — fix it before proceeding.

- [ ] **Step 4: Behaviour pass (manual, `make run`)**

- `/`: carousel arrows/dots/thumbnail hover work with no JS and with JS.
- `/games`: cartridge click AND drag load the TV; press-start only loads on demand; CRT toggle and fullscreen work; a case opens its dialog; dialog links to the game page.
- A playable game page: offline download widget works; screenshots zoom.
- `/devlog`: filters work; with JS off, full list.
- `/contact`: submit → `/contact/sent`; honeypot field still hidden.
- Keyboard: tab through header, console controls, dialogs — amber focus everywhere.
- `prefers-reduced-motion` (devtools emulation): no blinking cursor, no hover translations.
- Phone width (~390px): header wraps to a second row, grids collapse to one column, nothing scrolls horizontally.

- [ ] **Step 5: Visual pass against the references**

Open each `STITCH/*/screen.png` beside the live page and confirm the family resemblance page by page: home, games, devlog, awards, contact. Differences in *content* are expected (ours is real CMS data); differences in *structure, palette, typography, borders* are bugs.

- [ ] **Step 6: Commit**

```bash
git add internal/site/assets/site.css
git commit -m "style: drop styles the Stitch redesign left behind

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Self-review notes

- Spec coverage: dark-only (T3), fonts (T2), tokens/typography (T3), utilities (T4), header/footer (T3), hero+slider+log/achievements+sales/members (T5/T6), games vault+console+shelf+library+jewel (T7), game detail+offline widget (T8), devlog+sidebar filter (T9), devlog post/awards/contact/sent/404/offline (T10), theme-color+manifest (T3), SEO diff (T1/T11), dead-CSS sweep (T11), regen tags (T5), reduced-motion (T3/T4/T9/T10), no-JS passes (T9/T11). Out-of-scope items (search, login, pagination, Tailwind, sitemap) have no tasks by design.
- The contact form block in Task 10 is the one place the plan says "move, don't retype" — the field markup is already exact in `contact.html:16-92` and retyping it is how the frozen contract gets broken.
- `TestRenderLanding_DeclaresEveryTagItDependsOn` rejects unknown tags, so T5 must update it in the same commit as the new tags.
