package mediaapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"pixabros/internal/httpapi"
	"pixabros/internal/imaging"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

type UploadHandler struct {
	repo  *media.Repo
	files storage.Storage
}

func NewUploadHandler(repo *media.Repo, files storage.Storage) *UploadHandler {
	return &UploadHandler{repo: repo, files: files}
}

type uploadResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// maxUploadBodyBytes bounds the whole request body, with headroom over
// imaging's own 20 MiB decode cap for multipart overhead. It is a var
// (matching gamearchive's own size limits) so tests can shrink it.
var maxUploadBodyBytes int64 = 24 << 20

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)

	targetName := r.URL.Query().Get("target")
	target, ok := imaging.LookupTarget(targetName)
	if !ok {
		httpapi.WriteError(w, http.StatusBadRequest, "unknown_target", fmt.Sprintf("unknown upload target %q", targetName))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpapi.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "uploaded file exceeds the maximum allowed size")
			return
		}
		httpapi.WriteError(w, http.StatusBadRequest, "missing_file", "a file field is required")
		return
	}
	defer file.Close()

	webpBytes, err := imaging.ProcessUpload(file, target.Width, target.Height)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_image", "could not decode or process the uploaded image")
		return
	}

	key, err := randomMediaKey(targetName)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not generate a storage key")
		return
	}
	if err := h.files.Put(key, bytesReader(webpBytes)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not store the image")
		return
	}

	m, err := h.repo.Create(key, target.Width, target.Height)
	if err != nil {
		_ = h.files.Delete(key) // best-effort cleanup of the orphaned storage object
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not save media record")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, uploadResponse{
		ID:     m.ID,
		URL:    h.files.URL(m.Path),
		Width:  m.Width,
		Height: m.Height,
	})
}

func randomMediaKey(targetName string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("media/%s/%d-%s.webp", targetName, time.Now().Year(), hex.EncodeToString(b)), nil
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
