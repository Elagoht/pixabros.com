# Stitch Redesign — Design Spec

**Date:** 2026-08-17
**Status:** Approved by studio (chat), pending spec review
**References:** `STITCH/8_bit_legacy/DESIGN.md`, `STITCH/{home,games,devlog,awards,contact}_pixabros*/code.html`

## Goal

Re-skin the public site with the "8-Bit Legacy" design system produced in Google
Stitch: Modern Retro-Brutalism — dark console palette, thick borders, offset
step shadows, scanlines, sharp corners, chamfered cartridge cards, Space
Grotesk / Public Sans / Courier Prime typography.

The site's content, behaviour and SEO stay as they are. This is a visual layer
replacement plus two small renderer extensions (home previews, devlog filter
data). No new product surface.

## Non-negotiable constraints

1. **SEO byte-identical where it exists.** `internal/site/seo.go`,
   `internal/site/schema.go`, and the title/description/canonical/OG/Twitter/
   JSON-LD emission in `internal/site/templates.go` are not touched. The only
   intentional `<head>` change is the `theme-color` pair collapsing to one
   dark meta (browser chrome hint, not an SEO surface). Verification step
   (below) diffs every SEO-relevant head tag of each page before and after.
2. **All content comes from the CMS.** No Stitch placeholder copy or
   `lh3.googleusercontent.com` assets ship. Text that is currently hard-coded
   English chrome (nav labels, section titles that are not CMS-managed) may be
   rewritten in the Stitch voice ("Game Vault", "The Library", "Press Start"),
   mirroring how the current site already hard-codes "Games", "The shelf", etc.
3. **Images come from the media pipeline** (`/media/…` WebP derivatives,
   `loading="lazy"`, existing alt-text fallbacks). No new image derivatives.
4. **No login.** The Stitch `LOGIN` button is dropped. Public site has none
   today; none is added.
5. **Header menu stays on the right.** Logo/wordmark left, nav right —
   same as today, now in Stitch's visual language.
6. **Dark only.** The `prefers-color-scheme: light` palette is removed.
   `color-scheme` becomes `dark`; the light `theme-color` meta and the
   manifest's pinned colour move to the new background (`#101412`).
7. **No new runtime dependencies.** No Tailwind, no icon font, no Google
   Fonts CDN. Icons stay inline SVG; fonts are self-hosted (all three
   families are SIL OFL — licence files ship alongside).

## Design system (what `site.css` becomes)

### Tokens

CSS custom properties keep the existing naming architecture; values move to
the 8-Bit Legacy palette:

| Role | Value |
|---|---|
| Background / page | `#101412` |
| Card / console plastic | `#1A1A1A` (console-gray) |
| Surface containers (low→highest) | `#191c1a`, `#1d211e`, `#272b28`, `#323632`; lowest `#0b0f0c` |
| Accent (primary action) | `#F29F05` pixel-amber |
| Primary (links, hover) | `#ffb781` |
| On-surface / body text | `#e0e3de` |
| Muted text | `#dac2b3` |
| Outline / border-strong | `#a28d7f` / `#544338` |
| Status green | `#22C55E` crt-green |
| Error | `#ffb4ab` |

### Typography

- **Space Grotesk** 600/700 — display + headings, tight tracking.
- **Public Sans** 400/600 — body.
- **Courier Prime** 400/700 — labels, metadata, terminal bits, uppercase with
  wide tracking.
- Scale from DESIGN.md: display 48/1.1, headline 32/1.2 (+24 mobile), body
  18/1.6 and 16/1.5, label 14/1.4 and 12/1.2.
- WOFF2 files under `internal/site/assets/fonts/`, embedded like the current
  assets, `font-display: swap`.

### Shape language

- Corners square. Buttons/cards/inputs get 2–4px solid borders, never radii
  (a 2px radius on tiny chips is allowed where Stitch itself uses it).
- Elevation = step shadows: `box-shadow: 4px 4px 0 #000` (2px small, 8px
  large). No blur shadows.
- Buttons press physically: on `:active` translate 2px and collapse the step.
  Hover inverts (bg/text swap) per Stitch.
- Inputs recessed: darker background than card, 1–2px border, focus border
  pixel-amber.
- Scanline overlay (1px lines, 4px period, ~10%) on hero and CRT surfaces
  only; pure CSS, static (no flicker animation).
- Cartridge cards: `clip-path` chamfer on the top-right corner.
- Dashed pixel dividers (4px dash / 4px gap) for secondary separators.
- All decorative animation (pulses, scanline effects) respects
  `prefers-reduced-motion`.

## Page-by-page

### Layout / chrome (`layout.html`, `site.css`, `osd.js` untouched)

- Header keeps its behaviour contract (`data-osd`, ghost `CHxx`): sticky bar,
  thick bottom border + step shadow; left = wordmark (site name, Space Grotesk,
  amber) + power lamp; right = nav. Channel links keep `CH01–CH04` mono
  prefixes as decoration; active link = amber + 4px underline (Stitch style).
  On mobile the channel links wrap onto a second row — no hamburger menu, no
  new JS.
- Footer: Stitch bar — Courier copyright left, social marks right as blocky
  bordered buttons using the existing `brand-icon` partial and `rel="me
  noopener"` links. Data source unchanged (site links).
- `<head>`: only the `theme-color` pair collapses to one dark meta. Everything
  else byte-identical.

### Home (`landing.html`, `landing.go`)

- **Hero** (Stitch): `SYSTEM ONLINE` chip (decorative, crt-green), H1 =
  `hero_slogan` (fallback site name), lead = `hero_description`, buttons =
  CMS `hero_cta_text`/`hero_cta_link` (primary) + a static secondary
  "View catalog" → `/games`. Right: `hero_logo` image, `image-rendering:
  pixelated`, step-shadow drop. Missing CMS pieces degrade: no logo → text-only
  hero; no CTA → only the catalog button.
- **Featured Releases** = existing JS-free carousel re-skinned as Stitch snap
  cards: cover art, genre/year meta, tags as chips, price when for-sale, jam
  badge, hover border→amber. Arrows/dots/thumbnail-swap behaviour unchanged.
- **System Log + Achievements** (new data): latest 2 published devlog posts
  (date, title, blurb = plain-text clamp of the markdown content, link to the
  post) and latest 2 awards (title, issuer, year). Stitch two-column split
  with `>` cursor hover. `landing.go` gains these queries; its returned tag
  list grows from `{homepage, game:list, member:list, site_settings}` to
  also include `devlog:list` and `award:list` (the strings
  `internal/devlogapi` / `internal/awardsapi` already enqueue — verified
  against `landing.go:181`, `devlog.go:20`, `awards.go:89`).
- **Sales** and **Members** sections keep their data and logic, restyled as
  retro cards (sale card = Stitch game-box look; member = bordered card with
  avatar, roles as chips).

### Games index (`arcade.html`)

- Page head: "Game Vault" display title + Courier subtitle (existing copy
  voice: slot a cartridge…).
- **Console** (`partials.html "console"`) restyled as the Stitch TV: bezel,
  scanlines, slot bar with "Insert cartridge below", `CH-03` + power LED row.
  All `data-console*` hooks, press-to-start loading, CRT/fullscreen knobs,
  reset/eject, drag-and-drop stay exactly as implemented.
- **Playable now** = cartridge shelf: existing cartridges with
  `cartridge_art`, grip/notch anatomy, drag-or-click to load. Stitch shelf
  proportions.
- **The shelf** → "The Library": 3/4-aspect boxes using `cd_cover_art`,
  genre chip + title + blurb on a gradient, hover lift. Existing jewel-case
  `<dialog>` flow stays (it is the no-JS path); the dialog's inside face is
  restyled toward Stitch's modal: art, meta (released/genre/price/tags),
  blurb, "Examine" → `/games/{slug}`.

### Game detail (`game.html`)

- Breadcrumb + display title + jam badge + short lead, Stitch typography.
- Console partial (when playable) identical to the index machine; offline
  download widget (`data-offline-game`) keeps its DOM contract, restyled as
  a bordered panel with a block progress bar.
- Body: prose (markdown output untouched) + side panel with price,
  released/genre facts, tags, screenshots (zoom lightbox kept), store links
  ("Where to get it") as console buttons with brand icons.
- Non-playable cover image renders as a Stitch game-box.

### Devlog index (`devlog.html`, `devlog.go`, new filter script)

- Head: "> Devlog Terminal", status line.
- Featured = first post (large card, image, date + game, title, excerpt,
  "Read protocol"). Remaining posts = horizontal rows (image left, mono
  date + game, title, clamped excerpt, greyscale→colour image hover).
- **Sidebar** (Stitch): "Directories" = games referenced by posts with counts,
  "Archives" = years with counts. Both are client-side filters: clicking sets
  the active item and hides non-matching rows via `data-game` / `data-year`
  attributes. No-JS = plain full list, sidebar links are then inert text.
  No URL/query changes (static pre-render store stays authoritative); no
  search box; no fake status widget.
- `devlog.go` builds the counts view model alongside the post list.

### Devlog post (`devlog-post.html`)

- Retro prose styling for `{{.Data.ContentHTML}}` (styles only — markdown
  pipeline untouched), mono date/game header, back-to-devlog link.

### Awards (`awards.html`)

- "Hall of Fame" header with amber left rule + Stitch intro line.
- Cards: chamfered cartridge cards — picture (existing fit-art), year badge
  (mono, inset), title, issuer with a verified-style mark, optional link.
  Picture zoom lightbox kept.

### Contact (`contact.html`, `contact-sent.html`)

- "Comm Link" layout: left column = studio blurb + org email (site settings)
  + "Network Nodes" social blocks; right = "Signal Input" form.
- **Form contract frozen:** field names (`name`, `subject`, `phone`,
  `email`, `message`, `wants_callback`, honeypot), `method="post"
  action="/api/contact"`, `minlength`, required flags, status/submit hooks
  — unchanged. Labels may move to Stitch voice ("Player Name", "Quest
  Objective"…) with placeholders updated to match; autocomplete attributes
  stay.
- Submit renders as the amber "PRESS START" console button.
- Sent page: restyled confirmation panel.

### 404 / offline (`404.html`, `offline.html`)

Same copy, new styling; offline page must keep working under the service
worker with no network.

## Explicitly out of scope

Search, login/accounts, payments, pagination, light theme, theme toggle,
category taxonomy in the CMS, Tailwind or any build step for public CSS,
new image derivatives, sitemap/robots (none exist today; adding them would
change the SEO surface), admin UI changes, `/play/` and offline-download
logic changes, API changes.

## Verification

1. `go test ./...` — existing suites; any golden/template tests updated
   deliberately, not deleted.
2. `make test` (vitest on `sw.js` / `offline.js`) — must pass untouched.
3. `make run` and walk every route (`/`, `/games`, a game page, `/devlog`,
   a post, `/awards`, `/contact`, `/contact/sent`, `/404`, `/offline`).
4. **SEO diff:** render each public page before and after; `<head>` content
   (title, description, canonical, OG, Twitter, JSON-LD) must be identical.
5. No-JS pass: carousel scrolls, cases open, contact form posts, nav usable.
6. Keyboard pass: visible focus (amber), dialog focus traps, console controls.
7. `prefers-reduced-motion`: decorative animation disabled.
8. Visual pass against the five `STITCH/*/screen.png` references.

## Risks / notes

- `site.css` is rewritten wholesale; component class names are kept where JS
  or templates still reference them (`osd`, `console`, `carousel`, `case`,
  `jewel`, `zoom`, `contact` hooks), so behaviour survives the restyle.
- The landing page's new devlog/award sections must join the regeneration
   queue or the home page goes stale — tag wiring is part of the work.
- Fonts add ~150–250KB of embedded WOFF2 (three families, two weights each,
   latin subset); acceptable for a self-hosted single-binary site.
