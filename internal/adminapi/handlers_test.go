package adminapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
)

func setupHandlers(t *testing.T) (*AuthHandlers, *auth.SessionStore, *sql.DB, int64) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	admins := auth.NewAdminRepo(conn)
	sessions := auth.NewSessionStore(conn)

	hash, err := auth.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	res, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	adminID, _ := res.LastInsertId()

	return NewAuthHandlers(admins, sessions), sessions, conn, adminID
}

func TestLogin_Success(t *testing.T) {
	handlers, _, _, _ := setupHandlers(t)

	body, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName {
		t.Fatalf("expected a %q cookie to be set, got %v", sessionCookieName, cookies)
	}
	if !cookies[0].HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	handlers, _, _, _ := setupHandlers(t)

	body, _ := json.Marshal(map[string]string{"username": "furkan", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()

	handlers.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, err := sessions.Validate(token); err == nil {
		t.Error("session should be invalid after logout")
	}
}

func TestRequireSession_RejectsMissingCookie(t *testing.T) {
	_, sessions, _, _ := setupHandlers(t)

	protected := RequireSession(sessions, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireSession_AllowsValidCookieAndInjectsAdminID(t *testing.T) {
	_, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	var gotAdminID int64
	protected := RequireSession(sessions, func(w http.ResponseWriter, r *http.Request) {
		gotAdminID, _ = AdminIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotAdminID != adminID {
		t.Errorf("adminID in context = %d, want %d", gotAdminID, adminID)
	}
}

func TestChangePassword_Success(t *testing.T) {
	handlers, sessions, conn, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "s3cret-password",
		"new_password":     "new-password-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/change-password", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	req = req.WithContext(withAdminID(req.Context(), adminID))
	rec := httptest.NewRecorder()

	handlers.ChangePassword(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	var newHash string
	if err := conn.QueryRow(`SELECT password_hash FROM admins WHERE id = ?;`, adminID).Scan(&newHash); err != nil {
		t.Fatalf("query updated hash: %v", err)
	}
	if !auth.VerifyPassword(newHash, "new-password-123") {
		t.Error("password was not updated")
	}
}

func TestChangePassword_InvalidatesAllSessionsForAdmin(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	oldToken, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}
	callerToken, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "s3cret-password",
		"new_password":     "new-password-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/change-password", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: callerToken})
	req = req.WithContext(withAdminID(req.Context(), adminID))
	rec := httptest.NewRecorder()

	handlers.ChangePassword(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	if _, err := sessions.Validate(oldToken); err == nil {
		t.Error("old session token should be invalid after password change")
	}
	if _, err := sessions.Validate(callerToken); err == nil {
		t.Error("caller's own session token should also be invalid after password change")
	}
}

func TestChangePassword_TooLongNewPasswordReturnsWeakPassword(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "s3cret-password",
		"new_password":     strings.Repeat("a", 73),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/change-password", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	req = req.WithContext(withAdminID(req.Context(), adminID))
	rec := httptest.NewRecorder()

	handlers.ChangePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var body2 struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body2); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body2.Error.Code != "weak_password" {
		t.Errorf("error.code = %q, want %q", body2.Error.Code, "weak_password")
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "totally-wrong",
		"new_password":     "new-password-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/change-password", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	req = req.WithContext(withAdminID(req.Context(), adminID))
	rec := httptest.NewRecorder()

	handlers.ChangePassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWhoami_Success(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()

	RequireSession(sessions, handlers.Whoami)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Username != "furkan" {
		t.Errorf("username = %q, want %q", body.Username, "furkan")
	}
}

func TestWhoami_Unauthorized(t *testing.T) {
	handlers, sessions, _, _ := setupHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	rec := httptest.NewRecorder()

	RequireSession(sessions, handlers.Whoami)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
