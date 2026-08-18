package gameupload

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/gamearchive"
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
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error {
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

// zipWithFiles builds a zip containing an index.html plus the given extra
// entries, so a test can distinguish one build's files from another's.
func zipWithFiles(t *testing.T, extra map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	entries := map[string]string{"index.html": "<html></html>"}
	for name, content := range extra {
		entries[name] = content
	}
	for name, content := range entries {
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

// TestUpload_ReplacingBuildDropsStaleFiles proves a re-upload replaces the
// live directory wholesale: a file that existed only in the previous build
// must not survive alongside the new one, since a leftover .wasm/.js next to
// its replacement silently breaks the published game.
func TestUpload_ReplacingBuildDropsStaleFiles(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

	firstReq := uploadRequest(t, "pixel-quest", zipWithFiles(t, map[string]string{"old-build.wasm": "stale"}))
	firstRec := httptest.NewRecorder()
	handler.Upload(firstRec, firstReq)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first upload status = %d, want %d, body = %s", firstRec.Code, http.StatusNoContent, firstRec.Body.String())
	}
	stalePath := filepath.Join(gamesDir, "pixel-quest", "old-build.wasm")
	if _, err := httpFileExists(stalePath); err != nil {
		t.Fatalf("first build's file should exist after the first upload: %v", err)
	}

	secondReq := uploadRequest(t, "pixel-quest", zipWithFiles(t, map[string]string{"new-build.wasm": "fresh"}))
	secondRec := httptest.NewRecorder()
	handler.Upload(secondRec, secondReq)
	if secondRec.Code != http.StatusNoContent {
		t.Fatalf("second upload status = %d, want %d, body = %s", secondRec.Code, http.StatusNoContent, secondRec.Body.String())
	}

	if _, err := httpFileExists(stalePath); !os.IsNotExist(err) {
		t.Errorf("os.Stat(old-build.wasm) error = %v, want the previous build's file to be gone", err)
	}
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest", "new-build.wasm")); err != nil {
		t.Errorf("new build's file should exist: %v", err)
	}
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest", "index.html")); err != nil {
		t.Errorf("new build's index.html should exist: %v", err)
	}
	// The staging directory must not be left behind next to the live one.
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest.incoming")); !os.IsNotExist(err) {
		t.Errorf("os.Stat(staging dir) error = %v, want no staging directory left behind", err)
	}
}

// TestUpload_FailedUploadLeavesPreviousBuildIntact proves the previous build
// survives a rejected upload: extraction happens in a staging directory, so
// a corrupt archive can no longer wipe out a live, working game.
func TestUpload_FailedUploadLeavesPreviousBuildIntact(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

	firstReq := uploadRequest(t, "pixel-quest", zipWithFiles(t, map[string]string{"game.wasm": "working build"}))
	firstRec := httptest.NewRecorder()
	handler.Upload(firstRec, firstReq)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first upload status = %d, want %d, body = %s", firstRec.Code, http.StatusNoContent, firstRec.Body.String())
	}

	badReq := uploadRequest(t, "pixel-quest", []byte("not a real archive"))
	badRec := httptest.NewRecorder()
	handler.Upload(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid upload status = %d, want %d", badRec.Code, http.StatusBadRequest)
	}

	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest", "index.html")); err != nil {
		t.Errorf("previous build's index.html should have survived: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(gamesDir, "pixel-quest", "game.wasm"))
	if err != nil {
		t.Fatalf("previous build's game.wasm should have survived: %v", err)
	}
	if string(got) != "working build" {
		t.Errorf("game.wasm = %q, want the original build's contents unchanged", got)
	}
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest.incoming")); !os.IsNotExist(err) {
		t.Errorf("os.Stat(staging dir) error = %v, want the failed staging directory cleaned up", err)
	}
}

// uploadRequestWithContentType builds a multipart request carrying a field
// the handler does not look for, so FormFile fails with ErrMissingFile rather
// than a transport-level parse error.
func uploadRequestWithoutFileField(t *testing.T, slug string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("not_file", "payload"); err != nil {
		t.Fatalf("WriteField(): %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/"+slug+"/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("slug", slug)
	return req
}

// TestUpload_MissingFileFieldNotLogged proves the operator-facing log stays
// quiet for plain client mistakes: a multipart body without a "file" part is
// a bad request, not something to investigate on the server.
func TestUpload_MissingFileFieldNotLogged(t *testing.T) {
	var captured error
	handler := NewHandler(t.TempDir(), func(slug string, build gamearchive.Build) error { return nil },
		WithErrorLogger(func(err error) { captured = err }))

	rec := httptest.NewRecorder()
	handler.Upload(rec, uploadRequestWithoutFileField(t, "pixel-quest"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "missing_file") {
		t.Errorf("body = %s, want the missing_file code", rec.Body.String())
	}
	if captured != nil {
		t.Errorf("onError called with %v, want it untouched for a client-side mistake", captured)
	}
}

// TestUpload_UnreadableBodyLoggedAndCoded proves a body that cannot be parsed
// as multipart (here: the wrong Content-Type, the same shape as a boundary
// stripped by a proxy or a stream cut mid-part) is logged with its real
// cause and answered with its own code instead of pretending the client
// forgot the file field.
func TestUpload_UnreadableBodyLoggedAndCoded(t *testing.T) {
	var captured error
	handler := NewHandler(t.TempDir(), func(slug string, build gamearchive.Build) error { return nil },
		WithErrorLogger(func(err error) { captured = err }))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/pixel-quest/upload", strings.NewReader("junk"))
	req.Header.Set("Content-Type", "text/plain")
	req.SetPathValue("slug", "pixel-quest")
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unreadable_upload") {
		t.Errorf("body = %s, want the unreadable_upload code", rec.Body.String())
	}
	if captured == nil {
		t.Fatal("onError was not called, want the underlying parse error logged")
	}
}

func TestUpload_InvalidArchiveReturnsBadRequest(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

	req := uploadRequest(t, "pixel-quest", []byte("not a real archive"))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpload_OversizedBodyRejected(t *testing.T) {
	orig := maxUploadBodyBytes
	t.Cleanup(func() { maxUploadBodyBytes = orig })
	maxUploadBodyBytes = 64 // far smaller than any real multipart upload

	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

	req := uploadRequest(t, "pixel-quest", zipWithIndex(t))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func TestUpload_InvalidArchiveLogsErrorWithoutLeakingPaths(t *testing.T) {
	gamesDir := t.TempDir()
	var captured error
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil }, WithErrorLogger(func(err error) {
		captured = err
	}))

	req := uploadRequest(t, "pixel-quest", []byte("not a real archive"))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if captured == nil {
		t.Fatal("expected WithErrorLogger's callback to receive the underlying extraction error")
	}
	if !strings.Contains(captured.Error(), "pixel-quest") {
		t.Errorf("captured error = %q, want it to name the slug", captured.Error())
	}

	body := rec.Body.String()
	if strings.Contains(body, gamesDir) {
		t.Errorf("response body = %q, must not contain the server-side destination path", body)
	}
	if strings.Contains(body, "/") {
		t.Errorf("response body = %q, must not contain any filesystem path fragment", body)
	}
}

func TestUpload_PathTraversalSlugRejected(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

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
	handler := NewHandler(gamesDir, func(slug string, build gamearchive.Build) error { return nil })

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
