package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"pixabros/internal/auth"
	"pixabros/internal/db"
)

func TestRouter_LoginAndSingleOriginServing(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	hash, err := auth.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, "furkan", hash); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	adminDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(adminDir, "index.html"), []byte("<h1>admin</h1>"), 0o644); err != nil {
		t.Fatalf("write admin index.html: %v", err)
	}
	playDir := t.TempDir()
	publicDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(publicDir, "index.html"), []byte("<h1>public</h1>"), 0o644); err != nil {
		t.Fatalf("write public index.html: %v", err)
	}

	handler := New(Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		AdminUIDir: adminDir,
		PlayDir:    playDir,
		PublicDir:  publicDir,
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": "furkan", "password": "s3cret-password"})
	loginResp, err := srv.Client().Post(srv.URL+"/api/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request error = %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusOK)
	}
	if len(loginResp.Cookies()) == 0 {
		t.Fatal("expected a session cookie after login")
	}

	adminResp, err := srv.Client().Get(srv.URL + "/I-am-a-pixabro/")
	if err != nil {
		t.Fatalf("admin UI request error = %v", err)
	}
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("admin UI status = %d, want %d", adminResp.StatusCode, http.StatusOK)
	}

	publicResp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("public request error = %v", err)
	}
	if publicResp.StatusCode != http.StatusOK {
		t.Fatalf("public status = %d, want %d", publicResp.StatusCode, http.StatusOK)
	}
}
