package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestBuild_PublishesHashedStylesheet(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	url := bundle.URL("site.css")
	if !strings.HasPrefix(url, "/assets/build/site.") || !strings.HasSuffix(url, ".css") {
		t.Fatalf("URL(site.css) = %q, want /assets/build/site.<hash>.css", url)
	}

	// The hash has to be in the name, not just the URL: the file on disk is
	// what gets served.
	onDisk := filepath.Join(dir, strings.TrimPrefix(url, "/assets/"))
	if _, err := os.Stat(onDisk); err != nil {
		t.Errorf("published stylesheet is missing from disk: %v", err)
	}
}

// The URL must be stable across builds, or every deploy would invalidate a
// cache entry that did not need to change.
func TestBuild_SameInputProducesSameName(t *testing.T) {
	first, err := Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	second, err := Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if first.URL("site.css") != second.URL("site.css") {
		t.Errorf("URLs differ across builds: %q vs %q", first.URL("site.css"), second.URL("site.css"))
	}
}

func TestBuild_RemovesStaleFilesButOnlyInsideBuild(t *testing.T) {
	dir := t.TempDir()

	// A hand-placed file beside the build directory -- the admin logo lives
	// here, and pruning must never reach it.
	handPlaced := filepath.Join(dir, "logo.svg")
	if err := os.WriteFile(handPlaced, []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("seed logo: %v", err)
	}

	if _, err := Build(dir); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// A stylesheet from an imaginary earlier deploy.
	stale := filepath.Join(dir, buildDirName, "site.deadbeef.css")
	if err := os.WriteFile(stale, []byte("body{}"), 0o644); err != nil {
		t.Fatalf("seed stale: %v", err)
	}

	if _, err := Build(dir); err != nil {
		t.Fatalf("second Build() error = %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale stylesheet survived the rebuild (err = %v)", err)
	}
	if _, err := os.Stat(handPlaced); err != nil {
		t.Errorf("a hand-placed file outside build/ was removed: %v", err)
	}
}

func TestBuild_PublishesFontsUnderStableNames(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// The stylesheet references these paths literally, so they must not be
	// hashed. Every face the site sets type in is checked, not just one:
	// a face that failed to publish falls back silently in the browser.
	for _, face := range []string{"archivo", "public-sans", "vt323"} {
		name := "fonts/" + face + ".woff2"
		if url := bundle.URL(name); url != "/assets/build/"+name {
			t.Errorf("%s URL = %q, want an unhashed /assets/build/%s", face, url, name)
		}

		published, err := os.ReadFile(filepath.Join(dir, buildDirName, "fonts", face+".woff2"))
		if err != nil {
			t.Fatalf("read published %s: %v", face, err)
		}
		if len(published) < 4 || string(published[:4]) != "wOF2" {
			t.Errorf("published %s is not a woff2 file", face)
		}
	}
}

func TestBuild_MinifiesCSSWithoutBreakingIt(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	source, err := assetFS.ReadFile("assets/site.css")
	if err != nil {
		t.Fatalf("read source css: %v", err)
	}
	published, err := os.ReadFile(filepath.Join(dir, strings.TrimPrefix(bundle.URL("site.css"), "/assets/")))
	if err != nil {
		t.Fatalf("read published css: %v", err)
	}

	if len(published) >= len(source) {
		t.Errorf("published CSS is %d bytes, source is %d -- minification did nothing", len(published), len(source))
	}

	// Minification must not eat the things that silently break a stylesheet:
	// the design tokens themselves, and the font URL the CSS depends on.
	for _, needle := range []string{
		"--color-accent",
		"/assets/build/fonts/archivo.woff2",
		"/assets/build/fonts/public-sans.woff2",
		"/assets/build/fonts/vt323.woff2",
	} {
		if !strings.Contains(string(published), needle) {
			t.Errorf("minified CSS lost %q", needle)
		}
	}
	if strings.Contains(string(published), "Hand-written plain CSS") {
		t.Error("minified CSS still contains a source comment")
	}
}

func TestBundle_URLOfUnknownAssetIsEmpty(t *testing.T) {
	bundle, err := Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got := bundle.URL("nope.css"); got != "" {
		t.Errorf("URL(unknown) = %q, want empty", got)
	}
}

// The site is one room with the lights on or off, and the tokens are how it
// switches. What must never switch is the hardware: the banner, the machine
// and the cases stay dark in both, so anything drawn on them reads from the
// phosphor rather than from the room -- a pigment-dark amber on a black bezel
// is invisible, which is exactly the bug a light theme invites.
func TestStyles_LightRoomOverridesOnlyTokens(t *testing.T) {
	css := string(siteCSS(t))

	light := lightBlock(t, css)
	// Every declaration in the override block is a custom property. A rule in
	// there would be a second stylesheet nobody remembers to keep in step.
	for _, line := range strings.Split(stripComments(light), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == ":root {" || line == "}" || strings.HasPrefix(line, "--") {
			continue
		}
		t.Errorf("the light room declares something that is not a token: %q", line)
	}

	// The colours the dark surfaces use are declared once and never overridden.
	for _, fixed := range []string{"--phosphor:", "--on-dark:"} {
		if strings.Contains(light, fixed) {
			t.Errorf("%s is overridden by the lit room, so a dark surface changes with it", fixed)
		}
		if !strings.Contains(css, fixed) {
			t.Errorf("%s is never declared", fixed)
		}
	}

	// And the surfaces that stay dark say so, by taking the phosphor locally.
	for _, surface := range []string{".osd {", ".console {", ".jewel {", ".lightbox {"} {
		block := blockAfter(t, css, surface)
		if !strings.Contains(block, "--color-accent: var(--phosphor)") {
			t.Errorf("%s does not pin its accent to the phosphor:\n%s", surface, block)
		}
	}
}

func siteCSS(t *testing.T) []byte {
	t.Helper()
	source, err := assetFS.ReadFile("assets/site.css")
	if err != nil {
		t.Fatalf("read site.css: %v", err)
	}
	return source
}

// lightBlock returns the body of the prefers-color-scheme: light rule.
func lightBlock(t *testing.T, css string) string {
	t.Helper()
	const marker = "@media (prefers-color-scheme: light) {"
	start := strings.Index(css, marker)
	if start < 0 {
		t.Fatal("there is no light room at all")
	}
	rest := css[start+len(marker):]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		t.Fatal("the light room is not closed")
	}
	return rest[:end]
}

func blockAfter(t *testing.T, css, selector string) string {
	t.Helper()
	start := strings.Index(css, selector)
	if start < 0 {
		t.Fatalf("no %s rule", selector)
	}
	rest := css[start:]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		t.Fatalf("%s is not closed", selector)
	}
	return rest[:end]
}

// commentPattern matches a CSS comment, including one spanning lines.
var commentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)

func stripComments(css string) string {
	return commentPattern.ReplaceAllString(css, "")
}
