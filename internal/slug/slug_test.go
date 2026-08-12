package slug

import (
	"strings"
	"testing"
	"unicode"
)

func TestMake(t *testing.T) {
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
		if got := Make(c.title, "game"); got != c.want {
			t.Errorf("Make(%q, fallback) = %q, want %q", c.title, got, c.want)
		}
	}
}

// TestMake_TurkishTitlesStayDistinctAndASCII asserts the properties that
// matter beyond the exact strings: nothing non-ASCII survives, the fallback
// is not reached for a Turkish title, and two meaningfully different Turkish
// titles do not collapse onto the same slug (before transliteration
// "Çığır" and "Şeker Krallığı" both lost most of their letters).
func TestMake_TurkishTitlesStayDistinctAndASCII(t *testing.T) {
	titles := []string{"Şeker Krallığı", "Çığır", "Gölge Şövalyesi", "İstanbul Düşü"}

	seen := make(map[string]string, len(titles))
	for _, title := range titles {
		got := Make(title, "game")

		for _, r := range got {
			if r > unicode.MaxASCII {
				t.Errorf("Make(%q, fallback) = %q, contains non-ASCII rune %q", title, got, r)
			}
		}
		if got == "game" {
			t.Errorf("Make(%q, fallback) = %q, want a real slug rather than the bare fallback", title, got)
		}
		if strings.Trim(got, "-") != got || strings.Contains(got, "--") {
			t.Errorf("Make(%q, fallback) = %q, want no leading/trailing/doubled hyphens", title, got)
		}
		if other, ok := seen[got]; ok {
			t.Errorf("Make(%q, fallback) and Make(%q, fallback) both = %q, want distinct slugs", other, title, got)
		}
		seen[got] = title
	}
}

func TestMake_FallsBackWhenNothingSurvives(t *testing.T) {
	// A title made entirely of punctuation or non-Latin script leaves nothing
	// slug-able, and a slug is a URL segment, so it can never be empty.
	for _, title := range []string{"", "   ", "!!!", "---", "。。。"} {
		if got := Make(title, "post"); got != "post" {
			t.Errorf("Make(%q, %q) = %q, want the fallback", title, "post", got)
		}
	}
}

func TestUnique(t *testing.T) {
	conn := setupSlugTestDB(t)

	first, err := Unique(conn, "devlog_posts", "hello-world", "")
	if err != nil {
		t.Fatalf("Unique() error = %v", err)
	}
	if first != "hello-world" {
		t.Errorf("Unique() = %q, want %q for an unused slug", first, "hello-world")
	}

	if _, err := conn.Exec(
		`INSERT INTO devlog_posts (id, slug, title) VALUES ('a', 'hello-world', 'Hello World');`,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	second, err := Unique(conn, "devlog_posts", "hello-world", "")
	if err != nil {
		t.Fatalf("Unique() error = %v", err)
	}
	if second != "hello-world-2" {
		t.Errorf("Unique() = %q, want %q once the base is taken", second, "hello-world-2")
	}

	// Excluding the row that already owns the slug is what lets an update
	// keep an unchanged title without the slug growing a suffix each save.
	own, err := Unique(conn, "devlog_posts", "hello-world", "a")
	if err != nil {
		t.Fatalf("Unique() error = %v", err)
	}
	if own != "hello-world" {
		t.Errorf("Unique() excluding the owning row = %q, want %q", own, "hello-world")
	}
}

func TestUnique_KeepsCountingPastTheSecond(t *testing.T) {
	conn := setupSlugTestDB(t)

	for i, s := range []string{"news", "news-2", "news-3"} {
		if _, err := conn.Exec(
			`INSERT INTO devlog_posts (id, slug, title) VALUES (?, ?, 'n');`,
			string(rune('a'+i)), s,
		); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}

	got, err := Unique(conn, "devlog_posts", "news", "")
	if err != nil {
		t.Fatalf("Unique() error = %v", err)
	}
	if got != "news-4" {
		t.Errorf("Unique() = %q, want %q", got, "news-4")
	}
}
