package site

import (
	"database/sql"
	"strings"
	"testing"

	"pixabros/internal/id"
)

func renderLandingPage(t *testing.T, s *Site) (string, []string) {
	t.Helper()
	html, tags, err := s.renderLanding(PageLanding)
	if err != nil {
		t.Fatalf("renderLanding() error = %v", err)
	}
	return string(html), tags
}

func seedGame(t *testing.T, conn *sql.DB, title, slug string, published, forSale bool, tags string) string {
	t.Helper()
	gameID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, tags, is_published, is_for_sale, display_order)
		 VALUES (?, ?, ?, ?, ?, ?, 0);`,
		gameID, slug, title, tags, published, forSale,
	); err != nil {
		t.Fatalf("seed game %q: %v", title, err)
	}
	return gameID
}

func seedMember(t *testing.T, conn *sql.DB, name, tags, description string, published bool) string {
	t.Helper()
	memberID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO members (id, name, tags, description, links_json, display_order, is_published)
		 VALUES (?, ?, ?, ?, '[]', 0, ?);`,
		memberID, name, tags, description, published,
	); err != nil {
		t.Fatalf("seed member %q: %v", name, err)
	}
	return memberID
}

func TestRenderLanding_DeclaresEveryTagItDependsOn(t *testing.T) {
	conn := setupTestDB(t)
	_, tags := renderLandingPage(t, newTestSite(t, conn))

	want := map[string]bool{
		"homepage": false, "game:list": false,
		"member:list": false, "site_settings": false,
		"devlog:list": false, "award:list": false,
	}
	for _, tag := range tags {
		if _, ok := want[tag]; !ok {
			t.Errorf("unexpected tag %q", tag)
			continue
		}
		want[tag] = true
	}
	for tag, seen := range want {
		if !seen {
			t.Errorf("the landing page never declared the %q tag", tag)
		}
	}
}

// The repositories return drafts because the admin panel needs them. Filtering
// in the renderer is the only thing keeping unpublished work off the public
// site, so it gets its own test.
func TestRenderLanding_NeverShowsUnpublishedWork(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Shipped Game", "shipped", true, false, "")
	seedGame(t, conn, "Secret Prototype", "secret", false, false, "")
	seedMember(t, conn, "Public Person", "code", "Bio", true)
	seedMember(t, conn, "Hidden Person", "art", "Bio", false)

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "Shipped Game") {
		t.Error("a published game is missing from the landing page")
	}
	if strings.Contains(html, "Secret Prototype") {
		t.Error("an unpublished game leaked onto the public site")
	}
	if !strings.Contains(html, "Public Person") {
		t.Error("a published member is missing")
	}
	if strings.Contains(html, "Hidden Person") {
		t.Error("an unpublished member leaked onto the public site")
	}
}

// An unpublished game must not take a carousel slot either -- a slide with
// nothing in it would still render its arrows and dot.
func TestRenderLanding_DraftsDoNotTakeCarouselSlots(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Only Published", "only", true, false, "")
	seedGame(t, conn, "Draft One", "d1", false, false, "")
	seedGame(t, conn, "Draft Two", "d2", false, false, "")

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if got := strings.Count(html, `class=slide `); got > 1 {
		t.Errorf("slides rendered = %d, want 1", got)
	}
	// With a single slide there is nothing to page between.
	if strings.Contains(html, "carousel__dot") {
		t.Error("dots rendered for a single-slide carousel")
	}
}

func TestRenderLanding_UsesHomepageCopy(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type) VALUES
		 ('hero_slogan', 'We make small strange games', 'text'),
		 ('hero_description', 'A two person studio.', 'text'),
		 ('hero_cta_text', 'Play now', 'text'),
		 ('hero_cta_link', 'https://itch.io/pixabros', 'uri'),
		 ('portfolio_section_title', 'What we have built', 'text');`,
	); err != nil {
		t.Fatalf("seed homepage settings: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{
		"We make small strange games", "A two person studio.",
		"Play now", "https://itch.io/pixabros",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("landing page is missing %q", want)
		}
	}
}

func TestRenderLanding_ShowsTheConfiguredVision(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type) VALUES
		 ('vision_title', 'Our vision', 'text'),
		 ('vision_content', 'We want every game to leave one sharp idea behind.', 'text');`,
	); err != nil {
		t.Fatalf("seed homepage vision: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{"Our vision", "We want every game to leave one sharp idea behind."} {
		if !strings.Contains(html, want) {
			t.Errorf("landing page is missing vision copy %q", want)
		}
	}
}

func TestRenderLanding_PlacesTheTeamImmediatelyAfterTheVision(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type) VALUES
		 ('vision_title', 'Our vision', 'text'),
		 ('vision_content', 'A focused vision.', 'text');`,
	); err != nil {
		t.Fatalf("seed homepage vision: %v", err)
	}
	seedMember(t, conn, "A Brother", "Code", "Bio", true)
	seedGame(t, conn, "For Sale", "for-sale", true, true, "")

	html, _ := renderLandingPage(t, newTestSite(t, conn))
	vision := strings.Index(html, `id=vision-heading`)
	members := strings.Index(html, `id=members-heading`)
	sales := strings.Index(html, `id=sales-heading`)
	if vision == -1 || members == -1 || sales == -1 {
		t.Fatalf("expected vision, members and sales sections in rendered landing page")
	}
	if !(vision < members && members < sales) {
		t.Errorf("section order is vision=%d members=%d sales=%d; want vision, members, then later homepage sections", vision, members, sales)
	}
}

func TestRenderLanding_UsesAltTextForCarouselImagesAndLabelsThumbnails(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Alt Game", "alt-game", true, false, "")
	shotID := seedMedia(t, conn, "media/screenshot/alt-game.webp", "A tactical battle")
	if _, err := conn.Exec(
		`INSERT INTO game_screenshots (id, game_id, media_id, display_order)
		 VALUES (?, ?, ?, 0);`, id.New(), gameID, shotID,
	); err != nil {
		t.Fatalf("seed screenshot: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if !strings.Contains(html, `alt="A tactical battle" aria-hidden=true`) {
		t.Error("the large carousel screenshot does not use its CMS alt text")
	}
	if !strings.Contains(html, `alt="A tactical battle thumbnail"`) {
		t.Error("the carousel thumbnail is not distinguished from the large screenshot")
	}
}

// A call to action with nowhere to go is a button that does nothing.
func TestRenderLanding_HidesTheCTAWithoutALink(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type)
		 VALUES ('hero_cta_text', 'Click me', 'text');`,
	); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if strings.Contains(html, "Click me") {
		t.Error("a call to action rendered with no destination")
	}
}

func TestRenderLanding_ShowsOnlyGamesThatAreForSaleInTheSalesGrid(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Free Game", "free", true, false, "")
	forSale := seedGame(t, conn, "Paid Game", "paid", true, true, "")
	if _, err := conn.Exec(
		`UPDATE games SET price_display = '$9.99',
		 external_links_json = '[{"label":"Steam","url":"https://store.steampowered.com/app/1"}]'
		 WHERE id = ?;`, forSale,
	); err != nil {
		t.Fatalf("set sale fields: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	salesSection := html[strings.Index(html, "sales-grid"):]
	if !strings.Contains(salesSection, "Paid Game") {
		t.Error("a for-sale game is missing from the sales grid")
	}
	if strings.Contains(salesSection, "Free Game") {
		t.Error("a game that is not for sale appeared in the sales grid")
	}
	if !strings.Contains(html, "$9.99") {
		t.Error("the price is not shown")
	}
	// The card leads to the game's own page, where the stores are listed.
	// Jumping straight out to a shop skipped everything the site has to say
	// about the game.
	if !strings.Contains(salesSection, "href=/games/paid") {
		t.Error("the sales card does not lead to the game")
	}
	if strings.Contains(salesSection, "store.steampowered.com") {
		t.Error("the card still links straight out to a shop")
	}
}

// Sections with nothing in them are omitted rather than rendered as empty
// headings.
func TestRenderLanding_OmitsEmptySections(t *testing.T) {
	conn := setupTestDB(t)
	html, _ := renderLandingPage(t, newTestSite(t, conn))

	for _, absent := range []string{"sales-grid", "carousel__track", `class="members"`} {
		if strings.Contains(html, absent) {
			t.Errorf("an empty section rendered anyway: %q", absent)
		}
	}
}

func TestRenderLanding_RendersTagsAsPills(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Tagged Game", "tagged", true, false, "Pixel Art, Local Coop , ")

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "Pixel Art") || !strings.Contains(html, "Local Coop") {
		t.Error("tags were not rendered")
	}
	// A trailing comma must not produce an empty pill.
	if strings.Contains(html, "<li class=tag></li>") {
		t.Error("a blank tag rendered from a trailing comma")
	}
}

// The carousel works without JavaScript: the arrows and dots are anchors to a
// slide's id, computed at render time. A script only upgrades the click so it
// scrolls the track instead of yanking the page, so the anchors must survive.
func TestRenderLanding_CarouselArrowsPointAtNeighbouringSlides(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "One", "one", true, false, "")
	seedGame(t, conn, "Two", "two", true, false, "")
	seedGame(t, conn, "Three", "three", true, false, "")

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	// The dots are anchors, so the carousel is still navigable with scripting
	// off; the arrows are the script's own and stay hidden until it takes over.
	for _, want := range []string{"#slide-1", "#slide-2", "#slide-3"} {
		if !strings.Contains(html, want) {
			t.Errorf("no dot links to %q", want)
		}
	}
	if !strings.Contains(html, "id=slide-1") {
		t.Error("slides have no ids, so the dots have nothing to target")
	}
	if !strings.Contains(html, `href=#slide-2 data-carousel-target=slide-2`) {
		t.Error("a dot is missing either its href or the hook the script needs")
	}

	// One pair of arrows for the whole carousel, not a pair per card.
	if got := strings.Count(html, "data-carousel-prev"); got != 1 {
		t.Errorf("previous arrows = %d, want exactly 1 for the carousel", got)
	}
	if got := strings.Count(html, "data-carousel-next"); got != 1 {
		t.Errorf("next arrows = %d, want exactly 1 for the carousel", got)
	}
}

func TestSplitTags(t *testing.T) {
	cases := map[string]int{
		"":                    0,
		"   ":                 0,
		"one":                 1,
		"one, two":            2,
		"one, two, ":          2,
		" one ,, two , three": 3,
	}
	for input, want := range cases {
		if got := splitTags(input); len(got) != want {
			t.Errorf("splitTags(%q) = %v, want %d entries", input, got, want)
		}
	}
}

// The carousel's media area is 16:9 and the OG image is the wide one, so it
// wins over the portrait cover art rather than being cropped to a strip.
func TestRenderLanding_CarouselPrefersTheOpenGraphImage(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Wide Game", "wide-game", true, false, "")
	ogID := seedMedia(t, conn, "media/og/2026-wide.webp", "Wide art")
	coverID := seedMedia(t, conn, "media/cover/2026-tall.webp", "Tall art")
	if _, err := conn.Exec(
		`UPDATE games SET og_image_id = ?, cd_cover_art_id = ? WHERE id = ?;`,
		ogID, coverID, gameID,
	); err != nil {
		t.Fatalf("set art: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	slide := html[strings.Index(html, "slide__media"):strings.Index(html, "slide__info")]
	if !strings.Contains(slide, "2026-wide.webp") {
		t.Error("the carousel is not using the OpenGraph image")
	}
	if strings.Contains(slide, "2026-tall.webp") {
		t.Error("the portrait cover art is still filling the 16:9 media area")
	}
}

// With no OG image it still falls back, rather than showing a bare title.
func TestRenderLanding_CarouselFallsBackToCoverArt(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Cover Only", "cover-only", true, false, "")
	coverID := seedMedia(t, conn, "media/cover/2026-only.webp", "Cover art")
	if _, err := conn.Exec(
		`UPDATE games SET cd_cover_art_id = ? WHERE id = ?;`, coverID, gameID,
	); err != nil {
		t.Fatalf("set art: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "2026-only.webp") {
		t.Error("a game with only cover art shows no image at all")
	}
}

// The tags are the roles the panel collects, so they read as a title and sit
// under the name rather than after the biography.
func TestRenderLanding_MemberTagsSitBetweenNameAndBio(t *testing.T) {
	conn := setupTestDB(t)
	seedMember(t, conn, "Someone", "Code, Music", "A short biography line.", true)

	page, _ := renderLandingPage(t, newTestSite(t, conn))
	// The order being tested is the order on screen, so the head is cut away:
	// the structured data there names the same member for a crawler, in an
	// order that has nothing to do with the layout.
	html := page[strings.Index(page, "<body"):]

	name := strings.Index(html, "Someone")
	tag := strings.Index(html, "Music")
	bio := strings.Index(html, "A short biography line.")
	if name == -1 || tag == -1 || bio == -1 {
		t.Fatalf("member did not render fully: name=%d tag=%d bio=%d", name, tag, bio)
	}
	if !(name < tag && tag < bio) {
		t.Error("member order is not name, tags, bio")
	}
}

// Game pages exist now, so the portfolio has to lead to them -- the artwork is
// the way in, per the visual spec.
func TestRenderLanding_PortfolioCardsLinkToTheGame(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Linked Game", "linked-game", true, false, "")

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if strings.Count(html, "href=/games/linked-game") < 2 {
		t.Error("the card's artwork and its title should both lead to the game")
	}
}

// Members' own links come from the panel, so they belong on the card.
func TestRenderLanding_ShowsMemberLinks(t *testing.T) {
	conn := setupTestDB(t)
	memberID := seedMember(t, conn, "Someone", "Code", "A bio.", true)
	if _, err := conn.Exec(
		`UPDATE members SET links_json = '[{"label":"itch.io","url":"https://itch.io/someone"}]' WHERE id = ?;`,
		memberID,
	); err != nil {
		t.Fatalf("set links: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "https://itch.io/someone") {
		t.Error("the member's link is missing")
	}
	if !strings.Contains(html, "itch.io<") {
		t.Error("the link has no label")
	}
}

// A member with no links gets no empty list.
func TestRenderLanding_OmitsAnEmptyMemberLinkList(t *testing.T) {
	conn := setupTestDB(t)
	seedMember(t, conn, "Someone", "Code", "A bio.", true)

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if strings.Contains(html, "member__links") {
		t.Error("an empty link list rendered anyway")
	}
}

func TestCardTags_KeepsTheFirstFewAndCountsTheRest(t *testing.T) {
	shown, hidden := cardTags("a, b, c, d, e, f, g, h", 6)
	if len(shown) != 6 || hidden != 2 {
		t.Errorf("cardTags() = %v, %d, want 6 shown and 2 hidden", shown, hidden)
	}
	if shown[0] != "a" || shown[5] != "f" {
		t.Errorf("cardTags() kept %v, want the first six in order", shown)
	}
}

// Under the limit nothing is hidden, so no counter is drawn.
func TestCardTags_HidesNothingWhenTheyFit(t *testing.T) {
	for _, raw := range []string{"", "a", "a, b, c, d"} {
		shown, hidden := cardTags(raw, 4)
		if hidden != 0 {
			t.Errorf("cardTags(%q) hid %d, want none (shown: %v)", raw, hidden, shown)
		}
	}
}

// The real complaint: a game with ten tags covered its own artwork. The cap is
// per surface because a carousel slide and a sale card are different widths.
func TestRenderLanding_CapsTheTagsOnACard(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedGame(t, conn, "Overtagged", "overtagged", true, true,
		"turn based, strategy, tactics, unit management, dungeon, pvp, pve, 2d, pixel art, metal music")

	html, _, err := newTestSite(t, conn).renderLanding("index.html")
	if err != nil {
		t.Fatalf("renderLanding() error = %v", err)
	}
	body := string(html)

	// Ten tags, so both surfaces leave some out and say how many.
	if !strings.Contains(body, "+4") {
		t.Error("the carousel slide does not say how many tags it left out")
	}
	if !strings.Contains(body, "+6") {
		t.Error("the sale card does not say how many tags it left out")
	}
	// The last tag alphabetically late in the list must not be on either card.
	if strings.Contains(body, ">metal music<") {
		t.Error("a tag past the cap still reached a card")
	}
	// Both lists are clamped, which is the guarantee a count alone cannot make.
	if strings.Count(body, "tags--clamped") != 2 {
		t.Errorf("wanted both card tag lists clamped, got %d",
			strings.Count(body, "tags--clamped"))
	}
}

// The detail page has room for the whole list, so it is not capped: that is
// where a reader goes for everything about the game.
func TestRenderGame_ShowsEveryTag(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedGame(t, conn, "Overtagged", "overtagged", true, false,
		"turn based, strategy, tactics, unit management, dungeon, pvp, pve, 2d, pixel art, metal music")

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "overtagged")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	body := string(html)
	if !strings.Contains(body, ">metal music<") {
		t.Error("the detail page dropped a tag")
	}
	if strings.Contains(body, "tags--clamped") {
		t.Error("the detail page clamped its tag list")
	}
}

// The homepage button usually points at a page of this site, so a path is what
// an admin writes. It has to render as that path, not be mangled into
// something absolute.
func TestRenderLanding_UsesAPathForTheButton(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO homepage_settings (key, value, value_type) VALUES
		 ('hero_cta_text', 'Play now', 'text'),
		 ('hero_cta_link', '/games', 'link');`,
	); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	html, _ := renderLandingPage(t, newTestSite(t, conn))
	if !strings.Contains(html, `href=/games`) {
		t.Errorf("the button does not point at the path it was given: %s", html)
	}
}

func seedLandingPost(t *testing.T, conn *sql.DB, title, slug, publishedAt string, published bool) string {
	t.Helper()
	return seedPost(t, conn, title, slug, "Body of "+title, publishedAt, published, nil)
}

func TestRenderLanding_DeclaresDevlogAndAwardTags(t *testing.T) {
	conn := setupTestDB(t)
	_, tags := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{"devlog:list", "award:list"} {
		found := false
		for _, tag := range tags {
			if tag == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the landing page never declared the %q tag", want)
		}
	}
}

func TestRenderLanding_ShowsLatestTwoPostsAndAwards(t *testing.T) {
	conn := setupTestDB(t)
	seedLandingPost(t, conn, "Newest", "newest", "2026-08-01", true)
	seedLandingPost(t, conn, "Second", "second", "2026-07-01", true)
	seedLandingPost(t, conn, "Third", "third", "2026-06-01", true)
	seedLandingPost(t, conn, "Draft", "draft", "2026-08-02", false)
	seedAward(t, conn, "Best Game", "IndieCade", "2026-05-01", "", nil)
	seedAward(t, conn, "Best Audio", "PixelFest", "2025-05-01", "", nil)

	html, _ := renderLandingPage(t, newTestSite(t, conn))

	for _, want := range []string{"Newest", "Second", "Best Game", "Best Audio", "IndieCade"} {
		if !strings.Contains(html, want) {
			t.Errorf("landing page is missing %q", want)
		}
	}
	if strings.Contains(html, "Third") {
		t.Error("a third post leaked past the two-post budget")
	}
	if strings.Contains(html, ">Draft<") {
		t.Error("an unpublished post leaked onto the landing page")
	}
}

func TestRenderLanding_HidesLogAndAchievementsWhenEmpty(t *testing.T) {
	conn := setupTestDB(t)
	html, _ := renderLandingPage(t, newTestSite(t, conn))

	if strings.Contains(html, "System log") {
		t.Error("the system log heading rendered with no posts")
	}
	if strings.Contains(html, "Achievements") {
		t.Error("the achievements heading rendered with no awards")
	}
}
