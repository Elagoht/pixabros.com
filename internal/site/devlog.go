package site

import (
	"bytes"
	"fmt"
	"html/template"
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
	Image    imageView
}

type devlogIndexPage struct {
	Posts []postSummary
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
// about; renaming a game has to refresh this page too.
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
			Image: lookupImage(images, post.OGImageID, post.Title),
		}
		if post.GameID != nil {
			summary.GameName = gameNames[*post.GameID].Title
		}
		summaries = append(summaries, summary)
	}

	html, err := s.renderer.render("devlog.html", pageData{
		Title:       "Devlog — " + chrome.Name,
		Description: "Notes on what we are building at " + chrome.Name + ".",
		Path:        "/" + PageDevlog,
		Site:        chrome,
		Data:        devlogIndexPage{Posts: summaries},
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
		Title:       post.Title + " — " + chrome.Name,
		Description: excerpt(post.ContentMarkdown, 160),
		Path:        "/" + PageDevlog,
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
	if err := markdown.Convert([]byte(source), &out); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(out.String()), nil
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
