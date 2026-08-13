package site

import (
	"os"
	"path/filepath"
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
