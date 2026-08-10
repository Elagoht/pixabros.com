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
