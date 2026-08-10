package gameupload

import (
	"net/http"
	"os"
	"path/filepath"

	"pixabros/internal/gamearchive"
	"pixabros/internal/httpapi"
)

type Handler struct {
	gamesDir    string
	onExtracted func(slug string) error
}

func NewHandler(gamesDir string, onExtracted func(slug string) error) *Handler {
	return &Handler{gamesDir: gamesDir, onExtracted: onExtracted}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_slug", "a game slug is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_file", "a file field is required")
		return
	}
	defer file.Close()

	destDir := filepath.Join(h.gamesDir, slug)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare destination directory")
		return
	}

	if err := gamearchive.Extract(file, header.Filename, destDir); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_archive", err.Error())
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
