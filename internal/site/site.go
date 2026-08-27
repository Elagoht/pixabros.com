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
	"strings"

	"pixabros/internal/awards"
	"pixabros/internal/devlog"
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

	// PageOffline is what a visitor gets when they ask for a page this device
	// has never held. It is a real page so that the worker has something to
	// precache rather than a string built into the worker itself.
	PageOffline = "offline"
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
	devlog   *devlog.Repo
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
		devlog:   devlog.NewRepo(db),
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
		{Key: PageGames, Render: s.renderArcade},
		{Key: PageDevlog, Render: s.renderDevlogIndex},
		{Key: PageAwards, Render: s.renderAwards},
		{Key: PageContact, Render: s.renderContact},
		{Key: PageContactSent, Render: s.renderContactSent},
		{Key: PageOffline, Render: s.renderOffline},
	}
}

// registeredPages is the authoritative set of exact page keys. Keeping the
// HTML pages and machine-readable discovery documents in separate groups
// makes both lists legible while Register and DesiredPages still consume the
// same combined definition.
func (s *Site) registeredPages() []pageDef {
	pages := s.pages()
	return append(pages, s.discoveryPages()...)
}

// Register wires every page key to its renderer. Registration is a
// startup-only phase by convention, so this must run before the worker does.
func (s *Site) Register(registry *render.Registry) {
	for _, page := range s.registeredPages() {
		registry.Register(page.Key, page.Render)
	}
	// One renderer serves every game detail page; the key carries the slug.
	registry.RegisterPrefix(GamePagePrefix, s.renderGame)
	registry.RegisterPrefix(DevlogPagePrefix, s.renderDevlogPost)
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
		Title:       "Page not found, but the games are still where they were",
		Description: "The page you were looking for does not exist.",
		Keywords:    []string{"indie games", "game studio"},
		Robots:      RobotsNoindex,
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

	images, err := s.mediaByID()
	if err != nil {
		return SiteChrome{}, err
	}
	// A media setting stores an id; the templates and the structured data want
	// a path. Resolving it here means a deleted image leaves the field empty
	// rather than pointing at nothing.
	imageURL := func(key string) string {
		image, ok := images[values[key]]
		if !ok {
			return ""
		}
		return mediaURL(image.Path)
	}

	ogImage := imageURL("default_og_image")
	if ogImage == "" {
		// A share card with no picture is a bare link. The studio's own mark is
		// a poorer lead than artwork, and a better one than nothing.
		ogImage = imageURL("org_logo")
	}

	links := parseLinks(values["org_sameas_json"])
	views := make([]brandedLink, 0, len(links))
	for _, link := range links {
		views = append(views, brandLink(link))
	}

	return SiteChrome{
		Name:      name,
		Twitter:   values["twitter_handle"],
		Links:     links,
		LinkViews: views,

		// Trailing slashes are trimmed once, here, so nothing downstream has to
		// wonder whether the configured address ends in one.
		URL:     strings.TrimRight(strings.TrimSpace(values["site_url"]), "/"),
		Tagline: values["site_tagline"],

		LegalName:        values["org_legal_name"],
		Description:      values["org_description"],
		Email:            values["org_email"],
		FoundingDate:     values["org_founding_date"],
		FoundingLocation: values["org_founding_location"],

		LogoURL:    imageURL("org_logo"),
		OGImageURL: ogImage,
	}, nil
}
