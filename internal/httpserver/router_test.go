package httpserver

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
	"pixabros/internal/games"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

func TestRouter_LoginAndSingleOriginServing(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	hash, err := auth.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	adminDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(adminDir, "index.html"), []byte("<h1>admin</h1>"), 0o644); err != nil {
		t.Fatalf("write admin index.html: %v", err)
	}
	playDir := t.TempDir()
	renderedDir := t.TempDir()
	files := storage.NewLocalDisk(renderedDir, "/rendered")
	store := render.NewStore(conn, files)
	_, err = store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>public</h1>"), []string{"public"}, nil
	})
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      files,
		DB:         conn,
		Games:      games.NewRepo(conn),
		AdminUIDir: adminDir,
		PlayDir:    playDir,
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	loginResp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error = %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusOK)
	}
	if len(loginResp.Cookies()) == 0 {
		t.Fatal("expected a session cookie after login")
	}

	client := &http.Client{}
	cookieReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/admin/whoami", nil)
	for _, c := range loginResp.Cookies() {
		cookieReq.AddCookie(c)
	}
	whoamiResp, err := client.Do(cookieReq)
	if err != nil {
		t.Fatalf("whoami request error = %v", err)
	}
	if whoamiResp.StatusCode != http.StatusOK {
		t.Fatalf("whoami status = %d, want %d", whoamiResp.StatusCode, http.StatusOK)
	}

	anonResp, err := srv.Client().Get(srv.URL + "/api/admin/whoami")
	if err != nil {
		t.Fatalf("anonymous whoami request error = %v", err)
	}
	if anonResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous whoami status = %d, want %d", anonResp.StatusCode, http.StatusUnauthorized)
	}

	adminResp, err := srv.Client().Get(srv.URL + "/I-am-a-pixabro/")
	if err != nil {
		t.Fatalf("admin UI request error = %v", err)
	}
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("admin UI status = %d, want %d", adminResp.StatusCode, http.StatusOK)
	}

	publicResp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("public request error = %v", err)
	}
	if publicResp.StatusCode != http.StatusOK {
		t.Fatalf("public status = %d, want %d", publicResp.StatusCode, http.StatusOK)
	}
}

func TestRouter_GameArchiveUpload(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	hash, err := auth.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash)

	gamesRepo := games.NewRepo(conn)
	game, err := gamesRepo.Create(games.CreateInput{Title: "Pixel Quest"})
	if err != nil {
		t.Fatalf("games.Create() error = %v", err)
	}

	playDir := t.TempDir()
	renderedFiles := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, renderedFiles)

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		DB:         conn,
		Games:      gamesRepo,
		AdminUIDir: t.TempDir(),
		PlayDir:    playDir,
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	loginResp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error = %v", err)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f, _ := zw.Create("index.html")
	f.Write([]byte("<html></html>"))
	zw.Close()

	var multipartBuf bytes.Buffer
	mw := multipart.NewWriter(&multipartBuf)
	part, _ := mw.CreateFormFile("file", "build.zip")
	part.Write(zipBuf.Bytes())
	mw.Close()

	uploadURL := fmt.Sprintf("%s/api/admin/games/%s/upload", srv.URL, game.Slug)
	uploadReq, _ := http.NewRequest(http.MethodPost, uploadURL, &multipartBuf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	for _, c := range loginResp.Cookies() {
		uploadReq.AddCookie(c)
	}

	uploadResp, err := srv.Client().Do(uploadReq)
	if err != nil {
		t.Fatalf("upload request error = %v", err)
	}
	if uploadResp.StatusCode != http.StatusNoContent {
		t.Fatalf("upload status = %d, want %d", uploadResp.StatusCode, http.StatusNoContent)
	}

	updated, err := gamesRepo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	wantPath := filepath.Join(playDir, game.Slug)
	if updated.WebExportPath != wantPath {
		t.Errorf("WebExportPath = %q, want %q", updated.WebExportPath, wantPath)
	}

	var jobCount int
	conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, fmt.Sprintf("game:%d", game.ID)).Scan(&jobCount)
	if jobCount != 1 {
		t.Errorf("regen_jobs count for game:%d = %d, want 1", game.ID, jobCount)
	}
}

// TestRouter_AdminSPAServing covers the SPA fallback: react-router-dom's
// BrowserRouter puts real browser URLs like /I-am-a-pixabro/change-password in
// the address bar, so a direct navigation, refresh or bookmark of a
// client-side route must load the shell instead of 404ing, while a stale
// content-hashed asset URL must still 404 honestly.
func TestRouter_AdminSPAServing(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	const indexHTML = `<!doctype html><title>pixabros admin</title><div id="root"></div>`
	adminDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(adminDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatalf("write admin index.html: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(adminDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "assets", "index-abc123.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      files,
		AdminUIDir: adminDir,
		PlayDir:    t.TempDir(),
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	get := func(t *testing.T, path string) (*http.Response, string) {
		t.Helper()
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body of %s: %v", path, err)
		}
		return resp, string(body)
	}

	t.Run("client-side route serves the uncached SPA shell", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/change-password")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d (a client-side route must load the shell)", resp.StatusCode, http.StatusOK)
		}
		if body != indexHTML {
			t.Errorf("body = %q, want the built index.html %q", body, indexHTML)
		}
		if got := resp.Header.Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q, want %q (a redeploy must reach the browser)", got, "no-store")
		}
	})

	t.Run("nested client-side route serves the SPA shell", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/login")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != indexHTML {
			t.Errorf("body = %q, want the built index.html", body)
		}
	})

	t.Run("missing asset 404s instead of serving the shell", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/assets/nope.js")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (a stale hashed asset must 404)", resp.StatusCode, http.StatusNotFound)
		}
		if strings.Contains(body, "<div id=\"root\">") {
			t.Errorf("body = %q, want a 404 rather than the SPA shell", body)
		}
	})

	t.Run("real asset is served as-is", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/assets/index-abc123.js")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != "console.log(1)" {
			t.Errorf("body = %q, want the real asset's contents", body)
		}
	})

	t.Run("shell root serves index.html", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != indexHTML {
			t.Errorf("body = %q, want the built index.html", body)
		}
	})

	t.Run("index.html requested by name is served as-is", func(t *testing.T) {
		resp, body := get(t, "/I-am-a-pixabro/index.html")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != indexHTML {
			t.Errorf("body = %q, want the built index.html", body)
		}
	})

	t.Run("non-GET requests get no SPA fallback", func(t *testing.T) {
		resp, err := srv.Client().Post(srv.URL+"/I-am-a-pixabro/change-password", "application/json", bytes.NewReader([]byte("{}")))
		if err != nil {
			t.Fatalf("POST error = %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (only navigations get the shell)", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("the API is never answered with the SPA shell", func(t *testing.T) {
		resp, _ := get(t, "/api/admin/whoami")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})
}

func TestRouter_UnmatchedAPIRouteReturnsJSONNotFound(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Store:      store,
		Files:      files,
		AdminUIDir: t.TempDir(),
		PlayDir:    t.TempDir(),
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/admin/does-not-exist")
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "not_found")
	}
}

// Note: with a "/" root file-server fallback already registered (as this
// router does for the public site), a wrong-method request to an
// exact-method pattern like "POST /api/admin/login" was already falling
// through to that fallback *before* this catch-all was added -- Go's
// ServeMux only synthesizes its own 405 when literally no other pattern
// matches the request for any method, and here the "/" wildcard always
// matches. So this case was never a real 405 in this router; adding the
// /api/ catch-all just changes the fallback's content from a plain-text
// file-server 404 to this project's JSON error envelope, which is the
// improvement fix 2 asks for.
func TestRouter_WrongMethodOnRegisteredRouteReturnsJSONNotFound(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      files,
		AdminUIDir: t.TempDir(),
		PlayDir:    t.TempDir(),
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/admin/login")
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestRouter_CorrectMethodOnRegisteredRouteStillWorks(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      files,
		AdminUIDir: t.TempDir(),
		PlayDir:    t.TempDir(),
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"username": "nobody", "password": "wrong"})
	resp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()
	// The exact-method pattern must still win over the /api/ catch-all for
	// a correctly-matched request; a wrong login gives 401, never 404.
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestRouter_PlayDirWithoutIndexHTMLDoesNotListDirectory(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	playDir := t.TempDir()
	sub := filepath.Join(playDir, "some-game")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "data.bin"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Store:      store,
		Files:      files,
		AdminUIDir: t.TempDir(),
		PlayDir:    playDir,
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/play/some-game/")
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (directory listing must not be exposed)", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRouter_PlayDirWithIndexHTMLServesIt(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	playDir := t.TempDir()
	sub := filepath.Join(playDir, "some-game")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "index.html"), []byte("<h1>game</h1>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := render.NewStore(conn, files)

	handler := New(Dependencies{
		Store:      store,
		Files:      files,
		AdminUIDir: t.TempDir(),
		PlayDir:    playDir,
		AssetsDir:  t.TempDir(),
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/play/some-game/")
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServeImmutableAssets_SetsCacheControlOnSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.abc123.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	handler := serveImmutableAssets(dir)
	req := httptest.NewRequest(http.MethodGet, "/main.abc123.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got := rec.Header().Get("Cache-Control")
	if !strings.Contains(got, "immutable") || !strings.Contains(got, "max-age=31536000") {
		t.Errorf("Cache-Control = %q, want it to contain immutable and max-age=31536000", got)
	}
}

func TestServeImmutableAssets_DoesNotCacheNotFound(t *testing.T) {
	handler := serveImmutableAssets(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control = %q, want empty on a 404", got)
	}
}

func TestServeImmutableAssets_DoesNotListDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "some-dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	handler := serveImmutableAssets(dir)
	req := httptest.NewRequest(http.MethodGet, "/some-dir/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (directory listing must not be exposed)", rec.Code, http.StatusNotFound)
	}
}
