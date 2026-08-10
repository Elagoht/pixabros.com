package httpserver

import (
	"net/http"
	"os"
	"path/filepath"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	AdminUIDir string
	PlayDir    string
	AssetsDir  string
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))

	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such endpoint")
	}))

	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", noDirListing(deps.AdminUIDir)))
	mux.Handle("/play/", http.StripPrefix("/play/", noDirListing(deps.PlayDir)))
	mux.Handle("/assets/", http.StripPrefix("/assets/", serveImmutableAssets(deps.AssetsDir)))
	mux.Handle("/", render.ServePages(deps.Store, deps.Files))

	return mux
}

// noDirListing wraps a directory-backed file server and refuses to serve
// automatic directory listings: a directory that lacks its own index.html
// 404s instead of listing its contents.
func noDirListing(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath := filepath.Join(dir, filepath.Clean(r.URL.Path))
		info, err := os.Stat(requestedPath)
		if err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(requestedPath, "index.html")); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

// serveImmutableAssets serves static files (content-hashed CSS/JS) from dir
// with a long-lived immutable Cache-Control header, applied only to
// successful responses so a missing asset's 404 is never cached as if it
// were a permanent answer.
func serveImmutableAssets(dir string) http.Handler {
	base := noDirListing(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base.ServeHTTP(&immutableCacheWriter{ResponseWriter: w}, r)
	})
}

type immutableCacheWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *immutableCacheWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if status >= 200 && status < 300 {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *immutableCacheWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
