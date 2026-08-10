package mediaapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func multipartUploadRequest(t *testing.T, target string, pngData []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(pngData); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/media?target="+target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func solidPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func TestUpload_Success(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	repo := media.NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")
	handler := NewUploadHandler(repo, files)

	req := multipartUploadRequest(t, "avatar", solidPNG(t, 800, 400))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Width != 400 || resp.Height != 400 {
		t.Errorf("dimensions = %dx%d, want 400x400", resp.Width, resp.Height)
	}
	if resp.ID == 0 {
		t.Error("expected a non-zero media ID")
	}

	saved, err := repo.FindByID(resp.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Width != 400 || saved.Height != 400 {
		t.Errorf("saved dimensions = %dx%d, want 400x400", saved.Width, saved.Height)
	}
}

func TestUpload_UnknownTarget(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	handler := NewUploadHandler(media.NewRepo(conn), storage.NewLocalDisk(t.TempDir(), "/media"))
	req := multipartUploadRequest(t, "not_a_real_target", solidPNG(t, 10, 10))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
