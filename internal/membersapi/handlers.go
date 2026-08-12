package membersapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"pixabros/internal/httpapi"
	"pixabros/internal/id"
	"pixabros/internal/members"
	"pixabros/internal/render"
)

// regenTag is the cache tag every member mutation invalidates. Members have
// no page of their own -- they are listed on the site's team section -- so
// one list-level tag covers them.
const regenTag = "member:list"

type Handlers struct {
	repo *members.Repo
	db   *sql.DB
}

func NewHandlers(repo *members.Repo, db *sql.DB) *Handlers {
	return &Handlers{repo: repo, db: db}
}

type memberResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	AvatarID     *string `json:"avatar_id"`
	Tags         string  `json:"tags"`
	Description  string  `json:"description"`
	LinksJSON    string  `json:"links_json"`
	DisplayOrder int     `json:"display_order"`
	IsPublished  bool    `json:"is_published"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toMemberResponse(m members.Member) memberResponse {
	return memberResponse{
		ID:           m.ID,
		Name:         m.Name,
		AvatarID:     m.AvatarID,
		Tags:         m.Tags,
		Description:  m.Description,
		LinksJSON:    m.LinksJSON,
		DisplayOrder: m.DisplayOrder,
		IsPublished:  m.IsPublished,
		CreatedAt:    m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    m.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// List orders by ?sort= and ?dir=, defaulting to the manual display order.
// An unknown field is rejected rather than quietly ignored: falling back
// would make a mistyped column look like the data is simply unsorted.
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
	if errors.Is(err, members.ErrInvalidSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort",
			"sort must be one of: "+strings.Join(members.SortableFields(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list members")
		return
	}

	responses := make([]memberResponse, 0, len(list))
	for _, m := range list {
		responses = append(responses, toMemberResponse(m))
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}

// createRequest carries no avatar: media is attached once the member exists,
// the same way a game's artwork is.
type createRequest struct {
	Name         string `json:"name"`
	Tags         string `json:"tags"`
	Description  string `json:"description"`
	LinksJSON    string `json:"links_json"`
	DisplayOrder int    `json:"display_order"`
	IsPublished  bool   `json:"is_published"`
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "name is required")
		return
	}

	member, err := h.repo.Create(members.CreateInput{
		Name:         req.Name,
		Tags:         req.Tags,
		Description:  req.Description,
		LinksJSON:    req.LinksJSON,
		DisplayOrder: req.DisplayOrder,
		IsPublished:  req.IsPublished,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create member")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, toMemberResponse(member))
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	member, err := h.repo.FindByID(r.PathValue("id"))
	if errors.Is(err, members.ErrMemberNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "member not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load member")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toMemberResponse(member))
}

type updateRequest struct {
	Name         string  `json:"name"`
	AvatarID     *string `json:"avatar_id"`
	Tags         string  `json:"tags"`
	Description  string  `json:"description"`
	LinksJSON    string  `json:"links_json"`
	DisplayOrder int     `json:"display_order"`
	IsPublished  bool    `json:"is_published"`
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "name is required")
		return
	}
	// An avatar id that is not id-shaped can only be a caller mistake, and
	// storing it would leave a foreign key pointing at nothing.
	if req.AvatarID != nil && !id.IsValid(*req.AvatarID) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "avatar_id is not a valid id")
		return
	}

	member, err := h.repo.Update(r.PathValue("id"), members.UpdateInput{
		Name:         req.Name,
		AvatarID:     req.AvatarID,
		Tags:         req.Tags,
		Description:  req.Description,
		LinksJSON:    req.LinksJSON,
		DisplayOrder: req.DisplayOrder,
		IsPublished:  req.IsPublished,
	})
	if errors.Is(err, members.ErrMemberNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "member not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update member")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, toMemberResponse(member))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.repo.Delete(r.PathValue("id"))
	if errors.Is(err, members.ErrMemberNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "member not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete member")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type reorderRequest struct {
	IDs []string `json:"ids"`
}

// Reorder takes the complete ordered id list -- the admin UI always has every
// member loaded -- and sets each display_order to its index, in one
// transaction.
func (h *Handlers) Reorder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var req reorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if len(req.IDs) == 0 {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "ids is required")
		return
	}

	if err := h.repo.Reorder(req.IDs); err != nil {
		if errors.Is(err, members.ErrMemberNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "one of the given ids does not exist")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not reorder members")
		return
	}

	if err := render.EnqueueRegen(h.db, regenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
