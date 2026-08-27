package site

import (
	"database/sql"
	"strings"
	"testing"

	"pixabros/internal/id"
)

// setGameMeta fills in the three fields on an already-seeded game, so the
// tests below do not need a second seeder that repeats every other column.
func setGameMeta(t *testing.T, conn *sql.DB, gameID, genre, releaseDate, kind string) {
	t.Helper()
	if _, err := conn.Exec(
		`UPDATE games SET genre = ?, release_date = ?, kind = ? WHERE id = ?;`,
		genre, releaseDate, kind, gameID,
	); err != nil {
		t.Fatalf("set game meta: %v", err)
	}
}

func TestReleaseYear(t *testing.T) {
	cases := map[string]string{
		"2026-07-31": "2026",
		"1999-01-01": "1999",
		// Anything that is not a stored date yields nothing rather than a
		// guess: the card simply leaves the year out.
		"":           "",
		"2026":       "",
		"31-07-2026": "",
		"sometime":   "",
	}
	for input, want := range cases {
		if got := releaseYear(input); got != want {
			t.Errorf("releaseYear(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderGame_ShowsTheReleaseDateAndGenre(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "Tactics")
	setGameMeta(t, conn, gameID, "Turn-based tactics", "2026-07-31", "production")

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "dungrid-tactics")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{"Released", "31 July 2026", "Genre", "Turn-based tactics"} {
		if !strings.Contains(body, want) {
			t.Errorf("the detail page is missing %q", want)
		}
	}
	// A production game carries no badge; that is what makes the jam one mean
	// something.
	if strings.Contains(body, "Game jam") {
		t.Error("a production game was badged as a jam entry")
	}
}

func TestRenderGame_BadgesAJamEntry(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Jam Thing", "jam-thing", true, false, "")
	setGameMeta(t, conn, gameID, "", "", "gamejam")

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "jam-thing")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	if !strings.Contains(string(html), "Game jam entry") {
		t.Error("the jam entry is not marked as one")
	}
}

// A game with none of the three filled in must render exactly as it did
// before the fields existed: no empty rows, no stray separators.
func TestRenderGame_LeavesOutWhatIsNotFilledIn(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedGame(t, conn, "Bare", "bare", true, false, "")

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "bare")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	for _, unwanted := range []string{"Released", "Genre", "game-facts", "jam-badge"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("a game with nothing filled in still rendered %q", unwanted)
		}
	}
}

// The three fields follow a game onto every surface it appears on, so a
// reader meets the same facts wherever they find it.
func TestPages_CarryTheGameMetaEverywhere(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Jam Thing", "jam-thing", true, true, "Tactics")
	setGameMeta(t, conn, gameID, "Turn-based tactics", "2026-07-31", "gamejam")

	site := newTestSite(t, conn)
	for name, render := range map[string]func(string) ([]byte, []string, error){
		"the landing page": site.renderLanding,
		"the games page":   site.renderArcade,
	} {
		html, _, err := render(map[string]string{
			"the landing page": "index.html",
			"the games page":   PageGames,
		}[name])
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		body := string(html)

		if !strings.Contains(body, "Turn-based tactics") {
			t.Errorf("%s does not show the genre", name)
		}
		if !strings.Contains(body, "jam-badge") {
			t.Errorf("%s does not mark the jam entry", name)
		}
	}
}

// The compact surfaces show the year, not the full date: a card has room for
// one and not the other.
func TestRenderArcade_ShowsTheYearOnACartridge(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Playable", "playable", true, false, "")
	setGameMeta(t, conn, gameID, "Puzzle", "2026-07-31", "production")
	if _, err := conn.Exec(
		`UPDATE games SET is_browser_playable = 1, web_export_path = 'play/playable' WHERE id = ?;`,
		gameID,
	); err != nil {
		t.Fatalf("mark playable: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderArcade(PageGames)
	if err != nil {
		t.Fatalf("renderArcade() error = %v", err)
	}
	// Scoped to the caption: the same game also appears as a case further down
	// the page, and the open case does have room for the long date.
	body := string(html)
	// The minifier drops attribute quotes where it can, so the class is
	// matched on its own rather than as a whole attribute.
	start := strings.Index(body, "cartridge__caption")
	if start < 0 {
		t.Fatal("no cartridge caption on the page")
	}
	// The minifier also drops optional end tags, so there is no </p> to stop
	// at; the cartridge list's own </ul> is the next thing that is definitely
	// still there.
	caption := body[start:]
	if end := strings.Index(caption, "</ul>"); end >= 0 {
		caption = caption[:end]
	}

	if !strings.Contains(caption, "2026") {
		t.Errorf("the caption does not carry the release year: %s", caption)
	}
	if strings.Contains(caption, "31 July 2026") {
		t.Errorf("the caption used the long date, which does not fit a card: %s", caption)
	}
	if !strings.Contains(caption, "Puzzle") {
		t.Errorf("the caption does not carry the genre: %s", caption)
	}
}

func TestBrandFor(t *testing.T) {
	cases := map[string]string{
		// A creator's page lives on their own subdomain, which is the form
		// these links actually take.
		"https://elagoht.itch.io/dungrid-tactics": brandItch,
		"https://itch.io/games":                   brandItch,
		"https://store.steampowered.com/app/1/x/": brandSteam,
		"https://steamcommunity.com/app/1":        brandSteam,
		"https://fiuby.com/games/x":               brandFiuby,
		"https://www.fiuby.com/games/x":           brandFiuby,
		"https://example.com/anything":            brandOther,
		// A host that merely ends in the brand's name is not the brand.
		"https://itch.io.evil.test/x":     brandOther,
		"https://notitch.io/x":            brandOther,
		"https://fake-steampowered.com/x": brandOther,
		"":                                brandOther,
	}
	for rawURL, want := range cases {
		if got := brandFor(rawURL); got != want {
			t.Errorf("brandFor(%q) = %q, want %q", rawURL, got, want)
		}
	}
}

// The description goes through the same markdown pipeline a devlog post does,
// which also means raw HTML in it is still dropped.
func TestRenderGame_RendersTheDescriptionAsMarkdown(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Marked Up", "marked-up", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET full_description = ? WHERE id = ?;`,
		"## Features\n\n- One\n- Two\n\n**bold**\n\n<script>alert(1)</script>\n", gameID,
	); err != nil {
		t.Fatalf("set description: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "marked-up")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{"<h2", "<li>One", "<strong>bold</strong>"} {
		if !strings.Contains(body, want) {
			t.Errorf("the description was not rendered as markdown: missing %q", want)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Error("raw HTML from a description reached the page")
	}
}

func TestRenderGame_EmbedsTheTrailer(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Trailered", "trailered", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET video_url = ? WHERE id = ?;`,
		"https://youtu.be/9mjjowHX1-g?si=abc", gameID,
	); err != nil {
		t.Fatalf("set video: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "trailered")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	if !strings.Contains(body, "youtube-nocookie.com/embed/9mjjowHX1-g") {
		t.Error("the trailer was not embedded")
	}
	// Only the id crosses over, so the tracking parameter goes with the rest
	// of the URL.
	if strings.Contains(body, "si=abc") {
		t.Error("the stored URL's query string reached the page")
	}
}

// A link stored before the API validated it might not be a YouTube URL at all.
// It yields no player rather than being framed as-is.
func TestRenderGame_IgnoresATrailerThatIsNotYouTube(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Odd", "odd", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET video_url = 'https://evil.test/x' WHERE id = ?;`, gameID,
	); err != nil {
		t.Fatalf("set video: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "odd")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	if strings.Contains(string(html), "evil.test") {
		t.Error("a stored non-YouTube link was framed")
	}
}

// The tab shows the game's own cover, so a row of open game pages is told
// apart by artwork rather than by a truncated title.
func TestRenderGame_UsesTheCoverAsTheFavicon(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Covered", "covered", true, false, "")
	coverID := seedMedia(t, conn, "media/cd/2026-cover.webp", "Covered cover")
	if _, err := conn.Exec(
		`UPDATE games SET cd_cover_art_id = ? WHERE id = ?;`, coverID, gameID,
	); err != nil {
		t.Fatalf("attach cover: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "covered")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	if !strings.Contains(string(html), `rel=icon href=/media/cd/2026-cover.webp`) {
		t.Errorf("the cover is not the page's icon: %s", html)
	}
}

// Every other page falls back to the studio's mark. Leaving it off is what the
// site used to do, and it cost every page but the games their tab icon.
func TestRenderAwards_FallsBackToTheStudioMark(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})

	site := newTestSite(t, conn)
	html, _, err := site.renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}
	if want := "rel=icon href=" + site.renderer.bundle.URL("logo.svg"); !strings.Contains(string(html), want) {
		t.Errorf("the awards page does not wear the studio mark (%s): %s", want, html)
	}
}

// A game with neither a CD cover nor cartridge art would otherwise render an
// empty href, which asks the browser to fetch the page as its own icon.
func TestRenderGame_WithoutArtworkFallsBackToTheStudioMark(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedGame(t, conn, "Bare", "bare", true, false, "")

	site := newTestSite(t, conn)
	html, _, err := site.renderGame(GamePagePrefix + "bare")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	if want := "rel=icon href=" + site.renderer.bundle.URL("logo.svg"); !strings.Contains(string(html), want) {
		t.Errorf("a game with no artwork does not wear the studio mark (%s): %s", want, html)
	}
}

// The manifest is what makes the site installable, so it has to be linked from
// every page rather than from the landing page alone.
func TestRender_LinksTheManifestFromEveryPage(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedGame(t, conn, "Bare", "bare", true, false, "")
	site := newTestSite(t, conn)

	pages := map[string]func() ([]byte, error){
		"landing": func() ([]byte, error) { html, _, err := site.renderLanding(PageLanding); return html, err },
		"awards":  func() ([]byte, error) { html, _, err := site.renderAwards(PageAwards); return html, err },
		"game":    func() ([]byte, error) { html, _, err := site.renderGame(GamePagePrefix + "bare"); return html, err },
	}
	for name, render := range pages {
		html, err := render()
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		if !strings.Contains(string(html), `rel=manifest href=`+ManifestPath) {
			t.Errorf("the %s page does not link the manifest: %s", name, html)
		}
		// iOS ignores the manifest's icons and reads this one instead, so
		// without it an installed site gets a screenshot for a home screen icon.
		if want := "rel=apple-touch-icon href=" + site.renderer.bundle.URL("icon-192.png"); !strings.Contains(string(html), want) {
			t.Errorf("the %s page has no apple-touch-icon (%s)", name, want)
		}
		// The .ico is the fallback for browsers that cannot read the vector
		// mark, so like the manifest it belongs on every page.
		if want := "rel=icon type=image/x-icon href=" + site.renderer.bundle.URL("favicon.ico"); !strings.Contains(string(html), want) {
			t.Errorf("the %s page has no .ico icon fallback (%s)", name, want)
		}
	}
}

// The stores sit at the foot of the page with a mark each.
func TestRenderGame_MarksTheStoreLinks(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Linked", "linked", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET external_links_json = ? WHERE id = ?;`,
		`[{"label":"itch.io","url":"https://elagoht.itch.io/x"},`+
			`{"label":"Steam","url":"https://store.steampowered.com/app/1/x/"},`+
			`{"label":"Our site","url":"https://example.com/x"}]`, gameID,
	); err != nil {
		t.Fatalf("set links: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "linked")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	stores := body[strings.Index(body, "store-links__list"):]
	if got := strings.Count(stores, "<svg"); got != 3 {
		t.Errorf("wanted a mark on each of the three links, got %d", got)
	}
	for _, want := range []string{"itch.io", "Steam", "Our site"} {
		if !strings.Contains(body, want) {
			t.Errorf("the store list is missing %q", want)
		}
	}
}

func TestRenderGame_KeepsStoreLinksOutsideProse(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Linked", "linked", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET full_description = 'Description.',
		 external_links_json = '[{"label":"itch.io","url":"https://itch.io/x"}]'
		 WHERE id = ?;`, gameID,
	); err != nil {
		t.Fatalf("set game content: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "linked")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)
	proseEnd := strings.Index(body, `</div><div class=store-links>`)
	if proseEnd == -1 {
		t.Error("the store links are not a sibling immediately after the prose content")
	}
}

// A screenshot in the narrow side column is too small to read, so it opens
// full size the way an award's badge does.
func TestRenderGame_OpensAScreenshotFullSize(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Shot", "shot", true, false, "")
	shotID := seedMedia(t, conn, "media/screenshot/2026-one.webp", "A screenshot")
	if _, err := conn.Exec(
		`INSERT INTO game_screenshots (id, game_id, media_id, display_order)
		 VALUES (?, ?, ?, 0);`, id.New(), gameID, shotID,
	); err != nil {
		t.Fatalf("seed screenshot: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "shot")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{
		"data-zoom-src=/media/screenshot/2026-one.webp",
		"data-zoom-dialog",
		"lightbox.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the screenshot does not open full size: missing %q", want)
		}
	}
	// Without scripting the picture is still on the page.
	if !strings.Contains(body, "src=/media/screenshot/2026-one.webp") {
		t.Error("the screenshot itself is not on the page")
	}
}

// The page leads with the OG image, which is drawn at a wide banner's shape.
// The tab icon stays the portrait cover.
func TestRenderGame_LeadsWithTheOGImageAndKeepsTheCoverAsIcon(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Both", "both", true, false, "")
	ogID := seedMedia(t, conn, "media/og/2026-wide.webp", "Wide art")
	coverID := seedMedia(t, conn, "media/cd/2026-tall.webp", "Tall art")
	if _, err := conn.Exec(
		`UPDATE games SET og_image_id = ?, cd_cover_art_id = ? WHERE id = ?;`,
		ogID, coverID, gameID,
	); err != nil {
		t.Fatalf("attach art: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "both")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	if !strings.Contains(body, `class=game-cover><img src=/media/og/2026-wide.webp`) {
		t.Errorf("the page does not lead with the OG image: %s", body)
	}
	if !strings.Contains(body, "rel=icon href=/media/cd/2026-tall.webp") {
		t.Error("the tab icon is not the cover")
	}
}

// The trailer sits at the head of the description rather than in a block of
// its own, so the words read as being about the thing above them.
func TestRenderGame_PutsTheTrailerInsideTheDescription(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	gameID := seedGame(t, conn, "Both", "both", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET video_url = ?, full_description = 'Some prose.' WHERE id = ?;`,
		"https://youtu.be/9mjjowHX1-g", gameID,
	); err != nil {
		t.Fatalf("set video: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "both")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)

	prose := body[strings.Index(body, "class=prose"):]
	player := strings.Index(prose, "embed--video")
	words := strings.Index(prose, "Some prose.")
	if player < 0 || words < 0 {
		t.Fatalf("player at %d, words at %d, wanted both inside the description", player, words)
	}
	if player > words {
		t.Error("the trailer comes after the description rather than heading it")
	}
}

func TestBrandFor_RecognisesTheStudiosOwnPlaces(t *testing.T) {
	cases := map[string]string{
		"https://x.com/PixaBrosStudio":            brandX,
		"https://twitter.com/PixaBrosStudio":      brandX,
		"https://www.instagram.com/pixabros/":     brandInstagram,
		"https://www.youtube.com/@pixabros":       brandYouTube,
		"https://youtu.be/9mjjowHX1-g":            brandYouTube,
		"https://github.com/Elagoht/":             brandGitHub,
		"https://elagoht.github.io/thing":         brandGitHub,
		"https://elagoht.itch.io/dungrid-tactics": brandItch,
		"https://store.steampowered.com/app/1/":   brandSteam,
		"https://fiuby.com/x":                     brandFiuby,
		// A host that merely ends in a brand's name is not that brand.
		"https://x.com.evil.test/a": brandOther,
		"https://notgithub.com/a":   brandOther,
		"https://example.com/a":     brandOther,
	}
	for rawURL, want := range cases {
		if got := brandFor(rawURL); got != want {
			t.Errorf("brandFor(%q) = %q, want %q", rawURL, got, want)
		}
	}
}

// A recognised site is named after itself, which is shorter and clearer than
// its address. Anything else keeps the address, since that is all there is.
func TestBrandLink_NamesWhatItRecognises(t *testing.T) {
	if got := brandLink("https://github.com/Elagoht/"); got.Label != "GitHub" {
		t.Errorf("label = %q, want GitHub", got.Label)
	}
	if got := brandLink("https://example.com/blog"); got.Label != "example.com/blog" {
		t.Errorf("label = %q, want the address itself", got.Label)
	}
}

// The footer shows marks, not a wall of addresses. An unrecognised site keeps
// its name beside the globe, because a globe on its own means nothing.
func TestFooter_ShowsTheStudiosLinksAsMarks(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_name": "Pixabros",
		"org_sameas_json": `["https://x.com/PixaBrosStudio",` +
			`"https://github.com/Elagoht/","https://example.com/blog"]`,
	})

	html, _, err := newTestSite(t, conn).renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}
	body := string(html)

	footer := body[strings.Index(body, "site-social"):]
	if got := strings.Count(footer, "<svg"); got != 3 {
		t.Errorf("wanted a mark on each of the three links, got %d", got)
	}
	// A mark with no words needs a name a screen reader can read out.
	for _, want := range []string{`aria-label=X`, `aria-label=GitHub`} {
		if !strings.Contains(footer, want) {
			t.Errorf("the footer is missing %q", want)
		}
	}
	// The unrecognised one keeps its name on screen instead.
	if !strings.Contains(footer, "example.com/blog") {
		t.Error("an unrecognised link lost the only thing identifying it")
	}
	// And the addresses are still there to click.
	if !strings.Contains(footer, "https://x.com/PixaBrosStudio") {
		t.Error("the footer lost a link's address")
	}
}
