package site

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

//go:embed assets
var assetFS embed.FS

// buildDirName is the subdirectory generated files live in. It is separate
// because the parent assets directory also holds hand-placed files -- the
// admin panel's logo, for one -- and pruning must never be able to reach them.
const buildDirName = "build"

// hashLength is how much of the sha256 goes into a filename. Eight hex
// characters is ample for a handful of files and keeps the HTML readable.
const hashLength = 8

// minifiable maps an extension to the media type its minifier is registered
// under. A file type absent from here is published byte for byte.
var minifiable = map[string]string{
	".css": "text/css",
	".js":  "application/javascript",
}

// Bundle publishes the embedded assets under content-hashed names and
// remembers where they landed, so a template can ask for "site.css" and get
// back the URL that is safe to cache forever.
type Bundle struct {
	urls map[string]string
}

// Build writes every embedded asset into <dir>/build and removes any file
// there that it did not just write, so a stylesheet from an older deploy does
// not linger.
//
// Stylesheets are minified and then content-hashed, so the hash identifies the
// bytes actually served. Fonts keep their plain names: a font file's contents
// never change without its name changing, so it does not need a hash to bust
// caches, and a stable name is what lets the CSS reference it literally.
func Build(dir string) (*Bundle, error) {
	buildDir := filepath.Join(dir, buildDirName)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("create asset build dir: %w", err)
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)

	bundle := &Bundle{urls: map[string]string{}}
	written := map[string]bool{}

	err := fs.WalkDir(assetFS, "assets", func(embedPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		content, err := assetFS.ReadFile(embedPath)
		if err != nil {
			return err
		}

		// Path relative to the assets root: "site.css", "fonts/inter.woff2".
		name := strings.TrimPrefix(embedPath, "assets/")

		// Stylesheets and scripts are minified and hashed; anything else (the
		// fonts) is published verbatim under its own name.
		outName := name
		if mediaType, ok := minifiable[path.Ext(name)]; ok {
			minified, err := m.Bytes(mediaType, content)
			if err != nil {
				return fmt.Errorf("minify %s: %w", name, err)
			}
			content = minified
			outName = hashedName(name, content)
		}

		outPath := filepath.Join(buildDir, filepath.FromSlash(outName))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, content, 0o644); err != nil {
			return err
		}

		written[outPath] = true
		bundle.urls[name] = "/assets/" + buildDirName + "/" + outName
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := prune(buildDir, written); err != nil {
		return nil, err
	}
	return bundle, nil
}

// URL returns the public path for a logical asset name, e.g. "site.css" ->
// "/assets/build/site.a1b2c3d4.css". An unknown name returns "" rather than
// guessing: a broken stylesheet link is easier to spot than a plausible one
// pointing at nothing.
func (b *Bundle) URL(name string) string {
	return b.urls[name]
}

// hashedName turns "site.css" into "site.<hash>.css".
func hashedName(name string, content []byte) string {
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])[:hashLength]
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext) + "." + digest + ext
}

// prune removes files under buildDir that this build did not write. It is
// scoped to buildDir alone, which is why generated files live there and not
// beside hand-placed ones.
func prune(buildDir string, written map[string]bool) error {
	return filepath.WalkDir(buildDir, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || written[p] {
			return nil
		}
		return os.Remove(p)
	})
}
