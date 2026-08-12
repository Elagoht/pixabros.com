package contactapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/contact"
	"pixabros/internal/db"
	"pixabros/internal/id"
)

func setup(t *testing.T) (*Handlers, *contact.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	repo := contact.NewRepo(conn)
	return NewHandlers(repo), repo, conn
}

func seed(t *testing.T, conn *sql.DB, subject, createdAt string) string {
	t.Helper()
	submissionID := id.New()
	if _, err := conn.Exec(
		`INSERT INTO contact_submissions
			(id, subject, phone, email, message, wants_callback, ip_address, created_at)
		 VALUES (?, ?, '+900000000', 'a@example.com', 'Hello', 1, '203.0.113.7', ?);`,
		submissionID, subject, createdAt,
	); err != nil {
		t.Fatalf("seed submission: %v", err)
	}
	return submissionID
}

func TestList_ReturnsSubmissionsAndUnreadCount(t *testing.T) {
	handlers, repo, conn := setup(t)
	first := seed(t, conn, "One", "2026-08-01T10:00:00.000Z")
	seed(t, conn, "Two", "2026-08-02T10:00:00.000Z")
	repo.SetRead(first, true)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contact", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(got.Submissions))
	}
	// Newest first.
	if got.Submissions[0].Subject != "Two" {
		t.Errorf("first subject = %q, want %q", got.Submissions[0].Subject, "Two")
	}
	// The count is reported separately so the UI does not have to derive it
	// from a list it may have re-sorted or filtered.
	if got.Unread != 1 {
		t.Errorf("unread = %d, want 1", got.Unread)
	}
}

func TestList_EmptyInboxIsAnArrayNotNull(t *testing.T) {
	handlers, _, _ := setup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contact", nil)
	rec := httptest.NewRecorder()
	handlers.List(rec, req)

	if body := rec.Body.String(); !bytes.Contains([]byte(body), []byte(`"submissions":[]`)) {
		t.Errorf("empty inbox body = %q, want an empty submissions array", body)
	}
}

func TestList_RejectsUnknownSort(t *testing.T) {
	handlers, _, _ := setup(t)

	for _, query := range []string{"?sort=ip_address", "?sort=subject&dir=sideways"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/contact"+query, nil)
		rec := httptest.NewRecorder()
		handlers.List(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func setRead(t *testing.T, h *Handlers, submissionID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodPut, "/api/admin/contact/"+submissionID+"/read", bytes.NewReader([]byte(body)),
	)
	req.SetPathValue("id", submissionID)
	rec := httptest.NewRecorder()
	h.SetRead(rec, req)
	return rec
}

func TestSetRead_MarksReadAndUnread(t *testing.T) {
	handlers, _, conn := setup(t)
	submissionID := seed(t, conn, "One", "2026-08-01T10:00:00.000Z")

	rec := setRead(t, handlers, submissionID, `{"is_read":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got submissionResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if !got.IsRead {
		t.Error("is_read = false after marking read")
	}

	// Marking something unread again is how an inbox lets you return to it,
	// which is why the request field is a pointer: a plain bool could not tell
	// "false" apart from "omitted".
	rec = setRead(t, handlers, submissionID, `{"is_read":false}`)
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.IsRead {
		t.Error("is_read = true after marking unread")
	}
}

func TestSetRead_RequiresTheField(t *testing.T) {
	handlers, _, conn := setup(t)
	submissionID := seed(t, conn, "One", "2026-08-01T10:00:00.000Z")

	if rec := setRead(t, handlers, submissionID, `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSetRead_UnknownSubmissionIsNotFound(t *testing.T) {
	handlers, _, _ := setup(t)

	rec := setRead(t, handlers, "aaaaaaaaaaaaaaaaaaaaaaaa", `{"is_read":true}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGet_ReturnsTheWholeSubmission(t *testing.T) {
	handlers, _, conn := setup(t)
	submissionID := seed(t, conn, "Collaboration", "2026-08-01T10:00:00.000Z")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contact/"+submissionID, nil)
	req.SetPathValue("id", submissionID)
	rec := httptest.NewRecorder()
	handlers.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got submissionResponse
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Message != "Hello" || got.IPAddress != "203.0.113.7" || !got.WantsCallback {
		t.Errorf("Get() = %+v, want the full submission including ip and callback flag", got)
	}
}

// Reading a submission must not silently mark it read: that is an explicit
// action, so an unread item stays unread until it is toggled.
func TestGet_DoesNotMarkAsRead(t *testing.T) {
	handlers, repo, conn := setup(t)
	submissionID := seed(t, conn, "One", "2026-08-01T10:00:00.000Z")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contact/"+submissionID, nil)
	req.SetPathValue("id", submissionID)
	handlers.Get(httptest.NewRecorder(), req)

	after, _ := repo.FindByID(submissionID)
	if after.IsRead {
		t.Error("reading a submission marked it read; that must be explicit")
	}
}

func TestDelete_RemovesSubmission(t *testing.T) {
	handlers, repo, conn := setup(t)
	submissionID := seed(t, conn, "Spam", "2026-08-01T10:00:00.000Z")

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/contact/"+submissionID, nil)
	req.SetPathValue("id", submissionID)
	rec := httptest.NewRecorder()
	handlers.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if list, _ := repo.List("", false); len(list) != 0 {
		t.Errorf("submissions remaining = %d, want 0", len(list))
	}
}
