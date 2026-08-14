package site

import (
	"encoding/json"
	"net/http"
)

// ManifestPath is where the web app manifest is served. The extension is the
// registered one rather than .json: browsers key off the media type, and a
// name that says what the file is keeps the two from drifting apart.
//
// It is deliberately not a page key. Every other public URL is pre-rendered
// into the store, which serves everything as text/html; a manifest served as
// HTML is a manifest the browser ignores.
const ManifestPath = "/manifest.webmanifest"

// themeColor is what the browser paints before any CSS has loaded: the title
// bar of an installed window, and the splash screen behind the icon. It is the
// stylesheet's own --color-bg, and a test holds the two together.
//
// The dark room is the right one to promise here even though the site has a
// lit one: this colour is shown before the page renders, and a flash of the
// wrong background is worse than a consistent dark one.
const themeColor = "#050806"

// manifest is the web app manifest, in the field order the spec lists.
//
// Everything is emitted unconditionally except the description, so a browser
// reading it never has to guess whether an absent key means "unset" or "no".
type manifest struct {
	// ID is what the browser uses to tell an already-installed app from a new
	// one. Without it the start URL stands in, which means changing the start
	// URL would install a second copy of the same site.
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	ShortName       string         `json:"short_name"`
	Description     string         `json:"description,omitempty"`
	StartURL        string         `json:"start_url"`
	Scope           string         `json:"scope"`
	Display         string         `json:"display"`
	ThemeColor      string         `json:"theme_color"`
	BackgroundColor string         `json:"background_color"`
	Icons           []manifestIcon `json:"icons"`
}

type manifestIcon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type"`
}

// ManifestHandler serves the manifest, built from the current site settings on
// every request.
//
// It reads the settings rather than caching them because a manifest is fetched
// about once per install: a handful of rows is cheaper than a second copy of
// the chrome that could go stale after the studio renames the site.
func (s *Site) ManifestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chrome, err := s.chrome()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(s.manifest(chrome))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/manifest+json")
		// The manifest changes only when the studio edits its settings, and an
		// installed app re-reads it on its own schedule. Revalidating keeps a
		// rename from taking a cache lifetime to appear.
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	})
}

func (s *Site) manifest(chrome SiteChrome) manifest {
	return manifest{
		// The site is one app rooted at /, so its identity, its scope and the
		// page it opens on are all the same address.
		ID:        "/",
		Name:      chrome.Name,
		ShortName: chrome.Name,
		// The tagline is not offered here: a home screen icon shows a name and
		// nothing else, and this text is only read on an install prompt, where
		// what the studio does is more use than what it says about itself.
		Description:     chrome.Description,
		StartURL:        "/",
		Scope:           "/",
		Display:         "standalone",
		ThemeColor:      themeColor,
		BackgroundColor: themeColor,
		Icons: []manifestIcon{
			// The two sizes the install criteria name explicitly, and the mark
			// itself for every size in between -- a launcher that wants 48px
			// gets a drawing rather than a downscaled 192.
			{Src: s.renderer.bundle.URL("icon-192.png"), Sizes: "192x192", Type: "image/png"},
			{Src: s.renderer.bundle.URL("icon-512.png"), Sizes: "512x512", Type: "image/png"},
			{Src: s.renderer.bundle.URL("logo.svg"), Sizes: "any", Type: "image/svg+xml"},
		},
	}
}
