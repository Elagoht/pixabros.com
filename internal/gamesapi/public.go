package gamesapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
)

// PublicHandlers serve the game data the public site's scripts need. Nothing
// here requires a session: it is the same data the rendered pages already
// carry, in the shape a script can read.
type PublicHandlers struct {
	repo *games.Repo
}

func NewPublicHandlers(repo *games.Repo) *PublicHandlers {
	return &PublicHandlers{repo: repo}
}

// buildBody is the manifest of a playable build. Per-file byte counts are
// carried because the download's progress is read from them.
type buildBody struct {
	Version string      `json:"version"`
	Bytes   int64       `json:"bytes"`
	Files   []buildFile `json:"files"`
}

type buildFile struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// Build answers with what a game's playable build is made of, so the offline
// download can state the size before spending it and fetch each file by name.
//
// A draft and a game with no build are both 404: advertising a download that
// cannot be completed is worse than admitting there is nothing to download.
func (h *PublicHandlers) Build(w http.ResponseWriter, r *http.Request) {
	// The version in this answer is what decides whether a held download is
	// stale. A heuristically cached copy -- there is no validator here to make
	// no-cache enough -- would report the old version after a deploy, and the
	// visitor would go on playing a build the site has replaced.
	w.Header().Set("Cache-Control", "no-store")

	game, err := h.repo.FindBySlug(r.PathValue("slug"))
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such game")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}
	if !game.IsPublished || !game.IsBrowserPlayable || game.BuildVersion == "" {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such game")
		return
	}

	var files []buildFile
	if err := json.Unmarshal([]byte(game.BuildFilesJSON), &files); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not read the build manifest")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, buildBody{
		Version: game.BuildVersion,
		Bytes:   game.BuildBytes,
		Files:   files,
	})
}
