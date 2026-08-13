package site

import (
	"fmt"
	"strings"

	"pixabros/internal/games"
	"pixabros/internal/media"
	"pixabros/internal/members"
	"pixabros/internal/settings"
)

// Landing view models. Everything is resolved here -- image paths, tag lists,
// carousel neighbours -- so the template only lays things out.

type imageView struct {
	URL string
	Alt string
}

type heroView struct {
	Logo        imageView
	Slogan      string
	Description string
	CTAText     string
	CTALink     string
}

// slideView is one game in the portfolio carousel.
type slideView struct {
	gameMeta
	Title string
	Slug  string
	Tags  []string
	Cover imageView
	Shots []imageView
	// ID and its neighbours drive the arrows and dots. Computing them at
	// render time is what lets the carousel work without any JavaScript.
	ID     string
	PrevID string
	NextID string
	Index  int
}

type saleView struct {
	gameMeta
	Title string
	Slug  string
	Tags  []string
	Cover imageView
	Price string
}

type memberView struct {
	Name   string
	Avatar imageView
	Bio    string
	Tags   []string
	Links  []gameLink
}

type landingPage struct {
	Hero            heroView
	PortfolioTitle  string
	Slides          []slideView
	SalesTitle      string
	Sales           []saleView
	MembersTitle    string
	MembersSubtitle string
	Members         []memberView
	HasPortfolio    bool
	HasSales        bool
	HasMembers      bool
}

// renderLanding builds the site root.
//
// Tags: homepage covers the hero and section copy, game:list the portfolio and
// sales grids, member:list the team section, and site_settings the chrome.
// All four must match what the admin API enqueues.
func (s *Site) renderLanding(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}

	homepageGroup, err := settings.LookupGroup("homepage")
	if err != nil {
		return nil, nil, err
	}
	copyValues, err := s.settings.Values(homepageGroup)
	if err != nil {
		return nil, nil, fmt.Errorf("read homepage settings: %w", err)
	}

	images, err := s.mediaByID()
	if err != nil {
		return nil, nil, err
	}

	gameList, err := s.games.List("display_order", false)
	if err != nil {
		return nil, nil, fmt.Errorf("list games: %w", err)
	}
	// The repositories return drafts too, because the admin panel needs them.
	// Filtering here is the only thing keeping unpublished work off the public
	// site.
	published := make([]games.Game, 0, len(gameList))
	for _, game := range gameList {
		if game.IsPublished {
			published = append(published, game)
		}
	}

	slides, err := s.buildSlides(published, images)
	if err != nil {
		return nil, nil, err
	}

	memberList, err := s.members.List("display_order", false)
	if err != nil {
		return nil, nil, fmt.Errorf("list members: %w", err)
	}

	page := landingPage{
		Hero:            s.buildHero(copyValues, images),
		PortfolioTitle:  fallback(copyValues["portfolio_section_title"], "Our games"),
		Slides:          slides,
		SalesTitle:      fallback(copyValues["sales_section_title"], "Available now"),
		Sales:           buildSales(published, images),
		MembersTitle:    fallback(copyValues["members_section_title"], "The team"),
		MembersSubtitle: copyValues["members_section_subtitle"],
		Members:         buildMembers(memberList, images),
	}
	page.HasPortfolio = len(page.Slides) > 0
	page.HasSales = len(page.Sales) > 0
	page.HasMembers = len(page.Members) > 0

	html, err := s.renderer.render("landing.html", pageData{
		Title:       chrome.Name + " · game studio",
		Description: fallback(copyValues["hero_description"], "Games made by "+chrome.Name+"."),
		Path:        "/",
		Scripts:     []string{s.renderer.bundle.URL("carousel.js")},
		Site:        chrome,
		Data:        page,
	})
	if err != nil {
		return nil, nil, err
	}

	return html, []string{homepageTag, gameListTag, memberListTag, siteSettingsTag}, nil
}

// These must match the tags the admin API enqueues exactly.
const (
	homepageTag   = "homepage"
	gameListTag   = "game:list"
	memberListTag = "member:list"
)

// buildHero assembles the site's opening statement.
//
// There is deliberately no featured image beside it. The spec called for one,
// and it was taken from whichever game happened to be first: an arbitrary
// picture, linked to nothing, that said nothing about the studio. The
// portfolio below is where the games introduce themselves.
func (s *Site) buildHero(values map[string]string, images map[string]media.Media) heroView {
	hero := heroView{
		Slogan:      values["hero_slogan"],
		Description: values["hero_description"],
		CTAText:     values["hero_cta_text"],
		CTALink:     values["hero_cta_link"],
		Logo:        lookupImage(images, ptr(values["hero_logo"]), "Studio logo"),
	}

	// A CTA with no destination is a button that does nothing.
	if hero.CTALink == "" {
		hero.CTAText = ""
	}
	return hero
}

func (s *Site) buildSlides(published []games.Game, images map[string]media.Media) ([]slideView, error) {
	slides := make([]slideView, 0, len(published))
	for i, game := range published {
		shots, err := s.games.ListScreenshots(game.ID)
		if err != nil {
			return nil, fmt.Errorf("list screenshots for %q: %w", game.Slug, err)
		}

		// The visual spec calls for a 2x2 thumbnail grid, so four is the most
		// that can be shown.
		views := make([]imageView, 0, 4)
		for _, shot := range shots {
			if len(views) == 4 {
				break
			}
			if image, ok := images[shot.MediaID]; ok {
				views = append(views, imageView{URL: mediaURL(image.Path), Alt: altOr(image, game.Title+" screenshot")})
			}
		}

		slides = append(slides, slideView{
			gameMeta: metaFor(game),
			Title:    game.Title,
			Slug:     game.Slug,
			Tags:     splitTags(game.Tags),
			// The OG image comes first here: it is the wide 1200x630 one, and
			// the carousel's media area is 16:9. Cover art is portrait, so it
			// would be cropped to a strip.
			Cover: lookupImage(images, firstNonNil(game.OGImageID, game.CDCoverArtID, game.CartridgeArtID), game.Title),
			Shots: views,
			ID:    fmt.Sprintf("slide-%d", i+1),
			Index: i + 1,
		})
	}

	// Arrows link to the neighbouring slide by id, which is what makes the
	// carousel navigable with no JavaScript. It wraps, so the last slide's
	// next arrow returns to the first.
	for i := range slides {
		slides[i].PrevID = slides[(i-1+len(slides))%len(slides)].ID
		slides[i].NextID = slides[(i+1)%len(slides)].ID
	}
	return slides, nil
}

func buildSales(published []games.Game, images map[string]media.Media) []saleView {
	sales := make([]saleView, 0)
	for _, game := range published {
		if !game.IsForSale {
			continue
		}
		sales = append(sales, saleView{
			gameMeta: metaFor(game),
			Title:    game.Title,
			Slug:     game.Slug,
			Tags:     splitTags(game.Tags),
			Cover:    lookupImage(images, firstNonNil(game.CDCoverArtID, game.CartridgeArtID), game.Title),
			Price:    game.PriceDisplay,
		})
	}
	return sales
}

func buildMembers(list []members.Member, images map[string]media.Media) []memberView {
	views := make([]memberView, 0, len(list))
	for _, member := range list {
		if !member.IsPublished {
			continue
		}
		views = append(views, memberView{
			Name:   member.Name,
			Avatar: lookupImage(images, member.AvatarID, member.Name),
			Bio:    member.Description,
			Links:  parseGameLinks(member.LinksJSON),
			// There is no separate role column; the tags field is what the
			// admin panel collects roles into.
			Tags: splitTags(member.Tags),
		})
	}
	return views
}

// mediaByID loads every image once, so a page with dozens of thumbnails does
// not issue one query per image.
func (s *Site) mediaByID() (map[string]media.Media, error) {
	list, err := s.media.List()
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	byID := make(map[string]media.Media, len(list))
	for _, item := range list {
		byID[item.ID] = item.Media
	}
	return byID, nil
}

func lookupImage(images map[string]media.Media, id *string, fallbackAlt string) imageView {
	if id == nil || *id == "" {
		return imageView{}
	}
	image, ok := images[*id]
	if !ok {
		return imageView{}
	}
	return imageView{URL: mediaURL(image.Path), Alt: altOr(image, fallbackAlt)}
}

func altOr(image media.Media, fallbackAlt string) string {
	if image.AltText != "" {
		return image.AltText
	}
	return fallbackAlt
}

// splitTags turns the stored comma-separated list into pills, dropping the
// blanks that a trailing comma leaves behind.
func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

func fallback(value, orElse string) string {
	if strings.TrimSpace(value) == "" {
		return orElse
	}
	return value
}

func ptr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func firstNonNil(candidates ...*string) *string {
	for _, candidate := range candidates {
		if candidate != nil && *candidate != "" {
			return candidate
		}
	}
	return nil
}
