package gamesapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
	"pixabros/internal/render"
)

type Handlers struct {
	repo    *games.Repo
	db      *sql.DB
	playDir string
}

// NewHandlers takes playDir -- the directory uploaded game builds are
// extracted into -- so that deleting a game can also remove its extracted
// build, which is otherwise left publicly playable under /play/{slug}/.
func NewHandlers(repo *games.Repo, db *sql.DB, playDir string) *Handlers {
	return &Handlers{repo: repo, db: db, playDir: playDir}
}

type gameResponse struct {
	ID                int64  `json:"id"`
	Slug              string `json:"slug"`
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	CartridgeArtID    *int64 `json:"cartridge_art_id"`
	CDCoverArtID      *int64 `json:"cd_cover_art_id"`
	OGImageID         *int64 `json:"og_image_id"`
	WebExportPath     string `json:"web_export_path"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func toGameResponse(g games.Game) gameResponse {
	return gameResponse{
		ID:                g.ID,
		Slug:              g.Slug,
		Title:             g.Title,
		ShortDescription:  g.ShortDescription,
		FullDescription:   g.FullDescription,
		Tags:              g.Tags,
		IsBrowserPlayable: g.IsBrowserPlayable,
		IsDownloadable:    g.IsDownloadable,
		IsForSale:         g.IsForSale,
		PriceDisplay:      g.PriceDisplay,
		ExternalLinksJSON: g.ExternalLinksJSON,
		CartridgeArtID:    g.CartridgeArtID,
		CDCoverArtID:      g.CDCoverArtID,
		OGImageID:         g.OGImageID,
		WebExportPath:     g.WebExportPath,
		DisplayOrder:      g.DisplayOrder,
		IsPublished:       g.IsPublished,
		CreatedAt:         g.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         g.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list games")
		return
	}
	responses := make([]gameResponse, 0, len(list))
	for _, g := range list {
		responses = append(responses, toGameResponse(g))
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}

type createRequest struct {
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return
	}

	game, err := h.repo.Create(games.CreateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		IsBrowserPlayable: req.IsBrowserPlayable,
		IsDownloadable:    req.IsDownloadable,
		IsForSale:         req.IsForSale,
		PriceDisplay:      req.PriceDisplay,
		ExternalLinksJSON: req.ExternalLinksJSON,
		DisplayOrder:      req.DisplayOrder,
		IsPublished:       req.IsPublished,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create game")
		return
	}

	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, toGameResponse(game))
}

func parseIDPathValue(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	game, err := h.repo.FindByID(id)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toGameResponse(game))
}

type updateRequest struct {
	Title             string `json:"title"`
	ShortDescription  string `json:"short_description"`
	FullDescription   string `json:"full_description"`
	Tags              string `json:"tags"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
	IsDownloadable    bool   `json:"is_downloadable"`
	IsForSale         bool   `json:"is_for_sale"`
	PriceDisplay      string `json:"price_display"`
	ExternalLinksJSON string `json:"external_links_json"`
	CartridgeArtID    *int64 `json:"cartridge_art_id"`
	CDCoverArtID      *int64 `json:"cd_cover_art_id"`
	OGImageID         *int64 `json:"og_image_id"`
	DisplayOrder      int    `json:"display_order"`
	IsPublished       bool   `json:"is_published"`
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return
	}

	game, err := h.repo.Update(id, games.UpdateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		IsBrowserPlayable: req.IsBrowserPlayable,
		IsDownloadable:    req.IsDownloadable,
		IsForSale:         req.IsForSale,
		PriceDisplay:      req.PriceDisplay,
		ExternalLinksJSON: req.ExternalLinksJSON,
		CartridgeArtID:    req.CartridgeArtID,
		CDCoverArtID:      req.CDCoverArtID,
		OGImageID:         req.OGImageID,
		DisplayOrder:      req.DisplayOrder,
		IsPublished:       req.IsPublished,
	})
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update game")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%d", id)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, toGameResponse(game))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPathValue(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	// The game is loaded before it is deleted purely to learn its slug, which
	// is what names its extracted build directory on disk.
	game, err := h.repo.FindByID(id)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete game")
		return
	}

	if h.playDir != "" {
		os.RemoveAll(filepath.Join(h.playDir, game.Slug))
	}

	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
