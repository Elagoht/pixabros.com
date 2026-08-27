package site

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	PageRobots  = "robots.txt"
	PageLLMS    = "llms.txt"
	PageSitemap = "sitemap.xml"
	PageRSS     = "rss.xml"

	// AdminUIPrefix is where the admin panel is mounted. It lives here rather
	// than with the routes so the crawler policy below can name it, and the
	// panel's own routes can quote it from there.
	AdminUIPrefix = "/I-am-a-pixabro/"
)

// crawlerExclusions are the route families the crawler policy keeps out of the
// index. One list feeds both documents that express it -- robots.txt disallows
// each entry, and the HTTP layer answers the same paths with noindex -- so the
// two cannot drift apart. A path disallowed in robots.txt but served without
// noindex reads as a secret and is indexed anyway.
//
// An entry ending in "/" names a family and matches its bare form too; one
// without names a single page.
var crawlerExclusions = []string{
	AdminUIPrefix,
	"/api/",
	"/play/",
	"/" + PageOffline,
	"/" + PageContactSent,
}

// CrawlerExclusions returns the paths the crawler policy keeps out of the
// index. The HTTP layer walks the same list robots.txt was written from.
func CrawlerExclusions() []string {
	return append([]string(nil), crawlerExclusions...)
}

const rssSummaryMaxRunes = 300

// The link and URL patterns are deliberately crude, and two authored forms
// pass through imperfectly: a destination containing parentheses
// (`https://en.wikipedia.org/wiki/A_(b)`) leaves its closing paren behind,
// and an autolink (`<https://example.com>`) is swallowed whole by the tag
// stripper along with its brackets. Both degrade to cosmetic noise in a
// summary, never to live markup, so a real Markdown parser is not worth its
// weight here.
var (
	markdownLink                = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	bareAbsoluteURL             = regexp.MustCompile(`https?://[^\s<>()]+`)
	whitespaceBeforePunctuation = regexp.MustCompile(`\s+([.,;:!?])`)
)

func (s *Site) discoveryPages() []pageDef {
	return []pageDef{
		{Key: PageRobots, Render: s.renderRobots},
		{Key: PageLLMS, Render: s.renderLLMS},
		{Key: PageSitemap, Render: s.renderSitemap},
		{Key: PageRSS, Render: s.renderRSS},
	}
}

func (s *Site) renderRobots(_ string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}

	var body strings.Builder
	lines := []string{"User-agent: *", "Allow: /"}
	for _, path := range crawlerExclusions {
		lines = append(lines, "Disallow: "+path)
	}
	for _, line := range lines {
		body.WriteString(line)
		body.WriteByte('\n')
	}
	if sitemap := canonicalURL(chrome.URL, PageSitemap); sitemap != "" {
		body.WriteString("Sitemap: ")
		body.WriteString(sitemap)
		body.WriteByte('\n')
	}
	return []byte(body.String()), []string{siteSettingsTag}, nil
}

type llmsLink struct {
	label string
	url   string
}

func (s *Site) renderLLMS(_ string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	games, err := s.publishedGames()
	if err != nil {
		return nil, nil, err
	}
	posts, err := s.publishedPosts()
	if err != nil {
		return nil, nil, err
	}

	var body strings.Builder
	body.WriteString("# ")
	body.WriteString(chrome.Name)
	body.WriteString("\n\n")
	if description := plainTextSummary(chrome.Description, 0); description != "" {
		body.WriteString(description)
		body.WriteString("\n\n")
	}

	writeLLMSLinks(&body, "Site", []llmsLink{
		{label: "Home", url: canonicalURL(chrome.URL, PageLanding)},
		{label: "Games", url: canonicalURL(chrome.URL, PageGames)},
		{label: "Devlog", url: canonicalURL(chrome.URL, PageDevlog)},
		{label: "Awards", url: canonicalURL(chrome.URL, PageAwards)},
		{label: "Contact", url: canonicalURL(chrome.URL, PageContact)},
	})

	gameLinks := make([]llmsLink, 0, len(games))
	for _, game := range games {
		gameLinks = append(gameLinks, llmsLink{
			label: game.Title,
			url:   canonicalURL(chrome.URL, GamePagePrefix+game.Slug),
		})
	}
	writeLLMSLinks(&body, "Games", gameLinks)

	postLinks := make([]llmsLink, 0, len(posts))
	for _, post := range posts {
		postLinks = append(postLinks, llmsLink{
			label: post.Title,
			url:   canonicalURL(chrome.URL, DevlogPagePrefix+post.Slug),
		})
	}
	writeLLMSLinks(&body, "Devlog posts", postLinks)

	writeLLMSLinks(&body, "Feeds", []llmsLink{
		{label: "Sitemap", url: canonicalURL(chrome.URL, PageSitemap)},
		{label: "RSS feed", url: canonicalURL(chrome.URL, PageRSS)},
	})

	return []byte(body.String()), []string{siteSettingsTag, gameListTag, devlogListTag}, nil
}

func writeLLMSLinks(body *strings.Builder, heading string, links []llmsLink) {
	wroteHeading := false
	for _, link := range links {
		if link.url == "" {
			continue
		}
		if !wroteHeading {
			body.WriteString("## ")
			body.WriteString(heading)
			body.WriteString("\n\n")
			wroteHeading = true
		}
		body.WriteString("- [")
		body.WriteString(markdownLabel(link.label))
		body.WriteString("](")
		body.WriteString(llmsLinkTarget(link.url))
		body.WriteString(")\n")
	}
	if wroteHeading {
		body.WriteByte('\n')
	}
}

// llmsLinkTarget percent-escapes the few characters that would end a Markdown
// link destination early. Real slugs are [a-z0-9-] and site_url is configured
// by hand, so this is hardening rather than a fix -- but a stray space or
// bracket in an address should not be able to break out of its own link.
func llmsLinkTarget(target string) string {
	return strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(target)
}

func markdownLabel(label string) string {
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`, "<", "&lt;", ">", "&gt;").Replace(label)
}

type sitemapDocument struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	URLs    []sitemapItem `xml:"url"`
}

type sitemapItem struct {
	Location string `xml:"loc"`
	LastMod  string `xml:"lastmod,omitempty"`
}

func (s *Site) renderSitemap(_ string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	games, err := s.publishedGames()
	if err != nil {
		return nil, nil, err
	}
	posts, err := s.publishedPosts()
	if err != nil {
		return nil, nil, err
	}

	document := sitemapDocument{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, key := range []string{PageLanding, PageGames, PageDevlog, PageAwards, PageContact} {
		if location := canonicalURL(chrome.URL, key); location != "" {
			document.URLs = append(document.URLs, sitemapItem{Location: location})
		}
	}
	for _, game := range games {
		if location := canonicalURL(chrome.URL, GamePagePrefix+game.Slug); location != "" {
			document.URLs = append(document.URLs, sitemapItem{
				Location: location,
				LastMod:  game.UpdatedAt.UTC().Format("2006-01-02"),
			})
		}
	}
	for _, post := range posts {
		if location := canonicalURL(chrome.URL, DevlogPagePrefix+post.Slug); location != "" {
			document.URLs = append(document.URLs, sitemapItem{
				Location: location,
				LastMod:  post.UpdatedAt.UTC().Format("2006-01-02"),
			})
		}
	}

	body, err := xml.Marshal(document)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal sitemap: %w", err)
	}
	return append([]byte(xml.Header), body...), []string{siteSettingsTag, gameListTag, devlogListTag}, nil
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr,omitempty"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link,omitempty"`
	Description string `xml:"description"`
	Language    string `xml:"language,omitempty"`
	// LastBuildDate is the most recent edit among the posts carried, not the
	// moment of rendering: the feed is regenerated whenever a dependency
	// changes and the store computes the ETag from these bytes, so a build
	// timestamp would churn the ETag on every regeneration rather than when
	// the content actually moved.
	LastBuildDate string       `xml:"lastBuildDate,omitempty"`
	SelfLink      *rssSelfLink `xml:"http://www.w3.org/2005/Atom link,omitempty"`
	Items         []rssItem    `xml:"item"`
}

// rssSelfLink is the channel's own address, as an Atom element. Readers use it
// to tell the feed's canonical home from a copy scraped somewhere else.
type rssSelfLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link,omitempty"`
	GUID        *rssGUID `xml:"guid,omitempty"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

func (s *Site) renderRSS(_ string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	posts, err := s.publishedPosts()
	if err != nil {
		return nil, nil, err
	}

	description := plainTextSummary(chrome.Description, 0)
	if description == "" {
		description = "Notes from the " + chrome.Name + " devlog."
	}
	channel := rssChannel{
		Title:       chrome.Name + " Devlog",
		Link:        canonicalURL(chrome.URL, PageDevlog),
		Description: description,
		// The site is English only, so there is no locale to consult.
		Language: "en",
	}
	var changed time.Time
	for _, post := range posts {
		link := canonicalURL(chrome.URL, DevlogPagePrefix+post.Slug)
		item := rssItem{
			Title:       post.Title,
			Link:        link,
			PubDate:     rssPublicationTime(post.PublishedAt, post.CreatedAt).Format(time.RFC1123Z),
			Description: plainTextSummary(post.ContentMarkdown, rssSummaryMaxRunes),
		}
		if link != "" {
			item.GUID = &rssGUID{IsPermaLink: "true", Value: link}
		}
		if changed.Before(post.UpdatedAt) {
			changed = post.UpdatedAt
		}
		channel.Items = append(channel.Items, item)
	}
	if !changed.IsZero() {
		channel.LastBuildDate = changed.UTC().Format(time.RFC1123Z)
	}
	if self := canonicalURL(chrome.URL, PageRSS); self != "" {
		channel.SelfLink = &rssSelfLink{Href: self, Rel: "self"}
	}

	body, err := xml.Marshal(rssDocument{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: channel,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal RSS: %w", err)
	}
	return append([]byte(xml.Header), body...), []string{siteSettingsTag, devlogListTag}, nil
}

func rssPublicationTime(publishedAt string, createdAt time.Time) time.Time {
	if publishedAt != "" {
		if parsed, err := time.Parse("2006-01-02", publishedAt); err == nil {
			return parsed.UTC()
		}
	}
	return createdAt.UTC()
}

// plainTextSummary reduces authored Markdown to compact plain text. It keeps
// Markdown link labels while removing their destinations, bare absolute URLs,
// remaining markup, and excess whitespace.
func plainTextSummary(markdown string, maxRunes int) string {
	withoutLinks := markdownLink.ReplaceAllString(markdown, "$1")
	withoutURLs := bareAbsoluteURL.ReplaceAllString(stripTags(withoutLinks), "")
	text := strings.Join(strings.Fields(stripMarkdown(withoutURLs)), " ")
	text = whitespaceBeforePunctuation.ReplaceAllString(text, "$1")
	if maxRunes < 1 || utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	trimmed := strings.TrimSpace(string(runes[:maxRunes-1]))
	if space := strings.LastIndex(trimmed, " "); space > maxRunes/2 {
		trimmed = strings.TrimSpace(trimmed[:space])
	}
	return trimmed + "…"
}
