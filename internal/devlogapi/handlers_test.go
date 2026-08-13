package devlogapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/devlog"
	"pixabros/internal/id"
	"pixabros/internal/media"
	"pixabros/internal/ogimage"
	"pixabros/internal/storage"
)

func setup(t *testing.T) (*Handlers, *devlog.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	repo := devlog.NewRepo(conn)
	// A real image store, writing into a temporary directory: the generated
	// preview is part of what creating a post does, so stubbing it out would
	// test a code path that never runs in production.
	og := ogimage.NewStore(media.NewRepo(conn), storage.NewLocalDisk(t.TempDir(), ""), conn)
	return NewHandlers(repo, conn, og), repo, conn
}

func post(t *testing.T, h *Handlers, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/devlog", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	return rec
}

func put(t *testing.T, h *Handlers, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/devlog/"+key, bytes.NewReader(raw))
	req.SetPathValue("id", key)
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	return rec
}

func TestCreate_RequiresATitle(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, body := range []map[string]any{{}, {"title": "   "}} {
		if rec := post(t, handlers, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: status = %d, want %d", body, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestCreate_RejectsAMalformedDate(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, date := range []string{"12/08/2026", "2026-8-1", "yesterday"} {
		body := map[string]any{"title": "Post", "published_at": date}
		if rec := post(t, handlers, body); rec.Code != http.StatusBadRequest {
			t.Errorf("date %q: status = %d, want %d", date, rec.Code, http.StatusBadRequest)
		}
	}
}

// An empty date is not an error: the repo stamps one on first publish.
func TestCreate_AllowsAnEmptyDate(t *testing.T) {
	handlers, _, _ := setup(t)

	rec := post(t, handlers, map[string]any{"title": "Post", "published_at": ""})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestCreate_QueuesBothThePostAndTheIndex(t *testing.T) {
	handlers, _, conn := setup(t)

	rec := post(t, handlers, map[string]any{"title": "Kartuş Sistemi"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got postResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Slug != "kartus-sistemi" {
		t.Errorf("Slug = %q, want %q", got.Slug, "kartus-sistemi")
	}

	// A post has a page of its own as well as appearing on the index, so both
	// tags have to be invalidated.
	for _, tag := range []string{regenTagFor(got.ID), listRegenTag} {
		var count int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, tag).Scan(&count); err != nil {
			t.Fatalf("count regen jobs: %v", err)
		}
		if count != 1 {
			t.Errorf("regen jobs for %q = %d, want 1", tag, count)
		}
	}
}

// The admin addresses posts by id, but a link built from a public URL uses the
// slug, so both have to resolve.
func TestGet_ResolvesByIDAndBySlug(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Hello World"})

	for _, key := range []string{created.ID, created.Slug} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog/"+key, nil)
		req.SetPathValue("id", key)
		rec := httptest.NewRecorder()
		handlers.Get(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("key %q: status = %d, want %d", key, rec.Code, http.StatusOK)
		}
		var got postResponse
		json.Unmarshal(rec.Body.Bytes(), &got)
		if got.ID != created.ID {
			t.Errorf("key %q resolved to %q, want %q", key, got.ID, created.ID)
		}
	}
}

func TestGet_UnknownPostIsNotFound(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog/nope", nil)
	req.SetPathValue("id", "nope")
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdate_RejectsMalformedReferences(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	for _, field := range []string{"game_id", "og_image_id"} {
		body := map[string]any{"title": "Post", field: "nope"}
		if rec := put(t, handlers, created.ID, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", field, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestUpdate_UnknownReferenceIsABadRequest(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	body := map[string]any{"title": "Post", "game_id": "aaaaaaaaaaaaaaaaaaaaaaaa"}
	rec := put(t, handlers, created.ID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var parsed struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &parsed)
	if parsed.Error.Code != "unknown_reference" {
		t.Errorf("error.code = %q, want %q", parsed.Error.Code, "unknown_reference")
	}
}

func TestDelete_RemovesPost(t *testing.T) {
	handlers, repo, _ := setup(t)
	created, _ := repo.Create(devlog.CreateInput{Title: "Post"})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/devlog/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if list, _ := repo.List("", false); len(list) != 0 {
		t.Errorf("posts remaining = %d, want 0", len(list))
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, query := range []string{"?sort=password_hash", "?sort=title&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestList_ReturnsAnEmptyArrayNotNull(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/devlog", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("empty list body = %q, want %q", body, "[]\n")
	}
}

// A post without a preview shares as a bare link, so one is drawn from the
// title at creation.
func TestCreate_GeneratesAnOpenGraphImage(t *testing.T) {
	handlers, repo, conn := setup(t)

	rec := post(t, handlers, map[string]any{
		"title": "Fifty Two Days", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}

	list, err := repo.List("published_at", true)
	if err != nil || len(list) != 1 {
		t.Fatalf("List() = %d posts, err = %v", len(list), err)
	}
	if list[0].OGImageID == nil {
		t.Fatal("the post has no preview image")
	}

	var path string
	if err := conn.QueryRow(
		`SELECT path FROM media WHERE id = ?;`, *list[0].OGImageID,
	).Scan(&path); err != nil {
		t.Fatalf("look up the image: %v", err)
	}
	if !ogimage.IsGenerated(path) {
		t.Errorf("image path = %q, want one marked as generated", path)
	}
}

// Renaming a post has to redraw its preview, and take the old one with it --
// otherwise every rename leaves another orphaned picture behind.
func TestUpdate_RedrawsThePreviewWhenTheTitleChanges(t *testing.T) {
	handlers, repo, conn := setup(t)

	post(t, handlers, map[string]any{
		"title": "First Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	list, _ := repo.List("published_at", true)
	original := list[0]

	put(t, handlers, original.ID, map[string]any{
		"title": "Second Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})

	updated, err := repo.FindByID(original.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.OGImageID == nil {
		t.Fatal("the renamed post lost its preview")
	}
	if *updated.OGImageID == *original.OGImageID {
		t.Error("the preview still shows the old title")
	}

	var leftovers int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM media WHERE id = ?;`, *original.OGImageID,
	).Scan(&leftovers); err != nil {
		t.Fatalf("count old image: %v", err)
	}
	if leftovers != 0 {
		t.Error("the previous preview was left behind")
	}
}

// An admin who uploads their own picture has overridden the default, and a
// title edit must not throw that away.
func TestUpdate_KeepsAnUploadedImage(t *testing.T) {
	handlers, repo, conn := setup(t)

	post(t, handlers, map[string]any{
		"title": "First Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	list, _ := repo.List("published_at", true)
	original := list[0]

	uploaded := id.New()
	if _, err := conn.Exec(
		`INSERT INTO media (id, path, width, height) VALUES (?, 'media/og_image/2026-chosen.webp', 1200, 630);`,
		uploaded,
	); err != nil {
		t.Fatalf("seed uploaded image: %v", err)
	}
	put(t, handlers, original.ID, map[string]any{
		"title": "First Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20", "og_image_id": uploaded,
	})

	put(t, handlers, original.ID, map[string]any{
		"title": "Renamed", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})

	updated, _ := repo.FindByID(original.ID)
	if updated.OGImageID == nil || *updated.OGImageID != uploaded {
		t.Error("the uploaded image was replaced by a generated one")
	}
}

// seedGame inserts the little a card needs: the game's name.
func seedGame(t *testing.T, conn *sql.DB, title string) string {
	t.Helper()
	gameID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO games (id, slug, title) VALUES (?, ?, ?);`, gameID, title, title,
	); err != nil {
		t.Fatalf("seed game %q: %v", title, err)
	}
	return gameID
}

// The panel's edit form sends the post's current picture back with every save,
// so a rename arrives with og_image_id already filled in. Reading that as "the
// admin chose this picture" skipped the redraw entirely and left every renamed
// post sharing under its old title.
func TestUpdate_RedrawsWhenThePanelEchoesTheCurrentImage(t *testing.T) {
	handlers, repo, conn := setup(t)

	post(t, handlers, map[string]any{
		"title": "First Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	list, _ := repo.List("published_at", true)
	original := list[0]
	if original.OGImageID == nil {
		t.Fatal("the new post has no preview to echo back")
	}

	put(t, handlers, original.ID, map[string]any{
		"title": "Second Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20", "og_image_id": *original.OGImageID,
	})

	updated, _ := repo.FindByID(original.ID)
	if updated.OGImageID == nil {
		t.Fatal("the renamed post lost its preview")
	}
	if *updated.OGImageID == *original.OGImageID {
		t.Error("the preview still shows the old title")
	}

	var leftovers int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM media WHERE id = ?;`, *original.OGImageID,
	).Scan(&leftovers); err != nil {
		t.Fatalf("count old image: %v", err)
	}
	if leftovers != 0 {
		t.Error("the previous preview was left behind")
	}
}

// The card names the game beside the studio's mark, so moving a post to another
// game dates the picture just as a rename does.
func TestUpdate_RedrawsWhenTheGameChanges(t *testing.T) {
	handlers, repo, conn := setup(t)

	post(t, handlers, map[string]any{
		"title": "Steady Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	list, _ := repo.List("published_at", true)
	original := list[0]

	put(t, handlers, original.ID, map[string]any{
		"title": "Steady Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20", "og_image_id": *original.OGImageID,
		"game_id": seedGame(t, conn, "Dungrid Tactics"),
	})

	updated, _ := repo.FindByID(original.ID)
	if updated.OGImageID == nil || *updated.OGImageID == *original.OGImageID {
		t.Error("the preview does not name the game the post moved to")
	}
}

// An ordinary save that changes neither the title nor the game must reuse the
// picture. Redrawing on every save would churn a new file each time.
func TestUpdate_KeepsThePreviewWhenTheCardWouldNotChange(t *testing.T) {
	handlers, repo, conn := setup(t)

	gameID := seedGame(t, conn, "Dungrid Tactics")
	post(t, handlers, map[string]any{
		"title": "Steady Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20",
	})
	list, _ := repo.List("published_at", true)
	attached := put(t, handlers, list[0].ID, map[string]any{
		"title": "Steady Name", "content_markdown": "Body", "is_published": true,
		"published_at": "2026-04-20", "og_image_id": *list[0].OGImageID, "game_id": gameID,
	})
	if attached.Code != http.StatusOK {
		t.Fatalf("attach game: status = %d", attached.Code)
	}
	original, _ := repo.FindByID(list[0].ID)

	put(t, handlers, original.ID, map[string]any{
		"title": "Steady Name", "content_markdown": "Edited body", "is_published": true,
		"published_at": "2026-04-20", "og_image_id": *original.OGImageID, "game_id": gameID,
	})

	updated, _ := repo.FindByID(original.ID)
	if updated.OGImageID == nil || *updated.OGImageID != *original.OGImageID {
		t.Error("an edit that does not touch the card redrew it anyway")
	}
}
