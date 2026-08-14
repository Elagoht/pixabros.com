package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	maxArchiveSize   int64 = 500 << 20 // 500 MiB — raw uploaded archive size
	maxExtractedSize int64 = 2 << 30   // 2 GiB — total decompressed output across all entries
)

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

// Extract detects the archive format from filename's extension and extracts
// archive into destDir. It requires an index.html at the extracted root and
// rejects any entry whose path would escape destDir. On any failure, destDir
// is left empty.
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

func extractZip(data []byte, destDir string, budget *int64, into *collector) error {
	r, err := zip.NewReader(byteReaderAt(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	for _, f := range r.File {
		if skip(f.Name) {
			continue
		}
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
	}
	return nil
}

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

func extractTarReader(tr *tar.Reader, destDir string, budget *int64, into *collector) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if skip(hdr.Name) {
			continue
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
			n, hash, err := writeFile(target, tr, budget)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(destDir, target)
			if err != nil {
				return err
			}
			into.add(rel, n, hash)
		case tar.TypeSymlink, tar.TypeLink:
			// Links can point outside destDir or alias unexpected files, so
			// they are rejected outright rather than silently dropped.
			return fmt.Errorf("archive entry %q is a symlink or hard link, which is not supported", hdr.Name)
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

func clearDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}

func byteReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

func byteReaderAt(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
