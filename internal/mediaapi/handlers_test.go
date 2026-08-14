package mediaapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/id"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func setupMediaHandlers(t *testing.T) (*Handlers, *media.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	repo := media.NewRepo(conn)
	// Root = the bare data dir and baseURL = "" mirrors main.go exactly: the
	// stored key already carries the "media/" namespace.
	files := storage.NewLocalDisk(t.TempDir(), "")
	return NewHandlers(repo, files, conn), repo, conn
}

func TestGet_ReturnsMediaWithPublicURL(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/cartridge_art/2026-abc123.webp", 400, 560)
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got mediaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	want := mediaResponse{ID: saved.ID, URL: "/media/cartridge_art/2026-abc123.webp", Width: 400, Height: 560}
	if got != want {
		t.Errorf("Get() = %+v, want %+v", got, want)
	}
}

func TestGet_UnknownIDNotFound(t *testing.T) {
	handlers, _, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestList_AnnotatesUsageAndCountsOrphans(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)

	used, err := repo.Create("media/cartridge_art/used.webp", 400, 560)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := repo.Create("media/screenshot/loose.webp", 1280, 720); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id) VALUES ('g1', 'pixel', 'Pixel Quest', ?);`,
		used.ID,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got libraryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(got.Items))
	}
	// The count is reported separately because picking it out of the list by
	// eye is exactly the tedious part.
	if got.Orphaned != 1 {
		t.Errorf("orphaned = %d, want 1", got.Orphaned)
	}

	for _, item := range got.Items {
		if item.ID != used.ID {
			continue
		}
		if len(item.Usages) != 1 || item.Usages[0].Module != "games" ||
			item.Usages[0].Label != "Pixel Quest" {
			t.Errorf("usages = %+v, want one games usage labelled with the title", item.Usages)
		}
	}
}

// The UI iterates usages, so an unused image must carry an empty array rather
// than a JSON null.
func TestList_UnusedImageHasAnEmptyUsageArray(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	if _, err := repo.Create("media/screenshot/loose.webp", 1280, 720); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/media", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); !strings.Contains(body, `"usages":[]`) {
		t.Errorf("body = %s, want an empty usages array", body)
	}
}

func TestUpdate_SetsAltText(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/screenshot/a.webp", 1280, 720)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if saved.AltText != "" {
		t.Fatalf("AltText = %q, want empty on upload", saved.AltText)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{"alt_text":"Two players under a falling block"}`)))
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	after, _ := repo.FindByID(saved.ID)
	if after.AltText != "Two players under a falling block" {
		t.Errorf("AltText = %q, want the submitted text", after.AltText)
	}
}

// A pointer field is what lets alt text be cleared: a plain string could not
// tell "" apart from "the field was omitted".
func TestUpdate_CanClearAltTextButNotOmitIt(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, _ := repo.Create("media/screenshot/a.webp", 1280, 720)
	repo.SetAltText(saved.ID, "something")

	req := httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{"alt_text":""}`)))
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("clearing: status = %d, want %d", rec.Code, http.StatusOK)
	}
	after, _ := repo.FindByID(saved.ID)
	if after.AltText != "" {
		t.Errorf("AltText = %q, want it cleared", after.AltText)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/admin/media/"+saved.ID,
		bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", saved.ID)
	rec = httptest.NewRecorder()
	handlers.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("omitting: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDelete_RemovesAnUnusedImageAndItsFile(t *testing.T) {
	handlers, repo, _ := setupMediaHandlers(t)
	saved, err := repo.Create("media/screenshot/loose.webp", 1280, 720)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, err := repo.FindByID(saved.ID); !errors.Is(err, media.ErrMediaNotFound) {
		t.Errorf("FindByID() after delete error = %v, want ErrMediaNotFound", err)
	}
}

// Deleting a referenced image would blank a game's artwork -- and for a
// screenshot, whose media_id cascades, remove the screenshot row outright. It
// is refused rather than quietly breaking a page.
func TestDelete_RefusesAnImageThatIsStillUsed(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)
	saved, _ := repo.Create("media/cartridge_art/used.webp", 400, 560)
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cartridge_art_id) VALUES ('g1', 'pixel', 'Pixel Quest', ?);`,
		saved.ID,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/"+saved.ID, nil)
	req.SetPathValue("id", saved.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "still_in_use" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "still_in_use")
	}
	if _, err := repo.FindByID(saved.ID); err != nil {
		t.Errorf("the refused delete removed the row anyway: %v", err)
	}
}

func TestDelete_UnknownImageIsNotFound(t *testing.T) {
	handlers, _, _ := setupMediaHandlers(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/media/aaaaaaaaaaaaaaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// Changing an image's alt text changes what every page showing it says, so
// those pages have to be rebuilt. Nothing else notices: a page is only rebuilt
// when its tag is enqueued.
func TestUpdate_RebuildsEveryPageShowingTheImage(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)

	mediaID := seedImage(t, repo, "media/cd_cover_art/2026-cover.webp")
	gameID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title, cd_cover_art_id) VALUES (?, 'a-game', 'A Game', ?);`,
		gameID, mediaID,
	); err != nil {
		t.Fatalf("seed game: %v", err)
	}

	if rec := putAltText(t, handlers, mediaID, "A cover"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	queued := pendingTags(t, conn)
	// The listing shows the cover on a card; the game's own page shows it in
	// full. Both are now wrong.
	for _, want := range []string{"game:list", "game:" + gameID} {
		if !queued[want] {
			t.Errorf("no rebuild queued for %q, queued: %v", want, keysOf(queued))
		}
	}
}

// An image nothing points at yet -- just uploaded, not attached -- appears on
// no page, so editing it must not queue work for pages it is not on.
func TestUpdate_QueuesNothingForAnUnusedImage(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)

	mediaID := seedImage(t, repo, "media/og_image/2026-loose.webp")
	if rec := putAltText(t, handlers, mediaID, "Nowhere yet"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if queued := pendingTags(t, conn); len(queued) != 0 {
		t.Errorf("queued %v for an image on no page", keysOf(queued))
	}
}

// The team is on the homepage and members have no pages of their own, so a
// member's avatar rebuilds the listing and nothing else.
func TestUpdate_RebuildsTheListingForAMemberAvatar(t *testing.T) {
	handlers, repo, conn := setupMediaHandlers(t)

	mediaID := seedImage(t, repo, "media/avatar/2026-face.webp")
	if _, err := conn.Exec(
		`INSERT INTO members (id, name, links_json, avatar_id) VALUES (?, 'Someone', '[]', ?);`,
		id.New(), mediaID,
	); err != nil {
		t.Fatalf("seed member: %v", err)
	}

	if rec := putAltText(t, handlers, mediaID, "A face"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	queued := pendingTags(t, conn)
	if !queued["member:list"] {
		t.Errorf("no rebuild queued for member:list, queued: %v", keysOf(queued))
	}
}

func putAltText(t *testing.T, h *Handlers, mediaID, alt string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"alt_text": alt})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/media/"+mediaID, bytes.NewReader(body))
	req.SetPathValue("id", mediaID)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

func pendingTags(t *testing.T, conn *sql.DB) map[string]bool {
	t.Helper()
	rows, err := conn.Query(`SELECT tag FROM regen_jobs WHERE status = 'pending';`)
	if err != nil {
		t.Fatalf("read the queue: %v", err)
	}
	defer rows.Close()

	tags := map[string]bool{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			t.Fatalf("scan tag: %v", err)
		}
		tags[tag] = true
	}
	return tags
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func seedImage(t *testing.T, repo *media.Repo, path string) string {
	t.Helper()
	saved, err := repo.Create(path, 400, 400)
	if err != nil {
		t.Fatalf("seed image: %v", err)
	}
	return saved.ID
}
