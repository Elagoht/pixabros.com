package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
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
