// Package site renders the public website.
//
// Everything public-facing lives here: the templates, the stylesheet, and one
// renderer per page. Renderers are pure reads -- nothing in this package
// writes to a content table or enqueues a regen job. The admin API decides
// when content changed; this package only decides what a page looks like.
package site

import (
	"database/sql"
	"fmt"

	"pixabros/internal/awards"
	"pixabros/internal/games"
	"pixabros/internal/media"
	"pixabros/internal/members"
	"pixabros/internal/render"
	"pixabros/internal/settings"
)

// Page keys are both the storage key and the URL path minus its leading
// slash. "/" maps to index.html -- see render.ServePages.
const (
	PageLanding = "index.html"
	PageAwards  = "awards"
)

// siteSettingsTag is on every page, because the header and footer read from
// site_settings. It must match settingsapi's tag exactly or the chrome would
// never refresh.
const siteSettingsTag = "site_settings"

// Site owns the public renderers.
type Site struct {
	db       *sql.DB
	renderer *renderer
	settings *settings.Repo
	awards   *awards.Repo
	games    *games.Repo
	members  *members.Repo
	media    *media.Repo
}

func New(db *sql.DB, bundle *Bundle) (*Site, error) {
	r, err := newRenderer(bundle)
	if err != nil {
		return nil, err
	}
	return &Site{
		db:       db,
		renderer: r,
		settings: settings.NewRepo(db),
		awards:   awards.NewRepo(db),
		games:    games.NewRepo(db),
		members:  members.NewRepo(db),
		media:    media.NewRepo(db),
	}, nil
}

// pageDef ties a page key to the renderer that produces it.
type pageDef struct {
	Key    string
	Render render.Renderer
}

// pages is the one place a static page is declared. Register and
// DesiredPages both read it, so a new page cannot be registered without
// also being reconciled -- which would leave it rendering on change but
// never existing in the first place.
func (s *Site) pages() []pageDef {
	return []pageDef{
		{Key: PageLanding, Render: s.renderLanding},
		{Key: PageAwards, Render: s.renderAwards},
	}
}

// Register wires every page key to its renderer. Registration is a
// startup-only phase by convention, so this must run before the worker does.
func (s *Site) Register(registry *render.Registry) {
	for _, page := range s.pages() {
		registry.Register(page.Key, page.Render)
	}
}

// NotFoundBody renders the 404 page once, at startup. Per the architecture
// spec it stays outside the regen pipeline: it has no content dependencies
// beyond the chrome, so nothing can invalidate it.
func (s *Site) NotFoundBody() ([]byte, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, err
	}
	return s.renderer.render("404.html", pageData{
		Title:       "Page not found — " + chrome.Name,
		Description: "The page you were looking for does not exist.",
		Site:        chrome,
	})
}

// chrome reads the header and footer values every page shares.
func (s *Site) chrome() (SiteChrome, error) {
	group, err := settings.LookupGroup("site")
	if err != nil {
		return SiteChrome{}, err
	}
	values, err := s.settings.Values(group)
	if err != nil {
		return SiteChrome{}, fmt.Errorf("read site settings: %w", err)
	}

	name := values["site_name"]
	if name == "" {
		// A blank site name would render an empty header link with nothing to
		// click. The studio's name is a safer default than nothing.
		name = "Pixabros"
	}

	return SiteChrome{
		Name:    name,
		Twitter: values["twitter_handle"],
		Links:   parseLinks(values["org_sameas_json"]),
	}, nil
}
