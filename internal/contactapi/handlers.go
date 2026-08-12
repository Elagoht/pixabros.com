package contactapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"pixabros/internal/contact"
	"pixabros/internal/httpapi"
)

type Handlers struct {
	repo *contact.Repo
}

// NewHandlers takes no *sql.DB: nothing here enqueues a regeneration, because
// contact submissions never appear on the public site.
func NewHandlers(repo *contact.Repo) *Handlers {
	return &Handlers{repo: repo}
}

type submissionResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Subject       string `json:"subject"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Message       string `json:"message"`
	WantsCallback bool   `json:"wants_callback"`
	IsRead        bool   `json:"is_read"`
	IPAddress     string `json:"ip_address"`
	CreatedAt     string `json:"created_at"`
}

func toSubmissionResponse(s contact.Submission) submissionResponse {
	return submissionResponse{
		ID:            s.ID,
		Name:          s.Name,
		Subject:       s.Subject,
		Phone:         s.Phone,
		Email:         s.Email,
		Message:       s.Message,
		WantsCallback: s.WantsCallback,
		IsRead:        s.IsRead,
		IPAddress:     s.IPAddress,
		CreatedAt:     s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type listResponse struct {
	Submissions []submissionResponse `json:"submissions"`
	Unread      int                  `json:"unread"`
}

// List returns the inbox newest-first by default, along with the unread count
// so the UI does not have to derive it from a page of results it may have
// sorted differently.
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
	if errors.Is(err, contact.ErrInvalidSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort",
			"sort must be one of: "+strings.Join(contact.SortableFields(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list contact submissions")
		return
	}

	unread, err := h.repo.UnreadCount()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not count unread submissions")
		return
	}

	responses := make([]submissionResponse, 0, len(list))
	for _, s := range list {
		responses = append(responses, toSubmissionResponse(s))
	}
	httpapi.WriteJSON(w, http.StatusOK, listResponse{Submissions: responses, Unread: unread})
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	submission, err := h.repo.FindByID(r.PathValue("id"))
	if errors.Is(err, contact.ErrSubmissionNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "contact submission not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load contact submission")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toSubmissionResponse(submission))
}

type setReadRequest struct {
	// A pointer distinguishes "mark it unread" from "the field was omitted",
	// which a plain bool would collapse into the same false.
	IsRead *bool `json:"is_read"`
}

// SetRead is the only mutation the admin has over a submission: everything
// else is what the sender wrote and stays as it arrived.
func (h *Handlers) SetRead(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)

	var req setReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.IsRead == nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "is_read is required")
		return
	}

	submission, err := h.repo.SetRead(r.PathValue("id"), *req.IsRead)
	if errors.Is(err, contact.ErrSubmissionNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "contact submission not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update contact submission")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toSubmissionResponse(submission))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.repo.Delete(r.PathValue("id"))
	if errors.Is(err, contact.ErrSubmissionNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "contact submission not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete contact submission")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
