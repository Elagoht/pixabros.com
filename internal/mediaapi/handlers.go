package mediaapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"pixabros/internal/httpapi"
	"pixabros/internal/media"
	"pixabros/internal/mediarefs"
	"pixabros/internal/storage"
)

// Handlers serves the media library: listing what is stored, editing alt
// text, and deleting an image nothing uses. The upload path lives on
// UploadHandler in upload_handler.go, because an upload needs a target to
// decide its dimensions and so belongs to the module asking for it.
type Handlers struct {
	repo  *media.Repo
	files storage.Storage
	db    *sql.DB
}

func NewHandlers(repo *media.Repo, files storage.Storage, db *sql.DB) *Handlers {
	return &Handlers{repo: repo, files: files, db: db}
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

// libraryItem is a media row plus the two things only the library shows: where
// the image is used, and when it arrived.
type libraryItem struct {
	ID        string            `json:"id"`
	URL       string            `json:"url"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Format    string            `json:"format"`
	AltText   string            `json:"alt_text"`
	CreatedAt string            `json:"created_at"`
	Usages    []mediarefs.Usage `json:"usages"`
}

type libraryResponse struct {
	Items []libraryItem `json:"items"`
	// Orphaned is how many images nothing points at, which is the number worth
	// acting on and is tedious to count from the list by eye.
	Orphaned int `json:"orphaned"`
}

// List returns the whole library, newest first, each item annotated with what
// is using it.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	stored, err := h.repo.List()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list media")
		return
	}

	usages, err := mediarefs.Lookup(h.db)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not resolve media usage")
		return
	}

	items := make([]libraryItem, 0, len(stored))
	orphaned := 0
	for _, m := range stored {
		used := usages[m.ID]
		if len(used) == 0 {
			orphaned++
			// A nil slice would serialise as null; the UI iterates this.
			used = []mediarefs.Usage{}
		}
		items = append(items, libraryItem{
			ID:        m.ID,
			URL:       h.files.URL(m.Path),
			Width:     m.Width,
			Height:    m.Height,
			Format:    m.Format,
			AltText:   m.AltText,
			CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
			Usages:    used,
		})
	}

	httpapi.WriteJSON(w, http.StatusOK, libraryResponse{Items: items, Orphaned: orphaned})
}

type updateRequest struct {
	AltText *string `json:"alt_text"`
}

// Update edits alt text, the one field an upload cannot decide for itself. It
// matters because it is what a screen reader reads out on the public site.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	// A pointer distinguishes "clear the alt text" from "the field was
	// omitted", which a plain string would collapse into the same empty value.
	if req.AltText == nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "alt_text is required")
		return
	}

	m, err := h.repo.SetAltText(r.PathValue("id"), *req.AltText)
	if errors.Is(err, media.ErrMediaNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update media")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, mediaResponse{
		ID:     m.ID,
		URL:    h.files.URL(m.Path),
		Width:  m.Width,
		Height: m.Height,
	})
}

// Delete removes an image and its file, but only if nothing points at it.
// Deleting a referenced image would blank a game's artwork or -- because
// game_screenshots cascades -- remove a screenshot outright, so it is refused
// rather than quietly breaking a page.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	mediaID := r.PathValue("id")

	m, err := h.repo.FindByID(mediaID)
	if errors.Is(err, media.ErrMediaNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load media")
		return
	}

	usages, err := mediarefs.Lookup(h.db)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not resolve media usage")
		return
	}
	if len(usages[mediaID]) > 0 {
		httpapi.WriteError(w, http.StatusConflict, "still_in_use",
			"this image is still used; remove it where it is used before deleting it")
		return
	}

	// The row goes first: if the file delete fails afterwards the sweep will
	// retry it, whereas a deleted file with a surviving row would leave a
	// broken image referenced by nothing but visible in the library.
	if err := h.repo.Delete(mediaID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete media")
		return
	}
	if err := h.files.Delete(m.Path); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete the stored file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
