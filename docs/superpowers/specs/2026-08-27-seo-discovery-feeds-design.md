# SEO, Crawler Discovery, and Feeds Design

## Goal

Make every public Pixabros page accurately discoverable by search engines,
social crawlers, feed readers, and AI agents while explicitly preventing
indexing of private, machine-only, transient, and game-runtime routes. Add
automatically maintained `robots.txt`, `llms.txt`, XML sitemap, and RSS output,
and investigate the reported redirect and First Contentful Paint costs without
guessing at their cause.

## Constraints and decisions

- The public site is English-only. It must publish `hreflang="en"` and
  `hreflang="x-default"` self-references, and must not claim that a Turkish
  alternative exists.
- `site_url` remains the sole origin for canonical and absolute discovery URLs.
  When it is unset, the renderer must omit links that cannot be stated
  truthfully rather than inventing a production host.
- Existing title, description, canonical, Open Graph, Twitter Card, and JSON-LD
  behavior stays intact and is extended rather than replaced.
- Generated discovery documents participate in the existing render store and
  tag invalidation pipeline. They are not manually maintained copies and are
  not rebuilt from the database on every HTTP request.
- Only published games and published devlog posts may appear in public
  discovery documents.
- The site welcomes ordinary search and AI crawling. Private or non-document
  routes remain excluded regardless of crawler identity.
- No DNS mutation is part of this repository change.

## Page metadata

The common HTML layout will publish the following on every indexable public
page when the required values exist:

- the existing `title`, description, canonical, Open Graph, Twitter Card,
  icons, manifest, theme color, and JSON-LD;
- `<link rel="alternate" hreflang="en">` and an identical `x-default`
  alternate pointing to the page's canonical URL;
- `<link rel="alternate" type="application/rss+xml" title="Pixabros Devlog"
  href="/rss.xml">`;
- `robots` with `index, follow, max-image-preview:large, max-snippet:-1,
  max-video-preview:-1`;
- `author`, `publisher`, and `application-name`, using the configured site name
  and organization identity;
- `keywords`, using a short page-specific, deduplicated list rather than a
  repeated global keyword dump;
- `referrer=origin-when-cross-origin` and `color-scheme=dark`.

The renderer will own these defaults so an individual page cannot accidentally
omit site-wide policy. Page view models will supply page-specific keywords:

- landing and section pages use a small curated vocabulary describing indie
  games, game development, game jams, pixel art, and Pixabros;
- a game page adds its title, genre, tags, and relevant game/studio terms;
- a devlog post adds its title and associated game where available;
- keywords are trimmed, compared case-insensitively, deduplicated, and emitted
  as one comma-separated value.

`keywords` and the non-standard `publisher` meta are included for compatible
auditors and consumers, not represented as Google ranking signals. Publisher
identity continues to be expressed authoritatively through the existing
Schema.org `Organization` and its publisher relationships.

## Robots policy and `X-Robots-Tag`

The HTML meta policy and HTTP policy must agree. Indexable public HTML receives:

```text
X-Robots-Tag: index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1
```

The following response families receive:

```text
X-Robots-Tag: noindex, nofollow, noarchive
```

- the admin panel under `/I-am-a-pixabro/`;
- `/api/` responses;
- uploaded game runtimes under `/play/`;
- `/offline` and `/contact/sent`;
- every 404 response, including the styled public 404;
- service-worker and shell-support endpoints that are implementation resources,
  not documents.

Static assets, media, manifest, `robots.txt`, `llms.txt`, `sitemap.xml`, and
`rss.xml` do not need an indexable-document header. They must never inherit
`noindex` merely because they are not HTML. Route-aware response header policy
will live beside the existing CSP selection in `internal/httpserver`, with tests
covering all route families and 404 status handling.

`robots.txt` will be generated from the configured site URL and will:

- use `User-agent: *` with `Allow: /`;
- disallow `/I-am-a-pixabro/`, `/api/`, `/play/`, `/offline`, and
  `/contact/sent`;
- advertise the absolute `/sitemap.xml` URL when `site_url` is configured;
- avoid crawler-specific blocks for AI agents, allowing them under the same
  public/private boundary as other crawlers.

The exclusions are crawl-efficiency hints, not access control. Authentication
and `X-Robots-Tag` remain responsible for private/non-indexable responses.

## Generated discovery documents

Four new rendered keys will be registered and reconciled with the current page
registry:

- `robots.txt` — UTF-8 plain text;
- `llms.txt` — UTF-8 Markdown/plain text;
- `sitemap.xml` — XML sitemap;
- `rss.xml` — RSS 2.0 feed.

The render-serving layer currently labels every stored output as HTML. It will
select a safe content type by the exact generated key: `text/plain` for the two
text documents and the standard XML media types for sitemap and RSS, leaving
all existing page keys as `text/html; charset=utf-8`. Existing ETag and
`Cache-Control: no-cache` behavior will remain.

### Sitemap

The sitemap contains canonical URLs for the landing page, games index, devlog
index, awards, contact, each published game, and each published devlog post.
It excludes admin/API/play resources, offline/contact-sent/404 pages, and
unpublished content. `lastmod` is included for dynamic entries using their
stored update timestamp; static pages omit invented modification dates. Output
is deterministic: fixed section order followed by consistently sorted dynamic
entries.

The sitemap depends on `site_settings`, `game:list`, and `devlog:list`, so
changes to the origin or published content automatically regenerate it.

### RSS

`rss.xml` is an RSS 2.0 feed for published devlog posts in newest-first order.
The channel uses the configured studio name, description, site URL, and devlog
URL. Each item includes title, canonical permalink GUID/link, publication date,
and a safe plain-text summary derived from Markdown; it must not expose raw
untrusted markup. Empty publication dates fall back to the persisted creation
time. The feed depends on `site_settings` and `devlog:list`.

### `llms.txt`

`llms.txt` follows the simple Markdown convention: an H1 project name, a short
studio summary, and grouped links to the canonical home, games, devlog, awards,
contact, published game pages, published posts, sitemap, and RSS feed. It is a
navigation aid, not a license grant and not a substitute for robots policy.
It depends on `site_settings`, `game:list`, and `devlog:list`.

All XML is produced through Go's XML encoder or properly escaped templates;
titles, summaries, URLs, and configured strings must never be concatenated into
raw XML.

## Data and invalidation flow

No schema migration is needed. Existing game and devlog repositories already
hold publication state, slugs, titles, timestamps, descriptions, tags, and
content. Discovery renderers will read through those repositories and return
the same dependency tags already enqueued by admin mutations.

Startup `RefreshAll` renders the four documents after every deploy. During
normal operation, `game:list`, `devlog:list`, and `site_settings` invalidations
regenerate only the documents that declared those tags. A change to an
individual page still uses its existing detail tag; list invalidation covers
the discovery documents because membership, title, slug, and timestamps can
all affect them.

## Redirect and FCP investigation

Performance work is evidence-led and separated from metadata work:

1. Measure the production URL's redirect chain for HTTP/HTTPS and www/apex
   variants with response headers and timings. Identify whether each hop comes
   from Cloudflare or the Go application.
2. Preserve one canonical redirect at most. If the chain is controlled by
   Cloudflare, document the exact rule consolidation needed instead of adding a
   competing application redirect. If it is application-controlled, cover the
   corrected behavior with an HTTP test.
3. Run Lighthouse (or equivalent browser performance tracing) against the
   production page and a local build, then inspect render-blocking CSS, fonts,
   above-the-fold image dimensions/priority, cache headers, and script cost.
4. Implement only changes supported by the trace. Likely candidates include
   font preloading/subsetting, an explicit priority for the actual LCP image,
   and removal of unused blocking CSS; none is assumed before measurement.

Success is a redirect chain with no avoidable intermediate hop and a fresh
trace showing the before/after FCP result. A 1.9-second lab FCP is an observation
from one run, not by itself proof of a specific defect or a universal score.

## Email DNS records

SPF and DMARC findings concern DNS and sender identity, not the presence of a
mailbox or HTML metadata. They cannot be fixed by adding website files.

If the domain sends no email at all, the recommended defensive DNS posture is:

```dns
@      TXT  "v=spf1 -all"
_dmarc TXT  "v=DMARC1; p=reject; adkim=s; aspf=s"
```

These values must not be published until the owner confirms that no provider,
contact workflow, transactional service, or forwarding setup sends mail using
the domain. If a sender exists, its provider-specific SPF include and DKIM
records must be incorporated first. DNS verification and publication remain a
separate operational action outside this implementation.

## Testing and acceptance

Implementation follows red-green-refactor tests covering:

- common metadata, English/x-default hreflang, RSS discovery, publisher, and
  deterministic page-specific keywords;
- absence of hreflang when no canonical URL exists and absence of any Turkish
  alternate;
- route-specific `X-Robots-Tag`, including public 404 responses;
- exact content types, ETag behavior, and HEAD handling for all four generated
  documents;
- robots directives and sitemap advertisement with and without `site_url`;
- XML validity and escaping, publication filtering, canonical URL formation,
  stable ordering, `lastmod`, RSS dates, and safe summaries;
- `llms.txt` structure and exclusion of drafts;
- dependency tags and reconciliation of all generated keys;
- full Go test suite and production build;
- before/after redirect traces and browser performance evidence for any
  performance code change.

The implementation is accepted when generated resources are reachable at their
root paths with correct types, update after relevant content/settings changes,
contain no drafts or false language claims, public pages expose consistent
indexing signals, private/transient routes are explicitly non-indexable, and
all automated verification passes.

## Out of scope

- a Turkish public site or language switcher;
- DNS changes, mailbox creation, DKIM provisioning, or mail delivery setup;
- crawler cloaking or different content for named AI bots;
- adding keywords solely to manipulate rankings;
- redesigning the visual interface or weakening the site's security headers;
- performance changes not supported by a trace.

## Recorded deviations (2026-08-27)

Two points where the implementation deliberately differs from what is written
above. Both are intentional; the code is the authority on these.

- The layout emits `referrer=strict-origin-when-cross-origin`, not the
  `origin-when-cross-origin` written above: it is the modern browser default
  and strictly narrower, so the head leaks no more of an address than this
  spec asked for.
- The RSS `<link rel="alternate">` title is "`<site name>` devlog" rather than
  the literal "Pixabros Devlog", so it follows the configured site name the
  same way every other published title does.
