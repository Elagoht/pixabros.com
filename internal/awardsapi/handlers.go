package awardsapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"pixabros/internal/awards"
	"pixabros/internal/dbutil"
	"pixabros/internal/httpapi"
	"pixabros/internal/id"
	"pixabros/internal/render"
)

// regenTag is the cache tag every award mutation invalidates. Awards have no
// page of their own -- they render as a timeline -- so one list tag covers them.
const regenTag = "award:list"

// isoDate matches the YYYY-MM-DD the date picker submits. The column is TEXT,
// and awards are ordered by it as a string, so a differently shaped date
// would silently sort into the wrong place instead of failing loudly.
var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Handlers struct {
	repo *awards.Repo
	db   *sql.DB
}

func NewHandlers(repo *awards.Repo, db *sql.DB) *Handlers {
	return &Handlers{repo: repo, db: db}
}

type awardResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Issuer    string  `json:"issuer"`
	Date      string  `json:"date"`
	PictureID *string `json:"picture_id"`
	GameID    *string `json:"game_id"`
	Link      string  `json:"link"`
	CreatedAt string  `json:"created_at"`
}

func toAwardResponse(a awards.Award) awardResponse {
	return awardResponse{
		ID:        a.ID,
		Title:     a.Title,
		Issuer:    a.Issuer,
		Date:      a.Date,
		PictureID: a.PictureID,
		GameID:    a.GameID,
		Link:      a.Link,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// List orders by ?sort= and ?dir=, defaulting to the most recent award first.
// An unknown field is rejected rather than quietly ignored.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	field := r.URL.Query().Get("sort")

	var descending bool
	switch dir := r.URL.Query().Get("dir"); dir {
	case "", "asc":
		descending = false
	case "desc":
		descending = true
	default:
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort", `dir must be "asc" or "desc"`)
		return
	}

	list, err := h.repo.List(field, descending)
	if errors.Is(err, awards.ErrInvalidSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort",
			"sort must be one of: "+strings.Join(awards.SortableFields(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list awards")
		return
	}

	responses := make([]awardResponse, 0, len(list))
	for _, a := range list {
		responses = append(responses, toAwardResponse(a))
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}

type createRequest struct {
	Title  string `json:"title"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
	Link   string `json:"link"`
}

// requireFields validates what every award needs, returning false once it has
// written the error response.
func requireFields(w http.ResponseWriter, title, issuer, date string) bool {
	if strings.TrimSpace(title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return false
	}
	if strings.TrimSpace(issuer) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "issuer is required")
		return false
	}
	if !isoDate.MatchString(date) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_date", "date must be formatted YYYY-MM-DD")
		return false
	}
	return true
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if !requireFields(w, req.Title, req.Issuer, req.Date) {
		return
	}

	award, err := h.repo.Create(awards.CreateInput{
		Title:  req.Title,
		Issuer: req.Issuer,
		Date:   req.Date,
		Link:   req.Link,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create award")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, toAwardResponse(award))
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	award, err := h.repo.FindByID(r.PathValue("id"))
	if errors.Is(err, awards.ErrAwardNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "award not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load award")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toAwardResponse(award))
}

type updateRequest struct {
	Title     string  `json:"title"`
	Issuer    string  `json:"issuer"`
	Date      string  `json:"date"`
	Link      string  `json:"link"`
	PictureID *string `json:"picture_id"`
	GameID    *string `json:"game_id"`
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if !requireFields(w, req.Title, req.Issuer, req.Date) {
		return
	}
	for field, value := range map[string]*string{
		"picture_id": req.PictureID,
		"game_id":    req.GameID,
	} {
		if value != nil && !id.IsValid(*value) {
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", field+" is not a valid id")
			return
		}
	}

	award, err := h.repo.Update(r.PathValue("id"), awards.UpdateInput{
		Title:     req.Title,
		Issuer:    req.Issuer,
		Date:      req.Date,
		Link:      req.Link,
		PictureID: req.PictureID,
		GameID:    req.GameID,
	})
	if errors.Is(err, awards.ErrAwardNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "award not found")
		return
	}
	// A well-formed id for a game or image that does not exist is a caller
	// mistake, not a server fault.
	if dbutil.IsForeignKeyViolation(err) {
		httpapi.WriteError(w, http.StatusBadRequest, "unknown_reference",
			"game_id or picture_id refers to something that does not exist")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update award")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, toAwardResponse(award))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.repo.Delete(r.PathValue("id"))
	if errors.Is(err, awards.ErrAwardNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "award not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete award")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
