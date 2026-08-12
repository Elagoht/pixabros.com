package mediaapi

import (
	"errors"
	"net/http"

	"pixabros/internal/httpapi"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

// Handlers serves media metadata lookups. The upload path lives on
// UploadHandler in upload_handler.go; this type deliberately stays read-only
// so the two concerns can be mounted (and reasoned about) separately.
type Handlers struct {
	repo  *media.Repo
	files storage.Storage
}

func NewHandlers(repo *media.Repo, files storage.Storage) *Handlers {
	return &Handlers{repo: repo, files: files}
}

// mediaResponse matches uploadResponse's shape on purpose: the admin UI shows
// a thumbnail from an upload result and from a lookup with the same code.
type mediaResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	m, err := h.repo.FindByID(r.PathValue("id"))
	if errors.Is(err, media.ErrMediaNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load media")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, mediaResponse{
		ID:     m.ID,
		URL:    h.files.URL(m.Path),
		Width:  m.Width,
		Height: m.Height,
	})
}
