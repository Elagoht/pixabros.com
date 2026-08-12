package membersapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/members"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func setup(t *testing.T) (*Handlers, *members.Repo, *sql.DB) {
	t.Helper()
	conn := setupTestDB(t)
	repo := members.NewRepo(conn)
	return NewHandlers(repo, conn), repo, conn
}

func regenJobs(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var count int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM regen_jobs WHERE tag = ?;`, regenTag,
	).Scan(&count); err != nil {
		t.Fatalf("count regen jobs: %v", err)
	}
	return count
}

func TestCreate_RequiresAName(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, body := range []string{`{}`, `{"name":"   "}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/members", bytes.NewReader([]byte(body)))
		rec := httptest.NewRecorder()
		handlers.Create(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %s: status = %d, want %d", body, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestCreate_StoresMemberAndQueuesRegen(t *testing.T) {
	handlers, _, conn := setup(t)

	body, _ := json.Marshal(map[string]any{"name": "Furkan", "tags": "code"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/members", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got memberResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "Furkan" {
		t.Errorf("Name = %q, want %q", got.Name, "Furkan")
	}
	if got.IsPublished {
		t.Error("a new member should start unpublished")
	}
	if regenJobs(t, conn) != 1 {
		t.Errorf("regen jobs = %d, want 1", regenJobs(t, conn))
	}
}

func TestUpdate_RejectsAMalformedAvatarID(t *testing.T) {
	handlers, repo, _ := setup(t)
	member, _ := repo.Create(members.CreateInput{Name: "Furkan"})

	body, _ := json.Marshal(map[string]any{"name": "Furkan", "avatar_id": "not-an-id"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/members/"+member.ID, bytes.NewReader(body))
	req.SetPathValue("id", member.ID)
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestUpdate_UnknownMemberIsNotFound(t *testing.T) {
	handlers, _, _ := setup(t)

	body, _ := json.Marshal(map[string]any{"name": "Nobody"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/members/aaaaaaaaaaaaaaaaaaaaaaaa", bytes.NewReader(body))
	req.SetPathValue("id", "aaaaaaaaaaaaaaaaaaaaaaaa")
	rec := httptest.NewRecorder()
	handlers.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDelete_RemovesMemberAndQueuesRegen(t *testing.T) {
	handlers, repo, conn := setup(t)
	member, _ := repo.Create(members.CreateInput{Name: "Furkan"})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/members/"+member.ID, nil)
	req.SetPathValue("id", member.ID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if list, _ := repo.List("", false); len(list) != 0 {
		t.Errorf("members remaining = %d, want 0", len(list))
	}
	// The member was seeded through the repo, so the delete is the only
	// thing that should have queued a regen.
	if got := regenJobs(t, conn); got != 1 {
		t.Errorf("regen jobs = %d, want 1", got)
	}
}

func TestList_SortsByQueryParams(t *testing.T) {
	handlers, repo, _ := setup(t)
	for _, name := range []string{"berk", "Ada", "cem"} {
		if _, err := repo.Create(members.CreateInput{Name: name}); err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
	}

	namesFor := func(t *testing.T, query string) []string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/members"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var got []memberResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		names := make([]string, 0, len(got))
		for _, m := range got {
			names = append(names, m.Name)
		}
		return names
	}

	if got, want := namesFor(t, "?sort=name&dir=asc"), []string{"Ada", "berk", "cem"}; !slices.Equal(got, want) {
		t.Errorf("?sort=name&dir=asc = %v, want %v", got, want)
	}
	if got, want := namesFor(t, "?sort=name&dir=desc"), []string{"cem", "berk", "Ada"}; !slices.Equal(got, want) {
		t.Errorf("?sort=name&dir=desc = %v, want %v", got, want)
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, query := range []string{"?sort=password_hash", "?sort=name&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/members"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestList_ReturnsAnEmptyArrayNotNull(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/members", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	// The admin UI iterates this directly; a JSON null would break it.
	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("empty list body = %q, want %q", body, "[]\n")
	}
}

func TestReorder_AppliesTheGivenOrder(t *testing.T) {
	handlers, repo, _ := setup(t)
	first, _ := repo.Create(members.CreateInput{Name: "first"})
	second, _ := repo.Create(members.CreateInput{Name: "second"})

	body, _ := json.Marshal(reorderRequest{IDs: []string{second.ID, first.ID}})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/members/reorder", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Reorder(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	list, _ := repo.List("", false)
	if list[0].ID != second.ID || list[1].ID != first.ID {
		t.Errorf("order after reorder = %v", []string{list[0].Name, list[1].Name})
	}
}

func TestReorder_RequiresIDs(t *testing.T) {
	handlers, _, _ := setup(t)

	body, _ := json.Marshal(reorderRequest{IDs: nil})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/members/reorder", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.Reorder(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
