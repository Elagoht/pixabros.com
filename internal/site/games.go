package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"pixabros/internal/games"
	"pixabros/internal/youtube"
)

// Page keys for the games section. PageGames is the arcade; each published
// game also gets a detail page beneath it.
const (
	PageGames      = "games"
	GamePagePrefix = "games/"
)

// gameMeta is the shelf information a game carries wherever it appears: what
// kind of thing it is, what genre, and when it came out. It is embedded rather
// than repeated so a cartridge, a case, a carousel slide and the detail page
// all say the same things in the same words.
//
// Year is the compact form a card has room for; Released is the long one the
// detail page uses. Both are empty when the game has no date, and a template
// that guards on one is guarding on the other.
type gameMeta struct {
	Genre     string
	Released  string
	Year      string
	IsGameJam bool
}

func metaFor(game games.Game) gameMeta {
	return gameMeta{
		Genre:     strings.TrimSpace(game.Genre),
		Released:  formatDate(game.ReleaseDate),
		Year:      releaseYear(game.ReleaseDate),
		IsGameJam: game.Kind == games.KindGameJam,
	}
}

// releaseYear takes the year off a stored date. The API pins the shape, so
// anything else here is a row written before that check existed and is left
// out rather than guessed at.
func releaseYear(date string) string {
	if !isoDate.MatchString(date) {
		return ""
	}
	return date[:4]
}

type cartridgeView struct {
	gameMeta
	Title string
	Slug  string
	Art   imageView
	Tags  []string
	// PlayURL is the extracted build, served straight from disk. The arcade
	// loads it into the console in place rather than navigating away.
	PlayURL string
}

type caseView struct {
	gameMeta
	Title       string
	Slug        string
	Art         imageView
	Tags        []string
	Description string
	Price       string
	// Links are the stores and pages the game lives on, shown on the open
	// case's first page.
	Links []gameLink
}

// consoleView is what the shared console partial needs. Interactive is the
// arcade's version: an empty bay you load a cartridge into. A game's own page
// gets the machine already running.
type consoleView struct {
	Interactive bool
	SiteName    string
	Title       string
	Playable    bool
	PlayURL     string
	Cartridge   imageView
}

type arcadePage struct {
	Console    consoleView
	Cartridges []cartridgeView
	Cases      []caseView
}

// renderArcade builds /games: the skeuomorphic shelf.
//
// The visual spec splits this into "browser playable" cartridges and
// "downloadable" CD cases. Direct downloads were dropped as a product
// decision, so the CD shelf shows the rest of the catalogue instead -- games
// you cannot play in the browser but can still read about. Without that the
// shelf would always be empty and twelve of fourteen games would be invisible.
func (s *Site) renderArcade(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	images, err := s.mediaByID()
	if err != nil {
		return nil, nil, err
	}
	published, err := s.publishedGames()
	if err != nil {
		return nil, nil, err
	}

	page := arcadePage{
		Console: consoleView{Interactive: true, SiteName: chrome.Name, Playable: true},
	}
	for _, game := range published {
		if game.IsBrowserPlayable {
			page.Cartridges = append(page.Cartridges, cartridgeView{
				gameMeta: metaFor(game),
				Title:    game.Title,
				Slug:     game.Slug,
				Art:      lookupImage(images, firstNonNil(game.CartridgeArtID, game.CDCoverArtID), game.Title),
				Tags:     splitTags(game.Tags),
				PlayURL:  "/play/" + game.Slug + "/",
			})
		}
		// The shelf holds the whole catalogue, playable games included: the
		// cartridge is how you start one, the case is how you read about it.
		page.Cases = append(page.Cases, caseView{
			gameMeta:    metaFor(game),
			Title:       game.Title,
			Slug:        game.Slug,
			Art:         lookupImage(images, firstNonNil(game.CDCoverArtID, game.CartridgeArtID), game.Title),
			Tags:        splitTags(game.Tags),
			Description: game.ShortDescription,
			Price:       priceFor(game),
			Links:       parseGameLinks(game.ExternalLinksJSON),
		})
	}

	html, err := s.renderer.render("arcade.html", pageData{
		Title:       "Games · " + chrome.Name,
		Description: "Play our browser games and browse the rest of the catalogue.",
		Path:        "/" + PageGames,
		Scripts: []string{
			s.renderer.bundle.URL("arcade.js"),
			s.renderer.bundle.URL("cases.js"),
		},
		Site: chrome,
		Data: page,
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{gameListTag, siteSettingsTag}, nil
}

type gameLink struct {
	Label string
	URL   string
	// Brand picks the mark shown beside the label. The template knows how to
	// draw each of these and nothing else.
	Brand string
}

// The stores a game actually lives on get their own mark; everything else gets
// a globe, which is honest rather than a guess at an unknown site's branding.
const (
	brandItch  = "itch"
	brandSteam = "steam"
	brandFiuby = "fiuby"
	brandOther = "other"
)

// brandFor reads the store off a link's host.
//
// Matching on the registrable domain rather than the whole host is what makes
// a creator's own subdomain work: an itch.io page lives at
// elagoht.itch.io, not at itch.io.
func brandFor(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return brandOther
	}
	host := strings.ToLower(parsed.Hostname())

	switch {
	case host == "itch.io" || strings.HasSuffix(host, ".itch.io"):
		return brandItch
	case host == "steampowered.com" || strings.HasSuffix(host, ".steampowered.com"),
		host == "steamcommunity.com" || strings.HasSuffix(host, ".steamcommunity.com"):
		return brandSteam
	case host == "fiuby.com" || strings.HasSuffix(host, ".fiuby.com"):
		return brandFiuby
	default:
		return brandOther
	}
}

type gamePage struct {
	gameMeta
	Console consoleView
	Title   string
	Slug    string
	Tags    []string
	Short   string
	// Full is rendered markdown, the same pipeline a devlog post goes through:
	// raw HTML is dropped, and a YouTube link on its own line becomes a player.
	Full  template.HTML
	Video string
	// Cover leads the page, so it is the OG image: the one drawn at a wide
	// banner's shape. Icon is the portrait artwork, which is what a browser
	// tab wants.
	Cover       imageView
	Icon        imageView
	Shots       []imageView
	Playable    bool
	PlayURL     string
	Price       string
	Links       []gameLink
	HasSideInfo bool
}

// renderGame builds /games/{slug}.
//
// The page key carries the slug, which is what lets one renderer serve every
// game: render.Registry resolves the "games/" prefix to this function and
// passes the full key through.
func (s *Site) renderGame(pageKey string) ([]byte, []string, error) {
	slug := strings.TrimPrefix(pageKey, GamePagePrefix)
	if slug == "" || strings.Contains(slug, "/") {
		return nil, nil, fmt.Errorf("not a game page key: %q", pageKey)
	}

	game, err := s.games.FindBySlug(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("find game %q: %w", slug, err)
	}
	// A draft must never be reachable, even by typing its address. The
	// reconciler will not ask for the page, but a stale job could.
	if !game.IsPublished {
		return nil, nil, fmt.Errorf("game %q is not published", slug)
	}

	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	images, err := s.mediaByID()
	if err != nil {
		return nil, nil, err
	}

	shots, err := s.games.ListScreenshots(game.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list screenshots: %w", err)
	}
	shotViews := make([]imageView, 0, len(shots))
	for _, shot := range shots {
		if image, ok := images[shot.MediaID]; ok {
			shotViews = append(shotViews, imageView{
				URL: mediaURL(image.Path),
				Alt: altOr(image, game.Title+" screenshot"),
			})
		}
	}

	full, err := renderMarkdown(game.FullDescription)
	if err != nil {
		return nil, nil, fmt.Errorf("render description: %w", err)
	}

	page := gamePage{
		gameMeta: metaFor(game),
		Console: consoleView{
			SiteName:  chrome.Name,
			Title:     game.Title,
			Playable:  game.IsBrowserPlayable,
			PlayURL:   "/play/" + game.Slug + "/",
			Cartridge: lookupImage(images, firstNonNil(game.CartridgeArtID, game.CDCoverArtID), game.Title),
		},
		Title:    game.Title,
		Slug:     game.Slug,
		Tags:     splitTags(game.Tags),
		Short:    game.ShortDescription,
		Full:     full,
		Video:    videoEmbedURL(game.VideoURL),
		Cover:    lookupImage(images, game.OGImageID, game.Title),
		Icon:     lookupImage(images, firstNonNil(game.CDCoverArtID, game.CartridgeArtID), game.Title),
		Shots:    shotViews,
		Playable: game.IsBrowserPlayable,
		// The build is served straight from disk at /play/{slug}/, which is why
		// that prefix is not available to any public page.
		PlayURL: "/play/" + game.Slug + "/",
		Price:   priceFor(game),
		Links:   parseGameLinks(game.ExternalLinksJSON),
	}
	// Everything the side column can hold. The screenshots belong here too:
	// leaving them out meant a game whose only extra was screenshots rendered
	// no column, and so no screenshots.
	page.HasSideInfo = len(page.Tags) > 0 || page.Price != "" ||
		page.Released != "" || page.Genre != "" || len(page.Shots) > 0

	html, err := s.renderer.render("game.html", pageData{
		Title:       game.Title + " · " + chrome.Name,
		Description: fallback(game.ShortDescription, game.Title+" by "+chrome.Name+"."),
		Path:        "/" + PageGames,
		// The game's own cover in the tab, so a row of open game pages is
		// told apart by artwork rather than by a truncated title.
		Favicon: page.Icon.URL,
		Scripts: []string{
			s.renderer.bundle.URL("arcade.js"),
			s.renderer.bundle.URL("lightbox.js"),
		},
		Site: chrome,
		Data: page,
	})
	if err != nil {
		return nil, nil, err
	}

	return html, []string{"game:" + game.ID, siteSettingsTag}, nil
}

// publishedGames is the public view of the catalogue. The repository returns
// drafts too, because the admin panel needs them.
func (s *Site) publishedGames() ([]games.Game, error) {
	all, err := s.games.List("display_order", false)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	published := make([]games.Game, 0, len(all))
	for _, game := range all {
		if game.IsPublished {
			published = append(published, game)
		}
	}
	return published, nil
}

// priceFor only surfaces a price for a game that is actually on sale. A price
// on something you cannot buy is noise at best.
func priceFor(game games.Game) string {
	if !game.IsForSale {
		return ""
	}
	return game.PriceDisplay
}

// externalLink mirrors the {label,url,icon} shape the admin panel stores.
type externalLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func parseGameLinks(raw string) []gameLink {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var links []externalLink
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return nil
	}
	views := make([]gameLink, 0, len(links))
	for _, link := range links {
		if link.URL == "" {
			continue
		}
		views = append(views, gameLink{
			Label: fallback(link.Label, linkLabel(link.URL)),
			URL:   link.URL,
			Brand: brandFor(link.URL),
		})
	}
	return views
}

// videoEmbedURL turns a stored trailer link into the player's address.
//
// The admin API only accepts a YouTube URL, so a link that fails here is one
// stored before that check existed. It yields nothing rather than being
// framed as-is: the point of reading the id out is that nothing else about
// the URL reaches the page.
func videoEmbedURL(rawURL string) string {
	videoID, ok := youtube.ID(rawURL)
	if !ok {
		return ""
	}
	return youtube.EmbedURL(videoID)
}
