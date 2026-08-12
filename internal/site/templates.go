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
)

//go:embed templates
var templateFS embed.FS

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
type SiteChrome struct {
	Name    string
	Twitter string
	Links   []string
}

// pageData is what every template can rely on: the site chrome plus whatever
// the page itself adds under Data.
type pageData struct {
	Title       string
	Description string
	// Path marks the current nav item and is not otherwise displayed.
	Path string
	CSS  string
	Nav  []navItem
	Site SiteChrome
	Year int
	Data interface{}
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
var pageTemplates = []string{"awards.html", "404.html"}

func newRenderer(bundle *Bundle) (*renderer, error) {
	// Parsed once at startup: parsing per render would turn a template typo
	// into a runtime 500 on a page that had been working.
	pages := make(map[string]*template.Template, len(pageTemplates))
	for _, name := range pageTemplates {
		parsed, err := template.New(name).Funcs(funcMap()).ParseFS(
			templateFS, "templates/layout.html", "templates/"+name,
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
