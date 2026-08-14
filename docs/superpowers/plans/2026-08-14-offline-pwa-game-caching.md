# Çevrimdışı PWA ve Oyun Cache'leme — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sitenin kurulu PWA'sını çevrimdışı çalışır hale getirmek ve kullanıcının seçtiği oyun build'lerini indirip çevrimdışı oynanabilir yapmak.

**Architecture:** Tek kök service worker (`/sw.js`, scope `/`) site kabuğunu, gezilen sayfaları ve `/play/*`'ı yönetir. Oyun dosyalarını sayfadaki indirme betiği `game-{slug}-{version}` cache'ine yazar; worker yalnızca okur. Build'in dosya listesi ve içerik türevli sürümü, arşiv zaten dolaşılırken extractor tarafından üretilip `games` tablosunda saklanır.

**Tech Stack:** Go 1.26 (stdlib `net/http`, `archive/zip`, `archive/tar`, `crypto/sha256`), SQLite (modernc, elle yazılmış SQL), düz JavaScript (framework yok), vitest.

**Spec:** `docs/superpowers/specs/2026-08-14-offline-pwa-game-caching-design.md`

## Global Constraints

- **`any` tipi kullanılmaz** (kullanıcının global kuralı). Go tarafında mevcut kod `interface{}` kullanıyor; ona dokunma, yeni kodda somut tip kullan.
- **Public site İngilizce.** Kullanıcıya görünen her metin İngilizce; Türkçe yalnızca admin panelinde. Bu plandaki ve spec'teki Türkçe açıklamalar dokümantasyondur, kopyalanacak metin değil.
- **Commit mesajları tek satır, semantic** (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`), gövde yok, `Co-Authored-By` trailer'ı **yok**.
- **CSS elle yazılır**, düz CSS, `:root` token'ları üzerinden. Tailwind yalnızca admin-ui'da.
- **Public sayfalar satır içi script/style taşımaz** — `publicCSP` `script-src 'self'` ve `style-src 'self'`; her şey `/assets` altında hash'li dosyadır.
- **Sürüm damgası uzunluğu:** 16 hex karakter (`versionLength = 16`).
- **Cache aileleri:** `shell-{version}`, `pages` (~60 girdi), `media` (~120 girdi), `game-{slug}-{version}`.
- **Tamamlanma işareti anahtarı:** `/play/{slug}/__offline-complete`.
- **Migration numarası:** 0024 (mevcut son migration 0023).

---

### Task 1: Extractor build manifesti üretsin

`gamearchive.Extract` bugün her arşiv girdisini dolaşıp diske yazıyor ama hiçbir şey hatırlamıyor. Bu görev o yürüyüşe defter tutmayı ekliyor, macOS çöpünü ve motorun kendi service worker'ını diskten dışlıyor.

**Files:**
- Modify: `internal/gamearchive/extract.go`
- Modify: `internal/gameupload/upload_handler.go:79` (yalnızca çağrı yerini derlenir tutmak için)
- Test: `internal/gamearchive/extract_test.go`

**Interfaces:**
- Consumes: yok, ilk görev.
- Produces:
  ```go
  type File struct {
      Path  string `json:"path"`
      Bytes int64  `json:"bytes"`
  }
  type Build struct {
      Files   []File
      Bytes   int64
      Version string
  }
  func Extract(archive io.Reader, filename, destDir string) (Build, error)
  ```

- [ ] **Step 1: Manifest testini yaz**

`internal/gamearchive/extract_test.go` sonuna ekle. Dosyanın import bloğunda `archive/zip`, `bytes`, `io`, `os`, `path/filepath`, `reflect`, `sort`, `testing` bulunmalı; eksik olanları ekle. `writeZip` adında bir yardımcı zaten varsa yenisini ekleme, mevcut olanı kullan.

```go
// writeZip builds a zip in memory from name -> contents, so a test can state
// exactly what an upload contained.
func writeZip(t *testing.T, entries map[string]string) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(entries[name])); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return bytes.NewReader(buf.Bytes()), "build.zip"
}

// The offline download needs to know what a build is made of, and how big each
// piece is, before it asks anyone to spend 90 MB.
func TestExtract_ReportsEveryFileItWrote(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":   "<h1>game</h1>",
		"game.wasm":    "wasm-bytes",
		"sub/data.pck": "pack",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	want := []File{
		{Path: "game.wasm", Bytes: 10},
		{Path: "index.html", Bytes: 13},
		{Path: "sub/data.pck", Bytes: 4},
	}
	if !reflect.DeepEqual(build.Files, want) {
		t.Errorf("files = %+v, want %+v", build.Files, want)
	}
	if build.Bytes != 27 {
		t.Errorf("Bytes = %d, want 27", build.Bytes)
	}
}

// Sorting is not cosmetic: zip and tar list entries in whatever order the
// archiver chose, and a version derived from that order would change when
// nothing about the build did.
func TestExtract_VersionIsTheContentNotTheOrdering(t *testing.T) {
	first, name := writeZip(t, map[string]string{"index.html": "a", "z.js": "b"})
	firstBuild, err := Extract(first, name, t.TempDir())
	if err != nil {
		t.Fatalf("first Extract() error = %v", err)
	}

	second, _ := writeZip(t, map[string]string{"z.js": "b", "index.html": "a"})
	secondBuild, err := Extract(second, name, t.TempDir())
	if err != nil {
		t.Fatalf("second Extract() error = %v", err)
	}

	if firstBuild.Version != secondBuild.Version {
		t.Errorf("the same build got two versions: %q vs %q", firstBuild.Version, secondBuild.Version)
	}
	if len(firstBuild.Version) != versionLength {
		t.Errorf("version = %q, want %d characters", firstBuild.Version, versionLength)
	}

	changed, _ := writeZip(t, map[string]string{"index.html": "a", "z.js": "CHANGED"})
	changedBuild, err := Extract(changed, name, t.TempDir())
	if err != nil {
		t.Fatalf("changed Extract() error = %v", err)
	}
	if changedBuild.Version == firstBuild.Version {
		t.Error("a changed file did not change the version")
	}
}

// Half of a Finder-made zip is resource forks. They were being served
// publicly, and downloading them to somebody's phone for offline play would
// be worse still.
func TestExtract_LeavesMacOSDebrisOnTheFloor(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":            "<h1>game</h1>",
		"__MACOSX/._index.html": "resource fork",
		"._index.html":          "resource fork",
		".DS_Store":             "finder",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(build.Files) != 1 || build.Files[0].Path != "index.html" {
		t.Errorf("files = %+v, want index.html alone", build.Files)
	}
	for _, junked := range []string{"__MACOSX", "._index.html", ".DS_Store"} {
		if _, err := os.Stat(filepath.Join(dir, junked)); !os.IsNotExist(err) {
			t.Errorf("%s was written to disk (err = %v)", junked, err)
		}
	}
}

// An engine's own service worker would claim /play/{slug}/ and shadow the
// site's, which is the scope the offline feature depends on.
func TestExtract_DropsTheEnginesOwnServiceWorker(t *testing.T) {
	dir := t.TempDir()
	archive, name := writeZip(t, map[string]string{
		"index.html":               "<h1>game</h1>",
		"game.service.worker.js":   "self.addEventListener('install', function () {});",
	})

	build, err := Extract(archive, name, dir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(build.Files) != 1 || build.Files[0].Path != "index.html" {
		t.Errorf("files = %+v, want index.html alone", build.Files)
	}
	if _, err := os.Stat(filepath.Join(dir, "game.service.worker.js")); !os.IsNotExist(err) {
		t.Errorf("the engine's service worker was written to disk (err = %v)", err)
	}
}
```

- [ ] **Step 2: Testleri koştur, derlenmediğini gör**

Run: `go test ./internal/gamearchive/ -run TestExtract_ 2>&1 | head -20`
Expected: derleme hatası — `Extract` tek değer döndürüyor, `File`/`Build`/`versionLength` tanımsız.

- [ ] **Step 3: `Build`, `File` ve yardımcılarını ekle**

`internal/gamearchive/extract.go` içinde `maxExtractedSize` bloğunun altına:

```go
// versionLength is how much of the sha256 becomes a build's version. Sixteen
// hex characters is far past collision territory for a handful of builds and
// stays short enough to read in a cache name.
const versionLength = 16

// File is one file of an extracted build: a slash-separated path relative to
// the build's root, and how many bytes it holds.
type File struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// Build is what an extraction produced.
//
// Version is derived from the contents of every file rather than from a clock,
// so re-uploading the same archive yields the same version and nobody is asked
// to download a build they already hold.
type Build struct {
	Files   []File
	Bytes   int64
	Version string
}

// collector accumulates what an extraction wrote. The archive walk already
// visits every entry, so this is the only walk needed.
type collector struct {
	files  []File
	hashes []string // parallel to files: the sha256 of each file's contents
}

func (c *collector) add(path string, bytes int64, hash string) {
	c.files = append(c.files, File{Path: filepath.ToSlash(path), Bytes: bytes})
	c.hashes = append(c.hashes, hash)
}

// result sorts the files by path and derives the version from their contents.
func (c *collector) result() Build {
	order := make([]int, len(c.files))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return c.files[order[a]].Path < c.files[order[b]].Path
	})

	sum := sha256.New()
	out := Build{Files: make([]File, 0, len(c.files))}
	for _, i := range order {
		out.Files = append(out.Files, c.files[i])
		out.Bytes += c.files[i].Bytes
		fmt.Fprintf(sum, "%s\x00%s\n", c.files[i].Path, c.hashes[i])
	}
	out.Version = hex.EncodeToString(sum.Sum(nil))[:versionLength]
	return out
}

// skip reports whether an archive entry is written to disk at all.
//
// Two kinds are refused. macOS packaging debris -- the resource fork a Finder
// zip carries for every file -- is not part of the build and was being served
// publicly. An engine's own service worker is refused because it would claim
// the /play/{slug}/ scope and shadow the site's worker, which is the scope the
// offline feature depends on.
func skip(name string) bool {
	slashed := filepath.ToSlash(name)
	if strings.HasSuffix(slashed, ".service.worker.js") {
		return true
	}
	for _, part := range strings.Split(slashed, "/") {
		if part == "__MACOSX" || part == ".DS_Store" || strings.HasPrefix(part, "._") {
			return true
		}
	}
	return false
}
```

Import bloğuna `crypto/sha256`, `encoding/hex`, `sort` ekle.

- [ ] **Step 4: `writeFile` hash döndürsün**

`internal/gamearchive/extract.go` içindeki `writeFile`'ı değiştir:

```go
// writeFile copies r into target, hashing it on the way through so the caller
// gets the content hash without a second read of the file.
func writeFile(target string, r io.Reader, budget *int64) (int64, string, error) {
	f, err := os.Create(target)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, sum), io.LimitReader(r, *budget+1))
	if err != nil {
		return 0, "", err
	}
	*budget -= n
	if *budget < 0 {
		return 0, "", fmt.Errorf("archive exceeds the %d byte decompressed size limit", maxExtractedSize)
	}
	return n, hex.EncodeToString(sum.Sum(nil)), nil
}
```

- [ ] **Step 5: Walker'ları collector'a bağla**

`extractZip`, `extractTarGz`, `extractTar`, `extractTarReader` imzalarına `into *collector` ekle ve gövdelerini güncelle.

`extractZip` içinde, `for _, f := range r.File {` satırının hemen altına:

```go
		if skip(f.Name) {
			continue
		}
```

ve `writeFile` çağrısını değiştir:

```go
			n, hash, err := writeFile(target, src, budget)
			if err != nil {
				src.Close()
				return err
			}
			src.Close()
			rel, err := filepath.Rel(destDir, target)
			if err != nil {
				return err
			}
			into.add(rel, n, hash)
```

`extractTarReader` içinde `target, err := safeJoin(...)` çağrısından önce:

```go
		if skip(hdr.Name) {
			continue
		}
```

ve `case tar.TypeReg:` dalındaki `writeFile` çağrısını:

```go
			n, hash, err := writeFile(target, tr, budget)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(destDir, target)
			if err != nil {
				return err
			}
			into.add(rel, n, hash)
```

`extractTarGz` ve `extractTar` yalnızca `into`'yu geçirir:

```go
func extractTarGz(data []byte, destDir string, budget *int64, into *collector) error {
	gz, err := gzip.NewReader(byteReader(data))
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()
	return extractTarReader(tar.NewReader(gz), destDir, budget, into)
}

func extractTar(data []byte, destDir string, budget *int64, into *collector) error {
	return extractTarReader(tar.NewReader(byteReader(data)), destDir, budget, into)
}
```

- [ ] **Step 6: `Extract`'ın imzasını değiştir**

```go
func Extract(archive io.Reader, filename, destDir string) (Build, error) {
	limited := io.LimitReader(archive, maxArchiveSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Build{}, fmt.Errorf("read archive: %w", err)
	}
	if int64(len(data)) > maxArchiveSize {
		return Build{}, fmt.Errorf("archive exceeds the %d byte size limit", maxArchiveSize)
	}

	budget := maxExtractedSize
	into := &collector{}
	var extractErr error
	switch {
	case strings.HasSuffix(filename, ".zip"):
		extractErr = extractZip(data, destDir, &budget, into)
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
		extractErr = extractTarGz(data, destDir, &budget, into)
	case strings.HasSuffix(filename, ".tar"):
		extractErr = extractTar(data, destDir, &budget, into)
	default:
		return Build{}, fmt.Errorf("unsupported archive extension for %q", filename)
	}
	if extractErr != nil {
		clearDir(destDir)
		return Build{}, extractErr
	}

	if _, err := os.Stat(filepath.Join(destDir, "index.html")); err != nil {
		clearDir(destDir)
		return Build{}, fmt.Errorf("archive is missing index.html at its root")
	}
	return into.result(), nil
}
```

- [ ] **Step 7: Çağrı yerini derlenir tut**

`internal/gameupload/upload_handler.go:79`'u değiştir (manifest Task 3'te kullanılacak):

```go
	if _, err := gamearchive.Extract(file, header.Filename, stagingDir); err != nil {
```

- [ ] **Step 8: Testleri koştur**

Run: `go build ./... && go test ./internal/gamearchive/ ./internal/gameupload/ -v -run 'TestExtract_|TestUpload' 2>&1 | grep -E '^(--- |ok|FAIL)'`
Expected: hepsi PASS. Mevcut extract testleri de geçmeli — eğer eski bir test `Extract`'ı tek dönüşle çağırıyorsa imzasını düzelt.

- [ ] **Step 9: Commit**

```bash
git add internal/gamearchive/ internal/gameupload/
git commit -m "feat: have the extractor report what a build is made of"
```

---

### Task 2: Manifesti veritabanında sakla

**Files:**
- Create: `internal/db/migrations/0024_game_build_manifest.sql`
- Modify: `internal/games/repo.go` (`Game` struct, `gameColumns:88-93`, `scanGame:99`, `SetBuild:266`)
- Modify: `internal/gamesapi/handlers.go:403` (`SetBuild` çağrısı)
- Test: `internal/games/repo_test.go`

**Interfaces:**
- Consumes: Task 1'in `gamearchive.Build`'i (yalnızca kavramsal; repo `gamearchive`'i import etmez).
- Produces:
  ```go
  // games.Game kazanır:
  BuildVersion   string
  BuildBytes     int64
  BuildFilesJSON string

  type BuildInfo struct {
      Version   string
      Bytes     int64
      FilesJSON string
  }
  func (r *Repo) SetBuild(id string, path string, info BuildInfo) error
  ```

- [ ] **Step 1: Migration'ı yaz**

`internal/db/migrations/0024_game_build_manifest.sql`:

```sql
-- What a game's playable build is made of, so a visitor can be told the size
-- before being asked to download it and so a re-upload can be recognised.
--
-- build_version is derived from the contents of every file in the build, not
-- from a clock: re-uploading the same archive must not ask anyone to download
-- 90 MB they already hold. build_files_json follows external_links_json's
-- convention -- the list is only ever read whole, so a table of its own would
-- buy nothing.

ALTER TABLE games ADD COLUMN build_version TEXT NOT NULL DEFAULT '';
ALTER TABLE games ADD COLUMN build_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE games ADD COLUMN build_files_json TEXT NOT NULL DEFAULT '[]';
```

- [ ] **Step 2: Repo testini yaz**

`internal/games/repo_test.go` sonuna ekle:

```go
// The offline download reads all three, so a build that recorded its path but
// not its manifest would be a build nobody can be told the size of.
func TestSetBuild_RecordsTheManifest(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	game, err := repo.Create(CreateInput{Title: "Covered", Kind: KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	info := BuildInfo{
		Version:   "a1b2c3d4e5f60718",
		Bytes:     46137344,
		FilesJSON: `[{"path":"index.html","bytes":12873}]`,
	}
	if err := repo.SetBuild(game.ID, "/data/games/covered", info); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	stored, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if stored.BuildVersion != info.Version {
		t.Errorf("BuildVersion = %q, want %q", stored.BuildVersion, info.Version)
	}
	if stored.BuildBytes != info.Bytes {
		t.Errorf("BuildBytes = %d, want %d", stored.BuildBytes, info.Bytes)
	}
	if stored.BuildFilesJSON != info.FilesJSON {
		t.Errorf("BuildFilesJSON = %q, want %q", stored.BuildFilesJSON, info.FilesJSON)
	}
}

// Deleting a build clears the manifest with it. A game still advertising a
// 46 MB download it no longer has would hand visitors a 404 per file.
func TestSetBuild_ClearingAlsoClearsTheManifest(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	game, err := repo.Create(CreateInput{Title: "Covered", Kind: KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, "/data/games/covered", BuildInfo{
		Version: "a1b2c3d4e5f60718", Bytes: 1, FilesJSON: `[{"path":"index.html","bytes":1}]`,
	}); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	if err := repo.SetBuild(game.ID, "", BuildInfo{}); err != nil {
		t.Fatalf("clearing SetBuild() error = %v", err)
	}

	stored, err := repo.FindByID(game.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if stored.BuildVersion != "" || stored.BuildBytes != 0 {
		t.Errorf("build left behind: version %q, bytes %d", stored.BuildVersion, stored.BuildBytes)
	}
	// An empty manifest must be valid JSON, not an empty string: it is served
	// straight to the browser.
	if stored.BuildFilesJSON != "[]" {
		t.Errorf("BuildFilesJSON = %q, want []", stored.BuildFilesJSON)
	}
	if stored.IsBrowserPlayable {
		t.Error("a game with no build is still marked browser playable")
	}
}
```

`CreateInput`'un gerçek alanları için `internal/games/repo.go:44` civarına bak; `setupTestDB` ve `KindProduction` mevcut test dosyasında zaten var.

- [ ] **Step 3: Testi koştur, kırmızı olduğunu gör**

Run: `go test ./internal/games/ -run TestSetBuild_ 2>&1 | head -20`
Expected: derleme hatası — `BuildInfo` tanımsız, `SetBuild` iki argüman alıyor.

- [ ] **Step 4: `Game`'e alanları ve `BuildInfo`'yu ekle**

`internal/games/repo.go`, `WebExportPath string` satırının altına:

```go
	// BuildVersion, BuildBytes and BuildFilesJSON describe the extracted
	// build. The offline download reads all three: the version tells a
	// returning visitor whether the copy they hold is still current, and the
	// rest is what they would be spending.
	BuildVersion   string
	BuildBytes     int64
	BuildFilesJSON string
```

`Game` struct'ının altına:

```go
// BuildInfo is a build's manifest as the repository stores it. The zero value
// clears a build, which is what deleting one passes.
type BuildInfo struct {
	Version string
	Bytes   int64
	// FilesJSON is a JSON array of {"path","bytes"} objects. Empty is stored
	// as "[]" -- it is served straight to the browser, so it must always be
	// valid JSON.
	FilesJSON string
}
```

- [ ] **Step 5: Sütunları ve taramayı güncelle**

`gameColumns`'un son satırını değiştir:

```go
	web_export_path, build_version, build_bytes, build_files_json,
	display_order, is_published, created_at, updated_at`
```

`scanGame`'deki `row.Scan` çağrısında `&webExportPath,` satırını değiştir:

```go
		&webExportPath, &g.BuildVersion, &g.BuildBytes, &g.BuildFilesJSON,
		&g.DisplayOrder, &g.IsPublished,
```

(eski `&g.DisplayOrder, &g.IsPublished,` satırını sil, yukarıdaki onu kapsıyor.)

- [ ] **Step 6: `SetBuild`'i güncelle**

```go
// SetBuild records where a game's playable build was extracted to and what it
// is made of, and derives is_browser_playable from whether there is one.
func (r *Repo) SetBuild(id string, path string, info BuildInfo) error {
	filesJSON := info.FilesJSON
	if filesJSON == "" {
		filesJSON = "[]"
	}
	res, err := r.db.Exec(
		`UPDATE games SET web_export_path = ?, is_browser_playable = ?,
			build_version = ?, build_bytes = ?, build_files_json = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
		nullableString(path), path != "",
		info.Version, info.Bytes, filesJSON, id,
	)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}
```

Mevcut doküman yorumunu (`repo.go:261-265`) yukarıdakiyle değiştir.

- [ ] **Step 7: Silme çağrısını güncelle**

`internal/gamesapi/handlers.go:403`:

```go
	if err := h.repo.SetBuild(game.ID, "", games.BuildInfo{}); err != nil {
```

- [ ] **Step 8: Testleri koştur**

Run: `go build ./... && go test ./internal/games/ ./internal/gamesapi/ ./internal/db/ 2>&1 | tail -10`
Expected: hepsi `ok`.

- [ ] **Step 9: Commit**

```bash
git add internal/db/migrations/ internal/games/ internal/gamesapi/
git commit -m "feat: store a game build's manifest and content version"
```

---

### Task 3: Yüklemeyi manifeste bağla ve build uç noktasını aç

**Files:**
- Modify: `internal/gameupload/upload_handler.go` (`onExtracted` imzası)
- Modify: `internal/httpserver/router.go:145-157` (`onGameArchiveExtracted`), yeni public rota
- Create: `internal/gamesapi/public.go`
- Test: `internal/gamesapi/public_test.go`, `internal/gameupload/upload_handler_test.go`

**Interfaces:**
- Consumes: `gamearchive.Build` (Task 1), `games.BuildInfo` ve `games.Game.Build*` (Task 2).
- Produces:
  ```go
  func NewHandler(gamesDir string, onExtracted func(slug string, build gamearchive.Build) error, opts ...HandlerOption) *Handler

  // gamesapi
  func NewPublicHandlers(repo *games.Repo) *PublicHandlers
  func (h *PublicHandlers) Build(w http.ResponseWriter, r *http.Request)
  // rota: GET /api/games/{slug}/build
  ```

- [ ] **Step 1: Uç nokta testini yaz**

`internal/gamesapi/public_test.go`:

```go
package gamesapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pixabros/internal/games"
)

// buildResponse mirrors the payload the download script parses.
type buildResponse struct {
	Version string `json:"version"`
	Bytes   int64  `json:"bytes"`
	Files   []struct {
		Path  string `json:"path"`
		Bytes int64  `json:"bytes"`
	} `json:"files"`
}

// getBuild drives the handler through a mux, so {slug} is populated the same
// way the real router populates it.
func getBuild(t *testing.T, repo *games.Repo, slug string) *http.Response {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/games/{slug}/build", NewPublicHandlers(repo).Build)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/games/"+slug+"/build", nil))
	return recorder.Result()
}

func TestPublicBuild_ServesTheManifest(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)
	game, err := repo.Create(games.CreateInput{Title: "Covered", IsPublished: true, Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(game.ID, "/data/games/covered", games.BuildInfo{
		Version:   "a1b2c3d4e5f60718",
		Bytes:     46137344,
		FilesJSON: `[{"path":"index.html","bytes":12873}]`,
	}); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	response := getBuild(t, repo, game.Slug)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var body buildResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Version != "a1b2c3d4e5f60718" || body.Bytes != 46137344 {
		t.Errorf("version/bytes = %q/%d", body.Version, body.Bytes)
	}
	if len(body.Files) != 1 || body.Files[0].Path != "index.html" || body.Files[0].Bytes != 12873 {
		t.Errorf("files = %+v", body.Files)
	}
}

// A draft's build is not public, and neither is a published game that has no
// build: both would advertise a download that cannot be completed.
func TestPublicBuild_HidesDraftsAndBuildlessGames(t *testing.T) {
	conn := setupTestDB(t)
	repo := games.NewRepo(conn)

	draft, err := repo.Create(games.CreateInput{Title: "Draft", Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.SetBuild(draft.ID, "/data/games/draft", games.BuildInfo{
		Version: "a1b2c3d4e5f60718", Bytes: 1, FilesJSON: `[{"path":"index.html","bytes":1}]`,
	}); err != nil {
		t.Fatalf("SetBuild() error = %v", err)
	}

	buildless, err := repo.Create(games.CreateInput{Title: "Buildless", IsPublished: true, Kind: games.KindProduction})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for name, slug := range map[string]string{
		"draft":     draft.Slug,
		"buildless": buildless.Slug,
		"unknown":   "no-such-game",
	} {
		if got := getBuild(t, repo, slug).StatusCode; got != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", name, got)
		}
	}
}
```

`setupTestDB` `internal/gamesapi` içinde zaten var; yoksa `handlers_test.go`'daki karşılığını kullan.

- [ ] **Step 2: Testi koştur, kırmızı olduğunu gör**

Run: `go test ./internal/gamesapi/ -run TestPublicBuild_ 2>&1 | head -10`
Expected: derleme hatası — `NewPublicHandlers` tanımsız.

- [ ] **Step 3: Public handler'ı yaz**

`internal/gamesapi/public.go`:

```go
package gamesapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"pixabros/internal/games"
	"pixabros/internal/httpapi"
)

// PublicHandlers serve the game data the public site's scripts need. Nothing
// here requires a session: it is the same data the rendered pages already
// carry, in the shape a script can read.
type PublicHandlers struct {
	repo *games.Repo
}

func NewPublicHandlers(repo *games.Repo) *PublicHandlers {
	return &PublicHandlers{repo: repo}
}

// buildBody is the manifest of a playable build. Per-file byte counts are
// carried because the download's progress is read from them.
type buildBody struct {
	Version string      `json:"version"`
	Bytes   int64       `json:"bytes"`
	Files   []buildFile `json:"files"`
}

type buildFile struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// Build answers with what a game's playable build is made of, so the offline
// download can state the size before spending it and fetch each file by name.
//
// A draft and a game with no build are both 404: advertising a download that
// cannot be completed is worse than admitting there is nothing to download.
func (h *PublicHandlers) Build(w http.ResponseWriter, r *http.Request) {
	game, err := h.repo.FindBySlug(r.PathValue("slug"))
	if errors.Is(err, games.ErrGameNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such game")
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load game")
		return
	}
	if !game.IsPublished || !game.IsBrowserPlayable || game.BuildVersion == "" {
		httpapi.WriteError(w, http.StatusNotFound, "not_found", "no such game")
		return
	}

	var files []buildFile
	if err := json.Unmarshal([]byte(game.BuildFilesJSON), &files); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not read the build manifest")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, buildBody{
		Version: game.BuildVersion,
		Bytes:   game.BuildBytes,
		Files:   files,
	})
}
```

- [ ] **Step 4: Testi koştur, yeşile döndüğünü gör**

Run: `go test ./internal/gamesapi/ -run TestPublicBuild_ -v 2>&1 | grep -E '^(--- |ok|FAIL)'`
Expected: iki test de PASS.

- [ ] **Step 5: Yükleme kancasını manifest taşır hale getir**

`internal/gameupload/upload_handler.go`:

```go
type Handler struct {
	gamesDir    string
	onExtracted func(slug string, build gamearchive.Build) error
	onError     func(error)
}

func NewHandler(gamesDir string, onExtracted func(slug string, build gamearchive.Build) error, opts ...HandlerOption) *Handler {
	h := &Handler{gamesDir: gamesDir, onExtracted: onExtracted, onError: func(error) {}}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
```

`Upload` gövdesinde `gamearchive.Extract` çağrısını ve kanca çağrısını:

```go
	build, err := gamearchive.Extract(file, header.Filename, stagingDir)
	if err != nil {
```

```go
	if h.onExtracted != nil {
		if err := h.onExtracted(slug, build); err != nil {
```

- [ ] **Step 6: Router'ı bağla**

`internal/httpserver/router.go:145`:

```go
	onGameArchiveExtracted := func(slug string, build gamearchive.Build) error {
		game, err := deps.Games.FindBySlug(slug)
		if err != nil {
			return err
		}
		files, err := json.Marshal(build.Files)
		if err != nil {
			return err
		}
		if err := deps.Games.SetBuild(game.ID, filepath.Join(deps.PlayDir, slug), games.BuildInfo{
			Version:   build.Version,
			Bytes:     build.Bytes,
			FilesJSON: string(files),
		}); err != nil {
			return err
		}
		return render.EnqueueRegen(deps.DB, fmt.Sprintf("game:%s", game.ID))
	}
```

Import bloğuna `encoding/json` ve `pixabros/internal/gamearchive` ekle.

`mux.HandleFunc("POST /api/contact", publicContact.Submit)` satırının (`router.go:122`) altına:

```go
	publicGames := gamesapi.NewPublicHandlers(deps.Games)
	mux.HandleFunc("GET /api/games/{slug}/build", publicGames.Build)
```

- [ ] **Step 7: Uçtan uca doğrula**

Run: `go build ./... && go test ./... 2>&1 | grep -v "no test files" | grep -v "^ok" ; echo "EXIT $?"`
Expected: hiçbir FAIL satırı yok. `internal/gameupload` testlerinde `onExtracted` stub'ları iki argümanlı olacak şekilde güncellenmeli.

- [ ] **Step 8: Commit**

```bash
git add internal/gameupload/ internal/httpserver/ internal/gamesapi/
git commit -m "feat: publish what a game build is made of"
```

---

### Task 4: Kabuk uç noktası, çevrimdışı sayfası, worker rotası ve CSP

Worker'ın kendisi Task 5-6'da. Bu görev onun ihtiyaç duyduğu sunucu tarafını, worker olmadan da tek başına test edilebilecek şekilde kuruyor.

**Files:**
- Create: `internal/site/shell.go`, `internal/site/templates/offline.html`, `internal/site/shell_test.go`
- Modify: `internal/site/site.go` (`PageOffline` sabiti, `pages()`), `internal/site/templates.go` (`OfflineScript`)
- Modify: `internal/site/templates/layout.html`
- Modify: `internal/httpserver/router.go`, `internal/httpserver/security.go`
- Modify: `cmd/server/main.go`
- Test: `internal/httpserver/security_test.go`, `internal/httpserver/router_test.go`

**Interfaces:**
- Consumes: `site.Bundle` (mevcut), `site.ManifestPath` deseni (mevcut).
- Produces:
  ```go
  const PageOffline = "offline"
  const ServiceWorkerPath = "/sw.js"
  const ShellPath = "/api/shell"
  func (s *Site) ShellHandler() http.Handler          // {"version":"...","urls":[...]}
  func ServiceWorkerHandler(bundle *Bundle, assetsDir string) (http.Handler, error)
  // Dependencies kazanır: ServiceWorker http.Handler, Shell http.Handler
  ```

- [ ] **Step 1: Kabuk testini yaz**

`internal/site/shell_test.go`:

```go
package site

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// shellResponse mirrors the payload the worker parses. It is named apart from
// shell.go's own shellBody: both live in package site.
type shellResponse struct {
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

func fetchShell(t *testing.T, site *Site) shellResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	site.ShellHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, ShellPath, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body shellResponse
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("the shell list is not JSON: %v", err)
	}
	return body
}

// Everything a page needs to render with no network. A missing entry here is a
// page that opens offline unstyled or without its typeface.
func TestShellHandler_ListsEverythingAPageNeeds(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)

	body := fetchShell(t, site)

	for _, name := range []string{
		"site.css", "osd.js", "offline.js",
		"fonts/archivo.woff2", "fonts/public-sans.woff2", "fonts/vt323.woff2",
		"logo.svg", "icon-192.png", "icon-512.png",
	} {
		want := site.renderer.bundle.URL(name)
		if want == "" {
			t.Fatalf("%s is not in the bundle at all", name)
		}
		if !contains(body.URLs, want) {
			t.Errorf("the shell list is missing %s (%s)", name, want)
		}
	}
	// The page shown when a visitor asks for something never visited.
	if !contains(body.URLs, "/"+PageOffline) {
		t.Errorf("the shell list is missing the offline page: %v", body.URLs)
	}
}

// The worker compares this stamp to decide whether to re-precache. If it did
// not move when the stylesheet did, a visitor who never comes back online
// would open a page whose CSS is not in the cache.
func TestShellHandler_StampMovesWithTheAssets(t *testing.T) {
	conn := setupTestDB(t)
	site := newTestSite(t, conn)

	body := fetchShell(t, site)
	if body.Version == "" {
		t.Fatal("the shell list carries no version")
	}
	if fetchShell(t, site).Version != body.Version {
		t.Error("the stamp changed without the assets changing")
	}
	if strings.Contains(body.Version, "/") {
		t.Errorf("version = %q, want an opaque stamp", body.Version)
	}
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Testi koştur, kırmızı olduğunu gör**

Run: `go test ./internal/site/ -run TestShellHandler_ 2>&1 | head -10`
Expected: derleme hatası — `ShellPath`, `ShellHandler`, `PageOffline` tanımsız.

- [ ] **Step 3: Çevrimdışı sayfasının şablonunu yaz**

`internal/site/templates/offline.html`:

```html
{{define "main"}}
  <div class="page-header">
    <h1>You are offline</h1>
    <p class="page-header__lead">
      This page has not been opened on this device before, so there is no copy
      of it here. Anything you have already visited still works, and so does
      every game you made playable offline.
    </p>
  </div>
{{end}}
```

- [ ] **Step 4: Kabuk uç noktasını ve çevrimdışı sayfasını yaz**

`internal/site/shell.go`:

```go
package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// ShellPath is where the service worker asks what a page needs to render with
// no network.
const ShellPath = "/api/shell"

// stampLength is how much of the sha256 becomes the shell's version. It only
// has to change when the list does, so it can be short.
const stampLength = 12

// shellAssets are the bundle names every page depends on. They are listed
// rather than derived from the bundle's whole contents so that adding a
// page-specific script does not silently enlarge what every visitor
// downloads on install.
var shellAssets = []string{
	"site.css",
	"osd.js",
	"offline.js",
	"fonts/archivo.woff2",
	"fonts/public-sans.woff2",
	"fonts/vt323.woff2",
	"logo.svg",
	"icon-192.png",
	"icon-512.png",
}

type shellBody struct {
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

// ShellHandler answers with the URLs a page needs offline, and a stamp that
// changes when they do.
//
// The worker cannot hold these URLs itself: they are content-hashed, so they
// move whenever the stylesheet or a script changes. The stamp is what lets a
// worker notice the move without re-downloading the list's contents.
func (s *Site) ShellHandler() http.Handler {
	urls := make([]string, 0, len(shellAssets)+1)
	for _, name := range shellAssets {
		if url := s.renderer.bundle.URL(name); url != "" {
			urls = append(urls, url)
		}
	}
	urls = append(urls, "/"+PageOffline)

	sum := sha256.Sum256([]byte(strings.Join(urls, "\n")))
	body, err := json.Marshal(shellBody{
		Version: hex.EncodeToString(sum[:])[:stampLength],
		URLs:    urls,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// The list moves with a deploy, and a worker holding an old one would
		// precache assets that no longer exist.
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	})
}

// ServiceWorkerHandler serves the worker at a stable path.
//
// Everything else under /assets is content-hashed and cached forever. The
// worker cannot be: a browser decides whether to update one by re-fetching the
// same URL and comparing bytes, so a moving URL would mean a worker that never
// updates in place. The bytes are read once at startup because the bundle is
// built once at startup.
func ServiceWorkerHandler(bundle *Bundle, assetsDir string) (http.Handler, error) {
	url := bundle.URL("sw.js")
	if url == "" {
		return nil, errNoServiceWorker
	}
	body, err := os.ReadFile(filepath.Join(assetsDir, strings.TrimPrefix(url, "/assets/")))
	if err != nil {
		return nil, fmt.Errorf("read service worker: %w", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(body)
	}), nil
}

var errNoServiceWorker = errors.New("sw.js is not in the asset bundle")
```

Import bloğuna `errors`, `fmt`, `os`, `path/filepath` ekle.

`internal/site/site.go`'da `PageAwards` sabitinin yanına:

```go
	// PageOffline is what a visitor gets when they ask for a page this device
	// has never held. It is a real page so that the worker has something to
	// precache rather than a string built into the worker itself.
	PageOffline = "offline"
```

`internal/site/site.go`'daki `pages()` listesine ekle:

```go
		{Key: PageOffline, Render: s.renderOffline},
```

`internal/site/shell.go` sonuna:

```go
// renderOffline draws the page shown when a visitor asks for something this
// device has never held.
func (s *Site) renderOffline(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}
	html, err := s.renderer.render("offline.html", pageData{
		Title:       "You are offline",
		Description: "This page has not been opened on this device before.",
		Canonical:   canonicalURL(chrome.URL, PageOffline),
		Site:        chrome,
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{siteSettingsTag}, nil
}
```

- [ ] **Step 5: `offline.js` yer tutucusunu ekle ve her sayfaya bağla**

`internal/site/assets/offline.js` (Task 6 doldurur; bundle'ın adı çözebilmesi için şimdi var olmalı):

```javascript
// Offline support: registers the service worker and drives the per-game
// download control. Filled in by later tasks.
(function () {})();
```

`internal/site/templates.go`, `ChromeScript` alanının altına:

```go
	// OfflineScript registers the service worker and drives the offline
	// download control. It is on every page for the same reason ChromeScript
	// is: the worker has to be registered wherever a visitor lands.
	OfflineScript string
```

`renderer.render` içinde, `data.ChromeScript = r.bundle.URL("osd.js")` satırının altına:

```go
	data.OfflineScript = r.bundle.URL("offline.js")
```

`internal/site/templates/layout.html`, `<script src="{{.ChromeScript}}" defer></script>` satırının altına:

```html
    {{if .OfflineScript}}<script src="{{.OfflineScript}}" defer></script>{{end}}
```

- [ ] **Step 6: CSP testini yaz ve direktifi ekle**

`internal/httpserver/security_test.go`, `TestPublicCSP_AllowsItsOwnManifest`'in altına:

```go
// worker-src falls back to child-src and then to default-src, which is 'none'.
// A policy that does not name it refuses to start the service worker, and the
// site loses offline support without a single visible error.
func TestPublicCSP_AllowsItsOwnServiceWorker(t *testing.T) {
	if got := directives(publicCSP)["worker-src"]; got != "'self'" {
		t.Errorf("public worker-src = %q, want 'self'", got)
	}
}
```

`internal/httpserver/security.go`, `publicCSP` içinde `"manifest-src 'self'; " +` satırının altına:

```go
	// worker-src falls back to child-src and then to default-src, which is
	// 'none'. Without it the service worker never starts and offline support
	// disappears with no visible error.
	"worker-src 'self'; " +
```

- [ ] **Step 7: Rota testini yaz**

`internal/httpserver/router_test.go` sonuna:

```go
// The worker must be reachable at the root: a script served from /assets could
// only ever claim /assets as its scope.
func TestNew_ServesTheServiceWorkerFromTheRoot(t *testing.T) {
	deps := publicDeps(t)
	deps.ServiceWorker = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte("self.addEventListener('install', function () {});"))
	})
	srv := httptest.NewServer(New(deps))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/sw.js")
	if err != nil {
		t.Fatalf("GET /sw.js: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/javascript" {
		t.Errorf("Content-Type = %q, want application/javascript", got)
	}
	// A cached worker is a worker that never updates.
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "worker-src 'self'") {
		t.Errorf("the worker is served under a policy that blocks it: %q", got)
	}
}

// Tests and any deployment that never built the assets leave it unset.
func TestNew_WithoutAServiceWorkerLeavesThePathAlone(t *testing.T) {
	srv := httptest.NewServer(New(publicDeps(t)))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/sw.js")
	if err != nil {
		t.Fatalf("GET /sw.js: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
```

- [ ] **Step 8: Rotaları bağla**

`internal/httpserver/router.go`, `Dependencies`'e `Manifest`'in yanına:

```go
	// ServiceWorker serves the offline worker at the root, and Shell answers
	// with what it should precache. Nil leaves each route unmounted.
	ServiceWorker http.Handler
	Shell         http.Handler
```

`site.ManifestPath` rotasının altına:

```go
	if deps.ServiceWorker != nil {
		mux.Handle("GET "+site.ServiceWorkerPath, deps.ServiceWorker)
	}
	if deps.Shell != nil {
		mux.Handle("GET "+site.ShellPath, deps.Shell)
	}
```

`internal/site/shell.go`'a sabiti ekle:

```go
// ServiceWorkerPath is where the worker is served. It must be at the root:
// a worker's default scope is the directory it was served from.
const ServiceWorkerPath = "/sw.js"
```

`cmd/server/main.go`, `publicSite.Register(registry)` satırının altına:

```go
	serviceWorker, err := site.ServiceWorkerHandler(assets, cfg.DataDir+"/assets")
	if err != nil {
		log.Fatalf("build service worker handler: %v", err)
	}
```

ve `Dependencies` literaline:

```go
		ServiceWorker: serviceWorker,
		Shell:         publicSite.ShellHandler(),
```

- [ ] **Step 9: `sw.js` yer tutucusunu ekle**

`internal/site/assets/sw.js` (Task 5-6 doldurur; `ServiceWorkerHandler` başlangıçta onu bulamazsa sunucu açılmaz):

```javascript
// The site's offline worker. Filled in by later tasks.
self.addEventListener("install", function () {
  self.skipWaiting();
});
```

- [ ] **Step 10: Her şeyi koştur**

Run: `go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "no test files" | grep -v "^ok"; echo "EXIT $?"`
Expected: FAIL yok. `internal/site` golden dosyası artık `offline.js` script etiketi taşıdığı için değişecek: `UPDATE_GOLDEN=1 go test ./internal/site/ -run TestRenderAwards_MatchesGoldenFile` ile yenile, sonra tüm testleri tekrar koştur.

- [ ] **Step 11: Gerçek sunucuda doğrula**

```bash
go build -o /tmp/pixabros-check ./cmd/server && /tmp/pixabros-check &
sleep 3
curl -s http://localhost:8080/api/shell | head -c 400; echo
curl -s -o /dev/null -w '/sw.js %{http_code} %{content_type}\n' http://localhost:8080/sw.js
curl -s -o /dev/null -w '/offline %{http_code}\n' http://localhost:8080/offline
kill %1
```
Expected: kabuk listesi hash'li URL'ler ve bir sürüm damgası içeriyor; `/sw.js` 200 `application/javascript`; `/offline` 200.

- [ ] **Step 12: Commit**

```bash
git add internal/site/ internal/httpserver/ cmd/server/ 
git commit -m "feat: serve the offline shell list, page and worker route"
```

---

### Task 5: Vitest kurulumu ve worker'ın saf mantığı

Worker'ın karar veren kısımları tarayıcı gerektirmeyen saf fonksiyonlara ayrılıp test edilir. Kancalar Task 6'da.

**Files:**
- Create: `package.json`, `vitest.config.js`, `internal/site/assets/sw.test.js`
- Modify: `internal/site/assets/sw.js`, `.gitignore`, `Makefile`

**Interfaces:**
- Consumes: Task 4'ün `ShellPath`, `ServiceWorkerPath` sabitleri (yalnızca değer olarak).
- Produces: `self.SWLogic` üzerinde
  ```javascript
  classify(pathname)                  // "page" | "asset" | "media" | "play" | "bypass"
  gameCacheName(slug, version)        // "game-<slug>-<version>"
  parseGameCache(name)                // {slug, version} | null
  playSlug(pathname)                  // "<slug>" | null
  completeKey(slug)                   // "/play/<slug>/__offline-complete"
  isKeepable(name, shellVersion)      // bool
  ```

- [ ] **Step 1: npm projesini kur**

`package.json`:

```json
{
  "name": "pixabros-site-assets",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "vitest run"
  },
  "devDependencies": {
    "vitest": "^3.2.4"
  }
}
```

`vitest.config.js`:

```javascript
// Only the public site's scripts. admin-ui is its own npm project with its own
// vitest, dependencies and tsconfig; pointing one runner at both would mean
// one config serving two unrelated builds.
export default {
  test: {
    include: ["internal/site/assets/**/*.test.js"],
  },
};
```

`.gitignore` sonuna:

```
# The public site's test runner. The directory already existed empty.
/node_modules/
```

`Makefile`'daki `test` hedefini değiştir:

```makefile
test:
	go test ./...
	npm test
	npm --prefix admin-ui run test
```

- [ ] **Step 2: Saf mantığın testini yaz**

`internal/site/assets/sw.test.js`:

```javascript
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// sw.js is a classic service worker, not a module: it runs top-level
// self.addEventListener calls and cannot be imported. Evaluating it against a
// stub global gives the pure logic without a browser, and keeps the worker one
// file rather than splitting it just to be testable.
function loadWorker() {
  const source = readFileSync("internal/site/assets/sw.js", "utf8");
  const self = { addEventListener() {}, skipWaiting() {}, clients: { claim() {} } };
  new Function("self", source)(self);
  return self.SWLogic;
}

const SW = loadWorker();

describe("classify", () => {
  it("sends a page to the network first", () => {
    expect(SW.classify("/")).toBe("page");
    expect(SW.classify("/games")).toBe("page");
    expect(SW.classify("/games/tetrabros")).toBe("page");
  });

  it("sends a hashed asset to the cache first", () => {
    expect(SW.classify("/assets/build/site.f0c84817.css")).toBe("asset");
    expect(SW.classify("/assets/build/fonts/vt323.woff2")).toBe("asset");
  });

  // Media keys are not content-hashed, so a cache-first copy would outlive a
  // deleted image forever.
  it("sends an upload to the network first", () => {
    expect(SW.classify("/media/cd_cover_art/2026-abc.webp")).toBe("media");
  });

  // A game's iframe is a navigation too, so the prefix has to be checked
  // before the request type.
  it("sends a game build to its own cache", () => {
    expect(SW.classify("/play/tetrabros/")).toBe("play");
    expect(SW.classify("/play/tetrabros/tetrabros.wasm")).toBe("play");
  });

  it("never touches the API or the admin panel", () => {
    expect(SW.classify("/api/shell")).toBe("bypass");
    expect(SW.classify("/api/games/tetrabros/build")).toBe("bypass");
    expect(SW.classify("/I-am-a-pixabro/")).toBe("bypass");
    expect(SW.classify("/I-am-a-pixabro/games")).toBe("bypass");
  });
});

describe("game caches", () => {
  it("round-trips a name", () => {
    const name = SW.gameCacheName("tetrabros", "a1b2c3d4e5f60718");
    expect(SW.parseGameCache(name)).toEqual({
      slug: "tetrabros",
      version: "a1b2c3d4e5f60718",
    });
  });

  // Slugs carry hyphens, so a naive split on "-" would read the wrong slug.
  it("round-trips a hyphenated slug", () => {
    const name = SW.gameCacheName("dungrid-tactics", "0011223344556677");
    expect(SW.parseGameCache(name)).toEqual({
      slug: "dungrid-tactics",
      version: "0011223344556677",
    });
  });

  it("refuses a name from another family", () => {
    expect(SW.parseGameCache("pages")).toBeNull();
    expect(SW.parseGameCache("shell-abc123")).toBeNull();
  });

  it("reads the slug out of a play path", () => {
    expect(SW.playSlug("/play/tetrabros/tetrabros.wasm")).toBe("tetrabros");
    expect(SW.playSlug("/play/tetrabros/")).toBe("tetrabros");
    expect(SW.playSlug("/games/tetrabros")).toBeNull();
  });

  it("names the completion marker inside the build's own scope", () => {
    expect(SW.completeKey("tetrabros")).toBe("/play/tetrabros/__offline-complete");
  });
});

describe("isKeepable", () => {
  // Activate drops anything outside the four families, so a renamed cache from
  // an older worker cannot linger and hold storage the visitor cannot see.
  it("keeps the current shell and drops an older one", () => {
    expect(SW.isKeepable("shell-abc123def456", "abc123def456")).toBe(true);
    expect(SW.isKeepable("shell-000000000000", "abc123def456")).toBe(false);
  });

  it("keeps the bounded runtime caches", () => {
    expect(SW.isKeepable("pages", "abc123def456")).toBe(true);
    expect(SW.isKeepable("media", "abc123def456")).toBe(true);
  });

  // A downloaded game outlives shell versions on purpose: the visitor decides
  // when to spend 90 MB again, not a deploy.
  it("keeps every game cache whatever its version", () => {
    expect(SW.isKeepable("game-tetrabros-0011223344556677", "abc123def456")).toBe(true);
  });

  it("drops anything else", () => {
    expect(SW.isKeepable("leftovers", "abc123def456")).toBe(false);
  });
});
```

- [ ] **Step 3: Bağımlılığı kur ve testi koştur, kırmızı olduğunu gör**

Run: `npm install && npm test 2>&1 | tail -20`
Expected: FAIL — `SW.classify` tanımsız (`SWLogic` yok).

- [ ] **Step 4: Saf mantığı yaz**

`internal/site/assets/sw.js` içeriğini şununla değiştir:

```javascript
// The site's offline worker.
//
// One worker owns the whole origin, including the uploaded game builds under
// /play. An engine's own worker would otherwise claim that scope and shadow
// this one, which is why extraction drops any it finds in an archive.
//
// The decisions live in SWLogic as plain functions so they can be tested
// without a browser; the event handlers below are the thin part.

var SHELL_PATH = "/api/shell";
var PAGES_CACHE = "pages";
var MEDIA_CACHE = "media";
var PAGES_LIMIT = 60;
var MEDIA_LIMIT = 120;
var GAME_PREFIX = "game-";
var SHELL_PREFIX = "shell-";

self.SWLogic = {
  // classify decides which strategy a request gets. The URL prefix is checked
  // before the request type on purpose: a game's iframe is a navigation, but
  // it must be served from the game's cache rather than from the page cache.
  classify: function (pathname) {
    if (pathname.indexOf("/api/") === 0 || pathname.indexOf("/I-am-a-pixabro/") === 0) {
      return "bypass";
    }
    if (pathname.indexOf("/play/") === 0) {
      return "play";
    }
    if (pathname.indexOf("/assets/") === 0) {
      return "asset";
    }
    if (pathname.indexOf("/media/") === 0) {
      return "media";
    }
    return "page";
  },

  gameCacheName: function (slug, version) {
    return GAME_PREFIX + slug + "-" + version;
  },

  // parseGameCache splits a game cache name back apart. The version is taken
  // from the last hyphen rather than the first, because slugs carry hyphens
  // ("dungrid-tactics") and the version never does.
  parseGameCache: function (name) {
    if (name.indexOf(GAME_PREFIX) !== 0) {
      return null;
    }
    var rest = name.slice(GAME_PREFIX.length);
    var split = rest.lastIndexOf("-");
    if (split <= 0 || split === rest.length - 1) {
      return null;
    }
    return { slug: rest.slice(0, split), version: rest.slice(split + 1) };
  },

  playSlug: function (pathname) {
    if (pathname.indexOf("/play/") !== 0) {
      return null;
    }
    var rest = pathname.slice("/play/".length);
    var end = rest.indexOf("/");
    var slug = end === -1 ? rest : rest.slice(0, end);
    return slug === "" ? null : slug;
  },

  // completeKey is written last, after every file of a build has landed. A
  // cache without it is a download that was interrupted, and treating it as
  // ready would mean a game that does not start on a plane.
  //
  // It sits inside the build's own path so it cannot collide with a real file
  // that the manifest lists.
  completeKey: function (slug) {
    return "/play/" + slug + "/__offline-complete";
  },

  // isKeepable reports whether a cache survives activation. Anything outside
  // the four families is from an older worker and holds storage the visitor
  // has no way to see or free.
  isKeepable: function (name, shellVersion) {
    if (name === PAGES_CACHE || name === MEDIA_CACHE) {
      return true;
    }
    if (name.indexOf(GAME_PREFIX) === 0) {
      // A downloaded game outlives shell versions deliberately: the visitor
      // decides when to spend 90 MB again, not a deploy.
      return self.SWLogic.parseGameCache(name) !== null;
    }
    return name === SHELL_PREFIX + shellVersion;
  },
};

self.addEventListener("install", function () {
  self.skipWaiting();
});
```

- [ ] **Step 5: Testi koştur, yeşile döndüğünü gör**

Run: `npm test 2>&1 | tail -20`
Expected: bütün testler PASS.

- [ ] **Step 6: Go tarafının hâlâ derlendiğini doğrula**

Run: `go build ./... && go test ./internal/site/ 2>&1 | tail -5`
Expected: `ok`. `sw.js` minify edilip bundle'a giriyor; `TestBuild_*` testleri geçmeli.

- [ ] **Step 7: Commit**

```bash
git add package.json vitest.config.js .gitignore Makefile internal/site/assets/sw.js internal/site/assets/sw.test.js
git commit -m "test: cover the offline worker's routing and cache-name logic"
```

---

### Task 6: Worker'ın kancaları ve kayıt

**Files:**
- Modify: `internal/site/assets/sw.js`
- Modify: `internal/site/assets/offline.js`

**Interfaces:**
- Consumes: `self.SWLogic` (Task 5), `ShellPath` (Task 4).
- Produces: kayıtlı ve çalışan bir worker; Task 7 `offline.js` içindeki indirme arayüzünü buna ekler.

- [ ] **Step 1: Kancaları yaz**

`internal/site/assets/sw.js` sonundaki `self.addEventListener("install", ...)` bloğunu şununla değiştir:

```javascript
// fetchShell asks the server what a page needs offline. The worker cannot hold
// these URLs itself: they are content-hashed, so they move whenever a script
// or the stylesheet changes.
function fetchShell() {
  return fetch(SHELL_PATH, { cache: "no-store" }).then(function (response) {
    if (!response.ok) {
      throw new Error("shell list unavailable");
    }
    return response.json();
  });
}

// precache stores the shell under a name carrying its stamp, so an older shell
// is a different cache and activation can drop it whole.
function precache(shell) {
  return caches.open(SHELL_PREFIX + shell.version).then(function (cache) {
    return cache.addAll(shell.urls);
  });
}

// currentShellVersion is remembered between events so activate knows which
// shell cache to keep without asking the network again.
var currentShellVersion = "";

// shellCheckInFlight keeps a burst of navigations from starting a burst of
// shell fetches. The list is a few hundred bytes, but one request per
// navigation would still be one request nobody asked for.
var shellCheckInFlight = false;

// refreshShell re-reads the shell list and re-precaches when the stamp moved.
//
// Install alone is not enough. A visitor who installs the worker, then a
// deploy moves the stylesheet, then the visitor goes offline, would open a
// page asking for a stylesheet URL that was never cached -- an unstyled page.
// Checking after a navigation that reached the network closes that.
function refreshShell() {
  if (shellCheckInFlight) {
    return;
  }
  shellCheckInFlight = true;
  fetchShell()
    .then(function (shell) {
      if (shell.version === currentShellVersion) {
        return null;
      }
      var previous = currentShellVersion;
      currentShellVersion = shell.version;
      return precache(shell).then(function () {
        if (previous) {
          return caches.delete(SHELL_PREFIX + previous);
        }
        return null;
      });
    })
    .catch(function () {})
    .then(function () {
      shellCheckInFlight = false;
    });
}

self.addEventListener("install", function (event) {
  event.waitUntil(
    fetchShell()
      .then(function (shell) {
        currentShellVersion = shell.version;
        return precache(shell);
      })
      // A failed install would leave the visitor with no worker at all, which
      // is worse than a worker whose shell fills in as they browse.
      .catch(function () {})
      .then(function () {
        return self.skipWaiting();
      })
  );
});

self.addEventListener("activate", function (event) {
  event.waitUntil(
    fetchShell()
      .then(function (shell) {
        currentShellVersion = shell.version;
        return precache(shell);
      })
      .catch(function () {})
      .then(function () {
        return caches.keys();
      })
      .then(function (names) {
        return Promise.all(
          names.map(function (name) {
            if (self.SWLogic.isKeepable(name, currentShellVersion)) {
              return null;
            }
            return caches.delete(name);
          })
        );
      })
      .then(function () {
        return self.clients.claim();
      })
  );
});

// trim keeps a runtime cache bounded by dropping its oldest entries. Cache
// API returns keys in insertion order, which is what makes this possible
// without keeping a second ledger; a true LRU would mean writing a timestamp
// on every read, and these two caches are not worth that.
function trim(cache, limit) {
  return cache.keys().then(function (keys) {
    if (keys.length <= limit) {
      return null;
    }
    return Promise.all(
      keys.slice(0, keys.length - limit).map(function (key) {
        return cache.delete(key);
      })
    );
  });
}

// networkFirst is what freshness-sensitive things get. The whole site is
// pre-rendered and served with an ETag; serving a cached copy in preference to
// the network would undo that.
function networkFirst(request, cacheName, limit) {
  return fetch(request)
    .then(function (response) {
      if (!response.ok) {
        return response;
      }
      var copy = response.clone();
      caches.open(cacheName).then(function (cache) {
        return cache.put(request, copy).then(function () {
          return trim(cache, limit);
        });
      });
      return response;
    })
    .catch(function () {
      return caches.match(request).then(function (cached) {
        if (cached) {
          return cached;
        }
        if (request.mode === "navigate") {
          return caches.match("/offline");
        }
        return Response.error();
      });
    });
}

// cacheFirst is for content-hashed assets: their URL changes when their bytes
// do, so a hit is never stale.
function cacheFirst(request, cacheName) {
  return caches.match(request).then(function (cached) {
    if (cached) {
      return cached;
    }
    return fetch(request).then(function (response) {
      if (response.ok) {
        var copy = response.clone();
        caches.open(cacheName).then(function (cache) {
          return cache.put(request, copy);
        });
      }
      return response;
    });
  });
}

// playFirst serves a game build out of whichever version the visitor holds.
// The completion marker is never handed out as a file: it is bookkeeping, not
// part of the build.
function playFirst(request, pathname) {
  var slug = self.SWLogic.playSlug(pathname);
  if (!slug || pathname === self.SWLogic.completeKey(slug)) {
    return fetch(request);
  }
  return caches.match(request).then(function (cached) {
    return cached || fetch(request);
  });
}

self.addEventListener("fetch", function (event) {
  var request = event.request;
  if (request.method !== "GET") {
    return;
  }
  var url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  var kind = self.SWLogic.classify(url.pathname);
  if (kind === "bypass") {
    return;
  }
  if (kind === "asset") {
    event.respondWith(cacheFirst(request, SHELL_PREFIX + currentShellVersion));
    return;
  }
  if (kind === "media") {
    event.respondWith(networkFirst(request, MEDIA_CACHE, MEDIA_LIMIT));
    return;
  }
  if (kind === "play") {
    event.respondWith(playFirst(request, url.pathname));
    return;
  }
  event.respondWith(
    networkFirst(request, PAGES_CACHE, PAGES_LIMIT).then(function (response) {
      // Only a navigation that actually reached the network proves we are
      // online, which is the moment worth spending a shell check on.
      if (request.mode === "navigate" && response && response.ok) {
        refreshShell();
      }
      return response;
    })
  );
});
```

- [ ] **Step 2: Kaydı yaz**

`internal/site/assets/offline.js` içeriğini şununla değiştir:

```javascript
// Offline support on the page's side: registers the worker, and (in the next
// task) drives the per-game download control.
(function () {
  if (!("serviceWorker" in navigator)) {
    return;
  }

  // Registration waits for load so it never competes with the page's own
  // requests on a first visit, which is the visit that matters most.
  window.addEventListener("load", function () {
    navigator.serviceWorker.register("/sw.js").catch(function () {
      // A site that cannot register a worker is a site without offline
      // support, not a broken site. Nothing here is worth an error dialog.
    });
  });
})();
```

- [ ] **Step 3: Saf mantık testlerinin hâlâ geçtiğini doğrula**

Run: `npm test 2>&1 | tail -10`
Expected: bütün testler PASS, değişiklik gerekmeden.

`loadWorker`'daki stub yeterli: bu görevin eklediği her şey ya bir fonksiyon gövdesinin ya da bir `addEventListener` geri çağrısının içinde. `caches`, `fetch` ve `self.location` yalnızca çalışma anında okunuyor, modül değerlendirilirken değil. Test bir şey isterse kancalardan biri en üst seviyede iş yapıyor demektir — onu düzelt, stub'ı büyütme.

- [ ] **Step 4: Tarayıcıda elle doğrula**

```bash
make build && ./pixabros &
sleep 3
```

Chrome'da `http://localhost:8080/` aç, DevTools → Application → Service Workers: `/sw.js` **activated and is running** olmalı. Cache Storage altında `shell-<damga>` görünmeli ve içinde `site.<hash>.css` ile üç font olmalı. Sonra:
- Network sekmesinde "Offline" işaretle, ana sayfayı yenile → sayfa açılmalı, stilli.
- Hiç girmediğin bir adrese git (`/devlog/none`) → çevrimdışı sayfası gelmeli.
- `/I-am-a-pixabro/` iste → worker'ın karışmadığını doğrula (Network'te "from ServiceWorker" olmamalı).

Bittiğinde `kill %1`.

- [ ] **Step 5: Commit**

```bash
git add internal/site/assets/sw.js internal/site/assets/offline.js
git commit -m "feat: serve the site offline from a root service worker"
```

---

### Task 7: Oyun indirme arayüzü

**Files:**
- Modify: `internal/site/assets/offline.js`
- Create: `internal/site/assets/offline.test.js`
- Modify: `internal/site/assets/site.css`
- Modify: `internal/site/templates/game.html`

**Interfaces:**
- Consumes: `GET /api/games/{slug}/build` (Task 3), `self.SWLogic` adlandırması (Task 5), kayıtlı worker (Task 6).
- Produces: son kullanıcı özelliği; sonrasında görev yok.

- [ ] **Step 1: Durum mantığının testini yaz**

`internal/site/assets/offline.test.js`:

```javascript
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// offline.js is a plain script guarded by an IIFE. Evaluating it against a
// stub window exposes its pure helpers without a browser.
function loadOffline() {
  const source = readFileSync("internal/site/assets/offline.js", "utf8");
  const window = { addEventListener() {} };
  const navigator = {};
  const document = { querySelectorAll: () => [] };
  new Function("window", "navigator", "document", source)(window, navigator, document);
  return window.OfflineLogic;
}

const Offline = loadOffline();

describe("stateFor", () => {
  it("offers the download when nothing is held", () => {
    expect(Offline.stateFor(null, "a1b2c3d4e5f60718")).toBe("absent");
  });

  it("reports ready when the held version is current", () => {
    expect(Offline.stateFor("a1b2c3d4e5f60718", "a1b2c3d4e5f60718")).toBe("ready");
  });

  it("reports stale when a newer build exists", () => {
    expect(Offline.stateFor("0011223344556677", "a1b2c3d4e5f60718")).toBe("stale");
  });

  // Offline the build endpoint is unreachable, so there is no current version
  // to compare against. Claiming staleness that cannot be verified would be a
  // lie; a held copy is simply ready.
  it("never claims staleness it cannot verify", () => {
    expect(Offline.stateFor("0011223344556677", null)).toBe("ready");
    expect(Offline.stateFor(null, null)).toBe("unavailable");
  });
});

describe("formatBytes", () => {
  it("reads as a download size, not as a number", () => {
    expect(Offline.formatBytes(46137344)).toBe("44 MB");
    expect(Offline.formatBytes(1024)).toBe("1 KB");
    expect(Offline.formatBytes(0)).toBe("0 KB");
  });
});

describe("hasRoomFor", () => {
  // Refusing up front beats failing halfway through a 90 MB download.
  it("leaves headroom above the download", () => {
    expect(Offline.hasRoomFor({ quota: 1000, usage: 0 }, 100)).toBe(true);
    expect(Offline.hasRoomFor({ quota: 1000, usage: 950 }, 100)).toBe(false);
  });

  // A browser that will not estimate is not a browser that has no room.
  it("allows the attempt when the browser will not say", () => {
    expect(Offline.hasRoomFor(null, 100)).toBe(true);
    expect(Offline.hasRoomFor({}, 100)).toBe(true);
  });
});
```

- [ ] **Step 2: Testi koştur, kırmızı olduğunu gör**

Run: `npm test 2>&1 | tail -20`
Expected: FAIL — `window.OfflineLogic` tanımsız.

- [ ] **Step 3: Saf yardımcıları yaz**

`internal/site/assets/offline.js` içindeki IIFE'nin başına, `if (!("serviceWorker" in navigator))` kontrolünden **önce**:

```javascript
  var GAME_PREFIX = "game-";

  // The decisions live here as plain functions so they can be tested without a
  // browser. Everything below them is DOM and network.
  window.OfflineLogic = {
    // stateFor decides what the control says. current is null when the build
    // endpoint could not be reached, which is exactly the offline case: a held
    // copy is then reported ready rather than stale, because staleness cannot
    // be verified without the server and claiming it would be a lie.
    stateFor: function (held, current) {
      if (!held) {
        return current ? "absent" : "unavailable";
      }
      if (!current || held === current) {
        return "ready";
      }
      return "stale";
    },

    // formatBytes writes a download size the way a visitor reads one. Anything
    // under a megabyte is rounded up to a kilobyte rather than shown as 0.
    formatBytes: function (bytes) {
      var mb = bytes / (1024 * 1024);
      if (mb >= 1) {
        return Math.round(mb) + " MB";
      }
      return Math.round(bytes / 1024) + " KB";
    },

    // hasRoomFor refuses up front rather than failing halfway through 90 MB.
    // A browser that will not estimate is not a browser that has no room, so
    // an absent estimate allows the attempt.
    hasRoomFor: function (estimate, bytes) {
      if (!estimate || typeof estimate.quota !== "number" || typeof estimate.usage !== "number") {
        return true;
      }
      return estimate.quota - estimate.usage > bytes * 1.1;
    },
  };
```

- [ ] **Step 4: Testi koştur, yeşile döndüğünü gör**

Run: `npm test 2>&1 | tail -10`
Expected: bütün testler PASS.

- [ ] **Step 5: İndirme arayüzünü yaz**

`internal/site/assets/offline.js` içindeki kayıt bloğunun altına ekle:

```javascript
  var mount = document.querySelector("[data-offline-game]");
  if (!mount || !window.caches || !navigator.storage) {
    return;
  }
  var slug = mount.getAttribute("data-offline-game");

  var control = document.createElement("div");
  control.className = "offline";
  mount.appendChild(control);

  var manifest = null; // {version, bytes, files} once fetched, null offline
  var held = null; // the version this device holds, or null

  function gameCacheName(version) {
    return GAME_PREFIX + slug + "-" + version;
  }

  // Mirrors SWLogic.completeKey in sw.js: the worker refuses to hand this key
  // out as a file, and the download writes it last.
  function completeKey() {
    return "/play/" + slug + "/__offline-complete";
  }

  // heldVersion finds the completed copy on this device. A cache without its
  // completion marker is an interrupted download: it is deleted rather than
  // reported, because a half-built game does not start.
  function heldVersion() {
    return window.caches.keys().then(function (names) {
      var mine = names.filter(function (name) {
        return name.indexOf(GAME_PREFIX + slug + "-") === 0;
      });
      return mine.reduce(function (chain, name) {
        return chain.then(function (found) {
          if (found) {
            return found;
          }
          return window.caches.open(name).then(function (cache) {
            return cache.match(completeKey()).then(function (marker) {
              if (!marker) {
                return window.caches.delete(name).then(function () {
                  return null;
                });
              }
              return marker.text();
            });
          });
        });
      }, Promise.resolve(null));
    });
  }

  function loadManifest() {
    return fetch("/api/games/" + slug + "/build")
      .then(function (response) {
        return response.ok ? response.json() : null;
      })
      .catch(function () {
        return null;
      });
  }

  function button(label, onClick) {
    var el = document.createElement("button");
    el.type = "button";
    el.className = "offline__action";
    el.textContent = label;
    el.addEventListener("click", onClick);
    return el;
  }

  function say(text) {
    var el = document.createElement("p");
    el.className = "offline__status";
    el.textContent = text;
    return el;
  }

  function render() {
    control.textContent = "";
    var state = window.OfflineLogic.stateFor(held, manifest ? manifest.version : null);

    if (state === "unavailable") {
      return;
    }
    if (state === "absent") {
      control.appendChild(
        button("Make playable offline — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
      );
      return;
    }
    if (state === "ready") {
      control.appendChild(say("Playable offline"));
      control.appendChild(button("Remove", remove));
      return;
    }
    control.appendChild(say("A new version is available"));
    control.appendChild(
      button("Update — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
    );
    control.appendChild(button("Remove", remove));
  }

  function download() {
    control.textContent = "";
    var status = say("Downloading… 0%");
    control.appendChild(status);

    navigator.storage
      .estimate()
      .catch(function () {
        return null;
      })
      .then(function (estimate) {
        if (!window.OfflineLogic.hasRoomFor(estimate, manifest.bytes)) {
          throw new Error("not enough room on this device");
        }
        // Asking to persist makes the browser less willing to evict the copy.
        // It is a request, not a guarantee, and is refused outright on iOS.
        return navigator.storage.persist ? navigator.storage.persist() : null;
      })
      .then(function () {
        return window.caches.open(gameCacheName(manifest.version));
      })
      .then(function (cache) {
        var done = 0;
        return manifest.files
          .reduce(function (chain, file) {
            return chain.then(function () {
              var url = "/play/" + slug + "/" + file.path;
              return fetch(url).then(function (response) {
                if (!response.ok) {
                  throw new Error("could not fetch " + file.path);
                }
                return cache.put(url, response).then(function () {
                  done += file.bytes;
                  status.textContent =
                    "Downloading… " + Math.round((done / manifest.bytes) * 100) + "%";
                });
              });
            });
          }, Promise.resolve())
          .then(function () {
            // Written last, and only once every file has landed: this marker is
            // what separates a finished download from an interrupted one.
            return cache.put(completeKey(), new Response(manifest.version));
          });
      })
      .then(cacheThisPage)
      .then(function () {
        var previous = held;
        held = manifest.version;
        // The old copy goes only now, so an interrupted update never costs a
        // visitor the game they already had.
        if (previous && previous !== held) {
          return window.caches.delete(gameCacheName(previous));
        }
        return null;
      })
      .then(render)
      .catch(function (err) {
        window.caches.delete(gameCacheName(manifest.version));
        control.textContent = "";
        control.appendChild(say("Could not save this game for offline play: " + err.message));
        control.appendChild(
          button("Try again — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
        );
      });
  }

  // cacheThisPage stores the game's own page and the images on it alongside the
  // build. Without it "Playable offline" would be a half-truth: the game would
  // run, but the page you launch it from would not open.
  //
  // Failures here are swallowed: a cached build with an uncached page is worth
  // far more than a download reported as failed.
  function cacheThisPage() {
    var images = [];
    var nodes = document.querySelectorAll("img[src]");
    for (var i = 0; i < nodes.length; i++) {
      var src = nodes[i].getAttribute("src");
      if (src && src.indexOf("/media/") === 0) {
        images.push(src);
      }
    }

    return window.caches
      .open("pages")
      .then(function (cache) {
        return cache.add(window.location.pathname);
      })
      .then(function () {
        return window.caches.open("media");
      })
      .then(function (cache) {
        return Promise.all(
          images.map(function (src) {
            return cache.add(src).catch(function () {});
          })
        );
      })
      .catch(function () {});
  }

  function remove() {
    var version = held;
    window.caches.delete(gameCacheName(version)).then(function () {
      held = null;
      render();
    });
  }

  Promise.all([loadManifest(), heldVersion()]).then(function (results) {
    manifest = results[0];
    held = results[1];
    render();
  });
```

- [ ] **Step 6: Şablona bağlantı noktasını ekle**

`internal/site/templates/game.html` içinde, oyna butonunun bulunduğu bloğun yanına (`Playable` koşulunun içine):

```html
        {{if .Playable}}<div data-offline-game="{{.Slug}}"></div>{{end}}
```

- [ ] **Step 7: Stilleri ekle**

`internal/site/assets/site.css` sonuna:

```css
/* The offline control. It is injected by script, so it never exists for a
   visitor without JavaScript and needs no no-JS state of its own. */
.offline {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.75rem;
}

.offline__status {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.offline__action {
  padding: 0.4rem 0.8rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 0.875rem;
  cursor: pointer;
}

.offline__action:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
```

- [ ] **Step 8: Bütün testleri koştur**

Run: `go build ./... && go test ./... 2>&1 | grep -v "no test files" | grep -v "^ok"; npm test 2>&1 | tail -5`
Expected: Go tarafında FAIL yok (golden dosyası değiştiyse `UPDATE_GOLDEN=1` ile yenile), npm tarafında hepsi PASS.

- [ ] **Step 9: Tarayıcıda uçtan uca doğrula**

```bash
make build && ./pixabros &
sleep 3
```

Chrome'da `http://localhost:8080/games/tetrabros`:
1. "Make playable offline — 44 MB" butonu görünmeli.
2. Bas, yüzde ilerlemeli, "Playable offline" ile bitmeli.
3. Application → Cache Storage'da `game-tetrabros-<version>` olmalı ve içinde `__offline-complete` bulunmalı.
4. Network'ü Offline yap, sayfayı yenile, oyunu başlat → hem sayfa hem oyun çalışmalı (indirme sayfayı ve `/media/` görsellerini de cache'lemiş olmalı).
5. İndirme sırasında sekmeyi kapatıp yeniden aç → kontrol yine "Make playable offline" demeli (yarım cache silinmiş olmalı), "Playable offline" **dememeli**.
6. "Remove" bas → cache kaybolmalı, buton geri gelmeli.

Bittiğinde `kill %1`.

- [ ] **Step 10: Commit**

```bash
git add internal/site/assets/ internal/site/templates/game.html
git commit -m "feat: let a visitor download a game to play it offline"
```

---

## Notlar

- **Task 4 Step 10** ve **Task 7 Step 8**: `internal/site/testdata/awards.golden.html` bu plan boyunca iki kez değişecek (önce `offline.js` script etiketi, sonra CSS hash'i). `UPDATE_GOLDEN=1 go test ./internal/site/ -run TestRenderAwards_MatchesGoldenFile` ile yenile ve farkın yalnızca beklenen satırlar olduğunu gözle doğrula.
- **Mevcut build'ler manifestsiz.** Bu plan yalnızca yeni yüklemeleri kaydeder; diskte duran sekiz oyunun manifesti boş kalır ve indirme butonu onlarda görünmez. Panelden bir kez yeniden yüklemek yeterli. Bunun yerine açılışta eksik manifestleri diskten üreten bir geri doldurma isteniyorsa ayrı bir görev olarak konuşulmalı — bu plana dâhil değil.
- **iOS'ta `persist()` verilmez** ve tahliye agresiftir; kontrol bu yüzden "Playable offline" diyor, "saklandı" demiyor.
