package site

import (
	"strings"
	"testing"
)

// Every shape of link the YouTube share dialog produces has to work, because
// which one an author pastes is not their decision to think about.
func TestYouTubeID_AcceptsTheShapesYouTubeHandsOut(t *testing.T) {
	const want = "9mjjowHX1-g"

	for _, raw := range []string{
		"https://youtu.be/9mjjowHX1-g",
		"https://youtu.be/9mjjowHX1-g?si=wAFmtPxOuk8zR_SB",
		"https://www.youtube.com/watch?v=9mjjowHX1-g",
		"https://www.youtube.com/watch?v=9mjjowHX1-g&t=42s",
		"https://m.youtube.com/watch?v=9mjjowHX1-g",
		"https://www.youtube-nocookie.com/embed/9mjjowHX1-g?si=wAFmtPxOuk8zR_SB",
		"https://www.youtube.com/shorts/9mjjowHX1-g",
		"http://youtube.com/watch?v=9mjjowHX1-g",
	} {
		got, ok := youTubeID(raw)
		if !ok || got != want {
			t.Errorf("youTubeID(%q) = %q, %v, want %q, true", raw, got, ok, want)
		}
	}
}

// Anything that is not a YouTube link stays an ordinary link. The javascript:
// and data: cases are the ones that matter: a post must not be able to reach
// the iframe with a URL of its own choosing.
func TestYouTubeID_RejectsEverythingElse(t *testing.T) {
	for _, raw := range []string{
		"",
		"https://example.com/watch?v=9mjjowHX1-g",
		"https://youtube.com.evil.test/watch?v=9mjjowHX1-g",
		"https://notyoutube.com/embed/9mjjowHX1-g",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"https://youtu.be/",
		"https://youtu.be/short",
		"https://youtu.be/waytoolongtobeanid",
		`https://youtu.be/"onload="alert(1)`,
		"https://www.youtube.com/watch?v=",
		"https://youtu.be/9mjjowHX1-g extra words",
	} {
		if got, ok := youTubeID(raw); ok {
			t.Errorf("youTubeID(%q) = %q, true, want it rejected", raw, got)
		}
	}
}

// The id is the only thing that crosses from a post into the page, so the id
// pattern is what stops a quote from breaking out of the attribute.
func TestVideoID_AcceptsOnlyTheIDCharset(t *testing.T) {
	for _, id := range []string{"9mjjowHX1-g", "abcdefghijk", "___________"} {
		if !videoID.MatchString(id) {
			t.Errorf("videoID rejected %q", id)
		}
	}
	for _, id := range []string{`"onload="x`, "9mjjowHX1-g'", "short", "<script>abc"} {
		if videoID.MatchString(id) {
			t.Errorf("videoID accepted %q", id)
		}
	}
}

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
