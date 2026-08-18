package devlogapi

import (
	"bytes"
	"testing"
)

func TestSearchCache_GetReturnsWhatWasPut(t *testing.T) {
	cache := NewSearchCache(4)
	cache.Put("a", []byte("alpha"))
	body, ok := cache.Get("a")
	if !ok {
		t.Fatal("expected the entry to be present")
	}
	if !bytes.Equal(body, []byte("alpha")) {
		t.Fatalf("body = %q, want %q", body, "alpha")
	}
}

func TestSearchCache_GetMissesAnUnknownKey(t *testing.T) {
	cache := NewSearchCache(4)
	if _, ok := cache.Get("missing"); ok {
		t.Fatal("expected a miss for an unknown key")
	}
}

func TestSearchCache_EvictsTheLeastRecentlyUsed(t *testing.T) {
	cache := NewSearchCache(2)
	cache.Put("a", []byte("1"))
	cache.Put("b", []byte("2"))
	// Touch a, then insert c: b, the least recently used, is the eviction.
	cache.Get("a")
	cache.Put("c", []byte("3"))

	if _, ok := cache.Get("a"); !ok {
		t.Fatal("a was touched recently and should survive")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Fatal("c is the newest entry and should survive")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatal("b was least recently used and should have been evicted")
	}
}

func TestSearchCache_PutRefreshesAnExistingKey(t *testing.T) {
	cache := NewSearchCache(1)
	cache.Put("a", []byte("first"))
	cache.Put("a", []byte("second"))
	body, ok := cache.Get("a")
	if !ok {
		t.Fatal("expected the entry to be present")
	}
	if !bytes.Equal(body, []byte("second")) {
		t.Fatalf("body = %q, want %q", body, "second")
	}
}

func TestSearchCache_ClearDropsEverything(t *testing.T) {
	cache := NewSearchCache(4)
	cache.Put("a", []byte("1"))
	cache.Clear()
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected the cache to be empty after Clear")
	}
}
