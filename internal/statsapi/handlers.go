package statsapi

import (
	"net/http"

	"pixabros/internal/httpapi"
	"pixabros/internal/stats"
)

type Handlers struct {
	repo *stats.Repo
}

// NewHandlers takes no *sql.DB: this endpoint only reads, so nothing here
// enqueues a regeneration.
func NewHandlers(repo *stats.Repo) *Handlers {
	return &Handlers{repo: repo}
}

// Get returns every dashboard figure in one response. The dashboard shows them
// together, so splitting them across endpoints would only cost it round trips.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.Get()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not collect statistics")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s)
}
