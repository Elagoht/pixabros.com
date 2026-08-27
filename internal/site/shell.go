package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ShellPath is where the service worker asks what a page needs to render with
// no network.
const ShellPath = "/api/shell"

// ServiceWorkerPath is where the worker is served. It must be at the root:
// a worker's default scope is the directory it was served from.
const ServiceWorkerPath = "/sw.js"

// stampLength is how much of the sha256 becomes the shell's version. It only
// has to change when the list does, so it can be short.
const stampLength = 12

// shellAssets are the bundle names every page depends on. They are listed
// rather than derived from the bundle's whole contents so that adding a
// page-specific script does not silently enlarge what every visitor
// downloads on install.
var shellAssets = []string{
	"site.css",
	"osd.js",
	"offline.js",
	"fonts/space-grotesk.woff2",
	"fonts/public-sans.woff2",
	"fonts/courier-prime.woff2",
	"fonts/courier-prime-700.woff2",
	"logo.svg",
	"icon-192.png",
	"icon-512.png",
}

type shellBody struct {
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

// ShellHandler answers with the URLs a page needs offline, and a stamp that
// changes when they do.
//
// The worker cannot hold these URLs itself: they are content-hashed, so they
// move whenever the stylesheet or a script changes. The stamp is what lets a
// worker notice the move without re-downloading the list's contents.
func (s *Site) ShellHandler() http.Handler {
	urls := make([]string, 0, len(shellAssets)+1)
	for _, name := range shellAssets {
		if url := s.renderer.bundle.URL(name); url != "" {
			urls = append(urls, url)
		}
	}
	urls = append(urls, "/"+PageOffline)

	sum := sha256.Sum256([]byte(strings.Join(urls, "\n")))
	body, err := json.Marshal(shellBody{
		Version: hex.EncodeToString(sum[:])[:stampLength],
		URLs:    urls,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// The list moves with a deploy, and a worker holding an old one would
		// precache assets that no longer exist.
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	})
}

// ServiceWorkerHandler serves the worker at a stable path.
//
// Everything else under /assets is content-hashed and cached forever. The
// worker cannot be: a browser decides whether to update one by re-fetching the
// same URL and comparing bytes, so a moving URL would mean a worker that never
// updates in place. The bytes are read once at startup because the bundle is
// built once at startup.
func ServiceWorkerHandler(bundle *Bundle, assetsDir string) (http.Handler, error) {
	url := bundle.URL("sw.js")
	if url == "" {
		return nil, errNoServiceWorker
	}
	body, err := os.ReadFile(filepath.Join(assetsDir, strings.TrimPrefix(url, "/assets/")))
	if err != nil {
		return nil, fmt.Errorf("read service worker: %w", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	}), nil
}

var errNoServiceWorker = errors.New("sw.js is not in the asset bundle")

// renderOffline draws the page shown when a visitor asks for something this
// device has never held.
func (s *Site) renderOffline(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	html, err := s.renderer.render("offline.html", pageData{
		Title:       "You are offline",
		Description: "This page has not been opened on this device before.",
		Keywords:    []string{"offline games", "indie games"},
		Canonical:   canonicalURL(chrome.URL, PageOffline),
		Site:        chrome,
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{siteSettingsTag}, nil
}
