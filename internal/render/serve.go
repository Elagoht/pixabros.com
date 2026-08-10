package render

import (
	"io"
	"net/http"
	"strings"

	"pixabros/internal/storage"
)

// ServePages serves pre-rendered HTML pages tracked in rendered_pages,
// honoring If-None-Match for 304 responses. The request path (minus the
// leading slash) is used as the page_key; a request for "/" maps to
// "index.html".
func ServePages(store *Store, files storage.Storage) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			http.NotFound(w, r)
			return
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		body, err := files.Get(renderedFileKey(pageKey))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer body.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, body)
	})
}

// ServeImmutableAssets serves static files (content-hashed CSS/JS) from dir
// with a long-lived immutable Cache-Control header.
func ServeImmutableAssets(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}
