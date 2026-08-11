package games

import (
	"strings"
	"testing"
	"unicode"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Pixel Quest", "pixel-quest"},
		{"  Leading and Trailing  ", "leading-and-trailing"},
		{"Weird!!! Chars??? 123", "weird-chars-123"},
		{"", "game"},
		// Turkish titles are transliterated rather than having their
		// non-ASCII letters silently discarded. Slugs are immutable once
		// assigned, so a mangled slug would be permanent.
		{"Şeker Krallığı", "seker-kralligi"},
		{"Çığır", "cigir"},
		{"ÇIĞIR", "cigir"},
		{"Gölge Şövalyesi", "golge-sovalyesi"},
		{"İstanbul Düşü", "istanbul-dusu"},
		{"Uzay Gemisi Ömer", "uzay-gemisi-omer"},
	}
	for _, c := range cases {
		if got := Slugify(c.title); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

// TestSlugify_TurkishTitlesStayDistinctAndASCII asserts the properties that
// matter beyond the exact strings: nothing non-ASCII survives, the fallback
// is not reached for a Turkish title, and two meaningfully different Turkish
// titles do not collapse onto the same slug (before transliteration
// "Çığır" and "Şeker Krallığı" both lost most of their letters).
func TestSlugify_TurkishTitlesStayDistinctAndASCII(t *testing.T) {
	titles := []string{"Şeker Krallığı", "Çığır", "Gölge Şövalyesi", "İstanbul Düşü"}

	seen := make(map[string]string, len(titles))
	for _, title := range titles {
		slug := Slugify(title)

		for _, r := range slug {
			if r > unicode.MaxASCII {
				t.Errorf("Slugify(%q) = %q, contains non-ASCII rune %q", title, slug, r)
			}
		}
		if slug == "game" {
			t.Errorf("Slugify(%q) = %q, want a real slug rather than the bare fallback", title, slug)
		}
		if strings.Trim(slug, "-") != slug || strings.Contains(slug, "--") {
			t.Errorf("Slugify(%q) = %q, want no leading/trailing/doubled hyphens", title, slug)
		}
		if other, ok := seen[slug]; ok {
			t.Errorf("Slugify(%q) and Slugify(%q) both = %q, want distinct slugs", other, title, slug)
		}
		seen[slug] = title
	}
}
