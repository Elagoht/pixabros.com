package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip Create(%q): %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write(%q): %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip Close(): %v", err)
	}
	return buf.Bytes()
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader(%q): %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("tar write(%q): %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close(): %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close(): %v", err)
	}
	return buf.Bytes()
}

func TestExtract_Zip_Success(t *testing.T) {
	data := buildZip(t, map[string]string{
		"index.html":     "<html></html>",
		"assets/game.js": "console.log('hi');",
	})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.zip", dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
		t.Errorf("index.html not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "assets", "game.js")); err != nil {
		t.Errorf("assets/game.js not extracted: %v", err)
	}
}

func TestExtract_TarGz_Success(t *testing.T) {
	data := buildTarGz(t, map[string]string{"index.html": "<html></html>"})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.tar.gz", dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
		t.Errorf("index.html not extracted: %v", err)
	}
}

func TestExtract_MissingIndexHTML_RollsBack(t *testing.T) {
	data := buildZip(t, map[string]string{"readme.txt": "no index here"})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should fail when index.html is missing")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a failed extract, found %d entries", len(entries))
	}
}

func TestExtract_PathTraversal_Rejected(t *testing.T) {
	data := buildZip(t, map[string]string{
		"index.html":            "<html></html>",
		"../../etc/passwd-evil": "malicious",
	})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should reject an archive with a path-traversal entry")
	}
}

func TestExtract_UnsupportedExtension(t *testing.T) {
	if _, err := Extract(bytes.NewReader([]byte("irrelevant")), "build.rar", t.TempDir()); err == nil {
		t.Fatal("Extract() should reject an unsupported archive extension")
	}
}

func TestExtract_PathTraversal_Rejected_TarGz(t *testing.T) {
	data := buildTarGz(t, map[string]string{
		"index.html":            "<html></html>",
		"../../etc/passwd-evil": "malicious",
	})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.tar.gz", dest); err == nil {
		t.Fatal("Extract() should reject a tar.gz archive with a path-traversal entry")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a failed tar.gz extract with path traversal, found %d entries", len(entries))
	}
}

func TestExtract_TarSymlinkEntry_Rejected(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	index := "<html></html>"
	if err := tw.WriteHeader(&tar.Header{Name: "index.html", Mode: 0o644, Size: int64(len(index)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatalf("tar WriteHeader(index.html): %v", err)
	}
	if _, err := tw.Write([]byte(index)); err != nil {
		t.Fatalf("tar write(index.html): %v", err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "secrets", Mode: 0o777, Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}); err != nil {
		t.Fatalf("tar WriteHeader(symlink): %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close(): %v", err)
	}

	dest := t.TempDir()
	if _, err := Extract(bytes.NewReader(buf.Bytes()), "build.tar", dest); err == nil {
		t.Fatal("Extract() should reject a tar archive containing a symlink entry")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a rejected symlink entry, found %d entries", len(entries))
	}
}

func TestExtract_OversizedArchive(t *testing.T) {
	// Test with a very small limit to trigger rejection without needing huge data
	origMaxArchiveSize := maxArchiveSize
	origMaxExtractedSize := maxExtractedSize
	defer func() {
		maxArchiveSize = origMaxArchiveSize
		maxExtractedSize = origMaxExtractedSize
	}()

	maxArchiveSize = 100 // 100 bytes limit
	maxExtractedSize = origMaxExtractedSize

	data := buildZip(t, map[string]string{
		"index.html": "<html></html>",
		"large.txt":  "x",
	})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should reject an archive exceeding the size limit")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a failed extract due to size, found %d entries", len(entries))
	}
}

func TestExtract_OversizedDecompressed(t *testing.T) {
	// Test with a very small decompressed limit to trigger rejection
	origMaxArchiveSize := maxArchiveSize
	origMaxExtractedSize := maxExtractedSize
	defer func() {
		maxArchiveSize = origMaxArchiveSize
		maxExtractedSize = origMaxExtractedSize
	}()

	maxArchiveSize = origMaxArchiveSize
	maxExtractedSize = 50 // 50 bytes limit for total decompressed output

	data := buildZip(t, map[string]string{
		"index.html": "<html></html>",                                // 14 bytes
		"file.txt":   "this is extra content that exceeds the limit", // lots more bytes
	})
	dest := t.TempDir()

	if _, err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should reject archive when decompressed size exceeds the limit")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a failed extract due to size, found %d entries", len(entries))
	}
}

// writeZip builds a zip in memory from name -> contents, so a test can state
// exactly what an upload contained.
func writeZip(t *testing.T, entries map[string]string) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(entries[name])); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return bytes.NewReader(buf.Bytes()), "build.zip"
}

// The offline download needs to know what a build is made of, and how big each
// piece is, before it asks anyone to spend 90 MB.
func TestExtract_ReportsEveryFileItWrote(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":   "<h1>game</h1>",
		"game.wasm":    "wasm-bytes",
		"sub/data.pck": "pack",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	want := []File{
		{Path: "game.wasm", Bytes: 10},
		{Path: "index.html", Bytes: 13},
		{Path: "sub/data.pck", Bytes: 4},
	}
	if !reflect.DeepEqual(build.Files, want) {
		t.Errorf("files = %+v, want %+v", build.Files, want)
	}
	if build.Bytes != 27 {
		t.Errorf("Bytes = %d, want 27", build.Bytes)
	}
}

// Sorting is not cosmetic: zip and tar list entries in whatever order the
// archiver chose, and a version derived from that order would change when
// nothing about the build did.
func TestExtract_VersionIsTheContentNotTheOrdering(t *testing.T) {
	first, name := writeZip(t, map[string]string{"index.html": "a", "z.js": "b"})
	firstBuild, err := Extract(first, name, t.TempDir())
	if err != nil {
		t.Fatalf("first Extract() error = %v", err)
	}

	second, _ := writeZip(t, map[string]string{"z.js": "b", "index.html": "a"})
	secondBuild, err := Extract(second, name, t.TempDir())
	if err != nil {
		t.Fatalf("second Extract() error = %v", err)
	}

	if firstBuild.Version != secondBuild.Version {
		t.Errorf("the same build got two versions: %q vs %q", firstBuild.Version, secondBuild.Version)
	}
	if len(firstBuild.Version) != versionLength {
		t.Errorf("version = %q, want %d characters", firstBuild.Version, versionLength)
	}

	changed, _ := writeZip(t, map[string]string{"index.html": "a", "z.js": "CHANGED"})
	changedBuild, err := Extract(changed, name, t.TempDir())
	if err != nil {
		t.Fatalf("changed Extract() error = %v", err)
	}
	if changedBuild.Version == firstBuild.Version {
		t.Error("a changed file did not change the version")
	}
}

// Half of a Finder-made zip is resource forks. They were being served
// publicly, and downloading them to somebody's phone for offline play would
// be worse still.
func TestExtract_LeavesMacOSDebrisOnTheFloor(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":            "<h1>game</h1>",
		"__MACOSX/._index.html": "resource fork",
		"._index.html":          "resource fork",
		".DS_Store":             "finder",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(build.Files) != 1 || build.Files[0].Path != "index.html" {
		t.Errorf("files = %+v, want index.html alone", build.Files)
	}
	for _, junked := range []string{"__MACOSX", "._index.html", ".DS_Store"} {
		if _, err := os.Stat(filepath.Join(dir, junked)); !os.IsNotExist(err) {
			t.Errorf("%s was written to disk (err = %v)", junked, err)
		}
	}
}

// An engine's own service worker would claim /play/{slug}/ and shadow the
// site's, which is the scope the offline feature depends on.
func TestExtract_DropsTheEnginesOwnServiceWorker(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":             "<h1>game</h1>",
		"game.service.worker.js": "self.addEventListener('install', function () {});",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(build.Files) != 1 || build.Files[0].Path != "index.html" {
		t.Errorf("files = %+v, want index.html alone", build.Files)
	}
	if _, err := os.Stat(filepath.Join(dir, "game.service.worker.js")); !os.IsNotExist(err) {
		t.Errorf("the engine's service worker was written to disk (err = %v)", err)
	}
}
