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
	Title string `xml:"title"`
	// encoding/xml matches a namespace-less field against an element in any
	// namespace, so the Atom self link lands here too, as an empty entry.
	Links         []string         `xml:"link"`
	Description   string           `xml:"description"`
	Language      string           `xml:"language"`
	LastBuildDate string           `xml:"lastBuildDate"`
	Items         []decodedRSSItem `xml:"item"`
}

// channelLink is the channel's own address: the first link that says anything.
func (c decodedRSSChannel) channelLink() string {
	for _, link := range c.Links {
		if link != "" {
			return link
		}
	}
	return ""
}

// decodedRSSSelfLink cannot live in decodedRSSChannel: encoding/xml matches a
// namespace-less field against an element in any namespace, so a plain `link`
// field and the Atom `link` field in one struct corrupt each other's decode.
// The channel is read a second time through decodedRSSSelf, where the Atom
// link is the only `link` in sight.
type decodedRSSSelfLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type decodedRSSSelf struct {
	Channel struct {
		SelfLink *decodedRSSSelfLink `xml:"http://www.w3.org/2005/Atom link"`
	} `xml:"channel"`
}

type decodedRSSItem struct {
	Title       string          `xml:"title"`
	Link        string          `xml:"link"`
	GUID        *decodedRSSGUID `xml:"guid"`
	PubDate     string          `xml:"pubDate"`
	Description string          `xml:"description"`
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

func renderRSSSelfForTest(t *testing.T, site *Site) *decodedRSSSelfLink {
	t.Helper()
	body, _, err := site.renderRSS(PageRSS)
	if err != nil {
		t.Fatalf("renderRSS() error = %v", err)
	}
	var document decodedRSSSelf
	if err := xml.Unmarshal(body, &document); err != nil {
		t.Fatalf("RSS is invalid XML: %v\n%s", err, body)
	}
	return document.Channel.SelfLink
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

func TestRenderSitemap_WithoutSiteURLOmitsCanonicalLocations(t *testing.T) {
	conn := setupTestDB(t)
	seedGame(t, conn, "Public Game", "public-game", true, false, "")
	seedPost(t, conn, "Public Post", "public-post", "Body", "2026-06-02", true, nil)

	document := renderSitemapForTest(t, newTestSite(t, conn))
	if len(document.URLs) != 0 {
		t.Errorf("sitemap invented locations without site_url: %+v", document.URLs)
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
	if feed.Channel.Title != "Pixabros Devlog" || feed.Channel.channelLink() != "https://pixabros.com/devlog" || feed.Channel.Description != "Two brothers making independent games." {
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
		if item.GUID == nil || item.GUID.IsPermaLink != "true" || item.GUID.Value != item.Link {
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

// The channel says what language it is written in and when its content last
// changed, and points readers at its own address -- the bits feed validators
// ask for that the items alone do not carry.
func TestRenderRSS_StatesItsLanguageLatestChangeAndAddress(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_url":  "https://pixabros.com",
		"site_name": "Pixabros",
	})
	seedPost(t, conn, "Older Note", "older", "Older body", "2026-04-01", true, nil)
	seedPost(t, conn, "Newer Note", "newer", "Newer body", "2026-05-01", true, nil)
	setDiscoveryTimes(t, conn, "devlog_posts", "older", "2026-04-01T08:00:00.000Z", "2026-04-10T09:00:00.000Z")
	setDiscoveryTimes(t, conn, "devlog_posts", "newer", "2026-05-01T08:00:00.000Z", "2026-05-10T10:00:00.000Z")

	site := newTestSite(t, conn)
	feed := renderRSSForTest(t, site)
	if feed.Channel.Language != "en" {
		t.Errorf("channel language = %q, want en", feed.Channel.Language)
	}
	if want := "Sun, 10 May 2026 10:00:00 +0000"; feed.Channel.LastBuildDate != want {
		t.Errorf("lastBuildDate = %q, want %q", feed.Channel.LastBuildDate, want)
	}
	self := renderRSSSelfForTest(t, site)
	if self == nil || self.Rel != "self" || self.Href != "https://pixabros.com/rss.xml" {
		t.Errorf("atom self link = %+v", self)
	}
}

// Without a site address there is no truthful address to point the channel at,
// so the self link is left out entirely.
func TestRenderRSS_WithoutSiteURLOmitsTheSelfLink(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedPost(t, conn, "A Note", "a-note", "Body", "2026-05-01", true, nil)

	if self := renderRSSSelfForTest(t, newTestSite(t, conn)); self != nil {
		t.Errorf("self link = %+v, want none without a site address", self)
	}
}

func TestRenderRSS_SanitizesConfiguredChannelDescription(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_url":  "https://pixabros.com",
		"site_name": "Pixabros",
		"org_description": "A **bold** [studio](https://markdown.example) with " +
			"<em>HTML</em>, bare https://bare.example and <https://auto.example>.",
	})

	description := renderRSSForTest(t, newTestSite(t, conn)).Channel.Description
	if !strings.Contains(description, "A bold studio with HTML") {
		t.Errorf("channel description lost its readable text: %q", description)
	}
	for _, unsafe := range []string{"**", "](https", "<em>", "</em>", "https://"} {
		if strings.Contains(description, unsafe) {
			t.Errorf("channel description retained %q: %q", unsafe, description)
		}
	}
}

func TestRenderRSS_WithoutSiteURLOmitsCanonicalLinksAndGUIDs(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{"site_name": "Pixabros"})
	seedPost(t, conn, "Public Post", "public-post", "Body", "2026-06-02", true, nil)

	body, _, err := newTestSite(t, conn).renderRSS(PageRSS)
	if err != nil {
		t.Fatalf("renderRSS() error = %v", err)
	}
	for _, element := range []string{"<link>", "<guid"} {
		if strings.Contains(string(body), element) {
			t.Errorf("RSS retained %s without site_url:\n%s", element, body)
		}
	}
	var feed decodedRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("RSS is invalid XML: %v\n%s", err, body)
	}
	if got := feed.Channel.channelLink(); got != "" {
		t.Errorf("channel link = %q, want empty without site_url", got)
	}
	if len(feed.Channel.Items) != 1 {
		t.Fatalf("items = %+v, want one published post", feed.Channel.Items)
	}
	item := feed.Channel.Items[0]
	if item.Link != "" {
		t.Errorf("item link = %q, want empty without site_url", item.Link)
	}
	if item.GUID != nil {
		t.Errorf("item GUID = %+v, want omitted without site_url", item.GUID)
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

	blankOriginConn := setupTestDB(t)
	seedSiteSettings(t, blankOriginConn, map[string]string{
		"org_description": "A [studio](https://markdown.example), " +
			"https://bare.example, and <https://auto.example>.",
	})
	blankOriginBody, _, err := newTestSite(t, blankOriginConn).renderLLMS(PageLLMS)
	if err != nil {
		t.Fatal(err)
	}
	blankOriginText := string(blankOriginBody)
	if !strings.Contains(blankOriginText, "A studio") {
		t.Errorf("llms.txt lost the readable description:\n%s", blankOriginBody)
	}
	for _, unwanted := range []string{"http://", "https://", "](http", "<http"} {
		if strings.Contains(blankOriginText, unwanted) {
			t.Errorf("llms.txt retained %q without site_url:\n%s", unwanted, blankOriginBody)
		}
	}
}

// robots.txt and the X-Robots-Tag header are two documents stating one policy,
// so both are written from crawlerExclusions: everything on the list is
// disallowed here, and nothing off it is.
func TestRobotsTxt_DisallowsExactlyTheCrawlerExclusions(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)

	body, _, err := site.renderRobots(PageRobots)
	if err != nil {
		t.Fatalf("renderRobots() error = %v", err)
	}

	var disallowed []string
	for _, line := range strings.Split(string(body), "\n") {
		if path, ok := strings.CutPrefix(line, "Disallow: "); ok {
			disallowed = append(disallowed, path)
		}
	}
	if want := CrawlerExclusions(); !reflect.DeepEqual(disallowed, want) {
		t.Errorf("robots.txt disallows %v, want exactly %v", disallowed, want)
	}
}

// An address that would end a Markdown link destination early is
// percent-escaped, so it stays one link instead of breaking out of its own
// brackets.
func TestLLMS_EscapesLinkTargetsThatWouldEndTheLinkEarly(t *testing.T) {
	conn := setupTestDB(t)
	seedSiteSettings(t, conn, map[string]string{
		"site_url":  "https://pixabros.com/a (b)",
		"site_name": "Pixabros",
	})

	body, _, err := newTestSite(t, conn).renderLLMS(PageLLMS)
	if err != nil {
		t.Fatalf("renderLLMS() error = %v", err)
	}
	if !strings.Contains(string(body), "](https://pixabros.com/a%20%28b%29/games)") {
		t.Errorf("llms.txt did not escape the configured address:\n%s", body)
	}
}

// The truncation is what nearly every feed item goes through, so its
// boundaries are spelled out: the cut is rune-safe, backs up to a whole word
// when one is close enough, and never lands mid-character.
func TestPlainTextSummary_TruncatesOnRuneBoundaries(t *testing.T) {
	tests := []struct {
		name string
		body string
		max  int
		want string
	}{
		{
			name: "shorter than the limit is untouched",
			body: "A short body.",
			max:  rssSummaryMaxRunes,
			want: "A short body.",
		},
		{
			name: "exactly at the limit is untouched",
			body: strings.Repeat("a", rssSummaryMaxRunes),
			max:  rssSummaryMaxRunes,
			want: strings.Repeat("a", rssSummaryMaxRunes),
		},
		{
			name: "one rune over is cut to the limit with an ellipsis",
			body: strings.Repeat("a", rssSummaryMaxRunes+1),
			max:  rssSummaryMaxRunes,
			want: strings.Repeat("a", rssSummaryMaxRunes-1) + "…",
		},
		{
			name: "the cut backs up to a whole word",
			body: "one two threefourfive",
			max:  10,
			want: "one two…",
		},
		{
			name: "a space in the far half is not worth backing up to",
			body: "a bbbbbbbbbbbbbb",
			max:  10,
			want: "a bbbbbbb…",
		},
		{
			name: "multibyte runes are counted, not bytes",
			body: "一二三四五六七八",
			max:  5,
			want: "一二三四…",
		},
		{
			name: "no limit means no cut",
			body: strings.Repeat("a", rssSummaryMaxRunes*2),
			max:  0,
			want: strings.Repeat("a", rssSummaryMaxRunes*2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plainTextSummary(tt.body, tt.max); got != tt.want {
				t.Errorf("plainTextSummary cut wrong:\ngot  %q\nwant %q", got, tt.want)
			}
		})
	}
}
