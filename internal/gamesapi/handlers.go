package gamesapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
	"pixabros/internal/id"
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
	ID                string  `json:"id"`
	Slug              string  `json:"slug"`
	Title             string  `json:"title"`
	ShortDescription  string  `json:"short_description"`
	FullDescription   string  `json:"full_description"`
	Tags              string  `json:"tags"`
	Genre             string  `json:"genre"`
	ReleaseDate       string  `json:"release_date"`
	Kind              string  `json:"kind"`
	IsBrowserPlayable bool    `json:"is_browser_playable"`
	IsDownloadable    bool    `json:"is_downloadable"`
	IsForSale         bool    `json:"is_for_sale"`
	PriceDisplay      string  `json:"price_display"`
	ExternalLinksJSON string  `json:"external_links_json"`
	CartridgeArtID    *string `json:"cartridge_art_id"`
	CDCoverArtID      *string `json:"cd_cover_art_id"`
	OGImageID         *string `json:"og_image_id"`
	WebExportPath     string  `json:"web_export_path"`
	DisplayOrder      int     `json:"display_order"`
	IsPublished       bool    `json:"is_published"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toGameResponse(g games.Game) gameResponse {
	return gameResponse{
		ID:                g.ID,
		Slug:              g.Slug,
		Title:             g.Title,
		ShortDescription:  g.ShortDescription,
		FullDescription:   g.FullDescription,
		Tags:              g.Tags,
		Genre:             g.Genre,
		ReleaseDate:       g.ReleaseDate,
		Kind:              g.Kind,
		IsBrowserPlayable: g.IsBrowserPlayable,
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

// List orders by ?sort= and ?dir=, defaulting to the manual display order.
// An unknown field is rejected rather than quietly ignored: silently falling
// back would make a mistyped column look like the data is simply unsorted.
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
	if errors.Is(err, games.ErrInvalidSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort",
			"sort must be one of: "+strings.Join(games.SortableFields(), ", "))
		return
	}
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
	Genre             string `json:"genre"`
	ReleaseDate       string `json:"release_date"`
	Kind              string `json:"kind"`
	IsBrowserPlayable bool   `json:"is_browser_playable"`
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
	if !checkReleaseAndKind(w, req.ReleaseDate, req.Kind) {
		return
	}

	game, err := h.repo.Create(games.CreateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		Genre:             req.Genre,
		ReleaseDate:       req.ReleaseDate,
		Kind:              req.Kind,
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

func parseIDPathValue(r *http.Request) string {
	return r.PathValue("id")
}

// resolveGame reads the {id} path segment. The admin UI addresses a game by
// its immutable id, but a slug is still accepted so links built from a
// public /play/{slug}/ URL keep resolving. The two can never be confused:
// an id is exactly 24 lowercase alphanumeric characters, while a slug is
// derived from a title and is either shorter, longer, or hyphenated.
func (h *Handlers) resolveGame(r *http.Request) (games.Game, error) {
	raw := r.PathValue("id")
	if id.IsValid(raw) {
		return h.repo.FindByID(raw)
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
	Title             string  `json:"title"`
	ShortDescription  string  `json:"short_description"`
	FullDescription   string  `json:"full_description"`
	Tags              string  `json:"tags"`
	Genre             string  `json:"genre"`
	ReleaseDate       string  `json:"release_date"`
	Kind              string  `json:"kind"`
	IsBrowserPlayable bool    `json:"is_browser_playable"`
	IsDownloadable    bool    `json:"is_downloadable"`
	IsForSale         bool    `json:"is_for_sale"`
	PriceDisplay      string  `json:"price_display"`
	ExternalLinksJSON string  `json:"external_links_json"`
	CartridgeArtID    *string `json:"cartridge_art_id"`
	CDCoverArtID      *string `json:"cd_cover_art_id"`
	OGImageID         *string `json:"og_image_id"`
	DisplayOrder      int     `json:"display_order"`
	IsPublished       bool    `json:"is_published"`
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
	if !checkReleaseAndKind(w, req.ReleaseDate, req.Kind) {
		return
	}

	game, err := h.repo.Update(oldGame.ID, games.UpdateInput{
		Title:             req.Title,
		ShortDescription:  req.ShortDescription,
		FullDescription:   req.FullDescription,
		Tags:              req.Tags,
		Genre:             req.Genre,
		ReleaseDate:       req.ReleaseDate,
		Kind:              req.Kind,
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
				if err := h.repo.SetBuild(game.ID, newDir); err == nil {
					game.WebExportPath = newDir
				}
			}
		}
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%s", oldGame.ID)); err != nil {
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
	IDs []string `json:"ids"`
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

// DeleteBuild removes a game's extracted build from disk and clears the
// columns derived from it. is_browser_playable is not a field anyone can
// edit: a game is playable in the browser exactly while a build exists, so
// removing the files is what turns it off.
func (h *Handlers) DeleteBuild(w http.ResponseWriter, r *http.Request) {
	game, err := h.resolveGame(r)
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}

	// The directory is removed before the columns are cleared: if the delete
	// fails, the game keeps pointing at a build that is still playable,
	// rather than claiming to have none while the files remain served.
	if h.playDir != "" {
		if err := os.RemoveAll(filepath.Join(h.playDir, game.Slug)); err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not remove the build from disk")
			return
		}
	}

	if err := h.repo.SetBuild(game.ID, ""); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not clear the build")
		return
	}

	if err := render.EnqueueRegen(h.db, fmt.Sprintf("game:%s", game.ID)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	if err := render.EnqueueRegen(h.db, "game:list"); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isoDate matches the YYYY-MM-DD the date picker submits. release_date is
// stored and sorted as text, so a differently shaped date would sort into the
// wrong place rather than failing loudly.
var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// checkReleaseAndKind validates the two constrained fields, returning false
// once it has written the error response.
//
// Both are optional: a game with no release date yet is normal, and an empty
// kind means a caller that predates the field, which the repo reads as a
// production game. What is rejected is a value that is present and wrong,
// because the database's CHECK would otherwise turn it into a 500.
func checkReleaseAndKind(w http.ResponseWriter, releaseDate, kind string) bool {
	if releaseDate != "" && !isoDate.MatchString(releaseDate) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_date",
			"release_date must be formatted YYYY-MM-DD")
		return false
	}
	if kind != "" && !games.IsValidKind(kind) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_kind",
			`kind must be "`+games.KindProduction+`" or "`+games.KindGameJam+`"`)
		return false
	}
	return true
}
