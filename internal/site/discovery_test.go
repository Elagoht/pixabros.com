package site

import (
	"database/sql"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
)

type decodedSitemap struct {
	XMLName xml.Name         `xml:"urlset"`
	URLs    []decodedSiteURL `xml:"url"`
}

type decodedSiteURL struct {
	Location string `xml:"loc"`
	LastMod  string `xml:"lastmod"`
}

type decodedRSS struct {
	XMLName xml.Name          `xml:"rss"`
	Version string            `xml:"version,attr"`
	Channel decodedRSSChannel `xml:"channel"`
}

type decodedRSSChannel struct {
	Title       string           `xml:"title"`
	Link        string           `xml:"link"`
	Description string           `xml:"description"`
	Items       []decodedRSSItem `xml:"item"`
}

type decodedRSSItem struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	GUID        decodedRSSGUID `xml:"guid"`
	PubDate     string         `xml:"pubDate"`
	Description string         `xml:"description"`
}

type decodedRSSGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

func renderSitemapForTest(t *testing.T, site *Site) decodedSitemap {
	t.Helper()
	body, _, err := site.renderSitemap(PageSitemap)
	if err != nil {
		t.Fatalf("renderSitemap() error = %v", err)
	}
	var document decodedSitemap
	if err := xml.Unmarshal(body, &document); err != nil {
		t.Fatalf("sitemap is invalid XML: %v\n%s", err, body)
	}
	return document
}

func renderRSSForTest(t *testing.T, site *Site) decodedRSS {
	t.Helper()
	body, _, err := site.renderRSS(PageRSS)
	if err != nil {
		t.Fatalf("renderRSS() error = %v", err)
	}
	var feed decodedRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("RSS is invalid XML: %v\n%s", err, body)
	}
	return feed
}

func setDiscoveryTimes(t *testing.T, conn *sql.DB, table, slug, createdAt, updatedAt string) {
	t.Helper()
	if _, err := conn.Exec(
		"UPDATE "+table+" SET created_at = ?, updated_at = ? WHERE slug = ?",
		createdAt, updatedAt, slug,
	); err != nil {
		t.Fatalf("set %s timestamps for %q: %v", table, slug, err)
	}
}

func TestRenderSitemap_ContainsOnlyCanonicalPublishedPages(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_url": "https://pixabros.com"})
	seedGame(t, conn, "Public Game", "public-game", true, false, "")
	seedGame(t, conn, "Draft Game", "draft-game", false, false, "")
	seedPost(t, conn, "Public Post", "public-post", "Body", "2026-06-02", true, nil)
	seedPost(t, conn, "Draft Post", "draft-post", "Body", "2026-06-03", false, nil)

	document := renderSitemapForTest(t, newTestSite(t, conn))
	locations := make([]string, 0, len(document.URLs))
	for _, entry := range document.URLs {
		locations = append(locations, entry.Location)
	}
	want := []string{
		"https://pixabros.com/",
		"https://pixabros.com/games",
		"https://pixabros.com/devlog",
		"https://pixabros.com/awards",
		"https://pixabros.com/contact",
		"https://pixabros.com/games/public-game",
		"https://pixabros.com/devlog/public-post",
	}
	if !reflect.DeepEqual(locations, want) {
		t.Errorf("locations = %v, want %v", locations, want)
	}
}

func TestRenderSitemap_IsValidEscapedXMLWithStableOrder(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_url": "https://pixabros.com"})
	seedGame(t, conn, "Second < Game", "second<game", true, false, "")
	seedGame(t, conn, "First & Game", "first&game", true, false, "")
	if _, err := conn.Exec(`UPDATE games SET display_order = CASE slug WHEN 'first&game' THEN 1 ELSE 2 END`); err != nil {
		t.Fatal(err)
	}
	setDiscoveryTimes(t, conn, "games", "first&game", "2026-01-01T00:00:00.000Z", "2026-08-20T11:00:00.000Z")
	setDiscoveryTimes(t, conn, "games", "second<game", "2026-01-01T00:00:00.000Z", "2026-08-21T11:00:00.000Z")
	seedPost(t, conn, "Older & Post", "older&post", "Body", "2026-04-01", true, nil)
	seedPost(t, conn, "Newer < Post", "newer<post", "Body", "2026-05-01", true, nil)
	setDiscoveryTimes(t, conn, "devlog_posts", "older&post", "2026-04-01T00:00:00.000Z", "2026-08-22T11:00:00.000Z")
	setDiscoveryTimes(t, conn, "devlog_posts", "newer<post", "2026-05-01T00:00:00.000Z", "2026-08-23T11:00:00.000Z")

	site := newTestSite(t, conn)
	first := renderSitemapForTest(t, site)
	second := renderSitemapForTest(t, site)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("two renders from unchanged data differ:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if len(first.URLs) != 9 {
		t.Fatalf("sitemap entries = %+v, want five static and four dynamic entries", first.URLs)
	}

	gotDynamic := first.URLs[5:]
	wantDynamic := []decodedSiteURL{
		{Location: "https://pixabros.com/games/first&game", LastMod: "2026-08-20"},
		{Location: "https://pixabros.com/games/second<game", LastMod: "2026-08-21"},
		{Location: "https://pixabros.com/devlog/newer<post", LastMod: "2026-08-23"},
		{Location: "https://pixabros.com/devlog/older&post", LastMod: "2026-08-22"},
	}
	if !reflect.DeepEqual(gotDynamic, wantDynamic) {
		t.Errorf("dynamic entries = %+v, want %+v", gotDynamic, wantDynamic)
	}
}

func TestRenderRSS_ContainsOnlyPublishedPostsNewestFirst(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_url":        "https://pixabros.com",
		"site_name":       "Pixabros",
		"org_description": "Two brothers making independent games.",
	})
	seedPost(t, conn, "Older Note", "older", "Older body", "2026-04-01", true, nil)
	seedPost(t, conn, "Newer Note", "newer", "Newer body", "2026-05-01", true, nil)
	seedPost(t, conn, "Secret Draft", "secret", "Secret body", "2026-06-01", false, nil)

	feed := renderRSSForTest(t, newTestSite(t, conn))
	if feed.Version != "2.0" {
		t.Errorf("RSS version = %q, want 2.0", feed.Version)
	}
	if feed.Channel.Title != "Pixabros Devlog" || feed.Channel.Link != "https://pixabros.com/devlog" || feed.Channel.Description != "Two brothers making independent games." {
		t.Errorf("channel = %+v", feed.Channel)
	}
	if len(feed.Channel.Items) != 2 {
		t.Fatalf("items = %+v, want exactly two published posts", feed.Channel.Items)
	}
	if feed.Channel.Items[0].Title != "Newer Note" || feed.Channel.Items[1].Title != "Older Note" {
		t.Errorf("items are not newest first: %+v", feed.Channel.Items)
	}
	for _, item := range feed.Channel.Items {
		if strings.Contains(item.Title, "Secret") || strings.Contains(item.Description, "Secret") {
			t.Error("an unpublished post leaked into RSS")
		}
	}
}

func TestRenderRSS_ProducesValidXMLAndSafePlainTextDescriptions(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_url": "https://pixabros.com", "site_name": "Pixabros"})
	seedPost(t, conn, "Tools & <Tactics>", "tools-and-tactics",
		"## A [linked summary](https://example.com) with **bold** text <b>and HTML</b>.",
		"2026-05-02", true, nil)
	seedPost(t, conn, "Fallback Date", "fallback-date", "Plain body", "", true, nil)
	setDiscoveryTimes(t, conn, "devlog_posts", "fallback-date", "2026-04-03T14:15:16.000Z", "2026-04-03T14:15:16.000Z")

	feed := renderRSSForTest(t, newTestSite(t, conn))
	if len(feed.Channel.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(feed.Channel.Items))
	}
	for _, item := range feed.Channel.Items {
		if item.GUID.IsPermaLink != "true" || item.GUID.Value != item.Link {
			t.Errorf("GUID = %+v, link = %q", item.GUID, item.Link)
		}
		if _, err := time.Parse(time.RFC1123Z, item.PubDate); err != nil {
			t.Errorf("pubDate %q is not RFC1123Z: %v", item.PubDate, err)
		}
		if strings.ContainsAny(item.Description, "<>") || strings.Contains(item.Description, "](") {
			t.Errorf("description is not safe plain text: %q", item.Description)
		}
	}
	if feed.Channel.Items[0].Title != "Tools & <Tactics>" {
		t.Errorf("escaped title decoded as %q", feed.Channel.Items[0].Title)
	}
	if got, want := feed.Channel.Items[0].Description, "A linked summary with bold text and HTML."; got != want {
		t.Errorf("plain-text description = %q, want %q", got, want)
	}
	fallbackDate := feed.Channel.Items[1].PubDate
	if want := "Fri, 03 Apr 2026 14:15:16 +0000"; fallbackDate != want {
		t.Errorf("fallback pubDate = %q, want %q", fallbackDate, want)
	}
}

func TestRenderRobots_AllowsPublicCrawlingAndNamesSitemap(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_url": "https://pixabros.com/"})
	body, _, err := newTestSite(t, conn).renderRobots(PageRobots)
	if err != nil {
		t.Fatal(err)
	}
	want := "User-agent: *\n" +
		"Allow: /\n" +
		"Disallow: /I-am-a-pixabro/\n" +
		"Disallow: /api/\n" +
		"Disallow: /play/\n" +
		"Disallow: /offline\n" +
		"Disallow: /contact/sent\n" +
		"Sitemap: https://pixabros.com/sitemap.xml\n"
	if string(body) != want {
		t.Errorf("robots.txt =\n%s\nwant:\n%s", body, want)
	}
}

func TestRenderRobots_WithoutSiteURLDoesNotInventSitemapOrigin(t *testing.T) {
	conn := setupTestDB(t)
	body, _, err := newTestSite(t, conn).renderRobots(PageRobots)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Sitemap:") || strings.Contains(string(body), "http") {
		t.Errorf("robots.txt invented an absolute origin:\n%s", body)
	}
}

func TestRenderLLMS_LinksPublishedContentAndFeedsOnly(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_url":        "https://pixabros.com",
		"site_name":       "Pixabros",
		"org_description": "Two brothers making independent games.",
	})
	seedGame(t, conn, "Public Game", "public-game", true, false, "")
	seedGame(t, conn, "Draft Game", "draft-game", false, false, "")
	seedPost(t, conn, "Public Post", "public-post", "Body", "2026-06-02", true, nil)
	seedPost(t, conn, "Draft Post", "draft-post", "Body", "2026-06-03", false, nil)

	body, _, err := newTestSite(t, conn).renderLLMS(PageLLMS)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"# Pixabros", "Two brothers making independent games.",
		"[Home](https://pixabros.com/)", "[Games](https://pixabros.com/games)",
		"[Devlog](https://pixabros.com/devlog)", "[Awards](https://pixabros.com/awards)",
		"[Contact](https://pixabros.com/contact)", "[Public Game](https://pixabros.com/games/public-game)",
		"[Public Post](https://pixabros.com/devlog/public-post)",
		"[Sitemap](https://pixabros.com/sitemap.xml)", "[RSS feed](https://pixabros.com/rss.xml)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("llms.txt missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"Draft Game", "draft-game", "Draft Post", "draft-post"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("llms.txt leaked %q", unwanted)
		}
	}

	blankOriginBody, _, err := newTestSite(t, setupTestDB(t)).renderLLMS(PageLLMS)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blankOriginBody), "](http") {
		t.Errorf("llms.txt invented an absolute origin:\n%s", blankOriginBody)
	}
}
