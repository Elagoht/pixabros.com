package site

import (
	"database/sql"
	"strings"
	"testing"
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
