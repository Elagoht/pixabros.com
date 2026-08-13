package site

import (
	"strings"
	"testing"
)

func renderArcadePage(t *testing.T, s *Site) (string, []string) {
	t.Helper()
	html, tags, err := s.renderArcade(PageGames)
	if err != nil {
		t.Fatalf("renderArcade() error = %v", err)
	}
	return string(html), tags
}

func renderGamePage(t *testing.T, s *Site, slug string) (string, []string) {
	t.Helper()
	html, tags, err := s.renderGame(GamePagePrefix + slug)
	if err != nil {
		t.Fatalf("renderGame(%q) error = %v", slug, err)
	}
	return string(html), tags
}

// Browser-playable games are cartridges; everything else sits on the shelf as
// a case. Direct downloads were dropped as a product decision, so "the rest of
// the catalogue" is what the case grid shows.
func TestRenderArcade_SplitsPlayableFromTheRest(t *testing.T) {
	conn := setupTestDB(t)
	playable := seedGame(t, conn, "Playable One", "playable-one", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET is_browser_playable = 1, web_export_path = 'games/playable-one/build' WHERE id = ?;`,
		playable,
	); err != nil {
		t.Fatalf("mark playable: %v", err)
	}
	seedGame(t, conn, "Shelf Only", "shelf-only", true, false, "")

	html, tags := renderArcadePage(t, newTestSite(t, conn))

	cartridges := html[strings.Index(html, "cartridges"):strings.Index(html, "cases")]
	if !strings.Contains(cartridges, "Playable One") {
		t.Error("a browser-playable game is missing from the cartridge grid")
	}
	if strings.Contains(cartridges, "Shelf Only") {
		t.Error("a non-playable game was put on a cartridge")
	}

	// The shelf holds the whole catalogue: a cartridge is how you start a game,
	// a case is how you read about it, and a playable game has both.
	shelf := html[strings.Index(html, "cases"):]
	for _, title := range []string{"Playable One", "Shelf Only"} {
		if !strings.Contains(shelf, title) {
			t.Errorf("%q is missing from the shelf", title)
		}
	}

	if len(tags) != 2 || tags[0] != "game:list" || tags[1] != "site_settings" {
		t.Errorf("tags = %v, want [game:list site_settings]", tags)
	}
}

func TestRenderArcade_HidesDrafts(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Public Game", "public-game", true, false, "")
	seedGame(t, conn, "Draft Game", "draft-game", false, false, "")

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	if strings.Contains(html, "Draft Game") {
		t.Error("an unpublished game leaked onto the arcade page")
	}
	if !strings.Contains(html, "Public Game") {
		t.Error("a published game is missing")
	}
}

// Opening a case raises a dialog, which needs a script. The shelf must still
// be usable without one, so every case also carries a link to the game's own
// page -- the page that says everything the dialog does.
func TestRenderArcade_CasesRemainReachableWithoutJavaScript(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Shelf Game", "shelf-game", true, false, "")

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	if !strings.Contains(html, "id=case-shelf-game") {
		t.Error("the case dialog has no id for its button to open")
	}
	if !strings.Contains(html, `data-case-open=case-shelf-game`) {
		t.Error("nothing opens the case")
	}
	if !strings.Contains(html, "href=/games/shelf-game") {
		t.Error("the case does not link to the game, so it is a dead end without scripting")
	}
}

// The open case has two pages: the details, and the disc with the artwork
// clipped into it.
func TestRenderArcade_OpenCaseHasBothPages(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Shelf Game", "shelf-game", true, false, "Tactics")
	artID := seedMedia(t, conn, "media/cd/2026-cover.webp", "Cover art")
	if _, err := conn.Exec(
		`UPDATE games SET cd_cover_art_id = ?, short_description = 'A short blurb.',
		 external_links_json = '[{"label":"itch.io","url":"https://itch.io/x"}]'
		 WHERE id = ?;`, artID, gameID,
	); err != nil {
		t.Fatalf("set case fields: %v", err)
	}

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	dialog := html[strings.Index(html, "id=case-shelf-game"):]
	for _, want := range []string{
		"jewel__page--info", // the details page
		"jewel__page--disc", // the disc page
		"disc__hole",        // which is a disc, not a square
		"A short blurb.",
		"itch.io",
	} {
		if !strings.Contains(dialog, want) {
			t.Errorf("the open case is missing %q", want)
		}
	}
}

// The console loads a build in place, which needs a script -- but the shelf
// has to keep working without one, so every cartridge stays a real link to the
// game's own page.
func TestRenderArcade_CartridgesRemainLinksWithoutJavaScript(t *testing.T) {
	conn := setupTestDB(t)
	playable := seedGame(t, conn, "Playable", "playable", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET is_browser_playable = 1 WHERE id = ?;`, playable,
	); err != nil {
		t.Fatalf("mark playable: %v", err)
	}

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	if !strings.Contains(html, "href=/games/playable") {
		t.Error("a cartridge is not a link, so the shelf breaks without scripting")
	}
	// The script needs both the address to load and a name to show.
	if !strings.Contains(html, `data-play-url=/play/playable/`) {
		t.Error("a cartridge carries no build address for the console to load")
	}
	if !strings.Contains(html, "data-play-title") {
		t.Error("a cartridge carries no title for the console to show")
	}
}

func TestRenderArcade_RendersAnEmptyState(t *testing.T) {
	conn := setupTestDB(t)
	html, _ := renderArcadePage(t, newTestSite(t, conn))

	if !strings.Contains(html, "No games published yet") {
		t.Error("an empty catalogue should say so")
	}
}

// The detail page's tag has to name the individual game, or editing it would
// never refresh its own page.
func TestRenderGame_DeclaresItsOwnGameTag(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Some Game", "some-game", true, false, "")

	_, tags := renderGamePage(t, newTestSite(t, conn), "some-game")

	want := "game:" + gameID
	found := false
	for _, tag := range tags {
		if tag == want {
			found = true
		}
	}
	if !found {
		t.Errorf("tags = %v, want one of them to be %q", tags, want)
	}
}

func TestRenderGame_EmbedsTheBuildOnlyWhenPlayable(t *testing.T) {
	conn := setupTestDB(t)
	playable := seedGame(t, conn, "Playable", "playable", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET is_browser_playable = 1 WHERE id = ?;`, playable,
	); err != nil {
		t.Fatalf("mark playable: %v", err)
	}
	seedGame(t, conn, "Not Playable", "not-playable", true, false, "")

	site := newTestSite(t, conn)

	playableHTML, _ := renderGamePage(t, site, "playable")
	if !strings.Contains(playableHTML, `src=/play/playable/`) {
		t.Error("a playable game does not embed its build")
	}

	staticHTML, _ := renderGamePage(t, site, "not-playable")
	if strings.Contains(staticHTML, "<iframe") {
		t.Error("a game with no build still rendered a player")
	}
}

// Typing a draft's address must not reach it, even though the reconciler would
// never ask for the page.
func TestRenderGame_RefusesToRenderADraft(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Draft", "draft", false, false, "")

	if _, _, err := newTestSite(t, conn).renderGame(GamePagePrefix + "draft"); err == nil {
		t.Error("an unpublished game rendered a public page")
	}
}

func TestRenderGame_RejectsKeysThatAreNotGamePages(t *testing.T) {
	site := newTestSite(t, setupTestDB(t))

	for _, key := range []string{"games/", "games/a/b"} {
		if _, _, err := site.renderGame(key); err == nil {
			t.Errorf("renderGame(%q) succeeded, want an error", key)
		}
	}
}

func TestRenderGame_ShowsPriceOnlyWhenForSale(t *testing.T) {
	conn := setupTestDB(t)
	free := seedGame(t, conn, "Free", "free", true, false, "")
	if _, err := conn.Exec(`UPDATE games SET price_display = '$5' WHERE id = ?;`, free); err != nil {
		t.Fatalf("set price: %v", err)
	}

	html, _ := renderGamePage(t, newTestSite(t, conn), "free")

	if strings.Contains(html, "$5") {
		t.Error("a price showed for a game that is not for sale")
	}
}

// Every published game needs a page, and unpublishing one must take its page
// away -- that is what stops a retired game staying online.
func TestDesiredPages_TracksPublishedGames(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Live", "live", true, false, "")
	seedGame(t, conn, "Draft", "draft", false, false, "")

	pages, err := newTestSite(t, conn).DesiredPages()
	if err != nil {
		t.Fatalf("DesiredPages() error = %v", err)
	}

	has := func(key string) bool {
		for _, page := range pages {
			if page == key {
				return true
			}
		}
		return false
	}
	if !has("games/live") {
		t.Error("a published game has no detail page")
	}
	if has("games/draft") {
		t.Error("an unpublished game was given a public page")
	}
	if !has(PageGames) || !has(PageLanding) || !has(PageAwards) {
		t.Error("a static page went missing from the desired set")
	}
}

// The console's controls are wired by the script through these hooks. A
// renamed or dropped attribute leaves a button that does nothing, which is
// invisible until someone clicks it.
func TestRenderArcade_ConsoleCarriesEveryControlHook(t *testing.T) {
	conn := setupTestDB(t)
	playable := seedGame(t, conn, "Playable", "playable", true, false, "")
	if _, err := conn.Exec(
		`UPDATE games SET is_browser_playable = 1 WHERE id = ?;`, playable,
	); err != nil {
		t.Fatalf("mark playable: %v", err)
	}

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	for _, hook := range []string{
		"data-console-screen",    // the frame the build runs in
		"data-console-stage",     // what goes fullscreen
		"data-console-idle",      // the "insert cartridge" message
		"data-console-cartridge", // the slot a cartridge is loaded into
		"data-console-reset",
		"data-console-eject",
		"data-console-crt",
		"data-console-fullscreen",
	} {
		if !strings.Contains(html, hook) {
			t.Errorf("the console is missing %q", hook)
		}
	}
}

// Reset, eject and the screen controls only mean anything once a game is
// running, so they start hidden rather than sitting there inert.
func TestRenderArcade_PlaybackControlsStartHidden(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Shelf", "shelf", true, false, "")

	html, _ := renderArcadePage(t, newTestSite(t, conn))

	for _, control := range []string{
		"data-console-reset", "data-console-eject", "data-console-controls",
	} {
		idx := strings.Index(html, control)
		if idx == -1 {
			t.Errorf("%q is missing entirely", control)
			continue
		}
		// The element's tag starts before the hook; hidden is on the same tag.
		start := strings.LastIndex(html[:idx], "<")
		if !strings.Contains(html[start:idx], "hidden") {
			t.Errorf("%q is visible before a game is loaded", control)
		}
	}
}
