package gamesapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
	"pixabros/internal/render"
)

type screenshotResponse struct {
	ID           string `json:"id"`
	GameID       string `json:"game_id"`
	MediaID      string `json:"media_id"`
	DisplayOrder int    `json:"display_order"`
}

type addScreenshotRequest struct {
	MediaID      string `json:"media_id"`
	DisplayOrder int    `json:"display_order"`
}

func (h *Handlers) AddScreenshot(w http.ResponseWriter, r *http.Request) {
	gameID := parseIDPathValue(r)

	var req addScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.MediaID == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "media_id is required")
		return
	}

	screenshot, err := h.repo.AddScreenshot(gameID, req.MediaID, req.DisplayOrder)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not add screenshot")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%s", gameID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, screenshotResponse{
		ID:           screenshot.ID,
		GameID:       screenshot.GameID,
		MediaID:      screenshot.MediaID,
		DisplayOrder: screenshot.DisplayOrder,
	})
}

func (h *Handlers) RemoveScreenshot(w http.ResponseWriter, r *http.Request) {
	gameID := parseIDPathValue(r)
	screenshotID := r.PathValue("screenshotID")

	if err := h.repo.RemoveScreenshot(gameID, screenshotID); err != nil {
		if errors.Is(err, games.ErrScreenshotNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "screenshot not found")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not remove screenshot")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%s", gameID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderScreenshots takes the complete ordered list of this game's
// screenshot ids and sets each one's display_order to its index, in one
// transaction, scoped to gameID.
func (h *Handlers) ReorderScreenshots(w http.ResponseWriter, r *http.Request) {
	gameID := parseIDPathValue(r)

	var req reorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if len(req.IDs) == 0 {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "ids is required")
		return
	}

	if err := h.repo.ReorderScreenshots(gameID, req.IDs); err != nil {
		if errors.Is(err, games.ErrScreenshotNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "one of the given ids does not belong to this game")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not reorder screenshots")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%s", gameID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListScreenshots checks the game exists before listing, so a request for an
// unknown game is a 404 rather than an empty array that looks like "this game
// simply has no screenshots".
func (h *Handlers) ListScreenshots(w http.ResponseWriter, r *http.Request) {
	gameID := parseIDPathValue(r)

	if _, err := h.repo.FindByID(gameID); err != nil {
		if errors.Is(err, games.ErrGameNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}

	list, err := h.repo.ListScreenshots(gameID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list screenshots")
		return
	}

	responses := make([]screenshotResponse, 0, len(list))
	for _, s := range list {
		responses = append(responses, screenshotResponse{
			ID:           s.ID,
			GameID:       s.GameID,
			MediaID:      s.MediaID,
			DisplayOrder: s.DisplayOrder,
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}
