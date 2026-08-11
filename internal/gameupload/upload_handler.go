package gameupload

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"pixabros/internal/gamearchive"
	"pixabros/internal/httpapi"
)

type Handler struct {
	gamesDir    string
	onExtracted func(slug string) error
	onError     func(error)
}

type HandlerOption func(*Handler)

// WithErrorLogger registers a callback invoked with the underlying error
// whenever archive extraction fails, so the operator can see the real
// cause without it being echoed to the client. The default is a no-op.
func WithErrorLogger(onError func(error)) HandlerOption {
	return func(h *Handler) { h.onError = onError }
}

func NewHandler(gamesDir string, onExtracted func(slug string) error, opts ...HandlerOption) *Handler {
	h := &Handler{gamesDir: gamesDir, onExtracted: onExtracted, onError: func(error) {}}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// maxUploadBodyBytes bounds the whole request body, sized to gamearchive's own
// 500 MiB archive cap plus a small allowance for multipart overhead. It is a
// var (matching gamearchive's own size limits) so tests can shrink it rather
// than having to stream half a gigabyte.
var maxUploadBodyBytes int64 = 501 << 20

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)

	slug := r.PathValue("slug")
	if slug == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_slug", "a game slug is required")
		return
	}
	if slug != filepath.Base(slug) || slug == "." || slug == ".." {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_slug", "slug must be a single path segment")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpapi.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "uploaded archive exceeds the maximum allowed size")
			return
		}
		httpapi.WriteError(w, http.StatusBadRequest, "missing_file", "a file field is required")
		return
	}
	defer file.Close()

	destDir := filepath.Join(h.gamesDir, slug)
	stagingDir := destDir + ".incoming"

	// Always start from a clean staging directory: a previous crashed
	// upload could have left one behind.
	os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare destination directory")
		return
	}

	if err := gamearchive.Extract(file, header.Filename, stagingDir); err != nil {
		// The raw error can be an *os.PathError carrying an absolute server
		// path, so it is logged rather than returned to the client.
		os.RemoveAll(stagingDir)
		h.onError(fmt.Errorf("extract archive for slug %q: %w", slug, err))
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_archive", "could not extract the uploaded archive")
		return
	}

	// Replace the live directory wholesale -- RemoveAll before Rename means
	// a re-upload can never merge with (and leave stale files from) the
	// previous build, and until this point the previous build was never
	// touched, so a failure above leaves it fully intact.
	if err := os.RemoveAll(destDir); err != nil {
		os.RemoveAll(stagingDir)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not replace the previous build")
		return
	}
	if err := os.Rename(stagingDir, destDir); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not finalize the uploaded build")
		return
	}

	if h.onExtracted != nil {
		if err := h.onExtracted(slug); err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "extracted but could not finish processing")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
