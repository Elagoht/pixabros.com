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

var _ Storage = (*LocalDisk)(nil)

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
