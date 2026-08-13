package site

import (
	"net/url"
	"strings"
)

// Which sites the studio's links point at, and what each one is called.
//
// A link the site can name gets that site's own mark; everything else gets a
// globe, which is honest rather than a guess at an unknown site's branding.
// The set is deliberately small: it covers the stores the games are sold on and
// the places the studio actually posts.
const (
	brandItch      = "itch"
	brandSteam     = "steam"
	brandFiuby     = "fiuby"
	brandX         = "x"
	brandInstagram = "instagram"
	brandYouTube   = "youtube"
	brandGitHub    = "github"
	brandOther     = "other"
)

// brandNames are what a link is called once it is recognised. A footer shows
// only the mark, so this is what a screen reader reads out.
var brandNames = map[string]string{
	brandItch:      "itch.io",
	brandSteam:     "Steam",
	brandFiuby:     "Fiuby",
	brandX:         "X",
	brandInstagram: "Instagram",
	brandYouTube:   "YouTube",
	brandGitHub:    "GitHub",
}

// brandHosts maps a registrable domain to the brand it belongs to. Matching on
// the domain rather than the whole host is what makes a creator's own subdomain
// work: an itch.io page lives at elagoht.itch.io, not at itch.io.
var brandHosts = map[string]string{
	"itch.io":            brandItch,
	"steampowered.com":   brandSteam,
	"steamcommunity.com": brandSteam,
	"fiuby.com":          brandFiuby,
	"x.com":              brandX,
	"twitter.com":        brandX,
	"instagram.com":      brandInstagram,
	"youtube.com":        brandYouTube,
	"youtu.be":           brandYouTube,
	"github.com":         brandGitHub,
	"github.io":          brandGitHub,
}

// brandFor reads the site off a link's host.
func brandFor(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return brandOther
	}
	host := strings.ToLower(parsed.Hostname())

	for domain, brand := range brandHosts {
		// The host is the domain itself or something beneath it. Checking the
		// dot as well is what stops "notitch.io" and "itch.io.evil.test" from
		// wearing a brand they have nothing to do with.
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return brand
		}
	}
	return brandOther
}

// brandLink builds the view for one of the studio's own links.
//
// A recognised site is named after itself, which is both shorter and clearer
// than the address: "GitHub" rather than "github.com/Elagoht". Anything else
// keeps the address as its label, because that is all there is to go on.
func brandLink(rawURL string) brandedLink {
	brand := brandFor(rawURL)
	label, named := brandNames[brand]
	if !named {
		label = linkLabel(rawURL)
	}
	return brandedLink{Label: label, URL: rawURL, Brand: brand}
}
