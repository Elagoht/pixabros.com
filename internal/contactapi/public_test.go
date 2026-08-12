package contactapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixabros/internal/contact"
	"pixabros/internal/db"
)

const longEnough = "This is a real message with more than one hundred characters in it, written so that validation lets it through."

func setupPublic(t *testing.T) (*PublicHandlers, *contact.Repo, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	repo := contact.NewRepo(conn)
	return NewPublicHandlers(repo), repo, conn
}

func postJSON(t *testing.T, h *PublicHandlers, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(string(encoded)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:1234"
	rec := httptest.NewRecorder()
	h.Submit(rec, req)
	return rec
}

func countSubmissions(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM contact_submissions;`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	return count
}

func TestSubmit_StoresAValidMessage(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	rec := postJSON(t, handlers, map[string]interface{}{
		"name": "Someone", "subject": "Hello", "email": "a@example.com",
		"message": longEnough,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if countSubmissions(t, conn) != 1 {
		t.Error("the message was not stored")
	}
}

// The spec sets a hundred-character floor to make drive-by spam more effort
// than it is worth.
func TestSubmit_RejectsAShortMessage(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	rec := postJSON(t, handlers, map[string]interface{}{
		"subject": "Hi", "message": "too short",
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if countSubmissions(t, conn) != 0 {
		t.Error("a too-short message was stored anyway")
	}
}

func TestSubmit_RequiresASubject(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	rec := postJSON(t, handlers, map[string]interface{}{"message": longEnough})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if countSubmissions(t, conn) != 0 {
		t.Error("a message with no subject was stored")
	}
}

// Asking to be called back with no way to be reached cannot be honoured.
func TestSubmit_RequiresContactDetailsForACallBack(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	rec := postJSON(t, handlers, map[string]interface{}{
		"subject": "Call me", "message": longEnough, "wants_callback": true,
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if countSubmissions(t, conn) != 0 {
		t.Error("a call-back request with no contact details was stored")
	}

	// With an email it goes through.
	ok := postJSON(t, handlers, map[string]interface{}{
		"subject": "Call me", "message": longEnough,
		"wants_callback": true, "email": "a@example.com",
	})
	if ok.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 once an email is given", ok.Code)
	}
}

// A caught bot is answered exactly like a success: telling it otherwise only
// teaches whoever wrote it.
func TestSubmit_SilentlyDropsAHoneypotSubmission(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	rec := postJSON(t, handlers, map[string]interface{}{
		"subject": "Hello", "message": longEnough, "website": "http://spam.example",
	})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 so the bot learns nothing", rec.Code)
	}
	if countSubmissions(t, conn) != 0 {
		t.Error("a honeypot submission was stored")
	}
}

func TestSubmit_RateLimitsByAddress(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	handlers.now = func() time.Time { return now }

	first := postJSON(t, handlers, map[string]interface{}{"subject": "One", "message": longEnough})
	if first.Code != http.StatusOK {
		t.Fatalf("first submission status = %d, want 200", first.Code)
	}

	second := postJSON(t, handlers, map[string]interface{}{"subject": "Two", "message": longEnough})
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second submission status = %d, want 429", second.Code)
	}
	if countSubmissions(t, conn) != 1 {
		t.Error("the rate-limited submission was stored anyway")
	}

	// Once the window has passed the same address is welcome again.
	now = now.Add(RateLimit + time.Second)
	third := postJSON(t, handlers, map[string]interface{}{"subject": "Three", "message": longEnough})
	if third.Code != http.StatusOK {
		t.Errorf("status after the window = %d, want 200", third.Code)
	}
}

// The form must work with JavaScript switched off, which means a plain form
// post has to be answered with a redirect rather than raw JSON.
func TestSubmit_RedirectsAPlainFormPost(t *testing.T) {
	handlers, _, conn := setupPublic(t)

	form := url.Values{}
	form.Set("subject", "Hello")
	form.Set("message", longEnough)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "203.0.113.9:1234"
	rec := httptest.NewRecorder()
	handlers.Submit(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/contact/sent" {
		t.Errorf("Location = %q, want /contact/sent", location)
	}
	if countSubmissions(t, conn) != 1 {
		t.Error("the form post was not stored")
	}
}

func TestSubmit_RecordsTheForwardedAddress(t *testing.T) {
	handlers, repo, _ := setupPublic(t)

	req := httptest.NewRequest(http.MethodPost, "/api/contact",
		strings.NewReader(`{"subject":"Hi","message":"`+longEnough+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CF-Connecting-IP", "198.51.100.4")
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handlers.Submit(rec, req)

	list, err := repo.List("created_at", true)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("submissions = %d, want 1", len(list))
	}
	// Cloudflare fronts the site, so the socket address is its edge, not the
	// visitor's.
	if list[0].IPAddress != "198.51.100.4" {
		t.Errorf("IPAddress = %q, want the forwarded address", list[0].IPAddress)
	}
}
