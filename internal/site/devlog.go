package site

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strings"

	"pixabros/internal/devlog"
)

const (
	PageDevlog       = "devlog"
	DevlogPagePrefix = "devlog/"
)

// devlogListTag and the per-post tag must match what devlogapi enqueues.
const devlogListTag = "devlog:list"

type postSummary struct {
	Title    string
	Slug     string
	Date     string
	GameName string
	// GameSlug and Year drive the sidebar's client-side filter; a post with
	// neither can only be found by scrolling, which is what it falls back to.
	GameSlug string
	Year     string
	Image    imageView
}

// directoryView is one game bucket in the devlog sidebar, for filtering the
// index client-side.
type directoryView struct {
	Title string
	Slug  string
	Count int
}

// archiveView is one year bucket in the devlog sidebar.
type archiveView struct {
	Year  string
	Count int
}

type devlogIndexPage struct {
	Featured    postSummary
	Rest        []postSummary
	Directories []directoryView
	Archive     []archiveView
}

type devlogPostPage struct {
	Title    string
	Date     string
	GameName string
	GameSlug string
	Image    imageView
	Body     template.HTML
}

// renderDevlogIndex builds /devlog.
//
// game:list is in the tag set because each row can name the game a post is
// about; renaming a game has to refresh this page too. The sidebar's filters
// are purely client-side over the statically rendered rows, so they add no
// tags of their own.
func (s *Site) renderDevlogIndex(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	images, err := s.mediaByID()
	if err != nil {
		return nil, nil, err
	}
	gameNames, err := s.gameNames()
	if err != nil {
		return nil, nil, err
	}

	posts, err := s.publishedPosts()
	if err != nil {
		return nil, nil, err
	}

	summaries := make([]postSummary, 0, len(posts))
	for _, post := range posts {
		summary := postSummary{
			Title: post.Title,
			Slug:  post.Slug,
			Date:  post.PublishedAt,
			Year:  yearOf(post.PublishedAt),
			Image: lookupImage(images, post.OGImageID, post.Title),
		}
		if post.GameID != nil {
			summary.GameName = gameNames[*post.GameID].Title
			summary.GameSlug = gameNames[*post.GameID].Slug
		}
		summaries = append(summaries, summary)
	}

	directories := make([]directoryView, 0)
	dirCounts := map[string]int{}
	dirTitles := map[string]string{}
	archive := make([]archiveView, 0)
	yearCounts := map[string]int{}
	for _, summary := range summaries {
		if summary.GameSlug != "" {
			if _, seen := dirTitles[summary.GameSlug]; !seen {
				directories = append(directories, directoryView{Title: summary.GameName, Slug: summary.GameSlug})
				dirTitles[summary.GameSlug] = summary.GameName
			}
			dirCounts[summary.GameSlug]++
		}
		if summary.Year != "" && yearCounts[summary.Year] == 0 {
			archive = append(archive, archiveView{Year: summary.Year})
		}
		if summary.Year != "" {
			yearCounts[summary.Year]++
		}
	}
	for i := range directories {
		directories[i].Count = dirCounts[directories[i].Slug]
	}
	for i := range archive {
		archive[i].Count = yearCounts[archive[i].Year]
	}

	var featured postSummary
	var rest []postSummary
	if len(summaries) > 0 {
		featured = summaries[0]
		rest = summaries[1:]
	}
	// A sidebar with one bucket is a list calling itself a filter.
	if len(directories) < 2 {
		directories = nil
	}
	if len(archive) < 2 {
		archive = nil
	}

	html, err := s.renderer.render("devlog.html", pageData{
		Title:       "Devlog: how our games get made, one post at a time",
		Description: "Notes on what we are building at " + chrome.Name + ".",
		Path:        "/" + PageDevlog,
		Canonical:   canonicalURL(chrome.URL, PageDevlog),
		Schema:      s.devlogIndexSchema(chrome, summaries),
		Site:        chrome,
		Data: devlogIndexPage{
			Featured:    featured,
			Rest:        rest,
			Directories: directories,
			Archive:     archive,
		},
		Scripts: []string{s.renderer.bundle.URL("devlog-filter.js")},
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{devlogListTag, gameListTag, siteSettingsTag}, nil
}

// renderDevlogPost builds /devlog/{slug}.
func (s *Site) renderDevlogPost(pageKey string) ([]byte, []string, error) {
	slug := strings.TrimPrefix(pageKey, DevlogPagePrefix)
	if slug == "" || strings.Contains(slug, "/") {
		return nil, nil, fmt.Errorf("not a devlog page key: %q", pageKey)
	}

	post, err := s.devlog.FindBySlug(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("find post %q: %w", slug, err)
	}
	if !post.IsPublished {
		return nil, nil, fmt.Errorf("post %q is not published", slug)
	}

	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	images, err := s.mediaByID()
	if err != nil {
		return nil, nil, err
	}

	body, err := renderMarkdown(post.ContentMarkdown)
	if err != nil {
		return nil, nil, err
	}

	page := devlogPostPage{
		Title: post.Title,
		Date:  post.PublishedAt,
		Image: lookupImage(images, post.OGImageID, post.Title),
		Body:  body,
	}

	tags := []string{"devlog:" + post.ID, siteSettingsTag}
	if post.GameID != nil {
		gameNames, err := s.gameNames()
		if err != nil {
			return nil, nil, err
		}
		if game, ok := gameNames[*post.GameID]; ok && game.Published {
			page.GameName = game.Title
			page.GameSlug = game.Slug
			// Only depend on the game list when a game is actually named.
			tags = append(tags, gameListTag)
		}
	}

	html, err := s.renderer.render("devlog-post.html", pageData{
		Title:       postSubject(post.Title, chrome.Name),
		Description: excerpt(post.ContentMarkdown, 160),
		Path:        "/" + PageDevlog,
		Canonical:   canonicalURL(chrome.URL, pageKey),
		OGType:      "article",
		OGImage:     page.Image.URL,
		Schema:      s.devlogPostSchema(chrome, post, page),
		Site:        chrome,
		Data:        page,
	})
	if err != nil {
		return nil, nil, err
	}
	return html, tags, nil
}

// publishedPosts returns the public view of the devlog, newest first. Posts
// with no publication date sort last rather than being dropped: the date is
// only set the first time a post is published.
func (s *Site) publishedPosts() ([]devlog.Post, error) {
	all, err := s.devlog.List("published_at", true)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	posts := make([]devlog.Post, 0, len(all))
	for _, post := range all {
		if post.IsPublished {
			posts = append(posts, post)
		}
	}
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].PublishedAt > posts[j].PublishedAt
	})
	return posts, nil
}

type gameRef struct {
	Title     string
	Slug      string
	Published bool
}

// gameNames resolves the game a post is about, in one query rather than one
// per post.
func (s *Site) gameNames() (map[string]gameRef, error) {
	all, err := s.games.List("display_order", false)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	byID := make(map[string]gameRef, len(all))
	for _, game := range all {
		byID[game.ID] = gameRef{Title: game.Title, Slug: game.Slug, Published: game.IsPublished}
	}
	return byID, nil
}

// renderMarkdown converts a post body to HTML.
//
// goldmark is configured without WithUnsafe(), so raw HTML in the source is
// dropped rather than passed through. That is the whole XSS defence and the
// reason no separate sanitiser is needed: a post cannot emit a tag the
// renderer did not produce itself.
func renderMarkdown(source string) (template.HTML, error) {
	var out bytes.Buffer
	if err := markdown.Convert([]byte(normaliseBlockMarkers(source)), &out); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(out.String()), nil
}

// blockMarkerNBSP matches a no-break space directly after a line's block
// marker -- a heading's #, a list's bullet, a blockquote's >.
var blockMarkerNBSP = regexp.MustCompile(`(?m)^([ \t]*(?:#{1,6}|[-*+]|[0-9]+[.)]|>))\x{00a0}`)

// normaliseBlockMarkers turns a no-break space after a block marker into an
// ordinary one.
//
// CommonMark requires a space or tab after "##" for it to be a heading, and a
// no-break space is neither -- so "## Key Features" renders as literal
// text rather than a heading. Text pasted from a word processor is full of
// them, and nobody types one on purpose right after a "#".
//
// The replacement is deliberately confined to that position. A no-break space
// inside a sentence is usually there for a reason ("10 km", "Fig. 4"), so the
// prose is left exactly as written.
func normaliseBlockMarkers(source string) string {
	return blockMarkerNBSP.ReplaceAllString(source, "$1 ")
}

// excerpt makes a plain-text meta description out of a markdown body.
//
// Tags are removed before punctuation is, because stripping ">" first would
// tear "</script>" into fragments and leave "<script" sitting in a meta
// description -- harmless, since html/template escapes it, but it reads like
// an injection attempt and is not prose.
func excerpt(source string, limit int) string {
	text := strings.Join(strings.Fields(stripMarkdown(stripTags(source))), " ")
	if len(text) <= limit {
		return text
	}
	trimmed := text[:limit]
	if idx := strings.LastIndex(trimmed, " "); idx > 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed + "…"
}

// stripTags removes anything between angle brackets. A description is prose;
// markup in it is never wanted, whoever wrote it.
func stripTags(source string) string {
	var out strings.Builder
	depth := 0
	for _, r := range source {
		switch {
		case r == '<':
			depth++
		case r == '>' && depth > 0:
			depth--
			out.WriteRune(' ')
		case depth == 0:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// stripMarkdown removes the punctuation that would read badly in a search
// result. It is deliberately crude -- this is a description, not a document.
func stripMarkdown(source string) string {
	replacer := strings.NewReplacer(
		"#", "", "*", "", "_", "", "`", "", ">", "", "[", "", "]", "",
	)
	return replacer.Replace(source)
}
