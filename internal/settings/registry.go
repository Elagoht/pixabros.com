// Package settings stores the two key/value tables behind the site: global
// SEO defaults and the homepage's own copy.
//
// The set of keys lives here in code rather than being whatever the admin
// typed. The public site's templates look a key up by its exact name, so a
// free-form editor would let a single mistyped key blank out part of a page
// with no error anywhere. Adding or renaming a setting is a line in this file.
package settings

import (
	"errors"
	"fmt"
)

// Kind decides how a value is validated and which control the admin gets. It
// is stored alongside the value as value_type so the orphan-media sweep can
// find image references without knowing this registry.
type Kind string

const (
	KindText Kind = "text"
	KindURI  Kind = "uri"
	// KindURIList is a JSON array of absolute URLs, stored as text. JSON-LD's
	// sameAs is exactly this: a bare list of profile addresses with no labels.
	// The kind exists so the value is really validated and the admin edits a
	// list of fields rather than hand-writing JSON.
	KindURIList Kind = "uri_list"
	KindMedia   Kind = "media"
	// KindLink is somewhere to send a visitor: a path on this site, or a full
	// address elsewhere. It exists because most links an admin writes point at
	// a page of their own, and demanding the whole address for those means
	// typing the domain in twice and breaking the link the day it changes.
	KindLink Kind = "link"
)

var ErrUnknownGroup = errors.New("unknown settings group")

// ErrUnknownKey is returned for a key this registry does not define, rather
// than silently storing it where nothing will ever read it.
var ErrUnknownKey = errors.New("unknown settings key")

type Definition struct {
	Key  string `json:"key"`
	Kind Kind   `json:"kind"`
	// Multiline asks the admin UI for a textarea instead of a single line.
	Multiline bool `json:"multiline"`
	// Target names the imaging target a KindMedia value is uploaded through,
	// which decides the stored dimensions. It lives here so the admin UI does
	// not need a second key-to-target table of its own to keep in step.
	Target string `json:"target,omitempty"`
}

type Group struct {
	// Name is the segment the API is addressed by: /api/admin/settings/{name}.
	Name string
	// Table is interpolated into queries, so it is a constant defined here and
	// never a value that reached the process from outside.
	Table       string
	RegenTag    string
	Definitions []Definition
}

// site holds global metadata: values that describe the studio rather than any
// one page. Keys follow the data-model spec's Site Settings section.
var site = Group{
	Name:     "site",
	Table:    "site_settings",
	RegenTag: "site_settings",
	Definitions: []Definition{
		{Key: "site_name", Kind: KindText},
		// The address the site is served from, with no trailing slash. Every
		// canonical link, every absolute image URL in a share card and every
		// @id in the structured data is built from it, so a blank one costs
		// the site all three.
		{Key: "site_url", Kind: KindURI},
		// The words appended to a page title that would otherwise be too short
		// for a search result to be worth reading.
		{Key: "site_tagline", Kind: KindText},
		{Key: "twitter_handle", Kind: KindText},
		{Key: "org_logo", Kind: KindMedia, Target: "org_logo"},
		{Key: "default_og_image", Kind: KindMedia, Target: "og_image"},
		{Key: "org_sameas_json", Kind: KindURIList},
		// Organization fields, all of them optional. Each one that is filled in
		// becomes a property of the studio's structured data; each one left
		// blank is simply left out rather than published empty.
		{Key: "org_legal_name", Kind: KindText},
		{Key: "org_description", Kind: KindText, Multiline: true},
		{Key: "org_email", Kind: KindText},
		{Key: "org_founding_date", Kind: KindText},
		{Key: "org_founding_location", Kind: KindText},
	},
}

// homepage holds copy that only the landing page uses. Keys follow the
// data-model spec's Homepage section.
var homepage = Group{
	Name:     "homepage",
	Table:    "homepage_settings",
	RegenTag: "homepage",
	Definitions: []Definition{
		{Key: "hero_logo", Kind: KindMedia, Target: "org_logo"},
		{Key: "hero_slogan", Kind: KindText},
		{Key: "hero_description", Kind: KindText, Multiline: true},
		{Key: "hero_cta_text", Kind: KindText},
		{Key: "hero_cta_link", Kind: KindLink},
		{Key: "portfolio_section_title", Kind: KindText},
		{Key: "sales_section_title", Kind: KindText},
		{Key: "members_section_title", Kind: KindText},
		{Key: "members_section_subtitle", Kind: KindText},
	},
}

var groups = map[string]Group{
	site.Name:     site,
	homepage.Name: homepage,
}

// GroupNames lists the addressable groups, for error messages.
func GroupNames() []string {
	return []string{site.Name, homepage.Name}
}

// LookupGroup resolves the {group} path segment.
func LookupGroup(name string) (Group, error) {
	group, ok := groups[name]
	if !ok {
		return Group{}, fmt.Errorf("%w: %q", ErrUnknownGroup, name)
	}
	return group, nil
}

// Define returns the definition for a key within this group.
func (g Group) Define(key string) (Definition, error) {
	for _, definition := range g.Definitions {
		if definition.Key == key {
			return definition, nil
		}
	}
	return Definition{}, fmt.Errorf("%w: %q", ErrUnknownKey, key)
}
