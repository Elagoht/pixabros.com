package gamearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
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

func byteReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

func byteReaderAt(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
