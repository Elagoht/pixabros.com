package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
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

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err != nil {
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

	if err := Extract(bytes.NewReader(data), "build.tar.gz", dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
		t.Errorf("index.html not extracted: %v", err)
	}
}

func TestExtract_MissingIndexHTML_RollsBack(t *testing.T) {
	data := buildZip(t, map[string]string{"readme.txt": "no index here"})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
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

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should reject an archive with a path-traversal entry")
	}
}

func TestExtract_UnsupportedExtension(t *testing.T) {
	if err := Extract(bytes.NewReader([]byte("irrelevant")), "build.rar", t.TempDir()); err == nil {
		t.Fatal("Extract() should reject an unsupported archive extension")
	}
}

func TestExtract_PathTraversal_Rejected_TarGz(t *testing.T) {
	data := buildTarGz(t, map[string]string{
		"index.html":            "<html></html>",
		"../../etc/passwd-evil": "malicious",
	})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.tar.gz", dest); err == nil {
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
	if err := Extract(bytes.NewReader(buf.Bytes()), "build.tar", dest); err == nil {
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

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
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

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
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
