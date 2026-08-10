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
