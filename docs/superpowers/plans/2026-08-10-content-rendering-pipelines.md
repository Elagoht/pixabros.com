# Content & Rendering Pipelines Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the image/WebP pipeline, game archive upload, tag-based static-page regeneration queue, ETag-aware public page serving, orphan-media sweep, and the devlog OG image generator on top of Plan A's backend core.

**Architecture:** Pure Go image processing (`golang.org/x/image/draw` for resize, a pure-Go WebP encoder for output), `archive/zip`/`archive/tar`+`compress/gzip` for game upload extraction, a DB-backed durable job queue (`regen_jobs`) drained by a single polling worker goroutine, and a tag-based dependency table (`page_tags`) that lets any future page register what it depends on.

**Tech Stack:** Go 1.22+, `golang.org/x/image` (draw, font, font/basicfont, math/fixed, webp — decode only), `github.com/mayahiro/go-webp` (pure-Go WebP encoder, no cgo).

**Depends on:** `docs/superpowers/plans/2026-08-10-backend-core-data-model.md` (Plan A) — this plan assumes Plan A's packages (`internal/db`, `internal/storage`, `internal/auth`, `internal/httpapi`, `internal/adminapi`, `internal/httpserver`) already exist and their tests pass.

## Global Constraints

- Never use Go's `any` type alias anywhere in this codebase — use `interface{}` or a concrete type instead (user's global CLAUDE.md rule).
- Only the resized WebP is ever kept — the original uploaded file is discarded after processing.
- Fixed target sizes (from the spec): avatar 400×400, cd_cover_art 600×600, cartridge_art 400×560, og_image 1200×630, screenshot 1280×720, award_picture 320×320, org_logo 512×512.
- Game archives accepted: `.zip`, `.tar`, `.tar.gz`/`.tgz`. Extraction is rejected (and rolled back) if there is no `index.html` at the archive root, or if any entry path escapes the destination directory.
- Regen jobs are durable (`regen_jobs` table), never in-memory only. Failed jobs are never auto-retried.
- Orphan media is deleted only if unreferenced **and** older than 24 hours (grace period for in-flight uploads).
- Git commits in this repo: self-committed, one-sentence semantic messages, no co-author/contributor trailer.

---

## File Structure

```
internal/
  imaging/
    imaging.go          # Decode (jpeg/png/gif/webp) + ResizeCropFill + EncodeWebP
    imaging_test.go
    targets.go           # named target sizes (avatar, cd_cover_art, ...)
    targets_test.go
  media/
    repo.go              # media table CRUD
    repo_test.go
    orphansweep.go        # daily sweep of unreferenced media rows/files
    orphansweep_test.go
  mediaapi/
    upload_handler.go     # POST /api/admin/media?target=... (multipart)
    upload_handler_test.go
  ogimage/
    ogimage.go            # template + title -> rendered OG image.Image
    ogimage_test.go
  gamearchive/
    extract.go             # zip/tar/tar.gz -> validated extraction to disk
    extract_test.go
  gameupload/
    upload_handler.go       # POST /api/admin/games/{slug}/upload (multipart archive)
    upload_handler_test.go
  render/
    registry.go              # Renderer type, Registry (exact + prefix match)
    registry_test.go
    store.go                  # rendered_pages + page_tags persistence, RenderAndPersist
    store_test.go
    queue.go                   # regen_jobs enqueue + worker loop
    queue_test.go
    serve.go                    # public HTTP handler: ETag/304 for rendered pages, immutable cache for assets
    serve_test.go
cmd/
  server/
    main.go                # (modify) wire imaging/media/render packages, start worker, mount handlers
```

Each package owns one pipeline stage. `render` is the one package with several files because its four concerns (registry, persistence, queue, HTTP serving) all operate on the same two tables and are easiest to review together; each file is still independently testable.

---

### Task 1: Image decode, crop-to-fill resize, WebP encode

**Files:**
- Create: `internal/imaging/imaging.go`
- Test: `internal/imaging/imaging_test.go`

**Interfaces:**
- Consumes: nothing from Plan A
- Produces: `imaging.Decode(r io.Reader) (image.Image, error)`, `imaging.ResizeCropFill(src image.Image, targetW, targetH int) image.Image`, `imaging.EncodeWebP(img image.Image) ([]byte, error)`, `imaging.ProcessUpload(r io.Reader, targetW, targetH int) (webpBytes []byte, err error)`

- [ ] **Step 1: Add dependencies**

```bash
go get golang.org/x/image
go get github.com/mayahiro/go-webp
```

- [ ] **Step 2: Write the failing tests**

`internal/imaging/imaging_test.go`:

```go
package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func solidPNG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func TestDecode_PNG(t *testing.T) {
	data := solidPNG(t, 10, 10, color.RGBA{R: 255, A: 255})
	img, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
		t.Errorf("Decode() bounds = %v, want 10x10", img.Bounds())
	}
}

func TestDecode_RejectsGarbage(t *testing.T) {
	if _, err := Decode(bytes.NewReader([]byte("not an image"))); err == nil {
		t.Error("Decode() with garbage input should return an error")
	}
}

func TestResizeCropFill_WideSourceToSquareTarget(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	out := ResizeCropFill(src, 400, 400)
	if out.Bounds().Dx() != 400 || out.Bounds().Dy() != 400 {
		t.Errorf("ResizeCropFill() bounds = %v, want 400x400", out.Bounds())
	}
}

func TestResizeCropFill_TallSourceToWideTarget(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 800))
	out := ResizeCropFill(src, 1200, 630)
	if out.Bounds().Dx() != 1200 || out.Bounds().Dy() != 630 {
		t.Errorf("ResizeCropFill() bounds = %v, want 1200x630", out.Bounds())
	}
}

func TestResizeCropFill_PreservesSolidColor(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fillColor := color.RGBA{R: 10, G: 200, B: 30, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, fillColor)
		}
	}
	out := ResizeCropFill(src, 50, 50)
	r, g, b, a := out.At(25, 25).RGBA()
	if uint8(r>>8) != fillColor.R || uint8(g>>8) != fillColor.G || uint8(b>>8) != fillColor.B || uint8(a>>8) != fillColor.A {
		t.Errorf("center pixel = (%d,%d,%d,%d), want (%d,%d,%d,%d)", r>>8, g>>8, b>>8, a>>8, fillColor.R, fillColor.G, fillColor.B, fillColor.A)
	}
}

func TestEncodeWebP_ProducesDecodableOutput(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	data, err := EncodeWebP(src)
	if err != nil {
		t.Fatalf("EncodeWebP() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("EncodeWebP() returned no bytes")
	}
}

func TestProcessUpload_EndToEnd(t *testing.T) {
	data := solidPNG(t, 800, 400, color.RGBA{G: 255, A: 255})
	webpBytes, err := ProcessUpload(bytes.NewReader(data), 400, 400)
	if err != nil {
		t.Fatalf("ProcessUpload() error = %v", err)
	}
	if len(webpBytes) == 0 {
		t.Error("ProcessUpload() returned no bytes")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/imaging/... -v`
Expected: FAIL — package `imaging` does not exist yet.

- [ ] **Step 4: Implement the pipeline**

`internal/imaging/imaging.go`:

```go
package imaging

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	webpenc "github.com/mayahiro/go-webp"
)

// Decode reads a JPEG, PNG, GIF, or WebP image from r.
func Decode(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		return img, nil
	}
	if img, err := webp.Decode(bytes.NewReader(data)); err == nil {
		return img, nil
	}
	return nil, fmt.Errorf("unsupported or corrupt image format")
}

// ResizeCropFill center-crops src to the target aspect ratio, then scales it
// to exactly targetW x targetH.
func ResizeCropFill(src image.Image, targetW, targetH int) image.Image {
	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	srcAspect := float64(srcW) / float64(srcH)
	targetAspect := float64(targetW) / float64(targetH)

	cropRect := srcBounds
	switch {
	case srcAspect > targetAspect:
		newW := int(float64(srcH) * targetAspect)
		offset := (srcW - newW) / 2
		cropRect = image.Rect(srcBounds.Min.X+offset, srcBounds.Min.Y, srcBounds.Min.X+offset+newW, srcBounds.Max.Y)
	case srcAspect < targetAspect:
		newH := int(float64(srcW) / targetAspect)
		offset := (srcH - newH) / 2
		cropRect = image.Rect(srcBounds.Min.X, srcBounds.Min.Y+offset, srcBounds.Max.X, srcBounds.Min.Y+offset+newH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Over, nil)
	return dst
}

// EncodeWebP encodes img as a lossy WebP image.
func EncodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := webpenc.Encode(&buf, img, &webpenc.Options{
		Compression: webpenc.CompressionLossy,
		Quality:     85,
	})
	if err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}

// ProcessUpload decodes r, resizes it to exactly targetW x targetH, and
// returns the resulting WebP bytes. This is the single entry point the
// upload handler calls.
func ProcessUpload(r io.Reader, targetW, targetH int) ([]byte, error) {
	src, err := Decode(r)
	if err != nil {
		return nil, err
	}
	resized := ResizeCropFill(src, targetW, targetH)
	return EncodeWebP(resized)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/imaging/... -v`
Expected: PASS. If `EncodeWebP`/`webpenc.Encode` fails to compile, check `go doc github.com/mayahiro/go-webp` for the current exact signature and adjust the call — the rest of the pipeline is independent of this one call.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/imaging/imaging.go internal/imaging/imaging_test.go
git commit -m "feat: add image decode, crop-fill resize, and webp encode pipeline"
```

---

### Task 2: Named target sizes

**Files:**
- Create: `internal/imaging/targets.go`
- Test: `internal/imaging/targets_test.go`

**Interfaces:**
- Consumes: nothing
- Produces: `imaging.Target{Name string, Width, Height int}`, `imaging.Targets map[string]Target`, `imaging.LookupTarget(name string) (Target, bool)`

- [ ] **Step 1: Write the failing test**

`internal/imaging/targets_test.go`:

```go
package imaging

import "testing"

func TestLookupTarget_KnownAndUnknown(t *testing.T) {
	cases := []struct {
		name          string
		wantW, wantH  int
	}{
		{"avatar", 400, 400},
		{"cd_cover_art", 600, 600},
		{"cartridge_art", 400, 560},
		{"og_image", 1200, 630},
		{"screenshot", 1280, 720},
		{"award_picture", 320, 320},
		{"org_logo", 512, 512},
	}
	for _, c := range cases {
		target, ok := LookupTarget(c.name)
		if !ok {
			t.Errorf("LookupTarget(%q) not found", c.name)
			continue
		}
		if target.Width != c.wantW || target.Height != c.wantH {
			t.Errorf("LookupTarget(%q) = %dx%d, want %dx%d", c.name, target.Width, target.Height, c.wantW, c.wantH)
		}
	}

	if _, ok := LookupTarget("does_not_exist"); ok {
		t.Error("LookupTarget() should return ok=false for an unknown name")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/imaging/... -v -run TestLookupTarget`
Expected: FAIL — `LookupTarget` undefined.

- [ ] **Step 3: Implement the target registry**

`internal/imaging/targets.go`:

```go
package imaging

type Target struct {
	Name   string
	Width  int
	Height int
}

var Targets = map[string]Target{
	"avatar":        {Name: "avatar", Width: 400, Height: 400},
	"cd_cover_art":  {Name: "cd_cover_art", Width: 600, Height: 600},
	"cartridge_art": {Name: "cartridge_art", Width: 400, Height: 560},
	"og_image":      {Name: "og_image", Width: 1200, Height: 630},
	"screenshot":    {Name: "screenshot", Width: 1280, Height: 720},
	"award_picture": {Name: "award_picture", Width: 320, Height: 320},
	"org_logo":      {Name: "org_logo", Width: 512, Height: 512},
}

func LookupTarget(name string) (Target, bool) {
	t, ok := Targets[name]
	return t, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/imaging/... -v -run TestLookupTarget`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/imaging/targets.go internal/imaging/targets_test.go
git commit -m "feat: add named image target size registry"
```

---

### Task 3: Media repository

**Files:**
- Create: `internal/media/repo.go`
- Test: `internal/media/repo_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate` (Plan A Task 2/3)
- Produces: `media.Media{ID int64, Path string, Width, Height int, Format, AltText string}`, `media.ErrMediaNotFound`, `media.NewRepo(db *sql.DB) *Repo`, `(*Repo).Create(path string, width, height int) (Media, error)`, `(*Repo).FindByID(id int64) (Media, error)`, `(*Repo).Delete(id int64) error`, `(*Repo).AllIDs() ([]int64, error)`

- [ ] **Step 1: Write the failing tests**

`internal/media/repo_test.go`:

```go
package media

import (
	"errors"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
)

func setupTestDB(t *testing.T) *sqlDBType {
	t.Helper()
	return nil // placeholder replaced below
}
```

Wait — write the real helper directly instead:

`internal/media/repo_test.go`:

```go
package media

import (
	"database/sql"
	"errors"
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

func TestRepo_CreateFindDelete(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	m, err := repo.Create("media/2026/abc.webp", 400, 400)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if m.ID == 0 {
		t.Fatal("Create() returned a zero ID")
	}
	if m.Format != "webp" {
		t.Errorf("Format = %q, want %q", m.Format, "webp")
	}

	found, err := repo.FindByID(m.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Path != "media/2026/abc.webp" {
		t.Errorf("Path = %q, want %q", found.Path, "media/2026/abc.webp")
	}

	if err := repo.Delete(m.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(m.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Errorf("FindByID() after Delete() error = %v, want ErrMediaNotFound", err)
	}
}

func TestRepo_AllIDs(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	first, _ := repo.Create("a.webp", 10, 10)
	second, _ := repo.Create("b.webp", 10, 10)

	ids, err := repo.AllIDs()
	if err != nil {
		t.Fatalf("AllIDs() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("AllIDs() returned %d ids, want 2", len(ids))
	}
	seen := map[int64]bool{ids[0]: true, ids[1]: true}
	if !seen[first.ID] || !seen[second.ID] {
		t.Errorf("AllIDs() = %v, want to contain %d and %d", ids, first.ID, second.ID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/media/... -v`
Expected: FAIL — package `media` does not exist yet.

- [ ] **Step 3: Implement the repository**

`internal/media/repo.go`:

```go
package media

import (
	"database/sql"
	"errors"
)

var ErrMediaNotFound = errors.New("media not found")

type Media struct {
	ID      int64
	Path    string
	Width   int
	Height  int
	Format  string
	AltText string
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(path string, width, height int) (Media, error) {
	res, err := r.db.Exec(
		`INSERT INTO media (path, width, height, format, alt_text) VALUES (?, ?, ?, 'webp', '');`,
		path, width, height,
	)
	if err != nil {
		return Media{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Media{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) FindByID(id int64) (Media, error) {
	var m Media
	err := r.db.QueryRow(
		`SELECT id, path, width, height, format, alt_text FROM media WHERE id = ?;`, id,
	).Scan(&m.ID, &m.Path, &m.Width, &m.Height, &m.Format, &m.AltText)
	if errors.Is(err, sql.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, err
	}
	return m, nil
}

func (r *Repo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM media WHERE id = ?;`, id)
	return err
}

func (r *Repo) AllIDs() ([]int64, error) {
	rows, err := r.db.Query(`SELECT id FROM media;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/media/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/media/repo.go internal/media/repo_test.go
git commit -m "feat: add media repository"
```

---

### Task 4: Media upload HTTP endpoint

**Files:**
- Create: `internal/mediaapi/upload_handler.go`
- Test: `internal/mediaapi/upload_handler_test.go`

**Interfaces:**
- Consumes: `imaging.ProcessUpload`, `imaging.LookupTarget` (Task 1, Task 2), `media.NewRepo`, `(*media.Repo).Create` (Task 3), `storage.Storage` (Plan A Task 4), `httpapi.WriteJSON`/`WriteError` (Plan A Task 9)
- Produces: `mediaapi.NewUploadHandler(repo *media.Repo, files storage.Storage) *UploadHandler`, `(*UploadHandler).Upload` (`func(http.ResponseWriter, *http.Request)`)

- [ ] **Step 1: Write the failing tests**

`internal/mediaapi/upload_handler_test.go`:

```go
package mediaapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

func multipartUploadRequest(t *testing.T, target string, pngData []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(pngData); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/media?target="+target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func solidPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func TestUpload_Success(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	repo := media.NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")
	handler := NewUploadHandler(repo, files)

	req := multipartUploadRequest(t, "avatar", solidPNG(t, 800, 400))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Width != 400 || resp.Height != 400 {
		t.Errorf("dimensions = %dx%d, want 400x400", resp.Width, resp.Height)
	}
	if resp.ID == 0 {
		t.Error("expected a non-zero media ID")
	}

	saved, err := repo.FindByID(resp.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if saved.Width != 400 || saved.Height != 400 {
		t.Errorf("saved dimensions = %dx%d, want 400x400", saved.Width, saved.Height)
	}
}

func TestUpload_UnknownTarget(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	handler := NewUploadHandler(media.NewRepo(conn), storage.NewLocalDisk(t.TempDir(), "/media"))
	req := multipartUploadRequest(t, "not_a_real_target", solidPNG(t, 10, 10))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mediaapi/... -v`
Expected: FAIL — package `mediaapi` does not exist yet.

- [ ] **Step 3: Implement the handler**

`internal/mediaapi/upload_handler.go`:

```go
package mediaapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"pixabros/internal/httpapi"
	"pixabros/internal/imaging"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

type UploadHandler struct {
	repo  *media.Repo
	files storage.Storage
}

func NewUploadHandler(repo *media.Repo, files storage.Storage) *UploadHandler {
	return &UploadHandler{repo: repo, files: files}
}

type uploadResponse struct {
	ID     int64  `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	targetName := r.URL.Query().Get("target")
	target, ok := imaging.LookupTarget(targetName)
	if !ok {
		httpapi.WriteError(w, http.StatusBadRequest, "unknown_target", fmt.Sprintf("unknown upload target %q", targetName))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_file", "a file field is required")
		return
	}
	defer file.Close()

	webpBytes, err := imaging.ProcessUpload(file, target.Width, target.Height)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_image", "could not decode or process the uploaded image")
		return
	}

	key, err := randomMediaKey(targetName)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not generate a storage key")
		return
	}
	if err := h.files.Put(key, bytesReader(webpBytes)); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not store the image")
		return
	}

	m, err := h.repo.Create(key, target.Width, target.Height)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not save media record")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, uploadResponse{
		ID:     m.ID,
		URL:    h.files.URL(m.Path),
		Width:  m.Width,
		Height: m.Height,
	})
}

func randomMediaKey(targetName string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("media/%s/%d-%s.webp", targetName, time.Now().Year(), hex.EncodeToString(b)), nil
}
```

Add the small `bytesReader` helper in the same file (kept local since it is only used here):

```go
import "bytes"

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
```

Merge this import and helper into the main import block and function list above rather than appending — the final file must compile as one unit with `bytes` imported alongside the others.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mediaapi/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mediaapi
git commit -m "feat: add media upload endpoint"
```

---

### Task 5: Orphan media sweep

**Files:**
- Create: `internal/media/orphansweep.go`
- Test: `internal/media/orphansweep_test.go`

**Interfaces:**
- Consumes: `media.Repo` (Task 3), `storage.Storage` (Plan A Task 4)
- Produces: `media.ReferenceLookup func() (map[int64]bool, error)` (type alias for a caller-supplied function that returns every media ID currently referenced by any module table), `media.SweepOrphans(repo *Repo, files storage.Storage, referenced ReferenceLookup, olderThan time.Duration, now time.Time) (deleted int, err error)`

- [ ] **Step 1: Write the failing tests**

`internal/media/orphansweep_test.go`:

```go
package media

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func TestSweepOrphans_DeletesOnlyUnreferencedAndOld(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	repo := NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")

	referenced, err := repo.Create("referenced.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(referenced.Path, strings.NewReader("x"))

	tooNew, err := repo.Create("too-new.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(tooNew.Path, strings.NewReader("x"))

	orphan, err := repo.Create("orphan.webp", 10, 10)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	files.Put(orphan.Path, strings.NewReader("x"))
	if _, err := conn.Exec(`UPDATE media SET created_at = ? WHERE id = ?;`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339), orphan.ID); err != nil {
		t.Fatalf("backdate orphan media: %v", err)
	}

	lookup := func() (map[int64]bool, error) {
		return map[int64]bool{referenced.ID: true}, nil
	}

	deleted, err := SweepOrphans(repo, files, lookup, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("SweepOrphans() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	if _, err := repo.FindByID(orphan.ID); err == nil {
		t.Error("orphan media row should have been deleted")
	}
	if _, err := files.Get(orphan.Path); err == nil {
		t.Error("orphan media file should have been deleted")
	}

	if _, err := repo.FindByID(referenced.ID); err != nil {
		t.Error("referenced media row should not have been deleted")
	}
	if _, err := repo.FindByID(tooNew.ID); err != nil {
		t.Error("too-new unreferenced media row should not have been deleted yet")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/media/... -v -run TestSweepOrphans`
Expected: FAIL — `SweepOrphans` undefined.

- [ ] **Step 3: Implement the sweep**

`internal/media/orphansweep.go`:

```go
package media

import (
	"time"

	"pixabros/internal/storage"
)

type ReferenceLookup func() (map[int64]bool, error)

// SweepOrphans deletes media rows (and their backing files) that are not
// referenced by any module table and were created more than olderThan ago.
func SweepOrphans(repo *Repo, files storage.Storage, referenced ReferenceLookup, olderThan time.Duration, now time.Time) (int, error) {
	referencedIDs, err := referenced()
	if err != nil {
		return 0, err
	}

	all, err := repo.allWithCreatedAt()
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, m := range all {
		if referencedIDs[m.ID] {
			continue
		}
		if now.Sub(m.CreatedAt) < olderThan {
			continue
		}
		if err := files.Delete(m.Path); err != nil {
			return deleted, err
		}
		if err := repo.Delete(m.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
```

This calls a new unexported helper `allWithCreatedAt` — add it to `internal/media/repo.go` (Modify):

```go
type mediaWithCreatedAt struct {
	Media
	CreatedAt time.Time
}

func (r *Repo) allWithCreatedAt() ([]mediaWithCreatedAt, error) {
	rows, err := r.db.Query(`SELECT id, path, width, height, format, alt_text, created_at FROM media;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []mediaWithCreatedAt
	for rows.Next() {
		var m mediaWithCreatedAt
		var createdAtStr string
		if err := rows.Scan(&m.ID, &m.Path, &m.Width, &m.Height, &m.Format, &m.AltText, &createdAtStr); err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339, normalizeTimestamp(createdAtStr))
		if err != nil {
			return nil, err
		}
		m.CreatedAt = createdAt
		result = append(result, m)
	}
	return result, rows.Err()
}
```

SQLite's `strftime('%Y-%m-%dT%H:%M:%fZ','now')` default produces a fractional-second timestamp that `time.RFC3339` cannot parse directly (RFC3339 expects `2006-01-02T15:04:05Z07:00`, no fraction). Add this small normalizer to `internal/media/repo.go` (it strips fractional seconds so `time.Parse` with `time.RFC3339` succeeds):

```go
import "strings"

func normalizeTimestamp(s string) string {
	if i := strings.Index(s, "."); i != -1 {
		return s[:i] + "Z"
	}
	return s
}
```

Add `"time"` and `"strings"` to `internal/media/repo.go`'s import block alongside the existing `"database/sql"` and `"errors"`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/media/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/media
git commit -m "feat: add orphan media sweep"
```

---

### Task 6: Devlog OG image generator

**Files:**
- Create: `internal/ogimage/ogimage.go`
- Test: `internal/ogimage/ogimage_test.go`

**Interfaces:**
- Consumes: `imaging.EncodeWebP` (Task 1)
- Produces: `ogimage.Generate(title string) image.Image`, `ogimage.GenerateWebP(title string) ([]byte, error)`

- [ ] **Step 1: Add the font dependency (already pulled in by `golang.org/x/image` in Task 1; no new `go get` needed)**

- [ ] **Step 2: Write the failing tests**

`internal/ogimage/ogimage_test.go`:

```go
package ogimage

import "testing"

func TestGenerate_ProducesCorrectDimensions(t *testing.T) {
	img := Generate("Pixel Quest devlog #3")
	if img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630 {
		t.Errorf("Generate() bounds = %v, want 1200x630", img.Bounds())
	}
}

func TestGenerate_HandlesLongTitles(t *testing.T) {
	img := Generate("This is a deliberately very long devlog title meant to wrap across multiple lines of the template")
	if img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630 {
		t.Errorf("Generate() bounds = %v, want 1200x630", img.Bounds())
	}
}

func TestGenerateWebP_ProducesBytes(t *testing.T) {
	data, err := GenerateWebP("Short title")
	if err != nil {
		t.Fatalf("GenerateWebP() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("GenerateWebP() returned no bytes")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/ogimage/... -v`
Expected: FAIL — package `ogimage` does not exist yet.

- [ ] **Step 4: Implement the generator**

`internal/ogimage/ogimage.go`:

```go
package ogimage

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"pixabros/internal/imaging"
)

const (
	Width  = 1200
	Height = 630
	margin = 60
)

func Generate(title string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 20, G: 20, B: 30, A: 255}), image.Point{}, draw.Src)

	face := basicfont.Face7x13
	lines := wrapText(title, face, Width-2*margin)
	lineHeight := face.Metrics().Height.Ceil()
	startY := Height/2 - (len(lines)*lineHeight)/2 + face.Metrics().Ascent.Ceil()

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 240, G: 240, B: 245, A: 255}),
		Face: face,
	}
	for i, line := range lines {
		textWidth := font.MeasureString(face, line).Ceil()
		x := (Width - textWidth) / 2
		y := startY + i*lineHeight
		drawer.Dot = fixed.P(x, y)
		drawer.DrawString(line)
	}
	return img
}

func GenerateWebP(title string) ([]byte, error) {
	return imaging.EncodeWebP(Generate(title))
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if font.MeasureString(face, candidate).Ceil() > maxWidth {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	lines = append(lines, current)
	return lines
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ogimage/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ogimage
git commit -m "feat: add devlog og image generator"
```

---

### Task 7: Game archive extraction

**Files:**
- Create: `internal/gamearchive/extract.go`
- Test: `internal/gamearchive/extract_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: `gamearchive.Extract(archive io.Reader, filename string, destDir string) error` — detects format from `filename`'s extension (`.zip`, `.tar`, `.tar.gz`/`.tgz`), extracts into `destDir`, validates an `index.html` exists at the extracted root, and removes `destDir`'s contents entirely on any failure.

- [ ] **Step 1: Write the failing tests**

`internal/gamearchive/extract_test.go`:

```go
package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip Create(%q): %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write(%q): %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip Close(): %v", err)
	}
	return buf.Bytes()
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader(%q): %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("tar write(%q): %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close(): %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close(): %v", err)
	}
	return buf.Bytes()
}

func TestExtract_Zip_Success(t *testing.T) {
	data := buildZip(t, map[string]string{
		"index.html":     "<html></html>",
		"assets/game.js": "console.log('hi');",
	})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
		t.Errorf("index.html not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "assets", "game.js")); err != nil {
		t.Errorf("assets/game.js not extracted: %v", err)
	}
}

func TestExtract_TarGz_Success(t *testing.T) {
	data := buildTarGz(t, map[string]string{"index.html": "<html></html>"})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.tar.gz", dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
		t.Errorf("index.html not extracted: %v", err)
	}
}

func TestExtract_MissingIndexHTML_RollsBack(t *testing.T) {
	data := buildZip(t, map[string]string{"readme.txt": "no index here"})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should fail when index.html is missing")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir should be empty after a failed extract, found %d entries", len(entries))
	}
}

func TestExtract_PathTraversal_Rejected(t *testing.T) {
	data := buildZip(t, map[string]string{
		"index.html":            "<html></html>",
		"../../etc/passwd-evil": "malicious",
	})
	dest := t.TempDir()

	if err := Extract(bytes.NewReader(data), "build.zip", dest); err == nil {
		t.Fatal("Extract() should reject an archive with a path-traversal entry")
	}
}

func TestExtract_UnsupportedExtension(t *testing.T) {
	if err := Extract(bytes.NewReader([]byte("irrelevant")), "build.rar", t.TempDir()); err == nil {
		t.Fatal("Extract() should reject an unsupported archive extension")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gamearchive/... -v`
Expected: FAIL — package `gamearchive` does not exist yet.

- [ ] **Step 3: Implement extraction**

`internal/gamearchive/extract.go`:

```go
package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract detects the archive format from filename's extension and extracts
// archive into destDir. It requires an index.html at the extracted root and
// rejects any entry whose path would escape destDir. On any failure, destDir
// is left empty.
func Extract(archive io.Reader, filename, destDir string) error {
	data, err := io.ReadAll(archive)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}

	var extractErr error
	switch {
	case strings.HasSuffix(filename, ".zip"):
		extractErr = extractZip(data, destDir)
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
		extractErr = extractTarGz(data, destDir)
	case strings.HasSuffix(filename, ".tar"):
		extractErr = extractTar(data, destDir)
	default:
		return fmt.Errorf("unsupported archive extension for %q", filename)
	}
	if extractErr != nil {
		clearDir(destDir)
		return extractErr
	}

	if _, err := os.Stat(filepath.Join(destDir, "index.html")); err != nil {
		clearDir(destDir)
		return fmt.Errorf("archive is missing index.html at its root")
	}
	return nil
}

func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(byteReaderAt(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	for _, f := range r.File {
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		if err := writeFile(target, src); err != nil {
			src.Close()
			return err
		}
		src.Close()
	}
	return nil
}

func extractTarGz(data []byte, destDir string) error {
	gz, err := gzip.NewReader(byteReader(data))
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()
	return extractTarReader(tar.NewReader(gz), destDir)
}

func extractTar(data []byte, destDir string) error {
	return extractTarReader(tar.NewReader(byteReader(data)), destDir)
}

func extractTarReader(tr *tar.Reader, destDir string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFile(target, tr); err != nil {
				return err
			}
		}
	}
}

func safeJoin(root, name string) (string, error) {
	full := filepath.Join(root, name)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination directory", name)
	}
	return full, nil
}

func writeFile(target string, r io.Reader) error {
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func clearDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}
```

Add these two tiny byte-reader adapters to the same file (a `*zip.Reader` needs an `io.ReaderAt`, a `*gzip.Reader`/`tar.NewReader` need an `io.Reader`):

```go
import "bytes"

func byteReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

func byteReaderAt(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
```

Fold this `bytes` import into the main import block above rather than appending a second one.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gamearchive/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gamearchive
git commit -m "feat: add game archive extraction with path traversal protection"
```

---

### Task 8: Game archive upload endpoint

**Files:**
- Create: `internal/gameupload/upload_handler.go`
- Test: `internal/gameupload/upload_handler_test.go`

**Interfaces:**
- Consumes: `gamearchive.Extract` (Task 7)
- Produces: `gameupload.NewHandler(gamesDir string, onExtracted func(slug string) error) *Handler`, `(*Handler).Upload` (`func(http.ResponseWriter, *http.Request)`, expects the slug as a `{slug}` path value set by the router)

- [ ] **Step 1: Write the failing tests**

`internal/gameupload/upload_handler_test.go`:

```go
package gameupload

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func zipWithIndex(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("index.html")
	if err != nil {
		t.Fatalf("zip Create(): %v", err)
	}
	if _, err := f.Write([]byte("<html></html>")); err != nil {
		t.Fatalf("zip write(): %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip Close(): %v", err)
	}
	return buf.Bytes()
}

func uploadRequest(t *testing.T, slug string, archiveData []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "build.zip")
	if err != nil {
		t.Fatalf("CreateFormFile(): %v", err)
	}
	if _, err := part.Write(archiveData); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/games/"+slug+"/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("slug", slug)
	return req
}

func TestUpload_ExtractsAndCallsCallback(t *testing.T) {
	gamesDir := t.TempDir()
	var calledWithSlug string
	handler := NewHandler(gamesDir, func(slug string) error {
		calledWithSlug = slug
		return nil
	})

	req := uploadRequest(t, "pixel-quest", zipWithIndex(t))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if calledWithSlug != "pixel-quest" {
		t.Errorf("callback slug = %q, want %q", calledWithSlug, "pixel-quest")
	}
	if _, err := httpFileExists(filepath.Join(gamesDir, "pixel-quest", "index.html")); err != nil {
		t.Errorf("expected index.html to be extracted: %v", err)
	}
}

func TestUpload_InvalidArchiveReturnsBadRequest(t *testing.T) {
	gamesDir := t.TempDir()
	handler := NewHandler(gamesDir, func(slug string) error { return nil })

	req := uploadRequest(t, "pixel-quest", []byte("not a real archive"))
	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
```

Add a tiny test-only helper in the same test file so the test above compiles:

```go
import "os"

func httpFileExists(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
```

Fold this `os` import into the test file's main import block.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gameupload/... -v`
Expected: FAIL — package `gameupload` does not exist yet.

- [ ] **Step 3: Implement the handler**

`internal/gameupload/upload_handler.go`:

```go
package gameupload

import (
	"net/http"
	"path/filepath"

	"pixabros/internal/gamearchive"
	"pixabros/internal/httpapi"
)

type Handler struct {
	gamesDir    string
	onExtracted func(slug string) error
}

func NewHandler(gamesDir string, onExtracted func(slug string) error) *Handler {
	return &Handler{gamesDir: gamesDir, onExtracted: onExtracted}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_slug", "a game slug is required")
		return
	}
	if slug != filepath.Base(slug) || slug == "." || slug == ".." {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_slug", "slug must be a single path segment")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_file", "a file field is required")
		return
	}
	defer file.Close()

	destDir := filepath.Join(h.gamesDir, slug)
	if err := gamearchive.Extract(file, header.Filename, destDir); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_archive", err.Error())
		return
	}

	if h.onExtracted != nil {
		if err := h.onExtracted(slug); err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "extracted but could not finish processing")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
```

`gamearchive.Extract` requires `destDir` to already exist for its internal `os.Stat`/`clearDir` calls to behave — add a directory creation step before extraction:

```go
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare destination directory")
		return
	}
```

Insert this right before the `gamearchive.Extract` call, and add `"os"` to the import block.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gameupload/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gameupload
git commit -m "feat: add game archive upload endpoint"
```

---

### Task 9: Render registry and persistence store

**Files:**
- Create: `internal/render/registry.go`
- Create: `internal/render/store.go`
- Test: `internal/render/registry_test.go`
- Test: `internal/render/store_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate` (Plan A Task 2/3), `storage.Storage` (Plan A Task 4)
- Produces: `render.Renderer func(pageKey string) (html []byte, tags []string, err error)`, `render.NewRegistry() *Registry`, `(*Registry).Register(pageKey string, r Renderer)`, `(*Registry).RegisterPrefix(prefix string, r Renderer)`, `(*Registry).Resolve(pageKey string) (Renderer, bool)`, `render.NewStore(db *sql.DB, files storage.Storage) *Store`, `(*Store).RenderAndPersist(pageKey string, r Renderer) (etag string, err error)`, `(*Store).ETag(pageKey string) (etag string, found bool, err error)`, `(*Store).PageKeysForTag(tag string) ([]string, error)`

- [ ] **Step 1: Write the failing tests for the registry**

`internal/render/registry_test.go`:

```go
package render

import "testing"

func TestRegistry_ExactMatch(t *testing.T) {
	reg := NewRegistry()
	reg.Register("homepage", func(pageKey string) ([]byte, []string, error) {
		return []byte("home"), []string{"homepage"}, nil
	})

	r, ok := reg.Resolve("homepage")
	if !ok {
		t.Fatal("Resolve() should find an exact match")
	}
	html, tags, err := r("homepage")
	if err != nil {
		t.Fatalf("renderer error = %v", err)
	}
	if string(html) != "home" {
		t.Errorf("html = %q, want %q", html, "home")
	}
	if len(tags) != 1 || tags[0] != "homepage" {
		t.Errorf("tags = %v, want [homepage]", tags)
	}
}

func TestRegistry_PrefixMatch(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterPrefix("games/", func(pageKey string) ([]byte, []string, error) {
		return []byte("game page for " + pageKey), []string{"game:" + pageKey[len("games/"):]}, nil
	})

	r, ok := reg.Resolve("games/pixel-quest")
	if !ok {
		t.Fatal("Resolve() should find a prefix match")
	}
	html, tags, err := r("games/pixel-quest")
	if err != nil {
		t.Fatalf("renderer error = %v", err)
	}
	if string(html) != "game page for games/pixel-quest" {
		t.Errorf("html = %q", html)
	}
	if len(tags) != 1 || tags[0] != "game:pixel-quest" {
		t.Errorf("tags = %v", tags)
	}
}

func TestRegistry_NoMatch(t *testing.T) {
	reg := NewRegistry()
	if _, ok := reg.Resolve("unknown"); ok {
		t.Error("Resolve() should not find a renderer for an unregistered page key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/... -v -run TestRegistry`
Expected: FAIL — package `render` does not exist yet.

- [ ] **Step 3: Implement the registry**

`internal/render/registry.go`:

```go
package render

import "strings"

type Renderer func(pageKey string) (html []byte, tags []string, err error)

type prefixEntry struct {
	prefix   string
	renderer Renderer
}

type Registry struct {
	exact    map[string]Renderer
	prefixes []prefixEntry
}

func NewRegistry() *Registry {
	return &Registry{exact: make(map[string]Renderer)}
}

func (r *Registry) Register(pageKey string, renderer Renderer) {
	r.exact[pageKey] = renderer
}

func (r *Registry) RegisterPrefix(prefix string, renderer Renderer) {
	r.prefixes = append(r.prefixes, prefixEntry{prefix: prefix, renderer: renderer})
}

func (r *Registry) Resolve(pageKey string) (Renderer, bool) {
	if renderer, ok := r.exact[pageKey]; ok {
		return renderer, true
	}
	var best prefixEntry
	found := false
	for _, entry := range r.prefixes {
		if strings.HasPrefix(pageKey, entry.prefix) && len(entry.prefix) > len(best.prefix) {
			best = entry
			found = true
		}
	}
	if found {
		return best.renderer, true
	}
	return nil, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/... -v -run TestRegistry`
Expected: PASS

- [ ] **Step 5: Write the failing tests for the store**

`internal/render/store_test.go`:

```go
package render

import (
	"path/filepath"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	return NewStore(conn, files)
}

func TestStore_RenderAndPersist_WritesFileAndTags(t *testing.T) {
	store := setupTestStore(t)

	renderer := func(pageKey string) ([]byte, []string, error) {
		return []byte("<h1>hi</h1>"), []string{"homepage", "site_settings"}, nil
	}

	etag, err := store.RenderAndPersist("index.html", renderer)
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}
	if etag == "" {
		t.Fatal("RenderAndPersist() returned an empty etag")
	}

	gotEtag, found, err := store.ETag("index.html")
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}
	if !found {
		t.Fatal("ETag() should find the just-rendered page")
	}
	if gotEtag != etag {
		t.Errorf("ETag() = %q, want %q", gotEtag, etag)
	}

	pages, err := store.PageKeysForTag("homepage")
	if err != nil {
		t.Fatalf("PageKeysForTag() error = %v", err)
	}
	if len(pages) != 1 || pages[0] != "index.html" {
		t.Errorf("PageKeysForTag(\"homepage\") = %v, want [index.html]", pages)
	}
}

func TestStore_RenderAndPersist_ReplacesOldTags(t *testing.T) {
	store := setupTestStore(t)

	first := func(pageKey string) ([]byte, []string, error) {
		return []byte("v1"), []string{"tag-a"}, nil
	}
	if _, err := store.RenderAndPersist("page.html", first); err != nil {
		t.Fatalf("first RenderAndPersist() error = %v", err)
	}

	second := func(pageKey string) ([]byte, []string, error) {
		return []byte("v2"), []string{"tag-b"}, nil
	}
	if _, err := store.RenderAndPersist("page.html", second); err != nil {
		t.Fatalf("second RenderAndPersist() error = %v", err)
	}

	pagesForOldTag, err := store.PageKeysForTag("tag-a")
	if err != nil {
		t.Fatalf("PageKeysForTag(\"tag-a\") error = %v", err)
	}
	if len(pagesForOldTag) != 0 {
		t.Errorf("PageKeysForTag(\"tag-a\") = %v, want empty after re-render dropped that tag", pagesForOldTag)
	}

	pagesForNewTag, err := store.PageKeysForTag("tag-b")
	if err != nil {
		t.Fatalf("PageKeysForTag(\"tag-b\") error = %v", err)
	}
	if len(pagesForNewTag) != 1 || pagesForNewTag[0] != "page.html" {
		t.Errorf("PageKeysForTag(\"tag-b\") = %v, want [page.html]", pagesForNewTag)
	}
}

func TestStore_ETag_NotFound(t *testing.T) {
	store := setupTestStore(t)
	_, found, err := store.ETag("missing.html")
	if err != nil {
		t.Fatalf("ETag() error = %v", err)
	}
	if found {
		t.Error("ETag() should report not-found for a page that was never rendered")
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/render/... -v -run TestStore`
Expected: FAIL — `NewStore` undefined.

- [ ] **Step 7: Implement the store**

`internal/render/store.go`:

```go
package render

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	"pixabros/internal/storage"
)

type Store struct {
	db    *sql.DB
	files storage.Storage
}

func NewStore(db *sql.DB, files storage.Storage) *Store {
	return &Store{db: db, files: files}
}

// RenderAndPersist calls renderer, writes the resulting HTML via Storage
// (keyed by pageKey), and replaces rendered_pages/page_tags bookkeeping for
// pageKey with the renderer's freshly declared dependencies.
func (s *Store) RenderAndPersist(pageKey string, renderer Renderer) (string, error) {
	html, tags, err := renderer(pageKey)
	if err != nil {
		return "", err
	}

	etag := computeETag(html)

	if err := s.files.Put(renderedFileKey(pageKey), bytes.NewReader(html)); err != nil {
		return "", err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO rendered_pages (page_key, etag) VALUES (?, ?)
		 ON CONFLICT(page_key) DO UPDATE SET etag = excluded.etag, rendered_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');`,
		pageKey, etag,
	); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_key = ?;`, pageKey); err != nil {
		return "", err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT INTO page_tags (page_key, tag) VALUES (?, ?);`, pageKey, tag); err != nil {
			return "", err
		}
	}

	return etag, tx.Commit()
}

func (s *Store) ETag(pageKey string) (string, bool, error) {
	var etag string
	err := s.db.QueryRow(`SELECT etag FROM rendered_pages WHERE page_key = ?;`, pageKey).Scan(&etag)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return etag, true, nil
}

func (s *Store) PageKeysForTag(tag string) ([]string, error) {
	rows, err := s.db.Query(`SELECT page_key FROM page_tags WHERE tag = ?;`, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pageKeys []string
	for rows.Next() {
		var pk string
		if err := rows.Scan(&pk); err != nil {
			return nil, err
		}
		pageKeys = append(pageKeys, pk)
	}
	return pageKeys, rows.Err()
}

func renderedFileKey(pageKey string) string {
	return "rendered/" + pageKey
}

func computeETag(html []byte) string {
	sum := sha256.Sum256(html)
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/render/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/render/registry.go internal/render/store.go internal/render/registry_test.go internal/render/store_test.go
git commit -m "feat: add render registry and rendered-page persistence store"
```

---

### Task 10: Regen job queue and worker

**Files:**
- Create: `internal/render/queue.go`
- Test: `internal/render/queue_test.go`

**Interfaces:**
- Consumes: `render.Registry`, `(*Registry).Resolve` (Task 9), `render.Store`, `(*Store).RenderAndPersist`, `(*Store).PageKeysForTag` (Task 9)
- Produces: `render.EnqueueRegen(db *sql.DB, tag string) error`, `render.NewWorker(db *sql.DB, registry *Registry, store *Store, pollInterval time.Duration) *Worker`, `(*Worker).ProcessOnce() (processed int, err error)`, `(*Worker).Run(ctx context.Context)`

- [ ] **Step 1: Write the failing tests**

`internal/render/queue_test.go`:

```go
package render

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func TestEnqueueRegen_InsertsPendingJob(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	if err := EnqueueRegen(conn, "homepage"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	var status, tag string
	err = conn.QueryRow(`SELECT tag, status FROM regen_jobs;`).Scan(&tag, &status)
	if err != nil {
		t.Fatalf("query regen_jobs: %v", err)
	}
	if tag != "homepage" || status != "pending" {
		t.Errorf("job = (%q, %q), want (\"homepage\", \"pending\")", tag, status)
	}
}

func TestWorker_ProcessOnce_RendersDependentPages(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	renderCount := 0
	registry.Register("index.html", func(pageKey string) ([]byte, []string, error) {
		renderCount++
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})

	// Seed page_tags as if index.html had been rendered once before, so the
	// worker knows it depends on "homepage".
	if _, err := store.RenderAndPersist("index.html", registry.exact["index.html"]); err != nil {
		t.Fatalf("seed RenderAndPersist() error = %v", err)
	}
	renderCount = 0

	if err := EnqueueRegen(conn, "homepage"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	processed, err := worker.ProcessOnce()
	if err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}
	if renderCount != 1 {
		t.Errorf("renderCount = %d, want 1", renderCount)
	}

	var status string
	if err := conn.QueryRow(`SELECT status FROM regen_jobs;`).Scan(&status); err != nil {
		t.Fatalf("query job status: %v", err)
	}
	if status != "done" {
		t.Errorf("status = %q, want %q", status, "done")
	}
}

func TestWorker_ProcessOnce_MarksJobFailedWhenNoRendererRegistered(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	registry := NewRegistry()

	// A page_tags row exists (as if rendered before) but no renderer is
	// registered anymore for it — this must fail the job, not panic.
	if _, err := conn.Exec(`INSERT INTO rendered_pages (page_key, etag) VALUES ('orphaned.html', 'x');`); err != nil {
		t.Fatalf("seed rendered_pages: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO page_tags (page_key, tag) VALUES ('orphaned.html', 'ghost-tag');`); err != nil {
		t.Fatalf("seed page_tags: %v", err)
	}

	if err := EnqueueRegen(conn, "ghost-tag"); err != nil {
		t.Fatalf("EnqueueRegen() error = %v", err)
	}

	worker := NewWorker(conn, registry, store, 10*time.Millisecond)
	if _, err := worker.ProcessOnce(); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	var status string
	var jobErr string
	if err := conn.QueryRow(`SELECT status, error FROM regen_jobs;`).Scan(&status, &jobErr); err != nil {
		t.Fatalf("query job: %v", err)
	}
	if status != "failed" {
		t.Errorf("status = %q, want %q", status, "failed")
	}
	if jobErr == "" {
		t.Error("expected a non-empty error message on the failed job")
	}
}

func TestWorker_Run_StopsOnContextCancel(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	worker := NewWorker(conn, NewRegistry(), NewStore(conn, files), 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/render/... -v -run "TestEnqueueRegen|TestWorker"`
Expected: FAIL — `EnqueueRegen`/`NewWorker` undefined.

- [ ] **Step 3: Implement the queue and worker**

`internal/render/queue.go`:

```go
package render

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func EnqueueRegen(db *sql.DB, tag string) error {
	_, err := db.Exec(`INSERT INTO regen_jobs (tag) VALUES (?);`, tag)
	return err
}

type Worker struct {
	db           *sql.DB
	registry     *Registry
	store        *Store
	pollInterval time.Duration
}

func NewWorker(db *sql.DB, registry *Registry, store *Store, pollInterval time.Duration) *Worker {
	return &Worker{db: db, registry: registry, store: store, pollInterval: pollInterval}
}

// ProcessOnce drains every pending regen job once. It returns the number of
// jobs it processed (successfully or not).
func (w *Worker) ProcessOnce() (int, error) {
	rows, err := w.db.Query(`SELECT id, tag FROM regen_jobs WHERE status = 'pending' ORDER BY id;`)
	if err != nil {
		return 0, err
	}
	type job struct {
		id  int64
		tag string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.tag); err != nil {
			rows.Close()
			return 0, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	for _, j := range jobs {
		if _, err := w.db.Exec(`UPDATE regen_jobs SET status = 'processing' WHERE id = ?;`, j.id); err != nil {
			return 0, err
		}
		if err := w.processTag(j.tag); err != nil {
			w.db.Exec(
				`UPDATE regen_jobs SET status = 'failed', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'), error = ? WHERE id = ?;`,
				err.Error(), j.id,
			)
			continue
		}
		w.db.Exec(
			`UPDATE regen_jobs SET status = 'done', processed_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
			j.id,
		)
	}
	return len(jobs), nil
}

func (w *Worker) processTag(tag string) error {
	pageKeys, err := w.store.PageKeysForTag(tag)
	if err != nil {
		return err
	}
	for _, pageKey := range pageKeys {
		renderer, ok := w.registry.Resolve(pageKey)
		if !ok {
			return fmt.Errorf("no renderer registered for page key %q (tag %q)", pageKey, tag)
		}
		if _, err := w.store.RenderAndPersist(pageKey, renderer); err != nil {
			return fmt.Errorf("render %q: %w", pageKey, err)
		}
	}
	return nil
}

// Run polls for pending jobs every pollInterval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessOnce()
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/queue.go internal/render/queue_test.go
git commit -m "feat: add durable regen job queue and polling worker"
```

---

### Task 11: Public page serving with ETag/304, immutable static assets, router wiring

**Files:**
- Create: `internal/render/serve.go`
- Test: `internal/render/serve_test.go`
- Modify: `internal/httpserver/router.go` (Plan A Task 10)
- Modify: `cmd/server/main.go` (Plan A Task 11)

**Interfaces:**
- Consumes: `render.Store`, `(*Store).ETag` (Task 9), `storage.Storage` (Plan A Task 4)
- Produces: `render.ServePages(store *Store, files storage.Storage) http.Handler`, `render.ServeImmutableAssets(dir string) http.Handler`

- [ ] **Step 1: Write the failing tests**

`internal/render/serve_test.go`:

```go
package render

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pixabros/internal/db"
	"pixabros/internal/storage"
)

func TestServePages_ServesRenderedHTMLWithETag(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	etag, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("ETag") != etag {
		t.Errorf("ETag header = %q, want %q", rec.Header().Get("ETag"), etag)
	}
	if !strings.Contains(rec.Body.String(), "<h1>home</h1>") {
		t.Errorf("body = %q, want it to contain the rendered HTML", rec.Body.String())
	}
}

func TestServePages_ReturnsNotModifiedOnMatchingETag(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}

	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)
	etag, err := store.RenderAndPersist("index.html", func(string) ([]byte, []string, error) {
		return []byte("<h1>home</h1>"), []string{"homepage"}, nil
	})
	if err != nil {
		t.Fatalf("RenderAndPersist() error = %v", err)
	}

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}
}

func TestServePages_UnknownPageReturns404(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate() error = %v", err)
	}
	files := storage.NewLocalDisk(t.TempDir(), "/rendered")
	store := NewStore(conn, files)

	handler := ServePages(store, files)
	req := httptest.NewRequest(http.MethodGet, "/never-rendered.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServeImmutableAssets_SetsCacheControl(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.abc123.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	handler := ServeImmutableAssets(dir)
	req := httptest.NewRequest(http.MethodGet, "/main.abc123.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got := rec.Header().Get("Cache-Control")
	if !strings.Contains(got, "immutable") || !strings.Contains(got, "max-age=31536000") {
		t.Errorf("Cache-Control = %q, want it to contain immutable and max-age=31536000", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/render/... -v -run "TestServe"`
Expected: FAIL — `ServePages`/`ServeImmutableAssets` undefined.

- [ ] **Step 3: Implement the serving handlers**

`internal/render/serve.go`:

```go
package render

import (
	"io"
	"net/http"
	"strings"

	"pixabros/internal/storage"
)

// ServePages serves pre-rendered HTML pages tracked in rendered_pages,
// honoring If-None-Match for 304 responses. The request path (minus the
// leading slash) is used as the page_key; a request for "/" maps to
// "index.html".
func ServePages(store *Store, files storage.Storage) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageKey := strings.TrimPrefix(r.URL.Path, "/")
		if pageKey == "" {
			pageKey = "index.html"
		}

		etag, found, err := store.ETag(pageKey)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		body, err := files.Get(renderedFileKey(pageKey))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer body.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, body)
	})
}

// ServeImmutableAssets serves static files (content-hashed CSS/JS) from dir
// with a long-lived immutable Cache-Control header.
func ServeImmutableAssets(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/... -v`
Expected: PASS

- [ ] **Step 5: Wire the new handlers into the router**

Modify `internal/httpserver/router.go` (Plan A): add `Store *render.Store` and `AssetsDir string` to `Dependencies`, and replace the naive `http.FileServer` mount for `"/"` with `render.ServePages`, adding a dedicated `/assets/` mount for immutable static files:

```go
import (
	"net/http"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
	"pixabros/internal/render"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	AdminUIDir string
	PlayDir    string
	AssetsDir  string
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))

	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", http.FileServer(http.Dir(deps.AdminUIDir))))
	mux.Handle("/play/", http.StripPrefix("/play/", http.FileServer(http.Dir(deps.PlayDir))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", render.ServeImmutableAssets(deps.AssetsDir)))
	mux.Handle("/", render.ServePages(deps.Store, deps.filesForStore()))

	return mux
}
```

`render.ServePages` needs the same `storage.Storage` the `Store` writes through; add a `Files storage.Storage` field to `Dependencies` instead of the `filesForStore()` placeholder above — replace that last two lines with:

```go
type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	Store      *render.Store
	Files      storage.Storage
	AdminUIDir string
	PlayDir    string
	AssetsDir  string
}
```

and

```go
	mux.Handle("/", render.ServePages(deps.Store, deps.Files))
```

adding `"pixabros/internal/storage"` to the import block.

Update `internal/httpserver/router_test.go` (Plan A) accordingly: it must now construct a `render.Store` (via `render.NewStore(conn, files)`) and pass `Files: files` instead of `PublicDir`, and pre-render an `index.html` via `store.RenderAndPersist` instead of writing a static file — run `go test ./internal/httpserver/... -v` after updating and fix any remaining compile errors before moving on.

- [ ] **Step 6: Wire the worker and new handlers into the server entrypoint**

Modify `cmd/server/main.go` (Plan A):

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/httpserver"
	"pixabros/internal/render"
	"pixabros/internal/storage"
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

	files := storage.NewLocalDisk(cfg.DataDir+"/media", "/media")
	renderedFiles := storage.NewLocalDisk(cfg.DataDir+"/rendered-store", "/rendered")
	store := render.NewStore(conn, renderedFiles)
	registry := render.NewRegistry()

	worker := render.NewWorker(conn, registry, store, 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		AssetsDir:  cfg.DataDir + "/assets",
	})

	_ = files // wired for media/game uploads by the per-module phases that register their handlers here

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
```

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: PASS for every package (including the Plan A packages modified in this task).

- [ ] **Step 8: Commit**

```bash
git add internal/render/serve.go internal/render/serve_test.go internal/httpserver cmd/server
git commit -m "feat: serve rendered pages with etag support and immutable static assets"
```

---

## Definition of Done

- `go build ./...` succeeds.
- `go test ./...` passes with no skipped packages.
- Uploading an image via the media endpoint produces a WebP file at the exact target dimensions and a `media` row.
- Uploading a `.zip`/`.tar`/`.tar.gz` game archive without `index.html` at the root is rejected and leaves no partial files behind.
- `EnqueueRegen` + `Worker.ProcessOnce` re-renders every page tracked against the enqueued tag and updates its ETag.
- A request for a page whose ETag matches `If-None-Match` gets `304`; a mismatched or missing ETag gets a full `200` response.
- No `any` type appears anywhere in the Go source (`grep -rn '\bany\b' --include='*.go' .` returns nothing outside comments/strings).
