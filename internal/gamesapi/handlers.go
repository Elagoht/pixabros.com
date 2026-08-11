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

// resolveGame reads the {id} path segment, which the admin UI's edit URL
// now sends as the game's slug (slugs change with the title, so they are
// what the URL is built from), but accepts a numeric id too so any other
// direct API caller keeps working unchanged.
func (h *Handlers) resolveGame(r *http.Request) (games.Game, error) {
	raw := r.PathValue("id")
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return h.repo.FindByID(id)
	}
	return h.repo.FindBySlug(raw)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	game, err := h.resolveGame(r)
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
	// Loaded before the update so a slug change below (see games.Repo.Update,
	// which regenerates the slug from the title) can move this game's
	// already-extracted build alongside it -- otherwise its /play/{slug}/
	// link would 404 the moment the title changes.
	oldGame, err := h.resolveGame(r)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
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

	game, err := h.repo.Update(oldGame.ID, games.UpdateInput{
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

	if game.Slug != oldGame.Slug && h.playDir != "" {
		oldDir := filepath.Join(h.playDir, oldGame.Slug)
		if _, statErr := os.Stat(oldDir); statErr == nil {
			newDir := filepath.Join(h.playDir, game.Slug)
			if err := os.Rename(oldDir, newDir); err == nil {
				if err := h.repo.SetWebExportPath(game.ID, newDir); err == nil {
					game.WebExportPath = newDir
				}
			}
		}
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%d", oldGame.ID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, toGameResponse(game))
}

type reorderRequest struct {
	IDs []int64 `json:"ids"`
}

// Reorder takes the complete ordered list of game ids -- the admin UI always
// has every game loaded already, so there is no partial-reorder case -- and
// sets each one's display_order to its index in the list, in one
// transaction. This replaces sending one full-body PUT per moved game just
// to swap two display_order values.
func (h *Handlers) Reorder(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, games.ErrGameNotFound) {
			httpapi.WriteError(w, http.StatusNotFound, "not_found", "one of the given ids does not exist")
			return
		}
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not reorder games")
		return
	}

	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	// The game is loaded before it is deleted purely to learn its slug, which
	// is what names its extracted build directory on disk.
	game, err := h.resolveGame(r)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}

	if err := h.repo.Delete(game.ID); err != nil {
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
