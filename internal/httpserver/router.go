package httpserver

import (
	"net/http"
	"os"
	"path/filepath"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	AdminUIDir string
	PlayDir    string
	PublicDir  string
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
	mux.Handle("/", http.FileServer(http.Dir(deps.PublicDir)))

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
