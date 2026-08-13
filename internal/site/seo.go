package site

import (
	"encoding/json"
	"html/template"
	"strings"
	"unicode/utf8"
)

// Everything a search result or a share card is built from lives here: the
// title rule, the canonical address, and the structured data.
//
// The values come from site settings rather than from constants, so the studio
// can change how it describes itself without a deploy. What cannot be
// configured is the shape: a title is always built the same way, and a page
// always states its own address.

// Title length. Search engines truncate a long title and skip a short one, so
// the rule is a floor as well as a ceiling.
const (
	titleMinLength = 65
	titleMaxLength = 100
)

// defaultTagline is what a title too short to be useful is padded with when the
// studio has not set one of its own.
const defaultTagline = "Brothers Makes Games"

// buildTitle assembles a page's title.
//
// Always the site's name first, so a tab or a result row is recognisable before
// it is read. Then the page's own subject. A result that would still be short
// gets the tagline, because a search engine skips a title that says too little.
// Anything past the ceiling is cut on a word and closed with an ellipsis, so it
// ends somewhere deliberate rather than mid-word where the browser would cut it.
func buildTitle(siteName, tagline, subject string) string {
	if siteName == "" {
		siteName = "Pixabros"
	}
	if tagline == "" {
		tagline = defaultTagline
	}

	title := siteName
	if subject = strings.TrimSpace(subject); subject != "" {
		title += " | " + subject
	}
	if utf8.RuneCountInString(title) < titleMinLength {
		title += " | " + tagline
	}
	return truncateRunes(title, titleMaxLength)
}

// gameSubject names a game the way a search result should read it.
//
// A game's own name is usually two words, which leaves a title short even after
// the tagline. What it is and who made it are both known for every game, so the
// subject says them rather than the title being padded with nothing.
func gameSubject(title, genre, siteName string) string {
	if genre = strings.TrimSpace(genre); genre != "" {
		return title + ", a " + strings.ToLower(genre) + " game by " + siteName
	}
	return title + ", a game by " + siteName
}

// postSubject does the same for a devlog post, whose title is written as a
// headline and rarely says where it was published.
func postSubject(title, siteName string) string {
	return title + ", from the " + siteName + " devlog"
}

// truncateRunes cuts to at most limit characters, ellipsis included, breaking
// on a space when there is one to break on.
func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	runes := []rune(s)
	cut := string(runes[:limit-1])
	// A cut mid-word reads as a mistake, so back up to the last space when one
	// is close enough to be the same phrase.
	if space := strings.LastIndex(cut, " "); space > limit/2 {
		cut = cut[:space]
	}
	return strings.TrimRight(cut, " ,.;:-") + "…"
}

// canonicalURL is the one address a page should be indexed under.
//
// Every page states its own, which is what stops a query string or a trailing
// slash being treated as a separate page. With no site address configured there
// is nothing truthful to state, so nothing is stated.
func canonicalURL(baseURL, path string) string {
	if baseURL == "" {
		return ""
	}
	base := strings.TrimRight(baseURL, "/")
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "index.html" {
		return base + "/"
	}
	return base + "/" + path
}

// absoluteURL turns a site-relative path into one a crawler or a chat client
// can fetch. Share cards and structured data are read off-site, where a
// relative path means nothing.
func absoluteURL(baseURL, path string) string {
	if path == "" || baseURL == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
}

// jsonLD is one block of structured data, ready for the template.
type jsonLD = template.JS

// schema is a node in a structured-data graph. A map rather than a struct
// because the vocabulary is open and a property is left out entirely when its
// value is unknown -- an empty string published as a name is worse than no name.
type schema map[string]any

// set records a property, skipping anything empty. This is what keeps blank
// settings out of the published graph.
func (s schema) set(key string, value any) {
	switch v := value.(type) {
	case nil:
		return
	case string:
		if strings.TrimSpace(v) == "" {
			return
		}
	case []string:
		if len(v) == 0 {
			return
		}
	case []any:
		if len(v) == 0 {
			return
		}
	case schema:
		if len(v) == 0 {
			return
		}
	}
	s[key] = value
}

// organization describes the studio. It is the publisher of everything on the
// site, so it is referenced by @id from the other nodes rather than repeated.
func (c SiteChrome) organization() schema {
	org := schema{
		"@type": "Organization",
		"@id":   c.URL + "#organization",
		"name":  c.Name,
	}
	org.set("url", c.URL)
	org.set("legalName", c.LegalName)
	org.set("description", c.Description)
	org.set("foundingDate", c.FoundingDate)
	org.set("foundingLocation", c.FoundingLocation)
	org.set("sameAs", c.Links)

	if c.LogoURL != "" {
		org.set("logo", schema{
			"@type": "ImageObject",
			"url":   absoluteURL(c.URL, c.LogoURL),
		})
	}
	if c.Email != "" {
		org.set("contactPoint", schema{
			"@type":       "ContactPoint",
			"contactType": "customer support",
			"email":       c.Email,
		})
	}
	return org
}

// website is the site itself, published once per page so a crawler that lands
// anywhere learns what the site is.
func (c SiteChrome) website() schema {
	site := schema{
		"@type":     "WebSite",
		"@id":       c.URL + "#website",
		"name":      c.Name,
		"publisher": schema{"@id": c.URL + "#organization"},
	}
	site.set("url", c.URL)
	site.set("description", c.Description)
	return site
}

// breadcrumbs builds the trail a search result shows in place of a bare URL.
// Pairs are label then path, in order, starting from the home page.
func (c SiteChrome) breadcrumbs(pairs ...string) schema {
	if len(pairs) < 4 || len(pairs)%2 != 0 {
		// A trail of one is the page itself, which tells a reader nothing.
		return nil
	}

	items := make([]any, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		item := schema{
			"@type":    "ListItem",
			"position": i/2 + 1,
			"name":     pairs[i],
		}
		item.set("item", canonicalURL(c.URL, pairs[i+1]))
		items = append(items, item)
	}
	return schema{"@type": "BreadcrumbList", "itemListElement": items}
}

// graph packs the page's nodes into one block.
//
// One script per page holding a @graph, rather than a script per node: the
// nodes reference each other by @id, and a single graph is what lets them.
func graph(nodes ...schema) jsonLD {
	kept := make([]schema, 0, len(nodes))
	for _, node := range nodes {
		if len(node) > 0 {
			kept = append(kept, node)
		}
	}
	if len(kept) == 0 {
		return ""
	}

	encoded, err := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   kept,
	})
	if err != nil {
		// Structured data is an enhancement; a page still serves without it.
		return ""
	}
	// The output is machine-written JSON, never author text, and it is served
	// inside a script whose type makes it a data block rather than code. The
	// one sequence that could still close the element early is escaped.
	return jsonLD(strings.ReplaceAll(string(encoded), "</", `<\/`))
}
