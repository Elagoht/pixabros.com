package httpserver

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"pixabros/internal/adminapi"
	"pixabros/internal/adminui"
	"pixabros/internal/auth"
	"pixabros/internal/awards"
	"pixabros/internal/awardsapi"
	"pixabros/internal/contact"
	"pixabros/internal/contactapi"
	"pixabros/internal/devlog"
	"pixabros/internal/devlogapi"
	"pixabros/internal/games"
	"pixabros/internal/gamesapi"
	"pixabros/internal/gameupload"
	"pixabros/internal/httpapi"
	"pixabros/internal/media"
	"pixabros/internal/mediaapi"
	"pixabros/internal/members"
	"pixabros/internal/membersapi"
	"pixabros/internal/ogimage"
	"pixabros/internal/render"
	"pixabros/internal/settings"
	"pixabros/internal/settingsapi"
	"pixabros/internal/stats"
	"pixabros/internal/statsapi"
	"pixabros/internal/storage"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	DB         *sql.DB
	Games      *games.Repo
	Members    *members.Repo
	Awards     *awards.Repo
	Devlog     *devlog.Repo
	Contact    *contact.Repo
	Stats      *stats.Repo
	Settings   *settings.Repo
	Media      *media.Repo
	MediaFiles storage.Storage
	MediaDir   string
	// NotFoundBody is the styled public 404 page. Empty falls back to plain
	// text, which is what non-public deployments and tests get.
	NotFoundBody []byte
	PlayDir      string
	AssetsDir    string
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))
	mux.HandleFunc("GET /api/admin/whoami", adminapi.RequireSession(deps.Sessions, authHandlers.Whoami))

	gamesHandlers := gamesapi.NewHandlers(deps.Games, deps.DB, deps.PlayDir)
	mux.HandleFunc("GET /api/admin/games", adminapi.RequireSession(deps.Sessions, gamesHandlers.List))
	mux.HandleFunc("POST /api/admin/games", adminapi.RequireSession(deps.Sessions, gamesHandlers.Create))
	// Registered above GET/PUT /api/admin/games/{id}: Go's ServeMux ranks a
	// fully static segment ("reorder") over a wildcard ("{id}") regardless of
	// registration order, but keeping it here keeps the list readable.
	mux.HandleFunc("PUT /api/admin/games/reorder", adminapi.RequireSession(deps.Sessions, gamesHandlers.Reorder))
	mux.HandleFunc("GET /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Get))
	mux.HandleFunc("PUT /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/games/{id}", adminapi.RequireSession(deps.Sessions, gamesHandlers.Delete))
	mux.HandleFunc("GET /api/admin/games/{id}/screenshots", adminapi.RequireSession(deps.Sessions, gamesHandlers.ListScreenshots))
	mux.HandleFunc("POST /api/admin/games/{id}/screenshots", adminapi.RequireSession(deps.Sessions, gamesHandlers.AddScreenshot))
	mux.HandleFunc("PUT /api/admin/games/{id}/screenshots/reorder", adminapi.RequireSession(deps.Sessions, gamesHandlers.ReorderScreenshots))
	mux.HandleFunc("DELETE /api/admin/games/{id}/screenshots/{screenshotID}", adminapi.RequireSession(deps.Sessions, gamesHandlers.RemoveScreenshot))

	membersHandlers := membersapi.NewHandlers(deps.Members, deps.DB)
	mux.HandleFunc("GET /api/admin/members", adminapi.RequireSession(deps.Sessions, membersHandlers.List))
	mux.HandleFunc("POST /api/admin/members", adminapi.RequireSession(deps.Sessions, membersHandlers.Create))
	// Above /members/{id} for the same reason as games/reorder: a static
	// segment outranks a wildcard regardless of registration order.
	mux.HandleFunc("PUT /api/admin/members/reorder", adminapi.RequireSession(deps.Sessions, membersHandlers.Reorder))
	mux.HandleFunc("GET /api/admin/members/{id}", adminapi.RequireSession(deps.Sessions, membersHandlers.Get))
	mux.HandleFunc("PUT /api/admin/members/{id}", adminapi.RequireSession(deps.Sessions, membersHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/members/{id}", adminapi.RequireSession(deps.Sessions, membersHandlers.Delete))

	awardsHandlers := awardsapi.NewHandlers(deps.Awards, deps.DB)
	mux.HandleFunc("GET /api/admin/awards", adminapi.RequireSession(deps.Sessions, awardsHandlers.List))
	mux.HandleFunc("POST /api/admin/awards", adminapi.RequireSession(deps.Sessions, awardsHandlers.Create))
	mux.HandleFunc("GET /api/admin/awards/{id}", adminapi.RequireSession(deps.Sessions, awardsHandlers.Get))
	mux.HandleFunc("PUT /api/admin/awards/{id}", adminapi.RequireSession(deps.Sessions, awardsHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/awards/{id}", adminapi.RequireSession(deps.Sessions, awardsHandlers.Delete))

	devlogHandlers := devlogapi.NewHandlers(deps.Devlog, deps.DB, ogimage.NewStore(deps.Media, deps.MediaFiles))
	mux.HandleFunc("GET /api/admin/devlog", adminapi.RequireSession(deps.Sessions, devlogHandlers.List))
	mux.HandleFunc("POST /api/admin/devlog", adminapi.RequireSession(deps.Sessions, devlogHandlers.Create))
	mux.HandleFunc("GET /api/admin/devlog/{id}", adminapi.RequireSession(deps.Sessions, devlogHandlers.Get))
	mux.HandleFunc("PUT /api/admin/devlog/{id}", adminapi.RequireSession(deps.Sessions, devlogHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/devlog/{id}", adminapi.RequireSession(deps.Sessions, devlogHandlers.Delete))

	contactHandlers := contactapi.NewHandlers(deps.Contact)
	statsHandlers := statsapi.NewHandlers(deps.Stats)
	publicContact := contactapi.NewPublicHandlers(deps.Contact)
	mux.HandleFunc("POST /api/contact", publicContact.Submit)

	mux.HandleFunc("GET /api/admin/stats", adminapi.RequireSession(deps.Sessions, statsHandlers.Get))

	mux.HandleFunc("GET /api/admin/contact", adminapi.RequireSession(deps.Sessions, contactHandlers.List))
	mux.HandleFunc("GET /api/admin/contact/{id}", adminapi.RequireSession(deps.Sessions, contactHandlers.Get))
	// Read state is the only thing the admin can change about a submission,
	// so it gets its own sub-resource rather than a general update.
	mux.HandleFunc("PUT /api/admin/contact/{id}/read", adminapi.RequireSession(deps.Sessions, contactHandlers.SetRead))
	mux.HandleFunc("DELETE /api/admin/contact/{id}", adminapi.RequireSession(deps.Sessions, contactHandlers.Delete))

	settingsHandlers := settingsapi.NewHandlers(deps.Settings, deps.DB)
	mux.HandleFunc("GET /api/admin/settings/{group}", adminapi.RequireSession(deps.Sessions, settingsHandlers.Get))
	mux.HandleFunc("PUT /api/admin/settings/{group}", adminapi.RequireSession(deps.Sessions, settingsHandlers.Update))

	mediaUploadHandler := mediaapi.NewUploadHandler(deps.Media, deps.MediaFiles)
	mediaHandlers := mediaapi.NewHandlers(deps.Media, deps.MediaFiles, deps.DB)
	mux.HandleFunc("POST /api/admin/media/upload", adminapi.RequireSession(deps.Sessions, mediaUploadHandler.Upload))
	mux.HandleFunc("GET /api/admin/media", adminapi.RequireSession(deps.Sessions, mediaHandlers.List))
	mux.HandleFunc("GET /api/admin/media/{id}", adminapi.RequireSession(deps.Sessions, mediaHandlers.Get))
	mux.HandleFunc("PUT /api/admin/media/{id}", adminapi.RequireSession(deps.Sessions, mediaHandlers.Update))
	mux.HandleFunc("DELETE /api/admin/media/{id}", adminapi.RequireSession(deps.Sessions, mediaHandlers.Delete))

	onGameArchiveExtracted := func(slug string) error {
		game, err := deps.Games.FindBySlug(slug)
		if err != nil {
			return err
		}
		if err := deps.Games.SetBuild(game.ID, filepath.Join(deps.PlayDir, slug)); err != nil {
			return err
		}
		return render.EnqueueRegen(deps.DB, fmt.Sprintf("game:%s", game.ID))
	}
	gameUploadHandler := gameupload.NewHandler(deps.PlayDir, onGameArchiveExtracted, gameupload.WithErrorLogger(func(err error) {
		log.Printf("game upload: %v", err)
	}))

	// requireGameSlug resolves {slug} to a real game before the upload handler
	// touches the filesystem at all: without it, uploading against a slug
	// nobody created extracts a whole archive into a publicly served
	// directory and only then 500s, leaving an orphaned build on disk.
	requireGameSlug := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			slug := r.PathValue("slug")
			if _, err := deps.Games.FindBySlug(slug); err != nil {
				if errors.Is(err, games.ErrGameNotFound) {
					httpapi.WriteError(w, http.StatusNotFound, "not_found", "no game with that slug")
					return
				}
				httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not look up game")
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("POST /api/admin/games/{slug}/upload", adminapi.RequireSession(deps.Sessions, requireGameSlug(gameUploadHandler.Upload)))
	// Addressed by {id} like the rest of the games API; only the upload above
	// needs the slug, because the extracted directory is named after it.
	mux.HandleFunc("DELETE /api/admin/games/{id}/build", adminapi.RequireSession(deps.Sessions, gamesHandlers.DeleteBuild))

	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such endpoint")
	}))

	adminFS, err := adminui.FS()
	if err != nil {
		// The panel is embedded at compile time, so this can only fail if the
		// binary was built wrong -- not something a request can cause.
		panic("admin panel is not embedded: " + err.Error())
	}
	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", serveAdminSPA(adminFS)))
	mux.Handle("/play/", http.StripPrefix("/play/", noDirListing(deps.PlayDir)))
	// Uploaded media is public site content (cartridge art, screenshots, OG
	// images all appear on the public MPA), so it is served without a session.
	// It gets plain noDirListing rather than the immutable-cache treatment
	// /assets/ uses: media keys are not content-hashed, so a replaced or
	// deleted image has to be able to actually disappear.
	// deps.MediaDir is empty in several pre-existing Dependencies{} test
	// literals that don't care about media at all. http.Dir("").Open rewrites
	// "" to ".", so mounting unconditionally would make this unauthenticated
	// public route serve the process's current working directory the moment
	// MediaDir is ever left unset. Production always sets a real MediaDir, but
	// the guard is cheap and this route is new and unauthenticated by design.
	if deps.MediaDir != "" {
		mux.Handle("/media/", http.StripPrefix("/media/", noDirListing(deps.MediaDir)))
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", serveImmutableAssets(deps.AssetsDir)))
	publicPages := render.ServePages(deps.Store, deps.Files)
	if len(deps.NotFoundBody) > 0 {
		publicPages = render.ServePages(deps.Store, deps.Files, render.WithNotFoundPage(deps.NotFoundBody))
	}
	mux.Handle("/", publicPages)

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
// serveAdminSPA serves the embedded admin panel. Anything that is not a real
// file falls back to index.html, because the SPA owns its own routing.
func serveAdminSPA(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix leaves a relative path here (e.g. "login" or
		// "assets/index-abc123.js"), so normalise to a rooted, cleaned
		// slash path before deciding anything from it.
		urlPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))

		if file, err := fsys.Open(strings.TrimPrefix(urlPath, "/")); err == nil {
			info, statErr := file.Stat()
			file.Close()
			if statErr == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}
		// A missing asset must 404 rather than fall through to index.html:
		// answering a stale script request with HTML breaks the panel in a way
		// that is very hard to read from the console.
		if urlPath == "/assets" || strings.HasPrefix(urlPath, "/assets/") {
			http.NotFound(w, r)
			return
		}

		index, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.Error(w, "admin panel not built", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
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
