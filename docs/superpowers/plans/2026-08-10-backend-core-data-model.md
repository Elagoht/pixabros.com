# Backend Core & Veri Modeli Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the Go backend skeleton for pixabros.com: SQLite schema for every module, a pluggable storage interface with a local-disk implementation, admin authentication (bcrypt + session cookies), and the single-origin router that will later host the API, admin SPA, game builds, and public site.

**Architecture:** Pure Go (`net/http` + Go 1.22 `ServeMux` pattern routing), `database/sql` with hand-written SQL and embedded migration files, `modernc.org/sqlite` (CGO-free) as the driver. No web framework, no ORM.

**Tech Stack:** Go 1.22+, `modernc.org/sqlite`, `golang.org/x/crypto/bcrypt`.

## Global Constraints

- Framework yok — saf Go (`net/http`, Go 1.22+ `http.ServeMux` pattern routing).
- CGO-free SQLite driver: `modernc.org/sqlite`.
- `database/sql` + el yazımı SQL — ORM veya query builder yok.
- Never use Go's `any` type alias anywhere in this codebase — use `interface{}` or a concrete type instead (user's global CLAUDE.md rule).
- Session auth: `HttpOnly, Secure, SameSite=Strict` cookie; passwords hashed with bcrypt.
- Storage access goes through the `storage.Storage` interface — never call `os.*` directly from handler/business code.
- API error responses use `{"error": {"code": "...", "message": "..."}}`.
- Git commits in this repo: self-committed by the assistant, one-sentence semantic commit messages, no detail/body, no co-author/contributor trailer.

## Scope

This plan (Plan A of three) builds only the shared foundation: full DB schema, storage interface, and admin login/logout/change-password. The following are explicitly **out of scope** here and land in later plans:
- Render/regen cache pipeline, media/WebP processing, game archive extraction, orphan-media sweep, devlog OG auto-generation → **Plan B (Content & Rendering Pipelines)**.
- Admin React SPA, Service Worker, per-module CRUD screens, contact-form rate limiting/honeypot → **Plan C (Admin SPA Shell)**.
- Any per-module page rendering (Landing, Play, Devlog, Awards, Contact) → future per-module plans built on top of Plan A/B/C.

---

## File Structure

```
go.mod
.gitignore
cmd/
  server/
    main.go                     # process entrypoint, wires config+db+router
  admincli/
    main.go                     # `admincli create-admin -username -password`
    main_test.go
internal/
  config/
    config.go                   # env-based Config loader
    config_test.go
  db/
    db.go                       # sql.DB open + pragmas
    db_test.go
    migrate.go                  # embedded migration runner
    migrate_test.go
    migrations/
      0001_admins.sql
      0002_sessions.sql
      0003_media.sql
      0004_site_settings.sql
      0005_homepage_settings.sql
      0006_members.sql
      0007_games.sql
      0008_game_screenshots.sql
      0009_devlog_posts.sql
      0010_awards.sql
      0011_contact_submissions.sql
      0012_rendered_pages.sql
      0013_page_tags.sql
      0014_regen_jobs.sql
  storage/
    storage.go                  # Storage interface
    localdisk.go                # LocalDisk implementation
    localdisk_test.go
  auth/
    password.go                 # bcrypt hash/verify
    password_test.go
    admin.go                    # AdminRepo (admins table)
    admin_test.go
    session.go                  # SessionStore (sessions table)
    session_test.go
    testhelpers_test.go         # setupTestDB/seedAdmin shared by this package's tests
  httpapi/
    json.go                     # WriteJSON/WriteError helpers, ErrorBody type
    json_test.go
  adminapi/
    handlers.go                 # Login/Logout/ChangePassword handlers
    middleware.go                # RequireSession
    handlers_test.go
  httpserver/
    router.go                   # single-origin route wiring
    router_test.go
```

Each `internal/*` package owns one responsibility (config, DB connection+migrations, storage, auth, generic HTTP JSON helpers, admin-auth HTTP handlers, top-level routing). `cmd/*` packages are thin entrypoints that only wire dependencies together.

---

### Task 1: Go module, config loader, `.gitignore`

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Config{Addr, DBPath, DataDir string}`, `config.Load() Config`

- [ ] **Step 1: Initialize the Go module**

```bash
go mod init pixabros
```

- [ ] **Step 2: Create `.gitignore`**

```gitignore
/data/
*.db
*.db-journal
*.db-wal
*.db-shm
/bin/
```

- [ ] **Step 3: Write the failing test for the config loader**

`internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.DBPath != "./data/pixabros.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "./data/pixabros.db")
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./data")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("PIXABROS_ADDR", ":9090")
	t.Setenv("PIXABROS_DB_PATH", "/tmp/custom.db")
	t.Setenv("PIXABROS_DATA_DIR", "/tmp/custom-data")

	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9090")
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/custom.db")
	}
	if cfg.DataDir != "/tmp/custom-data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/tmp/custom-data")
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL — package `config` / function `Load` undefined.

- [ ] **Step 5: Implement the config loader**

`internal/config/config.go`:

```go
package config

import "os"

type Config struct {
	Addr    string
	DBPath  string
	DataDir string
}

func Load() Config {
	return Config{
		Addr:    getEnv("PIXABROS_ADDR", ":8080"),
		DBPath:  getEnv("PIXABROS_DB_PATH", "./data/pixabros.db"),
		DataDir: getEnv("PIXABROS_DATA_DIR", "./data"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod .gitignore internal/config
git commit -m "feat: add env-based config loader"
```

---

### Task 2: SQLite connection helper

**Files:**
- Create: `internal/db/db.go`
- Test: `internal/db/db_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: `db.Open(path string) (*sql.DB, error)`

- [ ] **Step 1: Add the SQLite driver dependency**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: Write the failing test**

`internal/db/db_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"
)

func TestOpen_EnablesForeignKeysAndWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	var fk int
	if err := conn.QueryRow(`PRAGMA foreign_keys;`).Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	var mode string
	if err := conn.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode pragma: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/db/... -v`
Expected: FAIL — `Open` undefined.

- [ ] **Step 4: Implement `Open`**

`internal/db/db.go`:

```go
package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return nil, err
	}
	return conn, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/db/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/db/db.go internal/db/db_test.go
git commit -m "feat: add sqlite connection helper with wal and foreign keys"
```

---

### Task 3: Migration runner and full schema

**Files:**
- Create: `internal/db/migrations/0001_admins.sql` … `0014_regen_jobs.sql` (14 files, listed below)
- Create: `internal/db/migrate.go`
- Test: `internal/db/migrate_test.go`

**Interfaces:**
- Consumes: `db.Open` (Task 2)
- Produces: `db.Migrate(conn *sql.DB) error` — creates every table in the spec's data model; idempotent (safe to call on every process start).

- [ ] **Step 1: Write the failing test**

`internal/db/migrate_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"
)

func TestMigrate_CreatesAllTablesAndIsIdempotent(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := Migrate(conn); err != nil {
		t.Fatalf("second Migrate() call must be a no-op, got error = %v", err)
	}

	wantTables := []string{
		"admins", "sessions", "media", "site_settings", "homepage_settings",
		"members", "games", "game_screenshots", "devlog_posts", "awards",
		"contact_submissions", "rendered_pages", "page_tags", "regen_jobs",
	}
	for _, table := range wantTables {
		var name string
		err := conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?;`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/db/... -v -run TestMigrate`
Expected: FAIL — `Migrate` undefined.

- [ ] **Step 3: Create all 14 migration files**

`internal/db/migrations/0001_admins.sql`:

```sql
CREATE TABLE admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0002_sessions.sql`:

```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
```

`internal/db/migrations/0003_media.sql`:

```sql
CREATE TABLE media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    format TEXT NOT NULL DEFAULT 'webp',
    alt_text TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0004_site_settings.sql`:

```sql
CREATE TABLE site_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri'))
);
```

`internal/db/migrations/0005_homepage_settings.sql`:

```sql
CREATE TABLE homepage_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    value_type TEXT NOT NULL CHECK (value_type IN ('text','uri'))
);
```

`internal/db/migrations/0006_members.sql`:

```sql
CREATE TABLE members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    avatar_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    tags TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    links_json TEXT NOT NULL DEFAULT '[]',
    display_order INTEGER NOT NULL DEFAULT 0,
    is_published INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0007_games.sql`:

```sql
CREATE TABLE games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    short_description TEXT NOT NULL DEFAULT '',
    full_description TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    is_browser_playable INTEGER NOT NULL DEFAULT 0,
    is_downloadable INTEGER NOT NULL DEFAULT 0,
    is_for_sale INTEGER NOT NULL DEFAULT 0,
    price_display TEXT,
    external_links_json TEXT NOT NULL DEFAULT '[]',
    cartridge_art_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    cd_cover_art_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    og_image_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    web_export_path TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_published INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0008_game_screenshots.sql`:

```sql
CREATE TABLE game_screenshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0
);
```

`internal/db/migrations/0009_devlog_posts.sql`:

```sql
CREATE TABLE devlog_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_markdown TEXT NOT NULL DEFAULT '',
    game_id INTEGER REFERENCES games(id) ON DELETE SET NULL,
    og_image_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    is_published INTEGER NOT NULL DEFAULT 0,
    published_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0010_awards.sql`:

```sql
CREATE TABLE awards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    issuer TEXT NOT NULL,
    date TEXT NOT NULL,
    picture_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    game_id INTEGER REFERENCES games(id) ON DELETE SET NULL,
    link TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0011_contact_submissions.sql`:

```sql
CREATE TABLE contact_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    subject TEXT NOT NULL,
    phone TEXT,
    email TEXT,
    message TEXT NOT NULL,
    wants_callback INTEGER NOT NULL DEFAULT 0,
    is_read INTEGER NOT NULL DEFAULT 0,
    ip_address TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0012_rendered_pages.sql`:

```sql
CREATE TABLE rendered_pages (
    page_key TEXT PRIMARY KEY,
    etag TEXT NOT NULL,
    rendered_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

`internal/db/migrations/0013_page_tags.sql`:

```sql
CREATE TABLE page_tags (
    page_key TEXT NOT NULL REFERENCES rendered_pages(page_key) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (page_key, tag)
);

CREATE INDEX idx_page_tags_tag ON page_tags(tag);
```

`internal/db/migrations/0014_regen_jobs.sql`:

```sql
CREATE TABLE regen_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    processed_at TEXT,
    error TEXT
);

CREATE INDEX idx_regen_jobs_status ON regen_jobs(status);
```

- [ ] **Step 4: Implement the migration runner**

`internal/db/migrate.go`:

```go
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func Migrate(conn *sql.DB) error {
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied := map[int]bool{}
	rows, err := conn.Query(`SELECT version FROM schema_migrations;`)
	if err != nil {
		return fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		tx, err := conn.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?);`, m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.name, err)
		}
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var migrations []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		versionStr := strings.SplitN(e.Name(), "_", 2)[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename %q: %w", e.Name(), err)
		}
		content, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration{version: version, name: e.Name(), sql: string(content)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	return migrations, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/db/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/db
git commit -m "feat: add migration runner and full database schema"
```

---

### Task 4: Storage interface and local-disk implementation

**Files:**
- Create: `internal/storage/storage.go`
- Create: `internal/storage/localdisk.go`
- Test: `internal/storage/localdisk_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: `storage.Storage` interface (`Put(key string, r io.Reader) error`, `Get(key string) (io.ReadCloser, error)`, `Delete(key string) error`, `URL(key string) string`); `storage.NewLocalDisk(root, baseURL string) *LocalDisk`

- [ ] **Step 1: Write the failing tests**

`internal/storage/localdisk_test.go`:

```go
package storage

import (
	"io"
	"strings"
	"testing"
)

func TestLocalDisk_PutGetDelete(t *testing.T) {
	s := NewLocalDisk(t.TempDir(), "/media")

	if err := s.Put("a/b.txt", strings.NewReader("hello")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	r, err := s.Get("a/b.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	data, _ := io.ReadAll(r)
	r.Close()
	if string(data) != "hello" {
		t.Errorf("Get() = %q, want %q", data, "hello")
	}

	if err := s.Delete("a/b.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Get("a/b.txt"); err == nil {
		t.Error("Get() after Delete() should return an error")
	}
	if err := s.Delete("a/b.txt"); err != nil {
		t.Errorf("Delete() on a missing key should be a no-op, got error = %v", err)
	}
}

func TestLocalDisk_RejectsPathTraversal(t *testing.T) {
	s := NewLocalDisk(t.TempDir(), "/media")

	if err := s.Put("../escape.txt", strings.NewReader("x")); err == nil {
		t.Error("Put() with a path-traversal key should return an error")
	}
	if _, err := s.Get("../escape.txt"); err == nil {
		t.Error("Get() with a path-traversal key should return an error")
	}
}

func TestLocalDisk_URL(t *testing.T) {
	s := NewLocalDisk("/data", "/media")
	if got := s.URL("a/b.webp"); got != "/media/a/b.webp" {
		t.Errorf("URL() = %q, want %q", got, "/media/a/b.webp")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/storage/... -v`
Expected: FAIL — package `storage` has no `Storage` type / `NewLocalDisk` undefined.

- [ ] **Step 3: Implement the `Storage` interface**

`internal/storage/storage.go`:

```go
package storage

import "io"

type Storage interface {
	Put(key string, r io.Reader) error
	Get(key string) (io.ReadCloser, error)
	Delete(key string) error
	URL(key string) string
}
```

- [ ] **Step 4: Implement `LocalDisk`**

`internal/storage/localdisk.go`:

```go
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalDisk struct {
	root    string
	baseURL string
}

func NewLocalDisk(root, baseURL string) *LocalDisk {
	return &LocalDisk{root: root, baseURL: baseURL}
}

func (s *LocalDisk) Put(key string, r io.Reader) error {
	fullPath, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func (s *LocalDisk) Get(key string) (io.ReadCloser, error) {
	fullPath, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

func (s *LocalDisk) Delete(key string) error {
	fullPath, err := s.resolve(key)
	if err != nil {
		return err
	}
	err = os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalDisk) URL(key string) string {
	return s.baseURL + "/" + key
}

func (s *LocalDisk) resolve(key string) (string, error) {
	full := filepath.Join(s.root, key)
	rel, err := filepath.Rel(s.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid key %q: escapes storage root", key)
	}
	return full, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/storage/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/storage
git commit -m "feat: add storage interface and local-disk implementation"
```

---

### Task 5: Password hashing

**Files:**
- Create: `internal/auth/password.go`
- Test: `internal/auth/password_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: `auth.HashPassword(password string) (string, error)`, `auth.VerifyPassword(hash, password string) bool`

- [ ] **Step 1: Add the bcrypt dependency**

```bash
go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: Write the failing test**

`internal/auth/password_test.go`:

```go
package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Error("HashPassword() returned the plaintext password")
	}
	if !VerifyPassword(hash, "correct-horse-battery-staple") {
		t.Error("VerifyPassword() = false for the correct password, want true")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("VerifyPassword() = true for a wrong password, want false")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/auth/... -v -run TestHashAndVerifyPassword`
Expected: FAIL — `HashPassword`/`VerifyPassword` undefined.

- [ ] **Step 4: Implement password hashing**

`internal/auth/password.go`:

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/auth/... -v -run TestHashAndVerifyPassword`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/auth/password.go internal/auth/password_test.go
git commit -m "feat: add bcrypt password hashing"
```

---

### Task 6: Admin repository

**Files:**
- Create: `internal/auth/admin.go`
- Create: `internal/auth/testhelpers_test.go`
- Test: `internal/auth/admin_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate` (Task 2, Task 3)
- Produces: `auth.Admin{ID int64, Username, PasswordHash string}`, `auth.ErrAdminNotFound`, `auth.NewAdminRepo(db *sql.DB) *AdminRepo`, `(*AdminRepo).FindByUsername(username string) (Admin, error)`, `(*AdminRepo).FindByID(id int64) (Admin, error)`, `(*AdminRepo).UpdatePasswordHash(id int64, hash string) error`

- [ ] **Step 1: Add shared test helpers for the `auth` package**

`internal/auth/testhelpers_test.go`:

```go
package auth

import (
	"database/sql"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
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

func seedAdmin(t *testing.T, conn *sql.DB, username, passwordHash string) int64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, username, passwordHash)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("get inserted admin id: %v", err)
	}
	return id
}
```

- [ ] **Step 2: Write the failing test**

`internal/auth/admin_test.go`:

```go
package auth

import (
	"errors"
	"testing"
)

func TestAdminRepo_FindByUsernameAndID(t *testing.T) {
	conn := setupTestDB(t)
	seedAdmin(t, conn, "furkan", "hash123")

	repo := NewAdminRepo(conn)

	byUsername, err := repo.FindByUsername("furkan")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if byUsername.PasswordHash != "hash123" {
		t.Errorf("PasswordHash = %q, want %q", byUsername.PasswordHash, "hash123")
	}

	byID, err := repo.FindByID(byUsername.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if byID.Username != "furkan" {
		t.Errorf("Username = %q, want %q", byID.Username, "furkan")
	}

	if _, err := repo.FindByUsername("nobody"); !errors.Is(err, ErrAdminNotFound) {
		t.Errorf("FindByUsername() error = %v, want ErrAdminNotFound", err)
	}
	if _, err := repo.FindByID(9999); !errors.Is(err, ErrAdminNotFound) {
		t.Errorf("FindByID() error = %v, want ErrAdminNotFound", err)
	}
}

func TestAdminRepo_UpdatePasswordHash(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "old-hash")
	repo := NewAdminRepo(conn)

	if err := repo.UpdatePasswordHash(adminID, "new-hash"); err != nil {
		t.Fatalf("UpdatePasswordHash() error = %v", err)
	}

	updated, err := repo.FindByID(adminID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want %q", updated.PasswordHash, "new-hash")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/auth/... -v -run TestAdminRepo`
Expected: FAIL — `NewAdminRepo`/`ErrAdminNotFound` undefined.

- [ ] **Step 4: Implement the admin repository**

`internal/auth/admin.go`:

```go
package auth

import (
	"database/sql"
	"errors"
)

var ErrAdminNotFound = errors.New("admin not found")

type Admin struct {
	ID           int64
	Username     string
	PasswordHash string
}

type AdminRepo struct {
	db *sql.DB
}

func NewAdminRepo(db *sql.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) FindByUsername(username string) (Admin, error) {
	var a Admin
	err := r.db.QueryRow(
		`SELECT id, username, password_hash FROM admins WHERE username = ?;`, username,
	).Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrAdminNotFound
	}
	if err != nil {
		return Admin{}, err
	}
	return a, nil
}

func (r *AdminRepo) FindByID(id int64) (Admin, error) {
	var a Admin
	err := r.db.QueryRow(
		`SELECT id, username, password_hash FROM admins WHERE id = ?;`, id,
	).Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrAdminNotFound
	}
	if err != nil {
		return Admin{}, err
	}
	return a, nil
}

func (r *AdminRepo) UpdatePasswordHash(id int64, hash string) error {
	_, err := r.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?;`, hash, id)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/auth/... -v -run TestAdminRepo`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/auth/admin.go internal/auth/admin_test.go internal/auth/testhelpers_test.go
git commit -m "feat: add admin repository"
```

---

### Task 7: Session store

**Files:**
- Create: `internal/auth/session.go`
- Test: `internal/auth/session_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate` (Task 2, Task 3), `setupTestDB`/`seedAdmin` (Task 6)
- Produces: `auth.Session{AdminID int64, ExpiresAt time.Time}`, `auth.ErrSessionNotFound`, `auth.NewSessionStore(db *sql.DB) *SessionStore`, `(*SessionStore).Create(adminID int64) (token string, expiresAt time.Time, err error)`, `(*SessionStore).Validate(token string) (Session, error)`, `(*SessionStore).Delete(token string) error`

- [ ] **Step 1: Write the failing tests**

`internal/auth/session_test.go`:

```go
package auth

import (
	"errors"
	"testing"
	"time"
)

func TestSessionStore_CreateValidateDelete(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "hash")

	store := NewSessionStore(conn)
	token, expiresAt, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if token == "" {
		t.Fatal("Create() returned an empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Error("Create() expiresAt should be in the future")
	}

	sess, err := store.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if sess.AdminID != adminID {
		t.Errorf("Validate() AdminID = %d, want %d", sess.AdminID, adminID)
	}

	if err := store.Delete(token); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Validate(token); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() after Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionStore_ValidateRejectsExpired(t *testing.T) {
	conn := setupTestDB(t)
	adminID := seedAdmin(t, conn, "furkan", "hash")

	store := NewSessionStore(conn)
	token, _, err := store.Create(adminID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := conn.Exec(
		`UPDATE sessions SET expires_at = ? WHERE admin_id = ?;`,
		time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), adminID,
	); err != nil {
		t.Fatalf("force-expire session: %v", err)
	}

	if _, err := store.Validate(token); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionStore_ValidateRejectsUnknownToken(t *testing.T) {
	conn := setupTestDB(t)
	store := NewSessionStore(conn)

	if _, err := store.Validate("does-not-exist"); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Validate() error = %v, want ErrSessionNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/auth/... -v -run TestSessionStore`
Expected: FAIL — `NewSessionStore`/`ErrSessionNotFound` undefined.

- [ ] **Step 3: Implement the session store**

`internal/auth/session.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

const SessionTTL = 7 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found or expired")

type Session struct {
	AdminID   int64
	ExpiresAt time.Time
}

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(adminID int64) (token string, expiresAt time.Time, err error) {
	token, err = generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(SessionTTL)
	_, err = s.db.Exec(
		`INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES (?, ?, ?);`,
		adminID, hashToken(token), expiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *SessionStore) Validate(token string) (Session, error) {
	var adminID int64
	var expiresAtStr string
	err := s.db.QueryRow(
		`SELECT admin_id, expires_at FROM sessions WHERE token_hash = ?;`, hashToken(token),
	).Scan(&adminID, &expiresAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return Session{}, err
	}
	if time.Now().After(expiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return Session{AdminID: adminID, ExpiresAt: expiresAt}, nil
}

func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?;`, hashToken(token))
	return err
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth/... -v -run TestSessionStore`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/session.go internal/auth/session_test.go
git commit -m "feat: add session store"
```

---

### Task 8: Admin seed CLI

**Files:**
- Create: `cmd/admincli/main.go`
- Test: `cmd/admincli/main_test.go`

**Interfaces:**
- Consumes: `config.Load` (Task 1), `db.Open`, `db.Migrate` (Task 2, Task 3), `auth.HashPassword` (Task 5)
- Produces: `admincli create-admin -username <u> -password <p>` command; internal `createAdmin(username, password string) error` (tested directly)

- [ ] **Step 1: Write the failing test**

`cmd/admincli/main_test.go`:

```go
package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateAdmin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("PIXABROS_DB_PATH", dbPath)

	if err := createAdmin("furkan", "s3cret-password"); err != nil {
		t.Fatalf("createAdmin() error = %v", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer conn.Close()

	var username, hash string
	err = conn.QueryRow(
		`SELECT username, password_hash FROM admins WHERE username = ?;`, "furkan",
	).Scan(&username, &hash)
	if err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if username != "furkan" {
		t.Errorf("username = %q, want %q", username, "furkan")
	}
	if hash == "s3cret-password" {
		t.Error("password_hash was stored in plaintext")
	}
}

func TestCreateAdmin_DuplicateUsernameFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("PIXABROS_DB_PATH", dbPath)

	if err := createAdmin("furkan", "s3cret-password"); err != nil {
		t.Fatalf("first createAdmin() error = %v", err)
	}
	if err := createAdmin("furkan", "another-password"); err == nil {
		t.Error("second createAdmin() with the same username should error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/admincli/... -v`
Expected: FAIL — `createAdmin` undefined.

- [ ] **Step 3: Implement the CLI**

`cmd/admincli/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: admincli create-admin -username <u> -password <p>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-admin":
		createCmd := flag.NewFlagSet("create-admin", flag.ExitOnError)
		username := createCmd.String("username", "", "admin username")
		password := createCmd.String("password", "", "admin password")
		createCmd.Parse(os.Args[2:])

		if *username == "" || *password == "" {
			fmt.Fprintln(os.Stderr, "both -username and -password are required")
			os.Exit(1)
		}
		if err := createAdmin(*username, *password); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("admin created:", *username)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func createAdmin(username, password string) error {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = conn.Exec(`INSERT INTO admins (username, password_hash) VALUES (?, ?);`, username, hash)
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/admincli/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/admincli
git commit -m "feat: add admin seed cli"
```

---

### Task 9: Admin auth HTTP handlers, JSON helpers, session middleware

**Files:**
- Create: `internal/httpapi/json.go`
- Test: `internal/httpapi/json_test.go`
- Create: `internal/adminapi/handlers.go`
- Create: `internal/adminapi/middleware.go`
- Test: `internal/adminapi/handlers_test.go`

**Interfaces:**
- Consumes: `auth.AdminRepo`, `auth.SessionStore`, `auth.HashPassword`, `auth.VerifyPassword` (Task 5, Task 6, Task 7)
- Produces:
  - `httpapi.WriteJSON(w http.ResponseWriter, status int, body interface{})`, `httpapi.WriteError(w http.ResponseWriter, status int, code, message string)`, `httpapi.ErrorBody{Error ErrorDetail}`, `httpapi.ErrorDetail{Code, Message string}`
  - `adminapi.NewAuthHandlers(admins *auth.AdminRepo, sessions *auth.SessionStore) *AuthHandlers`, methods `.Login`, `.Logout`, `.ChangePassword` (all `http.HandlerFunc`-shaped: `func(http.ResponseWriter, *http.Request)`)
  - `adminapi.RequireSession(sessions *auth.SessionStore, next http.HandlerFunc) http.HandlerFunc`
  - `adminapi.AdminIDFromContext(ctx context.Context) (int64, bool)`

- [ ] **Step 1: Write the failing test for the JSON helpers**

`internal/httpapi/json_test.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, 400, "bad_request", "message is required")

	if rec.Code != 400 {
		t.Errorf("status = %d, want 400", rec.Code)
	}

	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "bad_request" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "bad_request")
	}
	if body.Error.Message != "message is required" {
		t.Errorf("error.message = %q, want %q", body.Error.Message, "message is required")
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, 201, map[string]string{"status": "created"})

	if rec.Code != 201 {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi/... -v`
Expected: FAIL — package `httpapi` does not exist yet.

- [ ] **Step 3: Implement the JSON helpers**

`internal/httpapi/json.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi
git commit -m "feat: add json response helpers"
```

- [ ] **Step 6: Write the failing tests for the admin auth handlers and middleware**

`internal/adminapi/handlers_test.go`:

```go
package adminapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
		"new_password":      "new-password-123",
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

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "totally-wrong",
		"new_password":      "new-password-123",
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
```

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./internal/adminapi/... -v`
Expected: FAIL — package `adminapi` does not exist yet.

- [ ] **Step 8: Implement the session middleware**

`internal/adminapi/middleware.go`:

```go
package adminapi

import (
	"context"
	"net/http"

	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
)

const sessionCookieName = "pixabros_session"

type contextKey string

const adminIDContextKey contextKey = "adminID"

func RequireSession(sessions *auth.SessionStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
			return
		}
		session, err := sessions.Validate(cookie.Value)
		if err != nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "session expired or invalid")
			return
		}
		next(w, r.WithContext(withAdminID(r.Context(), session.AdminID)))
	}
}

func AdminIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(adminIDContextKey).(int64)
	return id, ok
}

func withAdminID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, adminIDContextKey, id)
}
```

- [ ] **Step 9: Implement the auth handlers**

`internal/adminapi/handlers.go`:

```go
package adminapi

import (
	"encoding/json"
	"net/http"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
)

type AuthHandlers struct {
	admins   *auth.AdminRepo
	sessions *auth.SessionStore
}

func NewAuthHandlers(admins *auth.AdminRepo, sessions *auth.SessionStore) *AuthHandlers {
	return &AuthHandlers{admins: admins, sessions: sessions}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "username and password are required")
		return
	}

	admin, err := h.admins.FindByUsername(req.Username)
	if err != nil || !auth.VerifyPassword(admin.PasswordHash, req.Password) {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		return
	}

	token, expiresAt, err := h.sessions.Create(admin.ID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
	httpapi.WriteJSON(w, http.StatusOK, loginResponse{Username: admin.Username})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		h.sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	adminID, ok := AdminIDFromContext(r.Context())
	if !ok {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if len(req.NewPassword) < 8 {
		httpapi.WriteError(w, http.StatusBadRequest, "weak_password", "new password must be at least 8 characters")
		return
	}

	admin, err := h.admins.FindByID(adminID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load admin")
		return
	}
	if !auth.VerifyPassword(admin.PasswordHash, req.CurrentPassword) {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not hash password")
		return
	}
	if err := h.admins.UpdatePasswordHash(adminID, newHash); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update password")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 10: Run tests to verify they pass**

Run: `go test ./internal/adminapi/... -v`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/adminapi
git commit -m "feat: add admin auth handlers and session middleware"
```

---

### Task 10: Single-origin router

**Files:**
- Create: `internal/httpserver/router.go`
- Test: `internal/httpserver/router_test.go`

**Interfaces:**
- Consumes: `adminapi.NewAuthHandlers`, `adminapi.RequireSession` (Task 9), `auth.AdminRepo`, `auth.SessionStore` (Task 6, Task 7)
- Produces: `httpserver.Dependencies{Admins *auth.AdminRepo, Sessions *auth.SessionStore, AdminUIDir, PlayDir, PublicDir string}`, `httpserver.New(deps Dependencies) http.Handler`

- [ ] **Step 1: Write the failing test**

`internal/httpserver/router_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpserver/... -v`
Expected: FAIL — package `httpserver` does not exist yet.

- [ ] **Step 3: Implement the router**

`internal/httpserver/router.go`:

```go
package httpserver

import (
	"net/http"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	AdminUIDir string
	PlayDir    string
	PublicDir  string
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))

	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", http.FileServer(http.Dir(deps.AdminUIDir))))
	mux.Handle("/play/", http.StripPrefix("/play/", http.FileServer(http.Dir(deps.PlayDir))))
	mux.Handle("/", http.FileServer(http.Dir(deps.PublicDir)))

	return mux
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpserver/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/httpserver
git commit -m "feat: add single-origin router"
```

---

### Task 11: Server entrypoint and end-to-end verification

**Files:**
- Create: `cmd/server/main.go`

**Interfaces:**
- Consumes: `config.Load` (Task 1), `db.Open`, `db.Migrate` (Task 2, Task 3), `auth.NewAdminRepo`, `auth.NewSessionStore` (Task 6, Task 7), `httpserver.New`, `httpserver.Dependencies` (Task 10)
- Produces: the `server` binary — nothing later in this plan consumes it, this is the plan's final deliverable

- [ ] **Step 1: Implement the entrypoint**

`cmd/server/main.go`:

```go
package main

import (
	"log"
	"net/http"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/httpserver"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		PublicDir:  cfg.DataDir + "/rendered",
	})

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
```

- [ ] **Step 2: Build the binary**

Run: `go build ./...`
Expected: builds with no errors.

- [ ] **Step 3: Run the full test suite**

Run: `go test ./...`
Expected: PASS for every package.

- [ ] **Step 4: Manually verify the running server**

```bash
mkdir -p data/admin-dist data/games data/rendered
echo '<h1>public</h1>' > data/rendered/index.html
echo '<h1>admin</h1>' > data/admin-dist/index.html
go run ./cmd/admincli create-admin -username furkan -password "a-strong-password-1"
go run ./cmd/server &
sleep 1
curl -i -c /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"furkan","password":"a-strong-password-1"}'
curl -i http://localhost:8080/
curl -i http://localhost:8080/I-am-a-pixabro/
kill %1
```

Expected: login returns `200` with a `Set-Cookie: pixabros_session=...` header, `/` and `/I-am-a-pixabro/` both return `200` with the placeholder HTML.

- [ ] **Step 5: Commit**

```bash
git add cmd/server
git commit -m "feat: wire server entrypoint"
```

---

## Definition of Done

- `go build ./...` succeeds.
- `go test ./...` passes with no skipped packages.
- The manual verification in Task 11 Step 4 succeeds against a locally built binary.
- Every table in the spec's data model section exists after `db.Migrate` runs.
- No `any` type appears anywhere in the Go source (`grep -rn '\bany\b' --include='*.go' .` returns nothing outside of comments/strings).
