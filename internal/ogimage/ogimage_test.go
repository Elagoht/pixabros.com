package ogimage

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// solidLogo stands in for the studio's mark.
func solidLogo(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
		}
	}
	return img
}

func TestGenerate_ProducesCorrectDimensions(t *testing.T) {
	img := Generate("Pixel Quest devlog #3", nil)
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Errorf("Generate() bounds = %v, want %dx%d", img.Bounds(), Width, Height)
	}
}

// The card is two bands split horizontally, which is what stops it reading as
// a caption on a black rectangle.
func TestGenerate_DrawsTwoBands(t *testing.T) {
	img := Generate("A title", nil)

	top := color.RGBAModel.Convert(img.At(20, 40)).(color.RGBA)
	bottom := color.RGBAModel.Convert(img.At(20, Height-40)).(color.RGBA)

	if top == bottom {
		t.Fatalf("both bands are %v, want a dark one above a light one", top)
	}
	if !(top.R < 0x40 && top.G < 0x40 && top.B < 0x40) {
		t.Errorf("top band = %v, want it dark", top)
	}
	if !(bottom.R > 0xc0 && bottom.G > 0xc0 && bottom.B > 0xc0) {
		t.Errorf("bottom band = %v, want it light", bottom)
	}
}

func TestGenerate_PlacesTheLogoInTheDarkBand(t *testing.T) {
	withLogo := Generate("A title", solidLogo(200, 200))
	without := Generate("A title", nil)

	centre := image.Pt(Width/2, bandSplit/2)
	if withLogo.At(centre.X, centre.Y) == without.At(centre.X, centre.Y) {
		t.Error("the logo did not reach the middle of the dark band")
	}
}

// A missing logo is normal, not an error: the card still has to come out.
func TestGenerate_SurvivesWithoutALogo(t *testing.T) {
	if img := Generate("A title", nil); img == nil {
		t.Fatal("Generate() returned nothing without a logo")
	}
}

func TestGenerate_HandlesLongTitles(t *testing.T) {
	img := Generate(
		"This is a deliberately very long devlog title meant to wrap across several lines of the card",
		solidLogo(256, 256),
	)
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Errorf("Generate() bounds = %v, want %dx%d", img.Bounds(), Width, Height)
	}
}

func TestGenerateWebP_ProducesBytes(t *testing.T) {
	data, err := GenerateWebP("Short title", solidLogo(128, 128))
	if err != nil {
		t.Fatalf("GenerateWebP() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("GenerateWebP() returned no bytes")
	}
}

func TestWrapText_WrapsLongTitleIntoMultipleLines(t *testing.T) {
	face, err := newFace(titleSizeMax)
	if err != nil {
		t.Fatalf("newFace() error = %v", err)
	}
	defer face.Close()

	long := "This is a deliberately very long devlog title that goes on well past " +
		"the width of the card and must therefore wrap"
	lines := wrapText(long, face, Width-2*margin)
	if len(lines) < 2 {
		t.Fatalf("wrapText() returned %d line(s), want at least 2 (lines: %v)", len(lines), lines)
	}
}

// A word longer than the line is left whole: breaking it mid-word would read
// worse than letting it run on.
func TestWrapText_DoesNotSplitASingleLongWord(t *testing.T) {
	face, err := newFace(titleSizeMax)
	if err != nil {
		t.Fatalf("newFace() error = %v", err)
	}
	defer face.Close()

	giant := strings.Repeat("x", 300)
	lines := wrapText(giant, face, Width-2*margin)
	if len(lines) != 1 || lines[0] != giant {
		t.Errorf("wrapText() = %v, want the word unchanged on one line", lines)
	}

	if img := Generate(giant, nil); img.Bounds().Dx() != Width {
		t.Errorf("Generate() bounds = %v, want %dx%d", img.Bounds(), Width, Height)
	}
}

// The type shrinks rather than spilling out of the band.
func TestDrawTitle_ShrinksLongTitlesToFit(t *testing.T) {
	short, err := newFace(titleSizeMax)
	if err != nil {
		t.Fatalf("newFace() error = %v", err)
	}
	defer short.Close()

	long := strings.Repeat("A long devlog title ", 8)
	lines := wrapText(long, short, Width-2*margin)
	if len(lines) <= maxLines {
		t.Skip("the sample is not long enough to force a shrink")
	}

	smaller, err := newFace(titleSizeMin)
	if err != nil {
		t.Fatalf("newFace() error = %v", err)
	}
	defer smaller.Close()

	if len(wrapText(long, smaller, Width-2*margin)) >= len(lines) {
		t.Error("the smaller face did not fit more words per line")
	}
}
