# SEO Discovery and Feeds Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish complete, truthful page metadata and automatically maintained crawler, sitemap, RSS, and AI discovery resources while keeping private/transient routes explicitly non-indexable and addressing only measured performance defects.

**Architecture:** Extend the shared page renderer for HTML metadata, register four non-HTML renderers in the existing render registry, and reuse `site_settings`, `game:list`, and `devlog:list` invalidation tags. Extend the HTTP serving and security-header boundaries narrowly: content type is selected from the exact rendered key, while route/status-aware robots headers are applied by middleware. Performance changes form a final evidence-gated task after production redirect and browser traces.

**Tech Stack:** Go 1.22+ `net/http`, `encoding/xml`, `html/template`, SQLite repositories, existing content-addressed render store, Go tests, curl, Chrome Lighthouse.

**Spec:** `docs/superpowers/specs/2026-08-27-seo-discovery-feeds-design.md`

## Global Constraints

- The public site is English-only: emit only `en` and `x-default` hreflang values.
- `site_url` is the only source of absolute/canonical URLs; omit absolute discovery links when it is blank.
- Only published games and posts appear in sitemap, RSS, and `llms.txt`.
- Discovery documents use the existing persisted render/invalidation pipeline, never per-request database queries.
- Do not publish DNS records from this repository.
- Do not make performance changes without before/after trace evidence.
- Follow red-green-refactor for every production behavior change.

---

## File map

- `internal/site/templates.go`: common metadata fields/defaults and keyword normalization.
- `internal/site/templates/layout.html`: shared HTML `<head>` output.
- `internal/site/seo.go`: keyword helpers and canonical-related SEO utilities.
- `internal/site/*.{go,test.go}`: page-specific keyword inputs and metadata assertions.
- `internal/site/discovery.go`: robots, sitemap, RSS, and llms renderers.
- `internal/site/discovery_test.go`: discovery content, escaping, filtering, ordering, and tags.
- `internal/site/site.go`: generated-key constants and registry declarations.
- `internal/site/reconcile.go` and tests: generated keys in desired output.
- `internal/render/serve.go` and tests: content types for stored non-HTML resources.
- `internal/httpserver/security.go` and tests: route- and status-aware `X-Robots-Tag` policy.
- `docs/performance/2026-08-27-seo-performance.md`: redirect/FCP evidence and any Cloudflare-only remediation.

---

### Task 1: Shared page metadata and deterministic keywords

**Files:**
- Modify: `internal/site/templates.go`
- Modify: `internal/site/seo.go`
- Modify: `internal/site/templates/layout.html`
- Modify: `internal/site/seo_test.go`
- Modify: page renderers under `internal/site/landing.go`, `games.go`, `devlog.go`, `awards.go`, `contact.go`, and `shell.go`

**Interfaces:**
- Produces: `pageData.Keywords []string`
- Produces: `normalizeKeywords(values ...string) string`
- Consumes: existing `pageData.Canonical`, `SiteChrome.Name`, and page content fields.

- [ ] **Step 1: Write failing common-head tests**

Add focused assertions to `internal/site/seo_test.go`:

```go
func TestPages_PublishEnglishAndDefaultAlternates(t *testing.T) {
    conn := setupTestDB(t)
    seedSEOSettings(t, conn)
    html, _, err := newTestSite(t, conn).renderLanding(PageLanding)
    if err != nil { t.Fatal(err) }
    body := string(html)
    for _, want := range []string{
        `rel="alternate" hreflang="en" href="https://pixabros.com/"`,
        `rel="alternate" hreflang="x-default" href="https://pixabros.com/"`,
        `rel="alternate" type="application/rss+xml"`,
        `name="publisher" content="Pixabros"`,
        `name="application-name" content="Pixabros"`,
        `name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1"`,
    } {
        if !strings.Contains(body, want) { t.Errorf("missing %q", want) }
    }
    if strings.Contains(body, `hreflang="tr"`) { t.Error("published a nonexistent Turkish alternate") }
}

func TestPageWithoutCanonicalOmitsHreflang(t *testing.T) {
    conn := setupTestDB(t)
    html, _, err := newTestSite(t, conn).renderLanding(PageLanding)
    if err != nil { t.Fatal(err) }
    if strings.Contains(string(html), `hreflang=`) { t.Error("hreflang has no truthful absolute target") }
}
```

Add table tests for keyword trimming, comma-splitting, case-insensitive
deduplication, stable order, and blank removal. Add one game test proving title,
genre, and tags are present and one post test proving drafts/other page data do
not leak into its keywords.

- [ ] **Step 2: Run the metadata tests and verify RED**

Run:

```bash
go test ./internal/site -run 'TestPages_PublishEnglish|TestPageWithoutCanonical|TestNormalizeKeywords|TestRenderGame_Keywords|TestRenderDevlogPost_Keywords' -count=1
```

Expected: FAIL because the layout has no hreflang/RSS/robots/publisher fields
and `normalizeKeywords` does not exist.

- [ ] **Step 3: Implement minimal shared metadata support**

Add `Keywords []string` and `KeywordText string` to `pageData`. Implement:

```go
func normalizeKeywords(values ...string) string {
    seen := map[string]bool{}
    out := make([]string, 0, len(values))
    for _, value := range values {
        for _, part := range strings.Split(value, ",") {
            keyword := strings.TrimSpace(part)
            key := strings.ToLower(keyword)
            if keyword == "" || seen[key] { continue }
            seen[key] = true
            out = append(out, keyword)
        }
    }
    return strings.Join(out, ", ")
}
```

In `renderer.render`, fill `KeywordText` from page keywords plus `data.Site.Name`.
In `layout.html`, render canonical-dependent `en` and `x-default` links, RSS
discovery, robots, author, publisher, application name, keywords when nonempty,
referrer, and dark color scheme. Add concise curated keyword slices in each
page renderer; game and post renderers append their actual content fields.

- [ ] **Step 4: Verify GREEN and run the full site suite**

Run:

```bash
go test ./internal/site -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add internal/site
git commit -m "feat: complete public page metadata"
```

---

### Task 2: Correct content types for generated resources

**Files:**
- Modify: `internal/render/serve.go`
- Modify: `internal/render/serve_test.go`

**Interfaces:**
- Produces: `contentTypeForPageKey(pageKey string) string`
- Consumes: exact stored keys `robots.txt`, `llms.txt`, `sitemap.xml`, `rss.xml`.

- [ ] **Step 1: Write failing content-type and HEAD tests**

Add a table-driven test that persists each key and requests it through
`ServePages`:

```go
tests := map[string]string{
    "index.html":  "text/html; charset=utf-8",
    "robots.txt":  "text/plain; charset=utf-8",
    "llms.txt":    "text/plain; charset=utf-8",
    "sitemap.xml": "application/xml; charset=utf-8",
    "rss.xml":     "application/rss+xml; charset=utf-8",
}
```

Assert GET has the expected type/body and HEAD has the same type/ETag with an
empty body. Retain the existing conditional GET assertion.

- [ ] **Step 2: Run and verify RED**

```bash
go test ./internal/render -run 'TestServePages_ContentTypes|TestServePages_Head' -count=1
```

Expected: FAIL because every key is currently labeled HTML and HEAD currently
copies the body.

- [ ] **Step 3: Implement exact-key MIME selection and true HEAD behavior**

Add:

```go
func contentTypeForPageKey(key string) string {
    switch key {
    case "robots.txt", "llms.txt":
        return "text/plain; charset=utf-8"
    case "sitemap.xml":
        return "application/xml; charset=utf-8"
    case "rss.xml":
        return "application/rss+xml; charset=utf-8"
    default:
        return "text/html; charset=utf-8"
    }
}
```

Set the type before conditional handling and return after headers for HEAD,
without opening/copying the stored body.

- [ ] **Step 4: Verify GREEN**

```bash
go test ./internal/render -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add internal/render/serve.go internal/render/serve_test.go
git commit -m "feat: serve rendered discovery content types"
```

---

### Task 3: Register and generate robots, sitemap, RSS, and llms documents

**Files:**
- Create: `internal/site/discovery.go`
- Create: `internal/site/discovery_test.go`
- Modify: `internal/site/site.go`
- Modify: `internal/site/reconcile.go`
- Modify: `internal/site/reconcile_test.go`

**Interfaces:**
- Produces constants `PageRobots`, `PageLLMS`, `PageSitemap`, `PageRSS`.
- Produces renderers `renderRobots`, `renderLLMS`, `renderSitemap`, `renderRSS` matching `render.Renderer`.
- Consumes `Site.chrome()`, `games.Repo.List`, `devlog.Repo.List`, `canonicalURL`.
- Returns dependency tags from the set `site_settings`, `game:list`, `devlog:list`.

- [ ] **Step 1: Write failing registration/reconciliation tests**

Assert `Site.Register` resolves all four exact keys and `DesiredPages` includes
them exactly once. Assert the registry render calls return the expected tags.

- [ ] **Step 2: Write failing document behavior tests**

In `discovery_test.go`, seed one published and one draft game/post plus site
settings. Test:

```go
func TestRenderSitemap_ContainsOnlyCanonicalPublishedPages(t *testing.T)
func TestRenderSitemap_IsValidEscapedXMLWithStableOrder(t *testing.T)
func TestRenderRSS_ContainsOnlyPublishedPostsNewestFirst(t *testing.T)
func TestRenderRSS_ProducesValidXMLAndSafePlainTextDescriptions(t *testing.T)
func TestRenderRobots_AllowsPublicCrawlingAndNamesSitemap(t *testing.T)
func TestRenderRobots_WithoutSiteURLDoesNotInventSitemapOrigin(t *testing.T)
func TestRenderLLMS_LinksPublishedContentAndFeedsOnly(t *testing.T)
```

Decode sitemap and RSS using `encoding/xml`, rather than asserting raw XML
formatting. Include titles containing `&` and `<` to prove escaping. For RSS,
assert each GUID has `isPermaLink="true"`, RFC 1123Z dates, and no Markdown link
syntax or HTML tags in description text.

- [ ] **Step 3: Run and verify RED**

```bash
go test ./internal/site -run 'TestSite_Register.*Discovery|TestDesiredPages.*Discovery|TestRender(Sitemap|RSS|Robots|LLMS)' -count=1
```

Expected: FAIL because constants and renderers do not exist.

- [ ] **Step 4: Implement shared discovery data selection**

Create focused private helpers:

```go
func (s *Site) publishedGames() ([]games.Game, error)
func (s *Site) publishedPosts() ([]devlog.Post, error)
func plainTextSummary(markdown string, maxRunes int) string
```

Use existing repo lists, filter `IsPublished`, and sort explicitly so output is
deterministic. Do not add new public repository APIs or database migrations.

- [ ] **Step 5: Implement robots and llms renderers**

Build robots lines with `strings.Builder`; append absolute `Sitemap:` only when
`chrome.URL != ""`. Build llms Markdown from canonical helpers and skip every
absolute link when no configured origin exists. Return tags:

```go
[]string{siteSettingsTag, "game:list", devlogListTag}
```

for llms and the relevant subset for robots.

- [ ] **Step 6: Implement sitemap and RSS with `encoding/xml`**

Define private XML structs with explicit tags. Prefix XML output with
`xml.Header`. Sitemap uses static canonical section URLs plus published games
and posts, and dynamic entries format `UpdatedAt` as `YYYY-MM-DD`. RSS channel
uses published posts newest-first; publication date uses parsed `PublishedAt`
at UTC midnight, falling back to `CreatedAt`, then formats `time.RFC1123Z`.

- [ ] **Step 7: Register and reconcile the generated keys**

Add all four exact page definitions to `pages()` or a parallel
`discoveryPages()` collection read by both `Register` and `DesiredPages`.
Keep HTML static pages and generated documents understandable as separate
groups while preserving one authoritative registration/desired list.

- [ ] **Step 8: Verify GREEN**

```bash
go test ./internal/site -count=1
go test ./internal/render -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit Task 3**

```bash
git add internal/site internal/render
git commit -m "feat: generate crawler discovery and feeds"
```

---

### Task 4: Route- and status-aware `X-Robots-Tag`

**Files:**
- Modify: `internal/httpserver/security.go`
- Modify: `internal/httpserver/security_test.go`

**Interfaces:**
- Produces: `robotsPolicyFor(path string) string` for pre-response route policy.
- Produces: a status-capturing `ResponseWriter` wrapper that forces noindex for 404 responses.
- Consumes: existing `withSecurityHeaders` middleware and route constants.

- [ ] **Step 1: Write failing route policy tests**

Add a table test for:

```go
var indexRobots = "index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1"
var noindexRobots = "noindex, nofollow, noarchive"
```

Expect `noindexRobots` for admin, API, play, `/offline`, `/contact/sent`,
service worker, and shell endpoints. Expect `indexRobots` for `/`, `/games/x`,
and `/devlog/x`. Expect an empty header policy for assets, media, manifest,
robots, llms, sitemap, and RSS.

- [ ] **Step 2: Write failing end-to-end response tests**

Exercise `New` with `httptest` and assert:

- a stored public page receives `indexRobots`;
- admin/API/play responses receive `noindexRobots`;
- an unknown public route returning the styled 404 receives `noindexRobots`;
- discovery/static resources do not receive contradictory `X-Robots-Tag`.

- [ ] **Step 3: Run and verify RED**

```bash
go test ./internal/httpserver -run 'TestRobotsPolicyFor|TestNew_SendsRobots|TestNew_Noindexes404' -count=1
```

Expected: FAIL because the header is absent.

- [ ] **Step 4: Implement policy and late 404 override**

Set route-known policies before calling the next handler. Wrap the writer so
`WriteHeader(http.StatusNotFound)` replaces any index policy with
`noindexRobots` before headers are committed. Preserve optional interfaces used
by Go HTTP handlers (`http.Flusher`, `http.Hijacker`, `io.ReaderFrom`, and
`http.Pusher`) or use `httpsnoop` only if already present; do not add a new
dependency solely for this wrapper. Ensure implicit 200 on `Write` preserves
the route policy.

- [ ] **Step 5: Verify GREEN**

```bash
go test ./internal/httpserver -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add internal/httpserver/security.go internal/httpserver/security_test.go
git commit -m "feat: publish explicit crawler response policy"
```

---

### Task 5: Whole-system generation and invalidation verification

**Files:**
- Modify: `internal/site/reconcile_test.go`
- Modify: `internal/render/queue_test.go` only if existing helpers cannot express the integration assertion
- Modify: `internal/httpserver/router_test.go` or nearest existing router integration test

**Interfaces:**
- Consumes: registered discovery renderers, persisted render store, tag queue, and public router.
- Produces: no new production API.

- [ ] **Step 1: Write failing integration tests**

Build the real site/registry/reconciler against a migrated temporary database.
Assert startup refresh persists all four keys and the router serves each at its
root URL with its correct body and type. Update a published post, enqueue/drain
`devlog:list`, and assert RSS/sitemap/llms ETags change while robots remains
unchanged. Update `site_url`, enqueue/drain `site_settings`, and assert all four
absolute outputs update.

- [ ] **Step 2: Run and verify the integration test fails for any missing wiring**

```bash
go test ./internal/site ./internal/httpserver -run 'TestDiscoveryResources_EndToEnd|TestDiscoveryResources_Regenerate' -count=1
```

Expected: FAIL only if registry, desired keys, handler type, or tags are wired
incorrectly. If it passes immediately because earlier focused work fully covers
the behavior, keep it as an end-to-end regression and record that the failure
was proven by temporarily removing one discovery registration, then restore it
and rerun.

> Recorded 2026-08-27, at final review: the branch shipped without a note of
> the red run, so it was proven after the fact. With
> `{Key: PageRSS, Render: s.renderRSS}` removed from `discoveryPages`,
> `TestDiscoveryResources_EndToEnd` failed on its `/rss.xml` subtest and
> `TestDiscoveryResources_Regenerate` failed too; with the registration
> restored, both passed.

- [ ] **Step 3: Add the minimal missing wiring**

Correct only the integration gap revealed by Step 2. Do not refactor unrelated
queue or router behavior.

- [ ] **Step 4: Verify GREEN and the full Go suite**

```bash
go test ./... -count=1
go build ./cmd/server
```

Expected: all packages PASS and build exits 0.

- [ ] **Step 5: Commit Task 5**

```bash
git add internal/site internal/httpserver internal/render
git commit -m "test: verify discovery resources end to end"
```

---

### Task 6: Measure redirects and First Contentful Paint, then apply only proven fixes

**Files:**
- Create: `docs/performance/2026-08-27-seo-performance.md`
- Modify: only files directly identified by the trace; likely candidates are `internal/site/templates/layout.html`, `internal/site/assets/site.css`, `internal/site/assets.go`, or Cloudflare configuration documentation.
- Test: corresponding `internal/site/*_test.go` or `internal/httpserver/*_test.go` for each implemented behavior.

**Interfaces:**
- Consumes: production `https://pixabros.com` and its HTTP/www variants.
- Produces: reproducible before/after evidence and explicit Cloudflare actions when repository code cannot fix the issue.

- [ ] **Step 1: Capture redirect evidence**

Run and paste status, `Location`, `Server`, and timing output into the report:

```bash
for url in http://pixabros.com https://pixabros.com http://www.pixabros.com https://www.pixabros.com; do
  curl -sS -o /dev/null -D - -w 'url=%{url_effective} redirects=%{num_redirects} total=%{time_total}\n' -L --max-redirs 10 "$url"
done
```

Record every hop separately with `curl -I` if the combined output is ambiguous.

- [ ] **Step 2: Capture browser performance evidence**

Run Lighthouse at least three times on the same production URL/profile and save
the median FCP plus trace attribution. Also run against a local production
binary so Cloudflare/network cost can be separated from application cost.
Record CSS blocking time, font requests, above-the-fold image/LCP priority,
transfer sizes, and cache status.

- [ ] **Step 3: Choose the remediation boundary**

- If multiple hops are Cloudflare-controlled, document one canonical
  redirect rule (HTTP and alternate host directly to the final HTTPS host) and
  do not add a Go redirect.
- If Go creates an avoidable hop, first add an HTTP test that reproduces the
  chain, run it RED, implement the one-hop redirect, and run it GREEN.
- If the FCP trace identifies a blocking repository resource, first add the
  closest automated assertion (preload link, image priority/dimensions, or
  removed unused asset), run RED, make the minimal change, and run GREEN.
- If no repository-controlled cause is demonstrated, make no performance code
  change and state that result in the report.

- [ ] **Step 4: Capture after evidence**

Repeat the exact curl and three-run Lighthouse protocol. Put before/after
values, environment, timestamp, and remaining bottlenecks in the report. Do not
claim an FCP improvement from a non-comparable run.

- [ ] **Step 5: Run final verification**

```bash
go test ./... -count=1
go build ./cmd/server
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 6: Commit Task 6**

```bash
git add docs/performance internal/site internal/httpserver
git commit -m "perf: document and reduce discovery page latency"
```

---

## Final acceptance review

- [ ] Compare every requirement in the spec to Tasks 1–6.
- [ ] Request a code review using `superpowers:requesting-code-review`.
- [ ] Address review findings using `superpowers:receiving-code-review`.
- [ ] Run `superpowers:verification-before-completion` with fresh full test,
  build, generated-resource HTTP, XML parse, and performance evidence.
- [ ] Use `superpowers:finishing-a-development-branch` to decide merge/commit
  handling with the user.

