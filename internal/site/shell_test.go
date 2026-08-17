package site

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// shellResponse mirrors the payload the worker parses. It is named apart from
// shell.go's own shellBody: both live in package site.
type shellResponse struct {
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

func fetchShell(t *testing.T, site *Site) shellResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	site.ShellHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, ShellPath, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body shellResponse
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("the shell list is not JSON: %v", err)
	}
	return body
}

// Everything a page needs to render with no network. A missing entry here is a
// page that opens offline unstyled or without its typeface.
func TestShellHandler_ListsEverythingAPageNeeds(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)

	body := fetchShell(t, site)

	for _, name := range []string{
		"site.css", "osd.js", "offline.js",
		"fonts/space-grotesk.woff2", "fonts/public-sans.woff2",
		"fonts/courier-prime.woff2", "fonts/courier-prime-700.woff2",
		"logo.svg", "icon-192.png", "icon-512.png",
	} {
		want := site.renderer.bundle.URL(name)
		if want == "" {
			t.Fatalf("%s is not in the bundle at all", name)
		}
		if !contains(body.URLs, want) {
			t.Errorf("the shell list is missing %s (%s)", name, want)
		}
	}
	// The page shown when a visitor asks for something never visited.
	if !contains(body.URLs, "/"+PageOffline) {
		t.Errorf("the shell list is missing the offline page: %v", body.URLs)
	}
}

// The worker compares this stamp to decide whether to re-precache. If it did
// not move when the stylesheet did, a visitor who never comes back online
// would open a page whose CSS is not in the cache.
func TestShellHandler_StampMovesWithTheAssets(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)

	body := fetchShell(t, site)
	if body.Version == "" {
		t.Fatal("the shell list carries no version")
	}
	if fetchShell(t, site).Version != body.Version {
		t.Error("the stamp changed without the assets changing")
	}
	if strings.Contains(body.Version, "/") {
		t.Errorf("version = %q, want an opaque stamp", body.Version)
	}
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
