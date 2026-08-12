package render

import (
	"io"
	"net/http"
	"strings"

	"pixabros/internal/storage"
)

// ServeOption configures ServePages.
type ServeOption func(*serveConfig)

type serveConfig struct {
	notFoundBody []byte
}

// WithNotFoundPage supplies the body to send for an unknown path. Without it
// the handler falls back to net/http's plain-text 404, which is correct but
// looks like the site is broken rather than like a page is missing.
//
// The body is passed in rather than looked up, because the 404 page is
// rendered once at startup and deliberately sits outside the regen pipeline --
// it has no content dependencies that could invalidate it.
func WithNotFoundPage(body []byte) ServeOption {
	return func(c *serveConfig) { c.notFoundBody = body }
}

// ServePages serves pre-rendered HTML pages tracked in rendered_pages,
// honoring If-None-Match for 304 responses. The request path (minus the
// leading slash) is used as the page_key; a request for "/" maps to
// "index.html".
func ServePages(store *Store, files storage.Storage, opts ...ServeOption) http.Handler {
	cfg := &serveConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	notFound := func(w http.ResponseWriter, r *http.Request) {
		if cfg.notFoundBody == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write(cfg.notFoundBody)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pageKey := strings.TrimPrefix(r.URL.Path, "/")
		if pageKey == "" {
			pageKey = "index.html"
		}

		etag, found, err := store.ETag(pageKey)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !found {
			notFound(w, r)
			return
		}

		// RFC 9110 §8.8.3 requires an entity-tag to be a quoted string; an
		// intermediary may normalize or drop a bare one.
		quoted := `"` + etag + `"`
		w.Header().Set("ETag", quoted)
		w.Header().Set("Cache-Control", "no-cache")

		if ifNoneMatchHas(r.Header.Get("If-None-Match"), quoted) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		body, err := files.Get(renderedFileKey(etag))
		if err != nil {
			notFound(w, r)
			return
		}
		defer body.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, body)
	})
}

// ifNoneMatchHas reports whether header (a comma-separated If-None-Match
// list, each entry optionally weak-prefixed with "W/") contains quoted,
// or is the wildcard "*".
func ifNoneMatchHas(header, quoted string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "*" {
			return true
		}
		part = strings.TrimPrefix(part, "W/")
		if part == quoted {
			return true
		}
	}
	return false
}
