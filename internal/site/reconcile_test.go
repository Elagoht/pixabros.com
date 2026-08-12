package site

import (
	"errors"
	"testing"

	"pixabros/internal/render"
)

func newTestReconciler(t *testing.T, s *Site, store *render.Store) (*Reconciler, *[]error) {
	t.Helper()
	registry := render.NewRegistry()
	s.Register(registry)

	var reported []error
	r := NewReconciler(s.DesiredPages, store, registry, WithReconcileErrorLogger(func(err error) {
		reported = append(reported, err)
	}))
	return r, &reported
}

// This is the whole reason the package exists: the regen worker resolves a tag
// to pages that already declared it, so a page that has never been rendered
// can never be produced by a job. Without this the site stays 404 forever.
func TestReconcile_RendersAPageThatHasNeverExisted(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)
	store := newTestStore(t, conn)
	reconciler, reported := newTestReconciler(t, site, store)

	if exists, _ := store.HasPage(PageAwards); exists {
		t.Fatal("precondition: the page should not exist yet")
	}

	rendered, removed, err := reconciler.Reconcile()
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if rendered != 1 || removed != 0 {
		t.Errorf("rendered=%d removed=%d, want 1 and 0", rendered, removed)
	}
	if len(*reported) != 0 {
		t.Errorf("unexpected errors: %v", *reported)
	}

	exists, err := store.HasPage(PageAwards)
	if err != nil {
		t.Fatalf("HasPage() error = %v", err)
	}
	if !exists {
		t.Error("the page was still missing after reconciliation")
	}
}

// Reconciliation runs on every startup and after every batch of jobs, so it
// must not re-render pages that are already there -- that would rewrite every
// ETag and defeat the 304s.
func TestReconcile_LeavesExistingPagesAlone(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)
	store := newTestStore(t, conn)
	reconciler, _ := newTestReconciler(t, site, store)

	if _, _, err := reconciler.Reconcile(); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	firstETag, _, err := store.ETag(PageAwards)
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}

	rendered, removed, err := reconciler.Reconcile()
	if err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}
	if rendered != 0 || removed != 0 {
		t.Errorf("second run rendered=%d removed=%d, want 0 and 0", rendered, removed)
	}

	secondETag, _, err := store.ETag(PageAwards)
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}
	if firstETag != secondETag {
		t.Error("the ETag changed even though nothing did, which would break every cached copy")
	}
}

// A page that is no longer part of the site must stop being served rather than
// linger until someone notices.
func TestReconcile_ForgetsPagesThatShouldNoLongerExist(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)
	store := newTestStore(t, conn)
	reconciler, _ := newTestReconciler(t, site, store)

	// A page from an imaginary earlier version of the site.
	if _, err := conn.Exec(
		`INSERT INTO rendered_pages (page_key, etag) VALUES ('games/retired-game', 'abc');`,
	); err != nil {
		t.Fatalf("seed stale page: %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO page_tags (page_key, tag) VALUES ('games/retired-game', 'game:1');`,
	); err != nil {
		t.Fatalf("seed stale tag: %v", err)
	}

	_, removed, err := reconciler.Reconcile()
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	exists, err := store.HasPage("games/retired-game")
	if err != nil {
		t.Fatalf("HasPage() error = %v", err)
	}
	if exists {
		t.Error("the retired page is still being served")
	}

	// The page's tags go with it, or the worker would keep resolving a tag to
	// a page that no longer exists.
	var tagCount int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM page_tags WHERE page_key = 'games/retired-game';`,
	).Scan(&tagCount); err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if tagCount != 0 {
		t.Errorf("page_tags rows left behind = %d, want 0", tagCount)
	}
}

// One broken page must not take the rest of the site down with it.
func TestReconcile_ReportsAFailureAndKeepsGoing(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)
	store := newTestStore(t, conn)

	registry := render.NewRegistry()
	site.Register(registry)
	registry.Register("broken", func(pageKey string) ([]byte, []string, error) {
		return nil, nil, errors.New("this page cannot be built")
	})

	var reported []error
	// Ask for a page whose renderer fails, alongside the healthy one.
	desired := func() ([]string, error) {
		pages, err := site.DesiredPages()
		return append(pages, "broken"), err
	}
	reconciler := NewReconciler(desired, store, registry, WithReconcileErrorLogger(func(err error) {
		reported = append(reported, err)
	}))

	rendered, _, err := reconciler.Reconcile()
	if err != nil {
		t.Fatalf("Reconcile() should not fail wholesale: %v", err)
	}
	if rendered != 1 {
		t.Errorf("rendered = %d, want the healthy page to still be built", rendered)
	}
	if len(reported) != 1 {
		t.Fatalf("reported %d errors, want 1", len(reported))
	}
}
