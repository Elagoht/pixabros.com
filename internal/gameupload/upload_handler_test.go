package gameupload

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func zipWithIndex(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("index.html")
	if err != nil {
		t.Fatalf("zip Create(): %v", err)
	}
	if _, err := f.Write([]byte("<html></html>")); err != nil {
		t.Fatalf("zip write(): %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip Close(): %v", err)
	}
	return buf.Bytes()
}

func uploadRequest(t *testing.T, slug string, archiveData []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "build.zip")
	if err != nil {
		t.Fatalf("CreateFormFile(): %v", err)
	}
	if _, err := part.Write(archiveData); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/"+slug+"/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("slug", slug)
	return req
}

func httpFileExists(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func TestUpload_ExtractsAndCallsCallback(t *testing.T) {
	gamesDir := t.TempDir()
	var calledWithSlug string
	handler := NewHandler(gamesDir, func(slug string) error {
		calledWithSlug = slug
		return nil
	})

	req := uploadRequest(t, "pixel-quest", zipWithIndex(t))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if calledWithSlug != "pixel-quest" {
		t.Errorf("callback slug = %q, want %q", calledWithSlug, "pixel-quest")
	}
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest", "index.html")); err != nil {
		t.Errorf("expected index.html to be extracted: %v", err)
	}
}

func TestUpload_InvalidArchiveReturnsBadRequest(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string) error { return nil })

	req := uploadRequest(t, "pixel-quest", []byte("not a real archive"))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpload_PathTraversalSlugRejected(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string) error { return nil })

	req := uploadRequest(t, "../../etc", zipWithIndex(t))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Verify no directory was created outside gamesDir
	entries, err := os.ReadDir(gamesDir)
	if err != nil {
		t.Fatalf("ReadDir gamesDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("gamesDir should be empty, found %d entries", len(entries))
	}
}

func TestUpload_SlugWithSlashRejected(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string) error { return nil })

	req := uploadRequest(t, "foo/bar", zipWithIndex(t))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Verify no directory was created
	entries, err := os.ReadDir(gamesDir)
	if err != nil {
		t.Fatalf("ReadDir gamesDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("gamesDir should be empty, found %d entries", len(entries))
	}
}
