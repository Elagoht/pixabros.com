package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// cssHash matches the content hash in the published stylesheet URL.
var cssHash = regexp.MustCompile(`site\.[a-f0-9]{8}\.css`)

// renderAwardsPage runs the renderer the way the store would.
func renderAwardsPage(t *testing.T, s *Site) (string, []string) {
	t.Helper()
	html, tags, err := s.renderAwards(PageAwards)
	if err != nil {
		t.Fatalf("renderAwards() error = %v", err)
	}
	return string(html), tags
}

// The tags are the page's whole connection to the admin panel: get one wrong
// and editing an award silently never updates the site.
func TestRenderAwards_DeclaresTheTagsTheAdminAPIEnqueues(t *testing.T) {
	conn := setupTestDB(t)
	_, tags := renderAwardsPage(t, newTestSite(t, conn))

	want := map[string]bool{"award:list": false, "site_settings": false}
	for _, tag := range tags {
		if _, ok := want[tag]; !ok {
			t.Errorf("unexpected tag %q", tag)
			continue
		}
		want[tag] = true
	}
	for tag, seen := range want {
		if !seen {
			t.Errorf("page never declared the %q tag", tag)
		}
	}
}

func TestRenderAwards_ShowsAwardsNewestFirst(t *testing.T) {
	conn := setupTestDB(t)
	seedAward(t, conn, "Older Prize", "Some Jury", "2024-01-15", "", nil)
	seedAward(t, conn, "Newer Prize", "Another Jury", "2026-03-02", "", nil)

	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	newer := strings.Index(html, "Newer Prize")
	older := strings.Index(html, "Older Prize")
	if newer == -1 || older == -1 {
		t.Fatalf("both awards should appear; newer=%d older=%d", newer, older)
	}
	if newer > older {
		t.Error("awards are not newest first")
	}
}

func TestRenderAwards_RendersDatesInEnglishLongForm(t *testing.T) {
	conn := setupTestDB(t)
	seedAward(t, conn, "Best Game", "A Jury", "2026-06-21", "", nil)

	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "21 June 2026") {
		t.Error("the stored date was not rendered in English long form")
	}
}

func TestRenderAwards_UsesTheBadgeImageAndItsAltText(t *testing.T) {
	conn := setupTestDB(t)
	mediaID := seedMedia(t, conn, "media/award/2026-badge.webp", "A golden trophy")
	seedAward(t, conn, "Best Game", "A Jury", "2026-06-21", "", &mediaID)

	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "/media/award/2026-badge.webp") {
		t.Error("the badge image is missing")
	}
	if !strings.Contains(html, "A golden trophy") {
		t.Error("the image's alt text was not used")
	}
}

// The renderer falls back to a placeholder when a badge image cannot be
// loaded, but the schema is what actually prevents the situation: picture_id
// is a foreign key, so a dangling reference cannot be written, and deleting
// the image nulls the column rather than leaving a broken pointer. Both halves
// are asserted here, because the fallback is only belt-and-braces if the
// constraint really holds.
func TestRenderAwards_ADeletedBadgeLeavesTheAwardIntact(t *testing.T) {
	conn := setupTestDB(t)
	mediaID := seedMedia(t, conn, "media/award/2026-badge.webp", "A golden trophy")
	awardID := seedAward(t, conn, "Best Game", "A Jury", "2026-06-21", "", &mediaID)

	// A dangling reference is refused outright.
	if _, err := conn.Exec(
		`UPDATE awards SET picture_id = 'nonexistentmediaid000000' WHERE id = ?;`, awardID,
	); err == nil {
		t.Error("the database accepted a picture_id pointing at no media row")
	}

	// Deleting the image nulls the column (ON DELETE SET NULL).
	if _, err := conn.Exec(`DELETE FROM media WHERE id = ?;`, mediaID); err != nil {
		t.Fatalf("delete media: %v", err)
	}

	html, _ := renderAwardsPage(t, newTestSite(t, conn))
	if !strings.Contains(html, "Best Game") {
		t.Error("the award vanished after its badge image was deleted")
	}
	if strings.Contains(html, "2026-badge.webp") {
		t.Error("the page still points at a deleted image")
	}
}

func TestRenderAwards_RendersAnEmptyState(t *testing.T) {
	conn := setupTestDB(t)
	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	if !strings.Contains(html, "No awards to show") {
		t.Error("a site with no awards should still say so, not render a bare page")
	}
}

// Game links are deliberately not emitted yet: game detail pages do not exist,
// and a link to a 404 is worse than no link.
func TestRenderAwards_DoesNotLinkToGameDetailPages(t *testing.T) {
	conn := setupTestDB(t)
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title) VALUES ('g1', 'some-game', 'Some Game');`,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}
	awardID := seedAward(t, conn, "Best Game", "A Jury", "2026-06-21", "", nil)
	if _, err := conn.Exec(`UPDATE awards SET game_id = 'g1' WHERE id = ?;`, awardID); err != nil {
		t.Fatalf("link award to game: %v", err)
	}

	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	if strings.Contains(html, "/games/some-game") {
		t.Error("the page links to a game detail page that does not exist yet")
	}
}

// Golden file, per the architecture spec's test strategy. It holds the
// minified output, since that is what is stored and served.
func TestRenderAwards_MatchesGoldenFile(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_name":       "Pixabros",
		"twitter_handle":  "PixaBrosStudio",
		"org_sameas_json": `["https://itch.io/pixabros"]`,
	})
	mediaID := seedMedia(t, conn, "media/award/2026-badge.webp", "A golden trophy")
	// One with a picture and a link, one with neither, so both branches are
	// covered by the golden output.
	seedAward(t, conn, "Best Game", "Indie Jury", "2026-06-21", "https://example.com/award", &mediaID)
	seedAward(t, conn, "Honourable Mention", "Some Festival", "2025-02-10", "", nil)

	html, _ := renderAwardsPage(t, newTestSite(t, conn))

	// The stylesheet's content hash changes with every CSS edit, which is an
	// intended and frequent change. Normalising it keeps the golden file about
	// the HTML -- what it is actually there to protect -- instead of failing
	// every time a colour moves.
	html = cssHash.ReplaceAllString(html, "site.HASH.css")

	goldenPath := filepath.Join("testdata", "awards.golden.html")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(html), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("golden file updated")
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v (run with UPDATE_GOLDEN=1 to create it)", err)
	}
	if html != string(want) {
		t.Errorf("rendered output differs from the golden file.\n--- got ---\n%s\n--- want ---\n%s", html, want)
	}
}
