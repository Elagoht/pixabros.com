// Package ogimage draws the picture a devlog post shares as.
//
// The card is the studio's own: the mark on a dark band above, with the game's
// name beside it when the post is about one, and the post's title on a light
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

	// The dark band carries the mark, the light one the title.
	bandSplit = 300

	margin = 72

	// Title size, and how far it may shrink before wrapping to more lines.
	titleSizeMax = 54
	titleSizeMin = 32
	maxLines     = 3

	// The game's name, set beside the mark.
	gameSize = 40
	logoSize = 168
	// Space between the mark and the game's name.
	logoGap = 48
)

var (
	inkDark  = color.RGBA{R: 0x0f, G: 0x11, B: 0x15, A: 0xff}
	inkLight = color.RGBA{R: 0xf6, G: 0xf6, B: 0xf8, A: 0xff}
	onDark   = color.RGBA{R: 0xf1, G: 0xf1, B: 0xf3, A: 0xff}
)

// Generate draws the card. logo may be nil and game may be empty; the band
// above simply carries whatever it has, centred.
func Generate(title string, logo image.Image, game string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))

	draw.Draw(img, image.Rect(0, 0, Width, bandSplit), image.NewUniform(inkDark), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, bandSplit, Width, Height), image.NewUniform(inkLight), image.Point{}, draw.Src)

	drawMark(img, logo, strings.TrimSpace(game))
	drawTitle(img, title)
	return img
}

// drawMark centres the studio's logo in the dark band, with the game's name to
// its right when there is one. The pair is centred as a unit rather than the
// logo alone, so the band stays balanced either way.
func drawMark(dst *image.RGBA, logo image.Image, game string) {
	logoW, logoH := 0, 0
	if logo != nil {
		bounds := logo.Bounds()
		if bounds.Dx() > 0 && bounds.Dy() > 0 {
			scale := float64(logoSize) / float64(bounds.Dy())
			logoW = int(float64(bounds.Dx()) * scale)
			logoH = int(float64(bounds.Dy()) * scale)
		}
	}

	var face font.Face
	gameW := 0
	if game != "" {
		var err error
		face, err = newFace(gameSize)
		if err != nil {
			return
		}
		defer face.Close()
		gameW = font.MeasureString(face, game).Ceil()
	}

	total := logoW + gameW
	if gameW > 0 && logoW > 0 {
		total += logoGap
	}
	if total == 0 {
		return
	}
	// Nothing may run off the edge, so an over-wide pair is nudged in.
	x := (Width - total) / 2
	if x < margin/2 {
		x = margin / 2
	}

	if logoW > 0 {
		y := (bandSplit - logoH) / 2
		xdraw.CatmullRom.Scale(
			dst, image.Rect(x, y, x+logoW, y+logoH), logo, logo.Bounds(), draw.Over, nil,
		)
		x += logoW + logoGap
	}

	if gameW > 0 {
		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(onDark),
			Face: face,
			Dot:  fixed.P(x, bandSplit/2+face.Metrics().Ascent.Ceil()/2-face.Metrics().Descent.Ceil()/2),
		}
		drawer.DrawString(game)
	}
}

// drawTitle lays the post's title across the light band, centred, shrinking the
// type until it fits in at most maxLines.
func drawTitle(dst *image.RGBA, title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}

	maxWidth := Width - 2*margin

	for size := titleSizeMax; size >= titleSizeMin; size -= 3 {
		face, err := newFace(float64(size))
		if err != nil {
			return
		}
		lines := wrapText(title, face, maxWidth)
		if len(lines) > maxLines && size > titleSizeMin {
			face.Close()
			continue
		}

		lineHeight := int(float64(size) * 1.25)
		blockHeight := lineHeight * len(lines)
		bandHeight := Height - bandSplit
		startY := bandSplit + (bandHeight-blockHeight)/2 + face.Metrics().Ascent.Ceil()

		drawer := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(inkDark),
			Face: face,
		}
		for i, line := range lines {
			lineWidth := font.MeasureString(face, line).Ceil()
			drawer.Dot = fixed.P((Width-lineWidth)/2, startY+i*lineHeight)
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
func GenerateWebP(title string, logo image.Image, game string) ([]byte, error) {
	return imaging.EncodeWebP(Generate(title, logo, game))
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
