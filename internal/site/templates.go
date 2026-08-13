package site

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed templates
var templateFS embed.FS

// markdown is built once and shared. WithUnsafe() is deliberately absent: raw
// HTML in a post is dropped instead of passed through, which is what makes a
// separate sanitiser unnecessary. videoEmbeds is how a post gets a player
// anyway, without being able to write the tag itself.
//
// Tables are the one GFM extension turned on. The rest of GFM is left off on
// purpose: linkify in particular would rewrite every bare URL in every post
// already written, which is a change to existing content rather than a new
// thing an author can reach for.
//
// The table extension is configured to write an align attribute rather than its
// default inline style: the site's content policy has no style-src
// 'unsafe-inline', so an inline style would be blocked and a right-aligned
// column would silently come out left-aligned. The stylesheet reads the
// attribute instead.
var markdown = goldmark.New(goldmark.WithExtensions(
	extension.NewTable(
		extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute),
	),
	videoEmbeds{},
))

// navItem is one entry in the site header. Declaring them once here is what
// lets every page mark its own link current without repeating the list.
type navItem struct {
	Label string
	Path  string
}

var navItems = []navItem{
	{Label: "Games", Path: "/games"},
	{Label: "Devlog", Path: "/devlog"},
	{Label: "Awards", Path: "/awards"},
	{Label: "Contact", Path: "/contact"},
}

// SiteChrome is what the header and footer need, read from site_settings. It
// is the same on every page, so it is fetched once per render rather than
// threaded through each page's view model.
// SiteChrome is what every page knows about the studio: the header and footer
// read the first few fields, and everything below them exists so a page can
// state its own address, its share card and its structured data.
type SiteChrome struct {
	Name    string
	Twitter string
	Links   []string

	// URL is the site's own address, with no trailing slash. Empty when the
	// studio has not set one, in which case no canonical link and no absolute
	// URL can be built and none is published.
	URL     string
	Tagline string

	LegalName        string
	Description      string
	Email            string
	FoundingDate     string
	FoundingLocation string

	// LogoURL and OGImageURL are site-relative. The share card and the
	// structured data make them absolute against URL.
	LogoURL    string
	OGImageURL string
}

// pageData is what every template can rely on: the site chrome plus whatever
// the page itself adds under Data.
type pageData struct {
	// Title is the page's own subject, not the finished <title>: the site's
	// name and the length rule are applied by the renderer, in one place, so
	// no page can opt out of them.
	Title       string
	Description string
	// Path marks the current nav item and is not otherwise displayed.
	Path string
	CSS  string
	// Favicon overrides the site icon for one page. The game pages use their
	// own cover art; everywhere else leaves it empty.
	Favicon string
	// Canonical is the page's own address, and Path above is only the nav
	// highlight, which is why they are separate: a game's page highlights
	// Games but is canonical to itself.
	Canonical string
	// OGImage is the picture a share card leads with, site-relative. Empty
	// falls back to the site's default.
	OGImage string
	// Schema is the page's structured data, already encoded.
	Schema jsonLD
	// OGType is the Open Graph type. Most pages are a website; an article is
	// the exception, so the zero value is filled in rather than repeated.
	OGType string
	// PageClass goes on <main>, for the few rules that belong to one page
	// rather than to a component.
	PageClass string
	// Scripts are per-page: only the contact form needs one, and every other
	// page ships none at all.
	Scripts []string
	Nav     []navItem
	Site    SiteChrome
	Year    int
	Data    interface{}
}

// renderer holds everything the page renderers share.
type renderer struct {
	// pages maps a template file name to the layout parsed together with just
	// that page. Every page defines a template called "main", so parsing them
	// all into one set would leave only the last one -- each page needs its
	// own set for the layout's {{block "main"}} to resolve correctly.
	pages    map[string]*template.Template
	minifier *minify.M
	bundle   *Bundle
	// now is injectable so a golden-file test does not start failing on
	// 1 January, when the footer's copyright year changes.
	now func() time.Time
}

// pageTemplates lists every page template file. Adding a page means adding it
// here; a missing entry fails at startup rather than when someone visits.
var pageTemplates = []string{
	"landing.html", "arcade.html", "game.html",
	"devlog.html", "devlog-post.html", "awards.html",
	"contact.html", "contact-sent.html", "404.html",
}

func newRenderer(bundle *Bundle) (*renderer, error) {
	// Parsed once at startup: parsing per render would turn a template typo
	// into a runtime 500 on a page that had been working.
	pages := make(map[string]*template.Template, len(pageTemplates))
	for _, name := range pageTemplates {
		parsed, err := template.New(name).Funcs(funcMap()).ParseFS(
			// partials.html carries the pieces more than one page draws -- the
			// console, for one -- so the two cannot drift apart.
			templateFS, "templates/layout.html", "templates/partials.html",
			"templates/"+name,
		)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		pages[name] = parsed
	}

	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	return &renderer{pages: pages, minifier: m, bundle: bundle, now: time.Now}, nil
}

// render executes a page template inside the layout and returns the minified
// HTML.
//
// Every renderer goes through here, so no page can accidentally ship
// unminified, and the bytes handed to render.Store are exactly the bytes
// served -- which matters because the store computes the ETag from them.
func (r *renderer) render(templateName string, data pageData) ([]byte, error) {
	parsed, ok := r.pages[templateName]
	if !ok {
		return nil, fmt.Errorf("no template named %q", templateName)
	}

	data.CSS = r.bundle.URL("site.css")
	data.Nav = navItems
	data.Year = r.now().UTC().Year()

	// Applied here rather than by each page, so the title rule and the share
	// card's fallbacks hold for every page including ones added later.
	data.Title = buildTitle(data.Site.Name, data.Site.Tagline, data.Title)
	if data.OGType == "" {
		data.OGType = "website"
	}
	if data.OGImage == "" {
		data.OGImage = data.Site.OGImageURL
	}
	data.OGImage = absoluteURL(data.Site.URL, data.OGImage)

	var raw bytes.Buffer
	if err := parsed.ExecuteTemplate(&raw, "layout", data); err != nil {
		return nil, fmt.Errorf("execute %s: %w", templateName, err)
	}

	var out bytes.Buffer
	if err := r.minifier.Minify("text/html", &out, &raw); err != nil {
		return nil, fmt.Errorf("minify %s: %w", templateName, err)
	}
	return out.Bytes(), nil
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": formatDate,
		"mediaURL":   mediaURL,
		"linkLabel":  linkLabel,
	}
}

// isoDate matches the YYYY-MM-DD form award dates are stored in.
var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// formatDate renders a stored date in English long form. The site is English
// only, so there is no locale to consult.
//
// An unparseable value is returned unchanged rather than replaced with a
// zero date, which would quietly claim the award was won in year one.
func formatDate(value string) string {
	if !isoDate.MatchString(value) {
		return value
	}
	parsed, err := time.Parse("2006-01-02", value[:10])
	if err != nil {
		return value
	}
	return parsed.Format("2 January 2006")
}

// linkLabel shortens a URL for display. A uri_list setting carries no labels,
// and a footer full of "https://..." reads like unfinished markup.
func linkLabel(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return rawURL
	}
	return parsed.Host + strings.TrimSuffix(parsed.Path, "/")
}

// mediaURL turns a stored media path into its public URL. Paths already begin
// with "media/", so this only adds the leading slash.
func mediaURL(path string) string {
	if path == "" {
		return ""
	}
	return "/" + path
}
