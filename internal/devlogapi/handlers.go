package devlogapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"pixabros/internal/dbutil"
	"pixabros/internal/devlog"
	"pixabros/internal/httpapi"
	"pixabros/internal/id"
	"pixabros/internal/ogimage"
	"pixabros/internal/render"
)

// isoDate matches the YYYY-MM-DD the date picker submits. published_at is
// stored as TEXT and ordered as a string, so a differently shaped date would
// sort into the wrong place instead of failing loudly.
var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Handlers struct {
	repo *devlog.Repo
	// og draws a post's social preview. A post without one would share as a
	// bare link, so every post gets a generated image unless an admin uploads
	// their own.
	og *ogimage.Store
	db *sql.DB
	// cache holds the public search responses. Every mutation clears it so a
	// fresh post, rename or delete reaches the public index immediately.
	cache *searchCache
}

func NewHandlers(repo *devlog.Repo, db *sql.DB, og *ogimage.Store, cache *searchCache) *Handlers {
	return &Handlers{og: og, repo: repo, db: db, cache: cache}
}

type postResponse struct {
	ID              string  `json:"id"`
	Slug            string  `json:"slug"`
	Title           string  `json:"title"`
	ContentMarkdown string  `json:"content_markdown"`
	GameID          *string `json:"game_id"`
	OGImageID       *string `json:"og_image_id"`
	IsPublished     bool    `json:"is_published"`
	PublishedAt     string  `json:"published_at"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toPostResponse(p devlog.Post) postResponse {
	return postResponse{
		ID:              p.ID,
		Slug:            p.Slug,
		Title:           p.Title,
		ContentMarkdown: p.ContentMarkdown,
		GameID:          p.GameID,
		OGImageID:       p.OGImageID,
		IsPublished:     p.IsPublished,
		PublishedAt:     p.PublishedAt,
		CreatedAt:       p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// regenTagFor is the cache tag for one post's own page. Unlike members and
// awards, a devlog post has a page of its own, so both it and the index are
// invalidated on every change.
func regenTagFor(postID string) string {
	return fmt.Sprintf("devlog:%s", postID)
}

const listRegenTag = "devlog:list"

func (h *Handlers) enqueueRegen(postID string) error {
	if err := render.EnqueueRegen(h.db, regenTagFor(postID)); err != nil {
		return err
	}
	return render.EnqueueRegen(h.db, listRegenTag)
}

// List orders by ?sort= and ?dir=, defaulting to the newest post first.
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
	if errors.Is(err, devlog.ErrInvalidSort) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_sort",
			"sort must be one of: "+strings.Join(devlog.SortableFields(), ", "))
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not list devlog posts")
		return
	}

	responses := make([]postResponse, 0, len(list))
	for _, p := range list {
		responses = append(responses, toPostResponse(p))
	}
	httpapi.WriteJSON(w, http.StatusOK, responses)
}

// createRequest carries no game or image: both are attached once the post
// exists, the same way a game's artwork is.
type createRequest struct {
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	IsPublished     bool   `json:"is_published"`
	PublishedAt     string `json:"published_at"`
}

// checkTitleAndDate validates what every post needs, returning false once it
// has written the error response. An empty date is allowed: the repo stamps
// one when the post is first published.
func checkTitleAndDate(w http.ResponseWriter, title, publishedAt string) bool {
	if strings.TrimSpace(title) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "title is required")
		return false
	}
	if publishedAt != "" && !isoDate.MatchString(publishedAt) {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_date", "published_at must be formatted YYYY-MM-DD")
		return false
	}
	return true
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	// Post bodies carry prose, so this cap is larger than the 64 KiB the
	// other modules use.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if !checkTitleAndDate(w, req.Title, req.PublishedAt) {
		return
	}

	post, err := h.repo.Create(devlog.CreateInput{
		Title:           req.Title,
		ContentMarkdown: req.ContentMarkdown,
		IsPublished:     req.IsPublished,
		PublishedAt:     req.PublishedAt,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create devlog post")
		return
	}

	// The preview is drawn from the title, so it can only be made once the post
	// exists. A failure here is not worth losing the post over: the picture is
	// regenerated on the next edit.
	if h.og != nil {
		if imageID, err := h.og.Refresh(nil, post.Title, post.GameID); err == nil {
			if updated, err := h.repo.Update(post.ID, devlog.UpdateInput{
				Title:           post.Title,
				ContentMarkdown: post.ContentMarkdown,
				GameID:          post.GameID,
				OGImageID:       imageID,
				IsPublished:     post.IsPublished,
				PublishedAt:     post.PublishedAt,
			}); err == nil {
				post = updated
			}
		}
	}

	if err := h.enqueueRegen(post.ID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	h.cache.Clear()

	httpapi.WriteJSON(w, http.StatusCreated, toPostResponse(post))
}

// resolvePost reads the {id} path segment, accepting either the post's id or
// its slug so a link built from a public URL still resolves. The two can never
// be confused: an id is exactly 24 lowercase alphanumeric characters.
func (h *Handlers) resolvePost(r *http.Request) (devlog.Post, error) {
	raw := r.PathValue("id")
	if id.IsValid(raw) {
		return h.repo.FindByID(raw)
	}
	return h.repo.FindBySlug(raw)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	post, err := h.resolvePost(r)
	if errors.Is(err, devlog.ErrPostNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "devlog post not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load devlog post")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toPostResponse(post))
}

type updateRequest struct {
	Title           string  `json:"title"`
	ContentMarkdown string  `json:"content_markdown"`
	GameID          *string `json:"game_id"`
	OGImageID       *string `json:"og_image_id"`
	IsPublished     bool    `json:"is_published"`
	PublishedAt     string  `json:"published_at"`
}

// resolveOGImage decides which picture a post carries after an edit.
//
// The panel sends the post's current image back with every save, so "the caller
// did not pick a different one" means the id came back unchanged, not that it
// came back empty. That is the case where a rename has to be redrawn: an
// unchanged, generated card still shows the old name. The game counts too,
// because the card names it beside the studio's mark.
//
// Refresh does the deciding about what may be replaced: it leaves an uploaded
// picture alone and deletes the generated one it supersedes.
func (h *Handlers) resolveOGImage(existing devlog.Post, req updateRequest) *string {
	requested := req.OGImageID
	if requested == nil {
		requested = existing.OGImageID
	}

	if h.og == nil || !sameID(requested, existing.OGImageID) {
		return requested
	}
	if req.Title == existing.Title && sameID(req.GameID, existing.GameID) {
		return requested
	}

	refreshed, err := h.og.Refresh(existing.OGImageID, req.Title, req.GameID)
	if err != nil {
		// A card that could not be redrawn is not worth losing the edit over;
		// the post keeps the one it has.
		return requested
	}
	return refreshed
}

func sameID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if !checkTitleAndDate(w, req.Title, req.PublishedAt) {
		return
	}
	for field, value := range map[string]*string{
		"game_id":     req.GameID,
		"og_image_id": req.OGImageID,
	} {
		if value != nil && !id.IsValid(*value) {
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", field+" is not a valid id")
			return
		}
	}

	existing, err := h.resolvePost(r)
	if errors.Is(err, devlog.ErrPostNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "devlog post not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load devlog post")
		return
	}

	post, err := h.repo.Update(existing.ID, devlog.UpdateInput{
		Title:           req.Title,
		ContentMarkdown: req.ContentMarkdown,
		GameID:          req.GameID,
		OGImageID:       h.resolveOGImage(existing, req),
		IsPublished:     req.IsPublished,
		PublishedAt:     req.PublishedAt,
	})
	if errors.Is(err, devlog.ErrPostNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "devlog post not found")
		return
	}
	// A well-formed id for a game or image that does not exist is a caller
	// mistake, not a server fault.
	if dbutil.IsForeignKeyViolation(err) {
		httpapi.WriteError(w, http.StatusBadRequest, "unknown_reference",
			"game_id or og_image_id refers to something that does not exist")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update devlog post")
		return
	}

	if err := h.enqueueRegen(post.ID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	h.cache.Clear()

	httpapi.WriteJSON(w, http.StatusOK, toPostResponse(post))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	post, err := h.resolvePost(r)
	if errors.Is(err, devlog.ErrPostNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "devlog post not found")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load devlog post")
		return
	}

	if err := h.repo.Delete(post.ID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not delete devlog post")
		return
	}

	if err := h.enqueueRegen(post.ID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not enqueue regen")
		return
	}
	h.cache.Clear()

	w.WriteHeader(http.StatusNoContent)
}
