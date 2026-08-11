package games

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Pixel Quest", "pixel-quest"},
		{"  Leading and Trailing  ", "leading-and-trailing"},
		{"Weird!!! Chars??? 123", "weird-chars-123"},
		{"", "game"},
	}
	for _, c := range cases {
		if got := Slugify(c.title); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}
