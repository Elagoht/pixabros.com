package youtube

import "testing"

// Every shape of link the YouTube share dialog produces has to work, because
// which one an author pastes is not their decision to think about.
func TestID_AcceptsTheShapesYouTubeHandsOut(t *testing.T) {
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
		got, ok := ID(raw)
		if !ok || got != want {
			t.Errorf("ID(%q) = %q, %v, want %q, true", raw, got, ok, want)
		}
	}
}

// Anything that is not a YouTube link stays an ordinary link. The javascript:
// and data: cases are the ones that matter: a post must not be able to reach
// the iframe with a URL of its own choosing.
func TestID_RejectsEverythingElse(t *testing.T) {
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
		if got, ok := ID(raw); ok {
			t.Errorf("ID(%q) = %q, true, want it rejected", raw, got)
		}
	}
}

// The id is the only thing that crosses from a post into the page, so the id
// pattern is what stops a quote from breaking out of the attribute.
func TestIsID_AcceptsOnlyTheIDCharset(t *testing.T) {
	for _, id := range []string{"9mjjowHX1-g", "abcdefghijk", "___________"} {
		if !IsID(id) {
			t.Errorf("IsID rejected %q", id)
		}
	}
	for _, id := range []string{`"onload="x`, "9mjjowHX1-g'", "short", "<script>abc"} {
		if IsID(id) {
			t.Errorf("IsID accepted %q", id)
		}
	}
}
