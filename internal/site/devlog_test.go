package site

import (
	"database/sql"
	"strings"
	"testing"

	"pixabros/internal/id"
)

func seedPost(t *testing.T, conn *sql.DB, title, slug, body, publishedAt string, published bool, gameID *string) string {
	t.Helper()
	postID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO devlog_posts (id, slug, title, content_markdown, published_at, is_published, game_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?);`,
		postID, slug, title, body, publishedAt, published, gameID,
	); err != nil {
		t.Fatalf("seed post %q: %v", title, err)
	}
	return postID
}

func TestRenderDevlogIndex_ListsPublishedPostsNewestFirst(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "Older Note", "older", "Body", "2025-01-05", true, nil)
	seedPost(t, conn, "Newer Note", "newer", "Body", "2026-04-20", true, nil)
	seedPost(t, conn, "Draft Note", "draft", "Body", "", false, nil)

	html, tags, err := newTestSite(t, conn).renderDevlogIndex(PageDevlog)
	if err != nil {
		t.Fatalf("renderDevlogIndex() error = %v", err)
	}
	body := string(html)

	if strings.Contains(body, "Draft Note") {
		t.Error("an unpublished post leaked onto the public devlog")
	}
	newer, older := strings.Index(body, "Newer Note"), strings.Index(body, "Older Note")
	if newer == -1 || older == -1 {
		t.Fatal("both published posts should be listed")
	}
	if newer > older {
		t.Error("posts are not newest first")
	}

	wantTags := map[string]bool{"devlog:list": false, "game:list": false, "site_settings": false}
	for _, tag := range tags {
		wantTags[tag] = true
	}
	for tag, seen := range wantTags {
		if !seen {
			t.Errorf("the index never declared the %q tag", tag)
		}
	}
}

func TestRenderDevlogIndex_NamesTheGameAPostIsAbout(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Dungrid Tactics", "dungrid-tactics", true, false, "")
	seedPost(t, conn, "How combat works", "combat", "Body", "2026-04-20", true, &gameID)

	html, _, err := newTestSite(t, conn).renderDevlogIndex(PageDevlog)
	if err != nil {
		t.Fatalf("renderDevlogIndex() error = %v", err)
	}

	if !strings.Contains(string(html), "Dungrid Tactics") {
		t.Error("the related game is not named on the index row")
	}
}

func TestRenderDevlogIndex_RendersAnEmptyState(t *testing.T) {
	conn := setupTestDB(t)
	html, _, err := newTestSite(t, conn).renderDevlogIndex(PageDevlog)
	if err != nil {
		t.Fatalf("renderDevlogIndex() error = %v", err)
	}
	if !strings.Contains(string(html), "Nothing written up yet") {
		t.Error("an empty devlog should say so")
	}
}

func TestRenderDevlogPost_RendersMarkdown(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "A Post", "a-post",
		"## A heading\n\nSome **bold** text.\n\n- one\n- two\n", "2026-04-20", true, nil)

	html, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "a-post")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{"<h2", "<strong>bold</strong>", "<li>one"} {
		if !strings.Contains(body, want) {
			t.Errorf("markdown was not rendered: missing %q", want)
		}
	}
}

// goldmark runs without WithUnsafe(), so raw HTML in a post is dropped. That
// is the entire XSS defence -- there is no separate sanitiser to fall back on.
func TestRenderDevlogPost_DropsRawHTMLFromTheBody(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "Nasty", "nasty",
		"Hello\n\n<script>alert('xss')</script>\n\n<img src=x onerror=alert(1)>\n",
		"2026-04-20", true, nil)

	html, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "nasty")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}
	body := string(html)

	// Inside the post's own rendered body nothing from the source may have
	// become a tag at all. This is the assertion that matters: it is the only
	// region the post's markdown reaches.
	prose := body[strings.Index(body, "class=prose"):]
	if end := strings.Index(prose, "</div>"); end >= 0 {
		prose = prose[:end]
	}
	for _, tag := range []string{"<script", "<img", "onerror"} {
		if strings.Contains(prose, tag) {
			t.Errorf("%q from post content became markup: %s", tag, prose)
		}
	}

	// And nowhere in the document may a bare <script> appear: everything the
	// site emits itself carries a src or a type, so an attribute-less one
	// could only have come through as raw HTML.
	if strings.Contains(body, "<script>") {
		t.Error("a script tag from post content reached the page")
	}
	if strings.Contains(body, "onerror=") {
		t.Error("an inline event handler from post content reached the page")
	}
	if !strings.Contains(body, "Hello") {
		t.Error("the surrounding prose was lost")
	}
}

func TestRenderDevlogPost_DeclaresItsOwnTag(t *testing.T) {
	conn := setupTestDB(t)
	postID := seedPost(t, conn, "A Post", "a-post", "Body", "2026-04-20", true, nil)

	_, tags, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "a-post")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}

	found := false
	for _, tag := range tags {
		if tag == "devlog:"+postID {
			found = true
		}
	}
	if !found {
		t.Errorf("tags = %v, want one naming the post", tags)
	}
}

func TestRenderDevlogPost_RefusesADraft(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "Draft", "draft", "Body", "", false, nil)

	if _, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "draft"); err == nil {
		t.Error("an unpublished post rendered a public page")
	}
}

// A post about an unpublished game must not link to a page that will not
// exist.
func TestRenderDevlogPost_DoesNotLinkToAnUnpublishedGame(t *testing.T) {
	conn := setupTestDB(t)
	gameID := seedGame(t, conn, "Secret Game", "secret-game", false, false, "")
	seedPost(t, conn, "A Post", "a-post", "Body", "2026-04-20", true, &gameID)

	html, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "a-post")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}

	if strings.Contains(string(html), "/games/secret-game") {
		t.Error("the post links to a game page that does not exist")
	}
}

func TestDesiredPages_TracksPublishedPosts(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "Live", "live", "Body", "2026-04-20", true, nil)
	seedPost(t, conn, "Draft", "draft", "Body", "", false, nil)

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
	if !has("devlog/live") {
		t.Error("a published post has no page")
	}
	if has("devlog/draft") {
		t.Error("an unpublished post was given a page")
	}
}

func TestExcerpt(t *testing.T) {
	long := strings.Repeat("word ", 100)
	if got := excerpt(long, 60); len(got) > 62 {
		t.Errorf("excerpt is %d characters, want it trimmed to about 60", len(got))
	}
	if got := excerpt("## A **heading**", 160); strings.ContainsAny(got, "#*") {
		t.Errorf("excerpt = %q, want markdown punctuation stripped", got)
	}
}

// Text pasted from a word processor carries no-break spaces. CommonMark needs
// a real space after "##", so "## Heading" renders as literal text -- and
// this is not hypothetical: the studio's first real post hit it twice.
func TestRenderDevlogPost_TreatsANoBreakSpaceAfterAMarkerAsASpace(t *testing.T) {
	conn := setupTestDB(t)
	seedPost(t, conn, "Pasted", "pasted",
		"Intro paragraph.\n\n## About the Game\n\nBody.\n\n- First item\n", "2026-04-20", true, nil)

	html, _, err := newTestSite(t, conn).renderDevlogPost(DevlogPagePrefix + "pasted")
	if err != nil {
		t.Fatalf("renderDevlogPost() error = %v", err)
	}
	body := string(html)

	if !strings.Contains(body, "<h2") {
		t.Error("a heading written with a no-break space stayed literal text")
	}
	if strings.Contains(body, "##") {
		t.Error("the raw ## marker is still visible on the page")
	}
	if !strings.Contains(body, "<li>First item") {
		t.Error("a list item written with a no-break space did not become a list")
	}
}

// A no-break space inside a sentence is usually deliberate, so only the one
// straight after a block marker is touched.
func TestNormaliseBlockMarkers_LeavesProseAlone(t *testing.T) {
	source := "A distance of 10 km, and a heading:\n\n## Title\n"
	got := normaliseBlockMarkers(source)

	if !strings.Contains(got, "10 km") {
		t.Error("a no-break space inside a sentence was rewritten")
	}
	if !strings.Contains(got, "## Title") {
		t.Error("the no-break space after the heading marker was not normalised")
	}
}

func seedPostFor(t *testing.T, conn *sql.DB, title, slug, publishedAt string, published bool, gameID *string) string {
	t.Helper()
	return seedPost(t, conn, title, slug, "Body", publishedAt, published, gameID)
}

func renderDevlogIndexPage(t *testing.T, s *Site) string {
	t.Helper()
	html, _, err := s.renderDevlogIndex(PageDevlog)
	if err != nil {
		t.Fatalf("renderDevlogIndex() error = %v", err)
	}
	return string(html)
}

func TestRenderDevlogIndex_BuildsDirectoriesAndArchive(t *testing.T) {
	conn := setupTestDB(t)
	grimID := seedGame(t, conn, "Grimoire", "grimoire", true, false, "")
	vaultID := seedGame(t, conn, "Vault Zero", "vault-zero", true, false, "")
	seedPostFor(t, conn, "First", "first", "2026-05-01", true, &grimID)
	seedPostFor(t, conn, "Vault Log", "vault-log", "2026-06-01", true, &vaultID)
	seedPostFor(t, conn, "Second", "second", "2025-05-01", true, nil)
	seedPostFor(t, conn, "Third", "third", "2025-04-01", true, &grimID)

	html := renderDevlogIndexPage(t, newTestSite(t, conn))

	for _, want := range []string{
		`data-filter-game=grimoire`, "Grimoire", "[2]",
		`data-filter-game=vault-zero`, "Vault Zero", "[1]",
		`data-filter-year=2025`, `data-filter-year=2026`, "(2)",
		`data-game=grimoire`, `data-year=2025`,
		// The page tells the client how many posts there are and how many it
		// drew, so load-more knows whether a second page exists.
		`data-devlog-feed`, `data-total=4`, `data-per-page=10`, `data-devlog-more`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("devlog index is missing %q", want)
		}
	}
	if !strings.Contains(html, "devlog-filter") {
		t.Error("the filter script is not on the page")
	}
}

func TestRenderDevlogIndex_NoSidebarWithoutTwoBuckets(t *testing.T) {
	conn := setupTestDB(t)
	seedPostFor(t, conn, "Only One", "only-one", "2026-05-01", true, nil)

	html := renderDevlogIndexPage(t, newTestSite(t, conn))

	if strings.Contains(html, "Directories") {
		t.Error("directories rendered with a single unfiled post")
	}
}
