package site

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// seedSEOSettings gives the site the address and studio details every canonical
// link and every structured-data node is built from.
func seedSEOSettings(t *testing.T, conn *sql.DB) {
	t.Helper()
	seedSiteSettings(t, conn, map[string]string{
		"site_name":             "Pixabros",
		"site_url":              "https://pixabros.com",
		"site_tagline":          "Brothers Makes Games",
		"twitter_handle":        "PixaBrosStudio",
		"org_description":       "A two brother game studio.",
		"org_legal_name":        "Pixabros Studio",
		"org_email":             "hello@pixabros.com",
		"org_founding_date":     "2022-01-01",
		"org_founding_location": "Istanbul",
		"org_sameas_json":       `["https://itch.io/pixabros"]`,
	})
}

var titlePattern = regexp.MustCompile(`(?s)<title>(.*?)</title>`)

func titleOf(t *testing.T, html string) string {
	t.Helper()
	match := titlePattern.FindStringSubmatch(html)
	if match == nil {
		t.Fatal("the page has no title")
	}
	return match[1]
}

func TestBuildTitle_LeadsWithTheSiteName(t *testing.T) {
	got := buildTitle("Pixabros", "Brothers Makes Games", "Awards")
	if !strings.HasPrefix(got, "Pixabros | ") {
		t.Errorf("buildTitle() = %q, want it to lead with the site's name", got)
	}
}

func TestBuildTitle_DoesNotPadAConciseTitle(t *testing.T) {
	got := buildTitle("Pixabros", "Brothers Makes Games", "Awards")
	if got != "Pixabros | Awards" {
		t.Errorf("buildTitle() = %q, want a concise title without tagline padding", got)
	}
}

func TestRenderLanding_UsesAConciseSearchTitle(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})

	html, _, err := newTestSite(t, conn).renderLanding(PageLanding)
	if err != nil {
		t.Fatalf("renderLanding() error = %v", err)
	}
	if got := titleOf(t, string(html)); got != "Pixabros | We Are Two Brothers Making Games" {
		t.Errorf("landing title = %q, want the concise homepage title", got)
	}
}

func TestBuildTitle_NeverExceedsTheCeiling(t *testing.T) {
	subjects := []string{
		"Awards",
		strings.Repeat("A very long game title ", 12),
		"Deterministic tactics on two five by five grids with no dice rolls at all anywhere",
	}
	for _, subject := range subjects {
		got := buildTitle("Pixabros", "Brothers Makes Games", subject)
		if n := utf8.RuneCountInString(got); n > titleMaxLength {
			t.Errorf("buildTitle(%q) is %d characters: %q", subject, n, got)
		}
	}
}

// A cut title ends in an ellipsis so it reads as shortened rather than as a
// sentence that stopped.
func TestBuildTitle_ClosesACutTitleWithAnEllipsis(t *testing.T) {
	got := buildTitle("Pixabros", "Brothers Makes Games", strings.Repeat("word ", 40))
	if !strings.HasSuffix(got, "…") {
		t.Errorf("a cut title does not say it was cut: %q", got)
	}
	if strings.Contains(got, " …") {
		t.Errorf("the cut left a dangling space: %q", got)
	}
}

// The rule holds for every page, not just the ones a test remembered to check.
func TestPages_TitlesObeyTheLengthRule(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "Tactics")
	seedMember(t, conn, "Someone", "Code", "A bio.", true)
	seedAward(t, conn, "A Prize", "A Jury", "2026-01-01", "", nil)
	seedPost(t, conn, "A Post", "a-post", "Body text.", "2026-01-02", true, nil)

	site := newTestSite(t, conn)
	for key, render := range everyPage(site) {
		html, _, err := render(key)
		if err != nil {
			t.Fatalf("render %s: %v", key, err)
		}

		title := titleOf(t, string(html))
		length := utf8.RuneCountInString(title)
		if !strings.HasPrefix(title, "Pixabros | ") {
			t.Errorf("%s title does not lead with the site's name: %q", key, title)
		}
		if length > titleMaxLength {
			t.Errorf("%s title is %d characters, want at most %d: %q",
				key, length, titleMaxLength, title)
		}
	}
}

// Every page states the one address it should be indexed under. Without it a
// query string or a trailing slash reads as a second copy of the page.
func TestPages_AreCanonicalToThemselves(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "")
	seedPost(t, conn, "A Post", "a-post", "Body.", "2026-01-02", true, nil)

	site := newTestSite(t, conn)
	want := map[string]string{
		"index.html":            "https://pixabros.com/",
		"games":                 "https://pixabros.com/games",
		"games/dungrid-tactics": "https://pixabros.com/games/dungrid-tactics",
		"devlog":                "https://pixabros.com/devlog",
		"devlog/a-post":         "https://pixabros.com/devlog/a-post",
		"awards":                "https://pixabros.com/awards",
		"contact":               "https://pixabros.com/contact",
		"contact/sent":          "https://pixabros.com/contact/sent",
	}
	for key, render := range everyPage(site) {
		html, _, err := render(key)
		if err != nil {
			t.Fatalf("render %s: %v", key, err)
		}
		got := `rel=canonical href=` + want[key]
		if !strings.Contains(string(html), got) {
			t.Errorf("%s is not canonical to itself, wanted %q", key, got)
		}
	}
}

// An unset site address costs the canonical link and nothing else.
//
// A canonical pointing at a guess is worse than none, so that one is dropped.
// Everything that does not need an address -- the studio's name, its logo, the
// share card's picture -- is published either way. An optional setting must not
// silently switch off features that do not depend on it.
func TestPages_DegradeWithoutASiteAddress(t *testing.T) {
	conn := setupTestDB(t)
	logoID := seedMedia(t, conn, "media/org_logo/2026-logo.webp", "The logo")
	seedSiteSettings(t, conn, map[string]string{
		"site_name":       "Pixabros",
		"org_description": "A two brother game studio.",
		"org_logo":        logoID,
	})

	html, _, err := newTestSite(t, conn).renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}
	body := string(html)

	if strings.Contains(body, "rel=canonical") {
		t.Error("a canonical link was published with no site address to build it from")
	}
	if !strings.Contains(body, "og:image") {
		t.Error("the share card lost its picture along with the address")
	}

	org := nodeOfType(graphOf(t, body), "Organization")
	if org == nil {
		t.Fatal("the studio was not published at all")
	}
	if org["name"] != "Pixabros" || org["description"] != "A two brother game studio." {
		t.Errorf("the studio was published without what it does know: %v", org)
	}
	// A document-relative id still lets the nodes reference each other.
	if org["@id"] != "#organization" {
		t.Errorf("organization @id = %v, want a document-relative fragment", org["@id"])
	}
	if _, present := org["url"]; present {
		t.Errorf("a URL was published with no address to build one from: %v", org["url"])
	}
}

// graphOf pulls a page's structured data out and parses it, which is also a
// check that what was published is valid JSON.
func graphOf(t *testing.T, html string) []map[string]any {
	t.Helper()

	const open = `<script type=application/ld+json>`
	start := strings.Index(html, open)
	if start < 0 {
		t.Fatal("the page has no structured data")
	}
	rest := html[start+len(open):]
	end := strings.Index(rest, "</script>")
	if end < 0 {
		t.Fatal("the structured data block is not closed")
	}

	var doc struct {
		Context string           `json:"@context"`
		Graph   []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(rest[:end]), &doc); err != nil {
		t.Fatalf("the structured data is not valid JSON: %v\n%s", err, rest[:end])
	}
	if doc.Context != "https://schema.org" {
		t.Errorf("@context = %q", doc.Context)
	}
	return doc.Graph
}

func nodeOfType(graph []map[string]any, wanted string) map[string]any {
	for _, node := range graph {
		if node["@type"] == wanted {
			return node
		}
	}
	return nil
}

// Whoever made this and what this is: every page answers both, wherever a
// crawler happens to land.
func TestPages_AllCarryTheStudioAndTheSite(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "")
	seedPost(t, conn, "A Post", "a-post", "Body.", "2026-01-02", true, nil)

	site := newTestSite(t, conn)
	for key, render := range everyPage(site) {
		if key == PageContactSent {
			// A confirmation is not a page anyone should reach from a search.
			continue
		}
		html, _, err := render(key)
		if err != nil {
			t.Fatalf("render %s: %v", key, err)
		}

		graph := graphOf(t, string(html))
		if nodeOfType(graph, "Organization") == nil {
			t.Errorf("%s does not say who made it", key)
		}
		if nodeOfType(graph, "WebSite") == nil {
			t.Errorf("%s does not say what site it belongs to", key)
		}
	}
}

func TestLandingSchema_DescribesTheStudioAndItsPeople(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedMember(t, conn, "Furkan", "Code, Design", "Writes the code.", true)

	html, _, err := newTestSite(t, conn).renderLanding("index.html")
	if err != nil {
		t.Fatalf("renderLanding() error = %v", err)
	}

	org := nodeOfType(graphOf(t, string(html)), "Organization")
	if org == nil {
		t.Fatal("no organization on the landing page")
	}
	for key, want := range map[string]any{
		"name":             "Pixabros",
		"legalName":        "Pixabros Studio",
		"description":      "A two brother game studio.",
		"foundingDate":     "2022-01-01",
		"foundingLocation": "Istanbul",
	} {
		if org[key] != want {
			t.Errorf("organization %s = %v, want %v", key, org[key], want)
		}
	}

	members, ok := org["member"].([]any)
	if !ok || len(members) != 1 {
		t.Fatalf("organization member = %v, want the one seeded member", org["member"])
	}
	person, _ := members[0].(map[string]any)
	if person["name"] != "Furkan" {
		t.Errorf("member name = %v", person["name"])
	}
	if person["jobTitle"] != "Code, Design" {
		t.Errorf("member jobTitle = %v, want the member's tags", person["jobTitle"])
	}
}

// A setting the studio has not filled in is left out of the graph. Publishing
// an empty name or a blank description is worse than publishing neither.
func TestSchema_LeavesOutWhatIsNotConfigured(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_name": "Pixabros",
		"site_url":  "https://pixabros.com",
	})

	html, _, err := newTestSite(t, conn).renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}

	org := nodeOfType(graphOf(t, string(html)), "Organization")
	for _, key := range []string{"legalName", "description", "foundingDate", "contactPoint", "sameAs"} {
		if _, present := org[key]; present {
			t.Errorf("organization published an unconfigured %s: %v", key, org[key])
		}
	}
}

func TestGameSchema_DescribesTheGame(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	gameID := seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, true, "tactics, pvp")
	if _, err := conn.Exec(
		`UPDATE games SET genre = 'Turn-based tactics', release_date = '2026-07-31',
		 short_description = 'A tactics game.', price_display = '$4.99',
		 video_url = 'https://youtu.be/9mjjowHX1-g', is_browser_playable = 1,
		 external_links_json = '[{"label":"itch.io","url":"https://itch.io/x"}]'
		 WHERE id = ?;`, gameID,
	); err != nil {
		t.Fatalf("fill the game in: %v", err)
	}

	html, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "dungrid-tactics")
	if err != nil {
		t.Fatalf("renderGame() error = %v", err)
	}
	graph := graphOf(t, string(html))

	game := nodeOfType(graph, "VideoGame")
	if game == nil {
		t.Fatal("the game's page does not describe a game")
	}
	for key, want := range map[string]any{
		"name":          "Dungrid Tactics",
		"genre":         "Turn-based tactics",
		"datePublished": "2026-07-31",
		"url":           "https://pixabros.com/games/dungrid-tactics",
		"gamePlatform":  "Web browser",
	} {
		if game[key] != want {
			t.Errorf("game %s = %v, want %v", key, game[key], want)
		}
	}
	if game["trailer"] == nil {
		t.Error("the game has a trailer that the structured data does not mention")
	}
	if game["offers"] == nil {
		t.Error("the game is for sale and the structured data does not say so")
	}

	// The trail a search result shows in place of a bare URL.
	if crumbs := nodeOfType(graph, "BreadcrumbList"); crumbs == nil {
		t.Error("the game's page has no breadcrumb trail")
	}
}

func TestDevlogPostSchema_DescribesThePost(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedPost(t, conn, "How It Was Made", "how-it-was-made", "Body text here.", "2026-01-02", true, nil)

	html, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "how-it-was-made")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}
	body := string(html)

	post := nodeOfType(graphOf(t, body), "BlogPosting")
	if post == nil {
		t.Fatal("the post is not described as one")
	}
	if post["headline"] != "How It Was Made" {
		t.Errorf("headline = %v", post["headline"])
	}
	if post["datePublished"] != "2026-01-02" {
		t.Errorf("datePublished = %v", post["datePublished"])
	}
	// A post is an article, and a share card that calls it a website is wrong.
	if !strings.Contains(body, `content="article"`) {
		t.Error("the post's share card does not call it an article")
	}
}

// A share card read off-site cannot follow a relative path, so the picture is
// published as a whole address.
func TestPages_ShareCardImagesAreAbsolute(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	logoID := seedMedia(t, conn, "media/org_logo/2026-logo.webp", "The logo")
	seedSiteSettings(t, conn, map[string]string{"org_logo": logoID})

	html, _, err := newTestSite(t, conn).renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}
	want := `content="https://pixabros.com/media/org_logo/2026-logo.webp"`
	if !strings.Contains(string(html), want) {
		t.Errorf("the share card image is not an absolute address, wanted %s", want)
	}
}

func TestCanonicalURL(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"https://pixabros.com", "", "https://pixabros.com/"},
		{"https://pixabros.com", "index.html", "https://pixabros.com/"},
		{"https://pixabros.com/", "games", "https://pixabros.com/games"},
		{"https://pixabros.com", "/games", "https://pixabros.com/games"},
		{"", "games", ""},
	}
	for _, c := range cases {
		if got := canonicalURL(c.base, c.path); got != c.want {
			t.Errorf("canonicalURL(%q, %q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

// everyPage is the whole public site, so a rule meant for all of it is checked
// against all of it rather than against whichever pages a test remembered.
func everyPage(site *Site) map[string]func(string) ([]byte, []string, error) {
	return map[string]func(string) ([]byte, []string, error){
		"index.html":            site.renderLanding,
		PageGames:               site.renderArcade,
		"games/dungrid-tactics": site.renderGame,
		PageDevlog:              site.renderDevlogIndex,
		"devlog/a-post":         site.renderDevlogPost,
		PageAwards:              site.renderAwards,
		PageContact:             site.renderContact,
		PageContactSent:         site.renderContactSent,
	}
}

func TestBuildDescription_PadsOnlyAShortOne(t *testing.T) {
	studio := "A two brother game studio making small, sharp games."

	short := buildDescription("Get in touch.", "Pixabros", studio)
	if !strings.Contains(short, studio) {
		t.Errorf("a short description was not padded: %q", short)
	}

	long := "Dungrid Tactics is a fully deterministic five versus five turn-based " +
		"tactics game played on two grids, with no dice rolls anywhere."
	if got := buildDescription(long, "Pixabros", studio); strings.Contains(got, studio) {
		t.Errorf("a description already long enough was padded anyway: %q", got)
	}
}

// Padding must not make a page say the same sentence twice, which is what
// happens when a page's own summary already is the studio's description.
func TestBuildDescription_DoesNotRepeatItself(t *testing.T) {
	studio := "A two brother game studio."
	got := buildDescription(studio, "Pixabros", studio)
	if strings.Count(got, studio) != 1 {
		t.Errorf("the studio's description was repeated: %q", got)
	}
}

// The rule holds for every page, the same way the title rule does.
func TestPages_DescriptionsObeyTheLengthRule(t *testing.T) {
	conn := setupTestDB(t)
	seedSEOSettings(t, conn)
	seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "Tactics")
	seedMember(t, conn, "Someone", "Code", "A bio.", true)
	seedAward(t, conn, "A Prize", "A Jury", "2026-01-01", "", nil)
	seedPost(t, conn, "A Post", "a-post", "Body text.", "2026-01-02", true, nil)

	pattern := regexp.MustCompile(`<meta name=description content="([^"]*)"`)

	site := newTestSite(t, conn)
	for key, render := range everyPage(site) {
		html, _, err := render(key)
		if err != nil {
			t.Fatalf("render %s: %v", key, err)
		}

		match := pattern.FindStringSubmatch(string(html))
		if match == nil {
			t.Errorf("%s has no description at all", key)
			continue
		}
		if n := utf8.RuneCountInString(match[1]); n < descriptionMinLength || n > descriptionMaxLength {
			t.Errorf("%s description is %d characters, want %d to %d: %q",
				key, n, descriptionMinLength, descriptionMaxLength, match[1])
		}
	}
}
