package site

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
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
	for _, face := range []string{"space-grotesk", "public-sans", "courier-prime", "courier-prime-700"} {
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
		"/assets/build/fonts/space-grotesk.woff2",
		"/assets/build/fonts/public-sans.woff2",
		"/assets/build/fonts/courier-prime.woff2",
		"/assets/build/fonts/courier-prime-700.woff2",
	} {
		if !strings.Contains(string(published), needle) {
			t.Errorf("minified CSS lost %q", needle)
		}
	}
	if strings.Contains(string(published), "Hand-written plain CSS") {
		t.Error("minified CSS still contains a source comment")
	}
}

// The mark and the two icons the manifest points at are content-hashed like
// the stylesheet. /assets is served with an immutable cache, so a mark that
// kept a stable name could be replaced and nobody would ever see the new one.
func TestBuild_PublishesTheMarkAndIconsUnderHashedNames(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, name := range []string{"logo.svg", "icon-192.png", "icon-512.png"} {
		ext := path.Ext(name)
		hashed := regexp.MustCompile(`^/assets/build/` +
			regexp.QuoteMeta(strings.TrimSuffix(name, ext)) +
			`\.[0-9a-f]{` + fmt.Sprint(hashLength) + `}` + regexp.QuoteMeta(ext) + `$`)

		url := bundle.URL(name)
		if !hashed.MatchString(url) {
			t.Errorf("URL(%s) = %q, want /assets/build/<name>.<hash>%s", name, url, ext)
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, strings.TrimPrefix(url, "/assets/"))); err != nil {
			t.Errorf("published %s is missing from disk: %v", name, err)
		}
	}
}

// The mark is a pixel-art drawing built from one path per colour, which is
// several times larger as source than it needs to be on the wire.
func TestBuild_MinifiesTheMark(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	source, err := assetFS.ReadFile("assets/logo.svg")
	if err != nil {
		t.Fatalf("read source mark: %v", err)
	}
	published, err := os.ReadFile(filepath.Join(dir, strings.TrimPrefix(bundle.URL("logo.svg"), "/assets/")))
	if err != nil {
		t.Fatalf("read published mark: %v", err)
	}

	if len(published) >= len(source) {
		t.Errorf("published mark is %d bytes, source is %d -- minification did nothing", len(published), len(source))
	}
	// Minifying a drawing into something that is no longer a drawing would
	// still pass a size check.
	if !strings.Contains(string(published), "<svg") || !strings.Contains(string(published), "viewBox") {
		t.Error("the minified mark is not an SVG with a viewBox any more")
	}
}

// A PNG is already compressed; running it through a minifier it has no rule
// for must leave the bytes alone rather than quietly truncate them.
func TestBuild_PublishesIconsByteForByte(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	source, err := assetFS.ReadFile("assets/icon-512.png")
	if err != nil {
		t.Fatalf("read source icon: %v", err)
	}
	published, err := os.ReadFile(filepath.Join(dir, strings.TrimPrefix(bundle.URL("icon-512.png"), "/assets/")))
	if err != nil {
		t.Fatalf("read published icon: %v", err)
	}

	if !bytes.Equal(source, published) {
		t.Errorf("published icon is %d bytes, source is %d -- something rewrote it", len(published), len(source))
	}
}

// The vitest files live beside the scripts they cover, which puts them inside
// the embedded asset tree. Published, they would hand every visitor the
// repository's own paths under a cache that never expires -- and every test
// file added later would join them without anyone choosing to.
func TestBuild_NeverPublishesATestFile(t *testing.T) {
	dir := t.TempDir()

	bundle, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// The tests this is protecting against must actually exist, or the check
	// below passes by describing nothing.
	if _, err := assetFS.ReadFile("assets/sw" + testSuffix); err != nil {
		t.Fatalf("there is no test file beside the worker to exclude: %v", err)
	}

	for name, url := range bundle.urls {
		if strings.HasSuffix(name, testSuffix) {
			t.Errorf("Build published %q as %q", name, url)
		}
	}

	err = filepath.WalkDir(filepath.Join(dir, buildDirName), func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), testSuffix) {
			t.Errorf("a test file reached the published assets: %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk build dir: %v", err)
	}
}

// A test file published by an earlier build has to go, not merely stop being
// written: /assets is immutable-cached, so a copy left on disk goes on being
// served for a year.
func TestBuild_PrunesATestFileAnEarlierBuildPublished(t *testing.T) {
	dir := t.TempDir()
	if _, err := Build(dir); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	stale := filepath.Join(dir, buildDirName, "sw.deadbeef"+testSuffix)
	if err := os.WriteFile(stale, []byte("// from a build that still published these"), 0o644); err != nil {
		t.Fatalf("seed stale test file: %v", err)
	}

	if _, err := Build(dir); err != nil {
		t.Fatalf("second Build() error = %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the stale test file survived the rebuild (err = %v)", err)
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

// The site is dark-only by design: the lit room was retired with the Stitch
// redesign. A light block sneaking back in would ship half a theme.
func TestStylesheet_IsDarkOnly(t *testing.T) {
	css, err := assetFS.ReadFile("assets/site.css")
	if err != nil {
		t.Fatalf("read site.css: %v", err)
	}
	if strings.Contains(string(css), "(prefers-color-scheme: light)") {
		t.Error("site.css still styles a light room; the redesign is dark-only")
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
