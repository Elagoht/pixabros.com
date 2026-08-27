package httpserver

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"pixabros/internal/site"
)

// adminUIPrefix is where the panel is mounted, named in the site package so
// the route and the policy that covers it cannot be written differently.
const adminUIPrefix = site.AdminUIPrefix

// A document's head and its response header carry the same two policies, so
// the words are quoted from the site package rather than written again here.
const (
	indexRobots   = site.RobotsIndex
	noindexRobots = site.RobotsNoindex
)

// The four areas of this origin need different policies, so one header for
// everything would have to be as loose as the loosest of them. Splitting them
// keeps the public site, which is the part strangers reach, strict.

// publicCSP covers the rendered site. It can afford to be strict because the
// pages carry no inline script or style at all: everything is a hashed file
// under /assets, and the only third party in the whole site is the video
// player a devlog post can embed.
//
// frame-src names youtube-nocookie for that player and 'self' for the game
// console, which frames a build from /play on the same origin.
const publicCSP = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"frame-src 'self' https://www.youtube-nocookie.com; " +
	// manifest-src has no fallback but default-src, so leaving it out would
	// deny the site its own manifest -- and a blocked manifest fails silently:
	// the page renders, and the site simply cannot be installed.
	"manifest-src 'self'; " +
	// worker-src falls back to child-src and then to default-src, which is
	// 'none'. Without it the service worker never starts and offline support
	// disappears with no visible error.
	"worker-src 'self'; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'"

// panelCSP covers the admin SPA, which needs two things the public site does
// not: Google's font service, and inline style attributes, because React
// components set them for measured widths and positions. 'unsafe-inline' for
// style is a real loosening; it is confined to a screen that already requires
// a session, and it does not extend to script.
const panelCSP = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"img-src 'self'; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"connect-src 'self'; " +
	"frame-src 'self'; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'"

// apiCSP covers JSON. Nothing here is ever a document, so the policy only has
// to say that.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// playCSP covers uploaded game builds, and deliberately restricts only who may
// frame them.
//
// A build is a third-party bundle: engine exports run inline script, compile
// WebAssembly, and start workers from blob URLs. A content policy written here
// would break games rather than protect anyone, and the code inside a build is
// already trusted at upload time. What is worth pinning is that only this site
// may put a game in an iframe.
const playCSP = "frame-ancestors 'self'"

// withSecurityHeaders states what each response is allowed to load.
//
// nosniff rides along because it answers the same question from the other
// side: the policy says where content may come from, and nosniff says the
// browser must believe the type we gave it rather than guessing from bytes.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policyFor(r.URL.Path))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if robots := robotsPolicyFor(r.URL.Path); robots != "" {
			w.Header().Set("X-Robots-Tag", robots)
		} else {
			w.Header().Del("X-Robots-Tag")
		}
		next.ServeHTTP(wrapRobotsResponseWriter(w), r)
	})
}

// robotsResponseWriter can still force noindex when a handler discovers that
// the requested resource does not exist. The decision has to be made at
// WriteHeader: after that point headers are already on the wire.
type robotsResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *robotsResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	if status < 100 || status > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", status))
	}
	if status == http.StatusNotFound {
		w.Header().Set("X-Robots-Tag", noindexRobots)
	}
	w.ResponseWriter.WriteHeader(status)
	if status >= 100 && status <= 199 && status != http.StatusSwitchingProtocols {
		return
	}
	w.wroteHeader = true
}

func (w *robotsResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Unwrap lets http.ResponseController reach features added by the underlying
// server writer without teaching it about this middleware.
func (w *robotsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type robotsFlusher struct{ w *robotsResponseWriter }

func (f robotsFlusher) Flush() {
	w := f.w
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.ResponseWriter.(http.Flusher).Flush()
}

type robotsHijacker struct{ w *robotsResponseWriter }

func (h robotsHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.w.ResponseWriter.(http.Hijacker).Hijack()
}

type robotsReaderFrom struct{ w *robotsResponseWriter }

func (r robotsReaderFrom) ReadFrom(src io.Reader) (int64, error) {
	w := r.w
	n, err := w.ResponseWriter.(io.ReaderFrom).ReadFrom(src)
	if n > 0 {
		w.wroteHeader = true
	}
	return n, err
}

type robotsPusher struct{ w *robotsResponseWriter }

func (p robotsPusher) Push(target string, opts *http.PushOptions) error {
	return p.w.ResponseWriter.(http.Pusher).Push(target, opts)
}

// wrapRobotsResponseWriter keeps the optional interface set identical to the
// writer supplied by net/http. Handlers may use type assertions to select
// streaming, hijacking, copy, or HTTP/2 push behavior, so inventing or hiding
// any of those capabilities would change handler behavior.
func wrapRobotsResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	rw := &robotsResponseWriter{ResponseWriter: w}
	flusher := robotsFlusher{w: rw}
	hijacker := robotsHijacker{w: rw}
	readerFrom := robotsReaderFrom{w: rw}
	pusher := robotsPusher{w: rw}

	const (
		hasFlusher = 1 << iota
		hasHijacker
		hasReaderFrom
		hasPusher
	)
	features := 0
	if _, ok := w.(http.Flusher); ok {
		features |= hasFlusher
	}
	if _, ok := w.(http.Hijacker); ok {
		features |= hasHijacker
	}
	if _, ok := w.(io.ReaderFrom); ok {
		features |= hasReaderFrom
	}
	if _, ok := w.(http.Pusher); ok {
		features |= hasPusher
	}

	// The sixteen combinations are laid out by hand rather than delegated to a
	// library, on purpose. What keeps this table honest is the exhaustive mask
	// test in security_test.go: it asserts every combination forwards every
	// optional interface the underlying writer has -- and invents none it has
	// not.
	switch features {
	case hasFlusher:
		return struct {
			*robotsResponseWriter
			http.Flusher
		}{rw, flusher}
	case hasHijacker:
		return struct {
			*robotsResponseWriter
			http.Hijacker
		}{rw, hijacker}
	case hasReaderFrom:
		return struct {
			*robotsResponseWriter
			io.ReaderFrom
		}{rw, readerFrom}
	case hasPusher:
		return struct {
			*robotsResponseWriter
			http.Pusher
		}{rw, pusher}
	case hasFlusher | hasHijacker:
		return struct {
			*robotsResponseWriter
			http.Flusher
			http.Hijacker
		}{rw, flusher, hijacker}
	case hasFlusher | hasReaderFrom:
		return struct {
			*robotsResponseWriter
			http.Flusher
			io.ReaderFrom
		}{rw, flusher, readerFrom}
	case hasFlusher | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Flusher
			http.Pusher
		}{rw, flusher, pusher}
	case hasHijacker | hasReaderFrom:
		return struct {
			*robotsResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{rw, hijacker, readerFrom}
	case hasHijacker | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Hijacker
			http.Pusher
		}{rw, hijacker, pusher}
	case hasReaderFrom | hasPusher:
		return struct {
			*robotsResponseWriter
			io.ReaderFrom
			http.Pusher
		}{rw, readerFrom, pusher}
	case hasFlusher | hasHijacker | hasReaderFrom:
		return struct {
			*robotsResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rw, flusher, hijacker, readerFrom}
	case hasFlusher | hasHijacker | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{rw, flusher, hijacker, pusher}
	case hasFlusher | hasReaderFrom | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Flusher
			io.ReaderFrom
			http.Pusher
		}{rw, flusher, readerFrom, pusher}
	case hasHijacker | hasReaderFrom | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Hijacker
			io.ReaderFrom
			http.Pusher
		}{rw, hijacker, readerFrom, pusher}
	case hasFlusher | hasHijacker | hasReaderFrom | hasPusher:
		return struct {
			*robotsResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
			http.Pusher
		}{rw, flusher, hijacker, readerFrom, pusher}
	default:
		return rw
	}
}

func policyFor(path string) string {
	switch {
	case strings.HasPrefix(path, "/play/"):
		return playCSP
	case strings.HasPrefix(path, adminUIPrefix):
		return panelCSP
	case strings.HasPrefix(path, "/api/"):
		return apiCSP
	default:
		return publicCSP
	}
}

func robotsPolicyFor(path string) string {
	switch {
	case path == "/assets" || strings.HasPrefix(path, "/assets/"),
		path == "/media" || strings.HasPrefix(path, "/media/"),
		path == site.ManifestPath,
		path == "/"+site.PageRobots,
		path == "/"+site.PageLLMS,
		path == "/"+site.PageSitemap,
		path == "/"+site.PageRSS:
		return ""
	case inACrawlerExclusion(path),
		path == site.ServiceWorkerPath,
		path == site.ShellPath:
		return noindexRobots
	default:
		return indexRobots
	}
}

// inACrawlerExclusion reports whether a path falls in a family robots.txt
// disallows. The two policies read the same list -- see site.CrawlerExclusions
// -- so neither can name a path the other has never heard of.
func inACrawlerExclusion(path string) bool {
	for _, excluded := range site.CrawlerExclusions() {
		if strings.HasSuffix(excluded, "/") {
			// A family covers its bare form too: /api and /api/ are the same
			// place to a router that trims trailing slashes.
			if path == strings.TrimSuffix(excluded, "/") || strings.HasPrefix(path, excluded) {
				return true
			}
			continue
		}
		// A single page matches exactly, so /offline-recovery is not dragged
		// in by /offline.
		if path == excluded {
			return true
		}
	}
	return false
}
