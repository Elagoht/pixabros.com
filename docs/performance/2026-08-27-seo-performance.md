# SEO redirect and paint baseline — 2026-08-27

## Outcome

Production already sends every tested non-canonical origin directly to
`https://pixabros.com/` in one Cloudflare-served `301`. The canonical HTTPS URL
returns `200` without a redirect. No application redirect is involved, so no Go
redirect was added.

Three-run mobile Lighthouse medians were 1,965.8 ms FCP on production and
1,800.7 ms on a local production binary. A second, unchanged control measured
1,962.2 ms and 1,800.7 ms respectively. This 3.6 ms production movement happened
without a code change and is measurement noise, not an improvement.

Lighthouse attributes a median 309 ms opportunity to the single render-blocking
stylesheet, but the local transport is not comparable: production serves the
53,866-byte CSS compressed as about 12.7 KB over HTTP/2, while the local Go
server sends about 54.4 KB over HTTP/1.1. Safely splitting critical CSS would
also need a representative local copy of the 44.6 KB production page, which was
not available. The trace therefore does not prove a safe repository change.
No performance code or SEO behavior was changed.

All timestamps below are UTC on 2026-08-27. The network vantage reached
Cloudflare FRA, as shown by the redirect responses' `CF-RAY` suffix.

## Redirect protocol and evidence

Command, run unchanged before and after the remediation decision:

```bash
for url in http://pixabros.com https://pixabros.com \
  http://www.pixabros.com https://www.pixabros.com; do
  curl -sS -o /dev/null -D - \
    -w 'url=%{url_effective} redirects=%{num_redirects} total=%{time_total}\n' \
    -L --max-redirs 10 "$url"
done
```

Baseline capture: `2026-08-27T16:55:34Z`.

```text
request: http://pixabros.com
HTTP/1.1 301 Moved Permanently
Location: https://pixabros.com/
Server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=0.576400

request: https://pixabros.com
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=0 total=0.359354

request: http://www.pixabros.com
HTTP/1.1 301 Moved Permanently
Location: https://pixabros.com/
Server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=0.532499

request: https://www.pixabros.com
HTTP/2 301
location: https://pixabros.com/
server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=0.530329
```

Unchanged control capture: `2026-08-27T17:00:55Z`.

```text
request: http://pixabros.com
HTTP/1.1 301 Moved Permanently
Location: https://pixabros.com/
Server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=0.522858

request: https://pixabros.com
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=0 total=0.623804

request: http://www.pixabros.com
HTTP/1.1 301 Moved Permanently
Location: https://pixabros.com/
Server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=1.553860

request: https://www.pixabros.com
HTTP/2 301
location: https://pixabros.com/
server: cloudflare
HTTP/2 200
server: cloudflare
url=https://pixabros.com/ redirects=1 total=0.763688
```

Every non-canonical response has `Server: cloudflare`, a Cloudflare ray ID, and
a `Location` that is already the final origin. Repository search found only the
intentional favicon and post-submit application redirects:

```bash
rg -n 'http\.Redirect|StatusMovedPermanently|StatusPermanentRedirect|Location' \
  cmd internal deploy
```

There is no host- or scheme-canonicalization redirect in the Go server or
deployment scripts. Together, these are the available ownership signals that
the canonical redirect is edge-controlled. No Cloudflare state was read or
changed.

Operator action: none. Preserve the current single edge rule. If it ever needs
to be recreated, use one rule matching HTTP or `www.pixabros.com` and send the
request directly to `https://pixabros.com` while preserving path and query. Do
not add a Go redirect or a second edge hop.

## Lighthouse methodology

The in-app Browser control runtime was unavailable after checking the session's
tool surface, so the approved fallback was the official Lighthouse CLI. The
measurement tools were installed into a temporary directory outside the
repository:

```text
Lighthouse 12.8.2
Google Chrome for Testing 152.0.7977.64 (headless shell, macOS arm64)
Node.js 22.15.1
```

Chrome was provisioned with the official Puppeteer browsers utility. Lighthouse
used its mobile form factor, simulated throttling, a 412 by 823 viewport at
1.75 device scale, 150 ms RTT, 1,638.4 Kbit/s throughput, and 4x CPU slowdown.
Lighthouse clears service workers and Cache Storage for each navigation.

Production command, run three times per set:

```bash
export CHROME_PATH="$MEASURE_DIR/browser/chrome-headless-shell/.../chrome-headless-shell"
for run in 1 2 3; do
  npx --yes lighthouse@12.8.2 https://pixabros.com/ \
    --only-categories=performance \
    --output=json \
    --output-path="$MEASURE_DIR/production-$run.json" \
    --form-factor=mobile \
    --throttling-method=simulate \
    --screenEmulation.mobile=true \
    --chrome-flags='--headless --no-sandbox --disable-gpu' \
    --quiet
done
```

The local binary was built from commit `4fe6929` and run on a disposable copy
of the existing local data, so the source database was not modified:

```bash
cp -R /Users/furkanbaytekin/Desktop/pixabros.com/data "$MEASURE_DIR/local-data"
go build -o "$MEASURE_DIR/pixabros-server" ./cmd/server
PIXABROS_ADDR=127.0.0.1:18186 \
PIXABROS_DB_PATH="$MEASURE_DIR/local-data/pixabros.db" \
PIXABROS_DATA_DIR="$MEASURE_DIR/local-data" \
  "$MEASURE_DIR/pixabros-server"
```

The same Lighthouse flags were then used with
`http://127.0.0.1:18186/`. The local data had zero games, members, posts, and
awards, so its HTML was 3,122 bytes versus 44,617 bytes in production. The
local run is useful for origin response cost and shared asset behavior, but it
is not a strict page-content comparison.

Measurement windows:

- baseline production: `16:56:31Z`–`16:57:30Z`;
- baseline local: `16:58:14Z`–`16:58:47Z`;
- unchanged-control production: `17:01:08Z`–`17:01:49Z`;
- unchanged-control local: `17:01:49Z`–`17:02:20Z`.

## Run results

Values are milliseconds except score and bytes.

| Target/set | Run | FCP | LCP | Speed Index | TBT | document response | render-blocking audit | transfer bytes | score |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| production baseline | 1 | 2,010.127 | 2,010.127 | 4,400.657 | 0 | 183.446 | 355 | 2,224,775 | 95 |
| production baseline | 2 | 1,965.818 | 1,965.818 | 4,155.789 | 0 | 133.877 | 309 | 2,224,667 | 96 |
| production baseline | 3 | 1,688.362 | 1,688.362 | 1,808.632 | 0 | 452.240 | 0 | 2,224,722 | 99 |
| local baseline | 1 | 1,651.164 | 1,951.164 | 1,651.164 | 0 | 0.238 | 998 | 131,875 | 99 |
| local baseline | 2 | 1,800.843 | 1,950.843 | 1,800.843 | 0 | 0.262 | 1,035 | 131,875 | 98 |
| local baseline | 3 | 1,800.729 | 1,950.729 | 1,800.729 | 0 | 0.226 | 1,034 | 131,875 | 98 |
| production control | 1 | 2,042.746 | 2,042.746 | 4,253.110 | 0 | 187.409 | 314 | 2,224,693 | 95 |
| production control | 2 | 1,962.175 | 1,962.175 | 4,262.151 | 0 | 211.274 | 307 | 106,962* | 95 |
| production control | 3 | 1,960.311 | 1,960.311 | 4,098.065 | 0 | 115.423 | 302 | 2,225,052 | 96 |
| local control | 1 | 1,651.764 | 1,801.764 | 1,651.764 | 0 | 0.535 | 886 | 131,875 | 99 |
| local control | 2 | 1,800.660 | 1,950.660 | 1,800.660 | 0 | 0.223 | 1,035 | 131,875 | 98 |
| local control | 3 | 1,800.969 | 1,950.969 | 1,800.969 | 0 | 0.194 | 1,034 | 131,875 | 98 |

`*` The second production control trace recorded the below-fold images with
zero transfer bytes. The request count remained 41, and the other two runs
transferred the full image set. The three-run median therefore remains about
2.224 MB; the outlier did not affect FCP.

Three-run medians:

| Target/set | FCP | LCP | response | render blocking | transfer | score |
|---|---:|---:|---:|---:|---:|---:|
| production baseline | 1,965.818 ms | 1,965.818 ms | 183.446 ms | 309 ms | 2,224,722 B | 96 |
| local baseline | 1,800.729 ms | 1,950.843 ms | 0.238 ms | 1,034 ms | 131,875 B | 98 |
| production unchanged control | 1,962.175 ms | 1,962.175 ms | 187.409 ms | 307 ms | 2,224,693 B | 95 |
| local unchanged control | 1,800.660 ms | 1,950.660 ms | 0.223 ms | 1,034 ms | 131,875 B | 98 |

No before/after improvement is claimed because there was no performance code
change. The unchanged production control moved by -3.643 ms FCP (-0.19%).

## Trace attribution

### CSS and fonts

The median production baseline trace identifies exactly one blocking resource:

```text
https://pixabros.com/assets/build/site.f136e3fb.css
transfer=12,709 B resource=53,866 B priority=VeryHigh
render-blocking audit=309 ms; estimated FCP savings=300 ms
```

The unchanged control median reports 307 ms and the same 300 ms estimated FCP
savings. The local median assigns 1,034 ms to the same stylesheet because it is
sent uncompressed as 54,424 transfer bytes over HTTP/1.1. That transport
difference prevents using the local number as comparable proof for a CSS code
change.

The production page requests four self-hosted WOFF2 files, all at `VeryHigh`
priority, with a median combined transfer of about 66.4 KB:

- Space Grotesk: 22,320 resource bytes;
- Public Sans: 18,472 resource bytes;
- Courier Prime regular: 11,192 resource bytes;
- Courier Prime bold: 11,588 resource bytes.

`font-display-insight` passes with zero estimated savings. The CSS already uses
`font-display: swap`, so no font behavior changed.

### LCP image and transfer sizes

The production LCP is the 512 by 512 PixaBros logo. It is present in the initial
HTML, eagerly loaded, and dimensioned. Its request is about 15.2 KB and was
Medium in the median trace (High in one baseline trace). Lighthouse notes that
`fetchpriority=high` is absent but estimates **0 ms** LCP savings; the legacy
`prioritize-lcp-image` audit also estimates zero. A priority-only code change is
therefore not proven.

Median baseline transfer attribution is 41 requests / 2,224,722 bytes total:

- images: 31 requests / 2,132,926 bytes;
- fonts: 4 / 66,354 bytes;
- stylesheet: 1 / 12,709 bytes;
- document: 1 / 9,231 bytes;
- scripts: 4 / 3,517 bytes.

The page's non-hero images already have lazy-loading markup, and Lighthouse's
offscreen-image audit estimates zero FCP and LCP savings. The large page transfer
is worth watching, but this trace does not support changing loading behavior in
this task.

### Cache observations

Headers were captured with cold GETs at `2026-08-27T17:00:21Z`:

| Resource | `Cache-Control` | `CF-Cache-Status` |
|---|---|---|
| HTML document | `no-cache` | `DYNAMIC` |
| hashed CSS | `public, max-age=31536000, immutable` | `HIT` |
| all four fonts | `public, max-age=31536000, immutable` | `HIT` |
| LCP media logo | `max-age=14400` | `HIT` |

The content-hashed repository assets have the intended one-year immutable
cache. The four-hour media TTL is an edge policy and Lighthouse estimates zero
FCP savings from changing it (50 ms LCP on a repeat visit). It is not an FCP
remediation for this task.

### Application versus production cost

The median Lighthouse document-response audit was 183.446 ms in production and
0.238 ms locally; the unchanged control was 187.409 ms and 0.223 ms. The local
binary responds quickly, while TLS, network, Cloudflare, and the production
origin path account for the observable difference. Lighthouse still scores the
production response-time audit as passing. No repository server bottleneck was
demonstrated.

## Remediation decision

No RED/GREEN code cycle was performed because no performance code was changed.
The evidence boundary is:

- canonicalization is already a one-hop Cloudflare redirect and needs no Go
  handler;
- the required stylesheet is render-blocking, but the available local trace is
  not transport- or content-comparable, so critical-CSS or asynchronous-CSS
  work cannot be credited with a measured improvement;
- the LCP image is already early, eager, dimensioned, and carries browser
  Medium/High priority; the missing explicit hint has zero estimated savings;
- font display and offscreen image audits estimate zero savings;
- immutable repository assets are Cloudflare cache hits.

Changing stylesheet delivery without representative evidence could introduce a
flash of unstyled content or require weakening the current CSP for inline CSS.
Those risks are not justified by this data. SEO metadata, crawler policy,
discovery resources, and rendering behavior remain untouched.

## Verification

Fresh verification ran from `2026-08-27T17:06:51Z` through
`2026-08-27T17:07:05Z`:

```text
go test ./... -count=1
all packages passed; exit 0

go build ./cmd/server
exit 0

git diff --check
exit 0
```

The generated local `server` build artifact was removed after the successful
build; it is not part of the documentation-only change.

## Concerns and follow-up threshold

There is no blocking concern. A future CSS experiment should first reproduce
the production homepage content locally behind comparable compression and
HTTP/2, then run the same three-run profile before and after. Only a stable
median improvement should authorize critical-CSS extraction or another delivery
change.
