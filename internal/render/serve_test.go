package render

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

// setupServeTest builds an empty store, for the cases that only care about
// what happens when a page is absent.
func setupServeTest(t *testing.T) (*Store, storage.Storage) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	return NewStore(conn, files), files
}

func TestServePages_ServesRenderedHTMLWithETag(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	etag, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	wantETag := "\"" + etag + "\""
	if rec.Header().Get("ETag") != wantETag {
		t.Errorf("ETag header = %q, want %q", rec.Header().Get("ETag"), wantETag)
	}
	if !strings.Contains(rec.Body.String(), "<h1>home</h1>") {
		t.Errorf("body = %q, want it to contain the rendered HTML", rec.Body.String())
	}
}

func TestServePages_ReturnsNotModifiedOnMatchingETag(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	etag, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	req.Header.Set("If-None-Match", "\""+etag+"\"")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}
}

func TestServePages_ContentTypes(t *testing.T) {
	store, files := setupServeTest(t)

	tests := map[string]string{
		"index.html":  "text/html; charset=utf-8",
		"robots.txt":  "text/plain; charset=utf-8",
		"llms.txt":    "text/plain; charset=utf-8",
		"sitemap.xml": "application/xml; charset=utf-8",
		"rss.xml":     "application/rss+xml; charset=utf-8",
	}

	etags := make(map[string]string, len(tests))
	for key := range tests {
		body := []byte("body for " + key)
		etag, err := store.RenderAndPersist(key, func(string) ([]byte, []string, error) {
			return body, nil, nil
		})
		if err != nil {
			t.Fatalf("RenderAndPersist(%q) error = %v", key, err)
		}
		etags[key] = etag
	}

	handler := ServePages(store, files)
	for key, wantContentType := range tests {
		wantBody := "body for " + key
		wantETag := `"` + etags[key] + `"`

		getRec := httptest.NewRecorder()
		handler.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/"+key, nil))
		if getRec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", key, getRec.Code, http.StatusOK)
		}
		if got := getRec.Header().Get("Content-Type"); got != wantContentType {
			t.Errorf("GET %s Content-Type = %q, want %q", key, got, wantContentType)
		}
		if got := getRec.Header().Get("ETag"); got != wantETag {
			t.Errorf("GET %s ETag = %q, want %q", key, got, wantETag)
		}
		if got := getRec.Body.String(); got != wantBody {
			t.Errorf("GET %s body = %q, want %q", key, got, wantBody)
		}

		headRec := httptest.NewRecorder()
		handler.ServeHTTP(headRec, httptest.NewRequest(http.MethodHead, "/"+key, nil))
		if headRec.Code != http.StatusOK {
			t.Fatalf("HEAD %s status = %d, want %d", key, headRec.Code, http.StatusOK)
		}
		if got := headRec.Header().Get("Content-Type"); got != wantContentType {
			t.Errorf("HEAD %s Content-Type = %q, want %q", key, got, wantContentType)
		}
		if got := headRec.Header().Get("ETag"); got != wantETag {
			t.Errorf("HEAD %s ETag = %q, want %q", key, got, wantETag)
		}
		if got := headRec.Body.String(); got != "" {
			t.Errorf("HEAD %s body = %q, want empty body", key, got)
		}
	}
}

func TestServePages_UnknownPageReturns404(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/never-rendered.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServePages_RejectsNonGetHead(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	if _, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	}); err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestServePages_PrefixedPageKeysDoNotCollide guards the content-addressed
// rendered-file key: with page keys stored verbatim, "games" could not be both
// a file and a parent directory of "games/pixel-quest" on local disk.
func TestServePages_PrefixedPageKeysDoNotCollide(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	if _, err := store.RenderAndPersist("games", func(string) ([]byte, []string, error) {
		return []byte("<h1>games</h1>"), []string{"games"}, nil
	}); err != nil {
		t.Fatalf("RenderAndPersist(games) error = %v", err)
	}
	if _, err := store.RenderAndPersist("games/pixel-quest", func(string) ([]byte, []string, error) {
		return []byte("<h1>pixel quest</h1>"), []string{"games"}, nil
	}); err != nil {
		t.Fatalf("RenderAndPersist(games/pixel-quest) error = %v", err)
	}

	handler := ServePages(store, files)
	for path, want := range map[string]string{
		"/games":             "<h1>games</h1>",
		"/games/pixel-quest": "<h1>pixel quest</h1>",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET %s body = %q, want it to contain %q", path, rec.Body.String(), want)
		}
	}
}

// Without a supplied body the handler must behave exactly as it always has,
// so the fallback stays meaningful for anything not serving a public site.
func TestServePages_FallsBackToPlainNotFound(t *testing.T) {
	store, files := setupServeTest(t)

	handler := ServePages(store, files)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want the plain-text fallback", ct)
	}
}

// A styled 404 is what a visitor should see: the site's own header and footer,
// not a bare line of text that looks like the server is broken.
func TestServePages_ServesTheSuppliedNotFoundPage(t *testing.T) {
	store, files := setupServeTest(t)

	body := []byte("<!doctype html><title>Not found</title><body>missing</body>")
	handler := ServePages(store, files, WithNotFoundPage(body))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "missing") {
		t.Error("the supplied 404 body was not served")
	}
}
