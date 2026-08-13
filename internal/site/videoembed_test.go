package site

import (
	"strings"
	"testing"
)

func TestRenderMarkdown_TurnsALoneVideoLinkIntoAPlayer(t *testing.T) {
	html, err := renderMarkdown("Watch it.\n\nhttps://youtu.be/9mjjowHX1-g?si=abc\n\nThe end.\n")
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{
		`class="embed embed--video"`,
		`src="https://www.youtube-nocookie.com/embed/9mjjowHX1-g"`,
		`allowfullscreen`,
		`loading="lazy"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the player is missing %q\ngot: %s", want, body)
		}
	}
	// The share tracking parameter is dropped along with the rest of the URL.
	if strings.Contains(body, "si=abc") {
		t.Error("the link's query string reached the page")
	}
	if !strings.Contains(body, "Watch it.") || !strings.Contains(body, "The end.") {
		t.Error("the prose around the video was lost")
	}
}

// A video link mentioned inside a sentence stays text. Only a paragraph that is
// nothing but the link becomes a player.
func TestRenderMarkdown_LeavesAVideoLinkInsideASentenceAlone(t *testing.T) {
	html, err := renderMarkdown("See https://youtu.be/9mjjowHX1-g for the trailer.\n")
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	if strings.Contains(string(html), "<iframe") {
		t.Errorf("a mid-sentence link became a player: %s", html)
	}
}

// Markdown link syntax is what an editor produces when you paste a URL, so it
// has to embed as readily as the bare link does.
func TestRenderMarkdown_EmbedsAMarkdownLinkOnItsOwnLine(t *testing.T) {
	html, err := renderMarkdown("[Gameplay](https://www.youtube.com/watch?v=9mjjowHX1-g)\n")
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	body := string(html)
	if !strings.Contains(body, `embed/9mjjowHX1-g`) {
		t.Errorf("a markdown link on its own line did not embed: %s", body)
	}
	// The link text names the player, which is what a screen reader reads out.
	if !strings.Contains(body, `title="Gameplay"`) {
		t.Errorf("the link text did not become the player's name: %s", body)
	}
}

// The link text is the one piece of author writing that reaches the tag, so it
// must not be able to close the attribute it sits in.
func TestRenderMarkdown_EscapesThePlayersName(t *testing.T) {
	html, err := renderMarkdown(
		`["><script>alert(1)</script>](https://youtu.be/9mjjowHX1-g)` + "\n",
	)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	body := string(html)
	if !strings.Contains(body, "embed/9mjjowHX1-g") {
		t.Fatalf("the video did not embed at all: %s", body)
	}
	if strings.Contains(body, "<script>") {
		t.Errorf("the player's name broke out of its attribute: %s", body)
	}
}

// The extension must not become a way back in for raw HTML: an iframe written
// by hand is still dropped, whatever it points at.
func TestRenderMarkdown_StillDropsAHandWrittenIframe(t *testing.T) {
	html, err := renderMarkdown(
		`<iframe src="https://evil.test/x" onload="alert(1)"></iframe>` + "\n",
	)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	body := string(html)
	if strings.Contains(body, "evil.test") || strings.Contains(body, "onload") {
		t.Errorf("a hand-written iframe reached the page: %s", body)
	}
}

func TestRenderMarkdown_LeavesAnOrdinaryLinkAlone(t *testing.T) {
	html, err := renderMarkdown("https://elagoht.itch.io/dungrid-tactics\n")
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	if strings.Contains(string(html), "<iframe") {
		t.Errorf("a link to another site became a player: %s", html)
	}
}

// Tables are the one GFM extension the site turns on, so a description or a
// post can lay figures out in columns.
func TestRenderMarkdown_RendersATable(t *testing.T) {
	html, err := renderMarkdown(
		"| Mode | Players |\n" +
			"| --- | ---: |\n" +
			"| Endless | 1 |\n" +
			"| Local PvP | 2 |\n",
	)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	body := string(html)

	for _, want := range []string{"<table>", "<thead>", "<th", "Mode", "<tbody>", "<td", "Local PvP"} {
		if !strings.Contains(body, want) {
			t.Errorf("the table did not render: missing %q\ngot: %s", want, body)
		}
	}
	// The column asked to be right-aligned, and that has to survive into the
	// markup for the stylesheet to honour it.
	if !strings.Contains(body, `align="right"`) {
		t.Errorf("the column alignment was lost: %s", body)
	}
}

// Linkify is deliberately off: turning it on would rewrite every bare URL in
// every post already written, which is a change to existing content rather than
// a new thing an author can reach for.
func TestRenderMarkdown_LeavesABareURLAsText(t *testing.T) {
	html, err := renderMarkdown("Read more at https://example.com/docs for details.\n")
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	if strings.Contains(string(html), "<a href") {
		t.Errorf("a bare URL became a link: %s", html)
	}
}

// The site's content policy has no style-src 'unsafe-inline', so an inline
// style on a table cell would be blocked and the alignment lost. Nothing the
// markdown pipeline produces may carry one.
func TestRenderMarkdown_NeverEmitsAnInlineStyle(t *testing.T) {
	html, err := renderMarkdown(
		"| Left | Middle | Right |\n" +
			"| :--- | :----: | ----: |\n" +
			"| a | b | c |\n",
	)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}
	if strings.Contains(string(html), "style=") {
		t.Errorf("the content policy would block this: %s", html)
	}
}
