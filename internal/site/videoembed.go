package site

import (
	"fmt"
	"html"
	"strings"

	"pixabros/internal/youtube"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmrender "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A post cannot write an iframe: goldmark drops raw HTML, which is the site's
// whole XSS defence. So a video goes in as a plain YouTube link on a line of
// its own, and this turns it into a player.
//
// The safety of it rests on one thing: the only part of the link that survives
// is the video id, and an id is eleven characters of [A-Za-z0-9_-]. Everything
// else about the URL -- host, path, query, fragment -- is read and thrown away.
// The host we frame is a constant in this file, so a post can choose which
// video plays and nothing else.

var kindVideoEmbed = ast.NewNodeKind("VideoEmbed")

type videoEmbed struct {
	ast.BaseBlock
	videoID string
	// title names the player for a screen reader. It comes from the link text
	// when there is one, so it is the one piece of author text in the tag and
	// the only reason this file escapes anything.
	title string
}

func (n *videoEmbed) Kind() ast.NodeKind { return kindVideoEmbed }

func (n *videoEmbed) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"VideoID": n.videoID, "Title": n.title}, nil)
}

// videoTransformer replaces a paragraph that is nothing but a video link.
//
// It reads the paragraph's own source rather than its children, so it does not
// matter whether the link arrived as bare text, an autolink, or markdown link
// syntax. A link with prose around it stays a link: turning a mid-sentence
// mention into a player would be a surprise, not a feature.
type videoTransformer struct{}

func (videoTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()

	for node := doc.FirstChild(); node != nil; {
		next := node.NextSibling()

		if para, ok := node.(*ast.Paragraph); ok {
			if embed, ok := videoIn(para, source); ok {
				doc.ReplaceChild(doc, node, embed)
			}
		}
		node = next
	}
}

// defaultVideoTitle names a player the author gave no words for.
const defaultVideoTitle = "Embedded video"

// videoIn reports the video a paragraph is, if a video is all it is.
//
// A bare URL is plain text here, because goldmark does not linkify by default.
// The other two shapes -- an autolink and markdown link syntax -- arrive as a
// single child node holding the destination, and link syntax also gives the
// player a name.
func videoIn(para *ast.Paragraph, source []byte) (*videoEmbed, bool) {
	if id, ok := youtube.ID(paragraphSource(para, source)); ok {
		return &videoEmbed{videoID: id, title: defaultVideoTitle}, true
	}

	child := para.FirstChild()
	if child == nil || child != para.LastChild() {
		return nil, false
	}

	switch link := child.(type) {
	case *ast.AutoLink:
		if id, ok := youtube.ID(string(link.URL(source))); ok {
			return &videoEmbed{videoID: id, title: defaultVideoTitle}, true
		}
	case *ast.Link:
		id, ok := youtube.ID(string(link.Destination))
		if !ok {
			return nil, false
		}
		title := strings.TrimSpace(string(link.Text(source)))
		if title == "" {
			title = defaultVideoTitle
		}
		return &videoEmbed{videoID: id, title: title}, true
	}
	return nil, false
}

func paragraphSource(para *ast.Paragraph, source []byte) string {
	var out strings.Builder
	for i := range para.Lines().Len() {
		segment := para.Lines().At(i)
		out.Write(segment.Value(source))
	}
	return strings.TrimSpace(out.String())
}

type videoRenderer struct{}

func (videoRenderer) RegisterFuncs(reg gmrender.NodeRendererFuncRegisterer) {
	reg.Register(kindVideoEmbed, renderVideoEmbed)
}

// renderVideoEmbed writes the player.
//
// nocookie is the privacy-preserving host, and the wrapper is what holds the
// 16:9 box: the width and height a YouTube share dialog gives you are fixed
// pixels that overflow a phone.
func renderVideoEmbed(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	embed := node.(*videoEmbed)
	fmt.Fprintf(w,
		`<div class="embed embed--video">`+
			`<iframe src="%s"`+
			` title="%s" loading="lazy"`+
			` referrerpolicy="strict-origin-when-cross-origin"`+
			` allow="clipboard-write; encrypted-media; picture-in-picture"`+
			` allowfullscreen></iframe>`+
			`</div>`,
		youtube.EmbedURL(embed.videoID), html.EscapeString(embed.title),
	)
	return ast.WalkSkipChildren, nil
}

// videoEmbeds is the goldmark extension that wires the two together.
type videoEmbeds struct{}

func (videoEmbeds) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(videoTransformer{}, 500),
	))
	md.Renderer().AddOptions(gmrender.WithNodeRenderers(
		util.Prioritized(videoRenderer{}, 500),
	))
}
