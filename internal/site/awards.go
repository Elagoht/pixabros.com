package site

import (
	"encoding/json"
	"fmt"
)

// awardView is one row of the timeline, already resolved: the renderer looks
// up the badge image so the template never has to reach for a repository.
type awardView struct {
	Title  string
	Issuer string
	Date   string
	// Year is pulled out of the date so it can head the card: it is the thing
	// people scan a list of awards for.
	Year       string
	Link       string
	PictureURL string
	PictureAlt string
}

type awardsPage struct {
	Awards []awardView
}

// renderAwards builds the awards timeline.
//
// Tags: award:list covers every award change (there is no per-award tag), and
// site_settings covers the header and footer. Both strings must match what
// the admin API enqueues.
func (s *Site) renderAwards(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}

	// Newest first, per the data-model spec's ordering for awards.
	list, err := s.awards.List("date", true)
	if err != nil {
		return nil, nil, fmt.Errorf("list awards: %w", err)
	}

	views := make([]awardView, 0, len(list))
	for _, award := range list {
		view := awardView{
			Title:  award.Title,
			Issuer: award.Issuer,
			Date:   award.Date,
			Year:   yearOf(award.Date),
			Link:   award.Link,
		}

		if award.PictureID != nil {
			// A missing image must not take the page down: the badge simply
			// falls back to its placeholder.
			if image, err := s.media.FindByID(*award.PictureID); err == nil {
				view.PictureURL = mediaURL(image.Path)
				view.PictureAlt = image.AltText
				if view.PictureAlt == "" {
					view.PictureAlt = award.Title
				}
			}
		}

		// A game link is deliberately not emitted: game detail pages do not
		// exist yet, and a link to a 404 is worse than no link.
		views = append(views, view)
	}

	html, err := s.renderer.render("awards.html", pageData{
		Title:       "Awards and festival recognition for our games",
		Description: "Awards and recognition for games made by " + chrome.Name + ".",
		Path:        "/" + PageAwards,
		Canonical:   canonicalURL(chrome.URL, PageAwards),
		Schema:      awardsSchema(chrome, views),
		Scripts:     []string{s.renderer.bundle.URL("lightbox.js")},
		Site:        chrome,
		Data:        awardsPage{Awards: views},
	})
	if err != nil {
		return nil, nil, err
	}

	return html, []string{awardsListTag, siteSettingsTag}, nil
}

// awardsListTag must match internal/awardsapi's regenTag exactly, or an edit
// in the admin panel would never reach this page.
const awardsListTag = "award:list"

// parseLinks reads a uri_list setting, which is stored as a JSON array of
// absolute URLs. A malformed value yields no links rather than an error: bad
// data in one setting must not stop every page on the site from rendering.
func parseLinks(raw string) []string {
	if raw == "" {
		return nil
	}
	var links []string
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return nil
	}
	return links
}

// yearOf takes the year from a stored YYYY-MM-DD date. Anything else yields
// nothing rather than a misleading fragment.
func yearOf(date string) string {
	if len(date) < 4 {
		return ""
	}
	for _, r := range date[:4] {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return date[:4]
}
