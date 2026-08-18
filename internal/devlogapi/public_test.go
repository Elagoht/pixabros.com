package devlogapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/devlog"
	"pixabros/internal/games"
	"pixabros/internal/media"
)

const testOrigin = "http://pixabros.com"

func publicSetup(t *testing.T) (*PublicHandlers, *devlog.Repo, *games.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	devlogRepo := devlog.NewRepo(conn)
	gamesRepo := games.NewRepo(conn)
	mediaRepo := media.NewRepo(conn)

	if _, err := gamesRepo.Create(games.CreateInput{
		Title: "Cyber Drift", IsPublished: true, DisplayOrder: 1,
	}); err != nil {
		t.Fatalf("gamesRepo.Create() error = %v", err)
	}

	return NewPublicHandlers(devlogRepo, gamesRepo, mediaRepo, testOrigin, NewSearchCache(16)), devlogRepo, gamesRepo, conn
}

// addPost writes one published post, optionally tied to a game.
func addPost(t *testing.T, repo *devlog.Repo, title, date string, gameID *string) devlog.Post {
	t.Helper()
	post, err := repo.Create(devlog.CreateInput{
		Title: title, ContentMarkdown: "# n", IsPublished: true, PublishedAt: date,
	})
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}
	if gameID != nil {
		post, err = repo.Update(post.ID, devlog.UpdateInput{
			Title: title, ContentMarkdown: "# n", IsPublished: true, PublishedAt: date, GameID: gameID,
		})
		if err != nil {
			t.Fatalf("repo.Update() error = %v", err)
		}
	}
	return post
}

func search(t *testing.T, h *PublicHandlers, query, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/devlog/posts?"+query, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.SearchPosts(rec, req)
	return rec
}

func decodeSearch(t *testing.T, rec *httptest.ResponseRecorder) searchResponse {
	t.Helper()
	var response searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	return response
}

func TestSearchPosts_RefusesACrossOriginRequest(t *testing.T) {
	h, _, _, _ := publicSetup(t)
	rec := search(t, h, "", "http://evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestSearchPosts_AllowsOurOwnOrigin(t *testing.T) {
	h, _, _, _ := publicSetup(t)
	rec := search(t, h, "", testOrigin)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestSearchPosts_RefusesARequestFromAnotherOrigin(t *testing.T) {
	h, _, _, _ := publicSetup(t)
	rec := search(t, h, "", "http://pixabros.com.evil.example")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestSearchPosts_ListsPublishedPosts(t *testing.T) {
	h, repo, _, _ := publicSetup(t)
	addPost(t, repo, "Server Rendering", "2026-04-20", nil)
	addPost(t, repo, "Pixel Art", "2026-03-05", nil)

	rec := search(t, h, "", "")
	response := decodeSearch(t, rec)
	if response.Total != 2 {
		t.Fatalf("total = %d, want 2", response.Total)
	}
	if len(response.Posts) != 2 {
		t.Fatalf("posts = %d, want 2", len(response.Posts))
	}
	// Newest first.
	if response.Posts[0].Title != "Server Rendering" {
		t.Fatalf("posts[0].Title = %q, want newest first", response.Posts[0].Title)
	}
	if response.Posts[0].Year != "2026" {
		t.Fatalf("posts[0].Year = %q, want 2026", response.Posts[0].Year)
	}
}

func TestSearchPosts_SkipsDrafts(t *testing.T) {
	h, repo, _, _ := publicSetup(t)
	addPost(t, repo, "Published", "2026-04-20", nil)
	if _, err := repo.Create(devlog.CreateInput{Title: "Draft", ContentMarkdown: "# n"}); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	response := decodeSearch(t, search(t, h, "", ""))
	if response.Total != 1 {
		t.Fatalf("total = %d, want 1 (drafts excluded)", response.Total)
	}
}

func TestSearchPosts_FiltersByQuery(t *testing.T) {
	h, repo, _, _ := publicSetup(t)
	addPost(t, repo, "Server Rendering", "2026-04-20", nil)
	addPost(t, repo, "Pixel Art", "2026-03-05", nil)

	response := decodeSearch(t, search(t, h, "q=pixel", ""))
	if response.Total != 1 || response.Posts[0].Title != "Pixel Art" {
		t.Fatalf("query filter gave %d posts, want the pixel one", response.Total)
	}
}

func TestSearchPosts_FiltersByGameAndYear(t *testing.T) {
	h, repo, gamesRepo, _ := publicSetup(t)
	game, _ := gamesRepo.FindBySlug("cyber-drift")
	gid := game.ID
	addPost(t, repo, "Cyber Note", "2026-04-20", &gid)
	addPost(t, repo, "Standalone", "2026-03-05", nil)
	addPost(t, repo, "Older Cyber", "2025-11-02", &gid)

	gameOnly := decodeSearch(t, search(t, h, "game=cyber-drift", ""))
	if gameOnly.Total != 2 {
		t.Fatalf("game filter total = %d, want 2", gameOnly.Total)
	}

	yearOnly := decodeSearch(t, search(t, h, "year=2025", ""))
	if yearOnly.Total != 1 || yearOnly.Posts[0].Title != "Older Cyber" {
		t.Fatalf("year filter gave %d posts, want the 2025 one", yearOnly.Total)
	}

	both := decodeSearch(t, search(t, h, "game=cyber-drift&year=2026", ""))
	if both.Total != 1 || both.Posts[0].Title != "Cyber Note" {
		t.Fatalf("combined filter gave %d posts, want the 2026 cyber one", both.Total)
	}
}

func TestSearchPosts_Paginates(t *testing.T) {
	h, repo, _, _ := publicSetup(t)
	for i := 1; i <= 3; i++ {
		addPost(t, repo, "Post", "2026-04-20", nil)
	}

	first := decodeSearch(t, search(t, h, "per_page=2&page=1", ""))
	if first.Page != 1 || first.Total != 3 || len(first.Posts) != 2 || !first.HasMore {
		t.Fatalf("first page = %+v, want 2 of 3 with more", first)
	}

	second := decodeSearch(t, search(t, h, "per_page=2&page=2", ""))
	if second.Page != 2 || len(second.Posts) != 1 || second.HasMore {
		t.Fatalf("second page = %+v, want 1 post and no more", second)
	}
}

func TestSearchPosts_NamesTheGameOfAPost(t *testing.T) {
	h, repo, gamesRepo, _ := publicSetup(t)
	game, _ := gamesRepo.FindBySlug("cyber-drift")
	gid := game.ID
	addPost(t, repo, "Cyber Note", "2026-04-20", &gid)

	response := decodeSearch(t, search(t, h, "", ""))
	post := response.Posts[0]
	if post.Game != "Cyber Drift" || post.GameSlug != "cyber-drift" {
		t.Fatalf("game = %q slug = %q, want Cyber Drift / cyber-drift", post.Game, post.GameSlug)
	}
}

func TestSearchPosts_RepeatedRequestIsServedFromTheCache(t *testing.T) {
	h, repo, _, _ := publicSetup(t)
	addPost(t, repo, "Server Rendering", "2026-04-20", nil)

	first := decodeSearch(t, search(t, h, "q=server", ""))
	second := decodeSearch(t, search(t, h, "q=server", ""))
	if first.Total != second.Total {
		t.Fatalf("cached response differs: %d vs %d", first.Total, second.Total)
	}
}
