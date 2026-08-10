package render

import (
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	return NewStore(conn, files)
}

func TestStore_RenderAndPersist_WritesFileAndTags(t *testing.T) {
	store := setupTestStore(t)

	renderer := func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>hi</h1>"), []string{"homepage", "site_settings"}, nil
	}

	etag, err := store.RenderAndPersist("index.html", renderer)
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}
	if etag == "" {
		t.Fatal("RenderAndPersist() returned an empty etag")
	}

	gotEtag, found, err := store.ETag("index.html")
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}
	if !found {
		t.Fatal("ETag() should find the just-rendered page")
	}
	if gotEtag != etag {
		t.Errorf("ETag() = %q, want %q", gotEtag, etag)
	}

	pages, err := store.PageKeysForTag("homepage")
	if err != nil {
		t.Fatalf("PageKeysForTag() error = %v", err)
	}
	if len(pages) != 1 || pages[0] != "index.html" {
		t.Errorf("PageKeysForTag(\"homepage\") = %v, want [index.html]", pages)
	}
}

func TestStore_RenderAndPersist_ReplacesOldTags(t *testing.T) {
	store := setupTestStore(t)

	first := func(pageKey string) ([]byte, []string, error) {
		return []byte("v1"), []string{"tag-a"}, nil
	}
	if _, err := store.RenderAndPersist("page.html", first); err != nil {
		t.Fatalf("first RenderAndPersist() error = %v", err)
	}

	second := func(pageKey string) ([]byte, []string, error) {
		return []byte("v2"), []string{"tag-b"}, nil
	}
	if _, err := store.RenderAndPersist("page.html", second); err != nil {
		t.Fatalf("second RenderAndPersist() error = %v", err)
	}

	pagesForOldTag, err := store.PageKeysForTag("tag-a")
	if err != nil {
		t.Fatalf("PageKeysForTag(\"tag-a\") error = %v", err)
	}
	if len(pagesForOldTag) != 0 {
		t.Errorf("PageKeysForTag(\"tag-a\") = %v, want empty after re-render dropped that tag", pagesForOldTag)
	}

	pagesForNewTag, err := store.PageKeysForTag("tag-b")
	if err != nil {
		t.Fatalf("PageKeysForTag(\"tag-b\") error = %v", err)
	}
	if len(pagesForNewTag) != 1 || pagesForNewTag[0] != "page.html" {
		t.Errorf("PageKeysForTag(\"tag-b\") = %v, want [page.html]", pagesForNewTag)
	}
}

func TestStore_ETag_NotFound(t *testing.T) {
	store := setupTestStore(t)
	_, found, err := store.ETag("missing.html")
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}
	if found {
		t.Error("ETag() should report not-found for a page that was never rendered")
	}
}
