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
	if !strings.Contains(html, "store.steampowered.com") {
		t.Error("the store link is missing -- there is no on-site checkout, so it is the whole purchase path")
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

	for _, want := range []string{"#slide-1", "#slide-2", "#slide-3"} {
		if !strings.Contains(html, want) {
			t.Errorf("no control links to %q", want)
		}
	}
	if !strings.Contains(html, "id=slide-1") {
		t.Error("slides have no ids, so the arrows and dots have nothing to target")
	}
	// Every control the script hooks has to be a real anchor too, or the
	// carousel dies the moment scripting is off.
	if !strings.Contains(html, `href=#slide-2 data-carousel-target=slide-2`) {
		t.Error("a control is missing either its href or the hook the script needs")
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

func TestFirstExternalLink(t *testing.T) {
	cases := map[string]string{
		`[{"label":"Steam","url":"https://steam.example"}]`: "https://steam.example",
		`[{"label":"No URL","url":""},{"url":"https://b"}]`: "https://b",
		`[]`:          "",
		``:            "",
		`not json`:    "",
		`{"url":"x"}`: "",
	}
	for input, want := range cases {
		if got := firstExternalLink(input); got != want {
			t.Errorf("firstExternalLink(%q) = %q, want %q", input, got, want)
		}
	}
}
