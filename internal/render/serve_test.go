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
