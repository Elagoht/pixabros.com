package gamearchive

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The whole point of Scan is to produce, for a build already on disk, the
// manifest that build would have got had it been uploaded today. If the two
// ever disagree, a backfilled game advertises a download whose contents do not
// match what the browser will fetch -- so they are held together here.
func TestScan_MatchesWhatExtractProduced(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":     "<h1>game</h1>",
		"game.wasm":      "wasm-bytes",
		"sub/data.pck":   "pack",
		"sub/deep/a.png": "png",
	})

	extracted, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	scanned, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if !reflect.DeepEqual(scanned, extracted) {
		t.Errorf("Scan and Extract disagree:\n scanned = %+v\nextracted = %+v", scanned, extracted)
	}
}

// A build extracted before the debris filter existed still has the junk on
// disk. Listing it would send a visitor resource forks and an engine service
// worker as part of their offline copy.
func TestScan_LeavesOldDebrisOutOfTheManifest(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, map[string]string{
		"index.html":             "<h1>game</h1>",
		"game.wasm":              "wasm-bytes",
		"__MACOSX/._index.html":  "resource fork",
		"._game.wasm":            "resource fork",
		".DS_Store":              "finder",
		"game.service.worker.js": "self.addEventListener('install', function () {});",
	})

	scanned, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	want := []File{
		{Path: "game.wasm", Bytes: 10},
		{Path: "index.html", Bytes: 13},
	}
	if !reflect.DeepEqual(scanned.Files, want) {
		t.Errorf("files = %+v, want %+v", scanned.Files, want)
	}
}

// The same requirement Extract enforces: without an index.html at the root
// there is nothing for the console to open, so the directory is not a build.
func TestScan_RefusesADirectoryWithNoIndex(t *testing.T) {
	dir := t.TempDir()
	seed(t, dir, map[string]string{"game.wasm": "wasm-bytes"})

	if _, err := Scan(dir); err == nil {
		t.Error("Scan accepted a directory with no index.html")
	}
}

func TestScan_ErrorsOnAMissingDirectory(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "nothing-here")); err == nil {
		t.Error("Scan accepted a directory that does not exist")
	}
}

// seed writes files into dir, creating parents, so a test can state exactly
// what a build directory already holds.
func seed(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
