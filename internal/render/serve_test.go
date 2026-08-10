package render

import (
	"net/http"
	"net/http/httptest"
	"os"
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
	if rec.Header().Get("ETag") != etag {
		t.Errorf("ETag header = %q, want %q", rec.Header().Get("ETag"), etag)
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
	req.Header.Set("If-None-Match", etag)
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

func TestServeImmutableAssets_SetsCacheControl(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.abc123.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	handler := ServeImmutableAssets(dir)
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
