// Package ogimage draws the picture a devlog post shares as.
//
// The first version was white text on black in a 7x13 bitmap font, which read
// as a terminal error rather than a studio's post. This one is the studio's
// own card: the logo on a dark band above, the post's title large on a light
// band below.
package ogimage

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"pixabros/internal/imaging"
)

const (
	Width  = 1200
	Height = 630

	// The dark band carries the logo, the light one the title. Splitting them
	// is what makes the card read as a card rather than a caption.
	bandSplit = 300

	margin = 72
	// Title size, and how far it may shrink before wrapping to more lines. A
	// long title should still be readable across a timeline thumbnail.
	titleSizeMax = 84
	titleSizeMin = 46
	maxLines     = 3
)

var (
	inkDark  = color.RGBA{R: 0x0f, G: 0x11, B: 0x15, A: 0xff}
	inkLight = color.RGBA{R: 0xf6, G: 0xf6, B: 0xf8, A: 0xff}
	accent   = color.RGBA{R: 0xe8, G: 0x79, B: 0xf9, A: 0xff}
)

// Generate draws the card. logo may be nil, in which case the top band carries
// the studio's accent rule alone.
func Generate(title string, logo image.Image) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))

	// Two bands, split horizontally.
	draw.Draw(img, image.Rect(0, 0, Width, bandSplit), image.NewUniform(inkDark), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, bandSplit, Width, Height), image.NewUniform(inkLight), image.Point{}, draw.Src)
	// A rule on the seam, in the site's accent.
	draw.Draw(img, image.Rect(0, bandSplit-6, Width, bandSplit), image.NewUniform(accent), image.Point{}, draw.Src)

	if logo != nil {
		drawLogo(img, logo)
	}

	drawTitle(img, title)
	return img
}

// drawLogo centres the studio's mark in the dark band, scaled to fit it.
func drawLogo(dst *image.RGBA, logo image.Image) {
	const maxLogo = 180

	bounds := logo.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return
	}

	scale := float64(maxLogo) / float64(bounds.Dy())
	if w := float64(bounds.Dx()) * scale; w > float64(Width-2*margin) {
		scale = float64(Width-2*margin) / float64(bounds.Dx())
	}
	w := int(float64(bounds.Dx()) * scale)
	h := int(float64(bounds.Dy()) * scale)

	x := (Width - w) / 2
	y := (bandSplit - h) / 2
	xdraw.CatmullRom.Scale(dst, image.Rect(x, y, x+w, y+h), logo, bounds, draw.Over, nil)
}

// drawTitle lays the post's title across the light band, shrinking the type
// until it fits in at most maxLines.
func drawTitle(dst *image.RGBA, title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}

	maxWidth := Width - 2*margin

	for size := titleSizeMax; size >= titleSizeMin; size -= 4 {
		face, err := newFace(float64(size))
		if err != nil {
			return
		}
		lines := wrapText(title, face, maxWidth)
		if len(lines) > maxLines && size > titleSizeMin {
			face.Close()
			continue
		}

		lineHeight := int(float64(size) * 1.18)
		blockHeight := lineHeight * len(lines)
		bandHeight := Height - bandSplit
		startY := bandSplit + (bandHeight-blockHeight)/2 + face.Metrics().Ascent.Ceil()

		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(inkDark),
			Face: face,
		}
		for i, line := range lines {
			drawer.Dot = fixed.P(margin, startY+i*lineHeight)
			drawer.DrawString(line)
		}
		face.Close()
		return
	}
}

func newFace(size float64) (font.Face, error) {
	parsed, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// GenerateWebP draws the card and encodes it.
func GenerateWebP(title string, logo image.Image) ([]byte, error) {
	return imaging.EncodeWebP(Generate(title, logo))
}

// wrapText breaks text into lines that fit maxWidth. A single word longer than
// the line is left to overflow rather than being cut in half.
func wrapText(text string, face font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if font.MeasureString(face, candidate).Ceil() > maxWidth {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	return append(lines, current)
}
