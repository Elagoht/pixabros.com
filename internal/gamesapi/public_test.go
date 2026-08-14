package gamesapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pixabros/internal/games"
)

// buildResponse mirrors the payload the download script parses.
type buildResponse struct {
	Version string `json:"version"`
	Bytes   int64  `json:"bytes"`
	Files   []struct {
		Path  string `json:"path"`
		Bytes int64  `json:"bytes"`
	} `json:"files"`
}

// getBuild drives the handler through a mux, so {slug} is populated the same
// way the real router populates it.
func getBuild(t *testing.T, repo *games.Repo, slug string) *http.Response {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/games/{slug}/build", NewPublicHandlers(repo).Build)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/games/"+slug+"/build", nil))
	return recorder.Result()
}

func TestPublicBuild_ServesTheManifest(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, err := repo.Create(games.CreateInput{Title: "Covered", IsPublished: true, Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, "/data/games/covered", games.BuildInfo{
		Version:   "a1b2c3d4e5f60718",
		Bytes:     46137344,
		FilesJSON: `[{"path":"index.html","bytes":12873}]`,
	}); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	response := getBuild(t, repo, game.Slug)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var body buildResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Version != "a1b2c3d4e5f60718" || body.Bytes != 46137344 {
		t.Errorf("version/bytes = %q/%d", body.Version, body.Bytes)
	}
	if len(body.Files) != 1 || body.Files[0].Path != "index.html" || body.Files[0].Bytes != 12873 {
		t.Errorf("files = %+v", body.Files)
	}

	// The version in here decides whether a held download is stale. A copy the
	// browser kept on its own would report the old version after a deploy, and
	// the visitor would go on playing a build the site has replaced.
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
}

// A draft's build is not public, and neither is a published game that has no
// build: both would advertise a download that cannot be completed.
func TestPublicBuild_HidesDraftsAndBuildlessGames(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)

	draft, err := repo.Create(games.CreateInput{Title: "Draft", Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(draft.ID, "/data/games/draft", games.BuildInfo{
		Version: "a1b2c3d4e5f60718", Bytes: 1, FilesJSON: `[{"path":"index.html","bytes":1}]`,
	}); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	buildless, err := repo.Create(games.CreateInput{Title: "Buildless", IsPublished: true, Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for name, slug := range map[string]string{
		"draft":     draft.Slug,
		"buildless": buildless.Slug,
		"unknown":   "no-such-game",
	} {
		if got := getBuild(t, repo, slug).StatusCode; got != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", name, got)
		}
	}
}
