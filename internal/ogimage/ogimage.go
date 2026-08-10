package ogimage

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"pixabros/internal/imaging"
)

const (
	Width  = 1200
	Height = 630
	margin = 60
)

func Generate(title string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 20, G: 20, B: 30, A: 255}), image.Point{}, draw.Src)

	face := basicfont.Face7x13
	lines := wrapText(title, face, Width-2*margin)
	lineHeight := face.Metrics().Height.Ceil()
	startY := Height/2 - (len(lines)*lineHeight)/2 + face.Metrics().Ascent.Ceil()

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 240, G: 240, B: 245, A: 255}),
		Face: face,
	}
	for i, line := range lines {
		textWidth := font.MeasureString(face, line).Ceil()
		x := (Width - textWidth) / 2
		y := startY + i*lineHeight
		drawer.Dot = fixed.P(x, y)
		drawer.DrawString(line)
	}
	return img
}

func GenerateWebP(title string) ([]byte, error) {
	return imaging.EncodeWebP(Generate(title))
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
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
	lines = append(lines, current)
	return lines
}
