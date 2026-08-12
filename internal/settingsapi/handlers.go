package settingsapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"pixabros/internal/httpapi"
	"pixabros/internal/id"
	"pixabros/internal/render"
	"pixabros/internal/settings"
)

type Handlers struct {
	repo *settings.Repo
	db   *sql.DB
}

func NewHandlers(repo *settings.Repo, db *sql.DB) *Handlers {
	return &Handlers{repo: repo, db: db}
}

type groupResponse struct {
	Group       string                `json:"group"`
	Definitions []settings.Definition `json:"definitions"`
	Values      map[string]string     `json:"values"`
}

// Get returns both the values and the definitions, so the admin UI renders the
// form from the server's registry instead of keeping its own copy of the key
// list that could drift out of step.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	group, err := settings.LookupGroup(r.PathValue("group"))
	if errors.Is(err, settings.ErrUnknownGroup) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found",
			"group must be one of: "+strings.Join(settings.GroupNames(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not resolve settings group")
		return
	}

	values, err := h.repo.Values(group)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not read settings")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, groupResponse{
		Group:       group.Name,
		Definitions: group.Definitions,
		Values:      values,
	})
}

type updateRequest struct {
	Values map[string]string `json:"values"`
}

// validate checks each value against its declared kind. A URI that is not a
// URI, or a media id pointing at nothing, would render as a broken link or a
// missing image on the public site rather than failing here.
func (h *Handlers) validate(group settings.Group, values map[string]string) (string, string, bool) {
	for key, value := range values {
		definition, err := group.Define(key)
		if err != nil {
			return "unknown_key", "no such setting: " + key, false
		}
		// Every setting may be left blank; blank simply means "not set".
		if value == "" {
			continue
		}

		switch definition.Kind {
		case settings.KindURI:
			parsed, err := url.Parse(value)
			if err != nil || !parsed.IsAbs() || parsed.Host == "" {
				return "invalid_uri", key + " must be a full URL, including its scheme", false
			}
		case settings.KindMedia:
			if !id.IsValid(value) {
				return "invalid_body", key + " is not a valid media id", false
			}
			exists, err := h.repo.MediaExists(value)
			if err != nil {
				return "internal_error", "could not check media", false
			}
			if !exists {
				return "unknown_reference", key + " refers to an image that does not exist", false
			}
		case settings.KindText:
			// Free text: nothing to check beyond the size cap on the body.
		}
	}
	return "", "", true
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	// Settings carry prose (a hero description, a JSON list of profiles), so
	// this cap is larger than the 64 KiB the id-shaped modules use.
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)

	group, err := settings.LookupGroup(r.PathValue("group"))
	if errors.Is(err, settings.ErrUnknownGroup) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found",
			"group must be one of: "+strings.Join(settings.GroupNames(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not resolve settings group")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.Values == nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "values is required")
		return
	}

	if code, message, ok := h.validate(group, req.Values); !ok {
		status := http.StatusBadRequest
		if code == "internal_error" {
			status = http.StatusInternalServerError
		}
		httpapi.WriteError(w, status, code, message)
		return
	}

	if err := h.repo.Replace(group, req.Values); err != nil {
		if errors.Is(err, settings.ErrUnknownKey) {
			httpapi.WriteError(w, http.StatusBadRequest, "unknown_key", err.Error())
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not save settings")
		return
	}

	if err := render.EnqueueRegen(h.db, group.RegenTag); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	values, err := h.repo.Values(group)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not read settings")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, groupResponse{
		Group:       group.Name,
		Definitions: group.Definitions,
		Values:      values,
	})
}
