package ogimage

import (
	"strings"
	"testing"

	"golang.org/x/image/font/basicfont"
)

func TestGenerate_ProducesCorrectDimensions(t *testing.T) {
	img := Generate("Pixel Quest devlog #3")
	if img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630 {
		t.Errorf("Generate() bounds = %v, want 1200x630", img.Bounds())
	}
}

func TestGenerate_HandlesLongTitles(t *testing.T) {
	img := Generate("This is a deliberately very long devlog title meant to wrap across multiple lines of the template")
	if img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630 {
		t.Errorf("Generate() bounds = %v, want 1200x630", img.Bounds())
	}
}

func TestGenerateWebP_ProducesBytes(t *testing.T) {
	data, err := GenerateWebP("Short title")
	if err != nil {
		t.Fatalf("GenerateWebP() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("GenerateWebP() returned no bytes")
	}
}

func TestWrapText_WrapsLongTitleIntoMultipleLines(t *testing.T) {
	longTitle := "This is a deliberately very long devlog title that repeats several real words over and over so that it definitely exceeds the eleven hundred and eighty pixel width limit for a single line at seven pixels per monospace character and must wrap onto multiple lines"
	lines := wrapText(longTitle, basicfont.Face7x13, Width-2*margin)
	if len(lines) < 2 {
		t.Fatalf("wrapText() returned %d line(s), want at least 2 for a title this long (lines: %v)", len(lines), lines)
	}
}

func TestWrapText_SingleWordLongerThanMaxWidthDoesNotCrashOrSplit(t *testing.T) {
	oneGiantWord := strings.Repeat("x", 300) // no spaces; ~2100px at 7px/char, well over the 1080px limit
	lines := wrapText(oneGiantWord, basicfont.Face7x13, Width-2*margin)
	if len(lines) != 1 {
		t.Errorf("wrapText() returned %d lines for a single unbreakable word, want exactly 1", len(lines))
	}
	if lines[0] != oneGiantWord {
		t.Errorf("wrapText() = %q, want the word unchanged (not split)", lines[0])
	}
	// Also confirm Generate() doesn't panic on this input.
	img := Generate(oneGiantWord)
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Errorf("Generate() bounds = %v, want %dx%d", img.Bounds(), Width, Height)
	}
}
