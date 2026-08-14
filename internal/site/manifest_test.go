package site

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeManifest fetches the manifest through its handler and decodes it, so
// every test here checks the bytes a browser would actually receive.
func decodeManifest(t *testing.T, site *Site) (map[string]any, *http.Response) {
	t.Helper()

	recorder := httptest.NewRecorder()
	site.ManifestHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, ManifestPath, nil))
	response := recorder.Result()

	var manifest map[string]any
	if err := json.NewDecoder(response.Body).Decode(&manifest); err != nil {
		t.Fatalf("the manifest is not JSON: %v (body %q)", err, recorder.Body.String())
	}
	return manifest, response
}

// A manifest served as JSON is ignored: browsers require the manifest type.
func TestManifestHandler_ServesTheManifestMediaType(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})

	_, response := decodeManifest(t, newTestSite(t, conn))

	if got := response.Header.Get("Content-Type"); got != "application/manifest+json" {
		t.Errorf("Content-Type = %q, want application/manifest+json", got)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", response.StatusCode)
	}
}

// The installed app is named by the studio, not by the code: renaming the site
// in the panel has to rename the icon on the home screen too.
func TestManifest_TakesItsNameFromTheSettings(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_name":       "Pixabros",
		"site_tagline":    "Brothers Make Games",
		"org_description": "A two-person studio.",
	})

	manifest, _ := decodeManifest(t, newTestSite(t, conn))

	if manifest["name"] != "Pixabros" {
		t.Errorf("name = %v, want Pixabros", manifest["name"])
	}
	if manifest["short_name"] != "Pixabros" {
		t.Errorf("short_name = %v, want Pixabros", manifest["short_name"])
	}
	if manifest["description"] != "A two-person studio." {
		t.Errorf("description = %v, want the studio description", manifest["description"])
	}
}

// The install prompt needs all of these; a manifest missing any one of them is
// a manifest the browser reads and then declines to offer.
func TestManifest_MeetsTheInstallCriteria(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})

	site := newTestSite(t, conn)
	manifest, _ := decodeManifest(t, site)

	for key, want := range map[string]string{
		"start_url": "/",
		"scope":     "/",
		"display":   "standalone",
		"id":        "/",
	} {
		if manifest[key] != want {
			t.Errorf("%s = %v, want %q", key, manifest[key], want)
		}
	}

	icons, ok := manifest["icons"].([]any)
	if !ok {
		t.Fatalf("icons = %v, want a list", manifest["icons"])
	}
	// A raster icon at each of the two sizes the install criteria name, plus
	// the vector one that covers every size in between.
	wantSources := map[string]string{
		"192x192": site.renderer.bundle.URL("icon-192.png"),
		"512x512": site.renderer.bundle.URL("icon-512.png"),
		"any":     site.renderer.bundle.URL("logo.svg"),
	}
	for _, entry := range icons {
		icon, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("icon entry = %v, want an object", entry)
		}
		sizes, _ := icon["sizes"].(string)
		want, expected := wantSources[sizes]
		if !expected {
			t.Errorf("unexpected icon size %q", sizes)
			continue
		}
		if icon["src"] != want {
			t.Errorf("the %s icon points at %v, want %q", sizes, icon["src"], want)
		}
		delete(wantSources, sizes)
	}
	for sizes := range wantSources {
		t.Errorf("no %s icon in the manifest", sizes)
	}
}

// The splash screen and the title bar are drawn from these before any CSS
// loads, so they have to match the stylesheet's own background.
func TestManifest_CarriesTheSiteColours(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})

	manifest, _ := decodeManifest(t, newTestSite(t, conn))

	for _, key := range []string{"theme_color", "background_color"} {
		if manifest[key] != themeColor {
			t.Errorf("%s = %v, want %q", key, manifest[key], themeColor)
		}
	}
	// The colour is a promise about what the page will look like, so it has to
	// be the one the stylesheet actually paints.
	if !strings.Contains(string(siteCSS(t)), "--color-bg: "+themeColor) {
		t.Errorf("the stylesheet does not declare %s as its background", themeColor)
	}
}

// A blank site name would install an app with no name at all, which is the
// same fallback the header takes.
func TestManifest_NamesTheStudioWhenTheSettingIsBlank(t *testing.T) {
	conn := setupTestDB(t)

	manifest, _ := decodeManifest(t, newTestSite(t, conn))

	if manifest["name"] != "Pixabros" {
		t.Errorf("name = %v, want the Pixabros fallback", manifest["name"])
	}
}
