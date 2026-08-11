package httpserver

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
	"pixabros/internal/games"
	"pixabros/internal/gameupload"
	"pixabros/internal/httpapi"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	DB         *sql.DB
	Games      *games.Repo
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
	mux.HandleFunc("GET /api/admin/whoami", adminapi.RequireSession(deps.Sessions, authHandlers.Whoami))

	onGameArchiveExtracted := func(slug string) error {
		game, err := deps.Games.FindBySlug(slug)
		if err != nil {
			return err
		}
		if err := deps.Games.SetWebExportPath(game.ID, filepath.Join(deps.PlayDir, slug)); err != nil {
			return err
		}
		return render.EnqueueRegen(deps.DB, fmt.Sprintf("game:%d", game.ID))
	}
	gameUploadHandler := gameupload.NewHandler(deps.PlayDir, onGameArchiveExtracted)
	mux.HandleFunc("POST /api/admin/games/{slug}/upload", adminapi.RequireSession(deps.Sessions, gameUploadHandler.Upload))

	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such endpoint")
	}))

	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", serveAdminSPA(deps.AdminUIDir)))
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

// serveAdminSPA serves dir (the built admin SPA's output). A request that
// resolves to a real file on disk (including assets/*) is served as-is via
// noDirListing, and a genuine miss inside assets/ still 404s -- a stale
// hashed-asset URL failing as a clear 404 is far better than it being
// answered with the HTML shell and a confusing MIME-type error. Any other
// GET/HEAD path -- a client-side route like /login or /change-password that
// react-router-dom owns, not the filesystem -- falls back to index.html with
// a 200 so react-router can take over, with Cache-Control: no-store so a
// redeploy is never masked by a stale cached shell. Non-GET/HEAD requests
// get no SPA fallback at all and keep the plain file-server behaviour.
func serveAdminSPA(dir string) http.Handler {
	fileServer := noDirListing(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix leaves a relative path here (e.g. "login" or
		// "assets/index-abc123.js"), so normalise to a rooted, cleaned
		// slash path before deciding anything from it.
		urlPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		if info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(urlPath))); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}
		if urlPath == "/assets" || strings.HasPrefix(urlPath, "/assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
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
