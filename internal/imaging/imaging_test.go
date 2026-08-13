package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"golang.org/x/image/webp"
)

func solidPNG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func TestDecode_PNG(t *testing.T) {
	data := solidPNG(t, 10, 10, color.RGBA{R: 255, A: 255})
	img, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
		t.Errorf("Decode() bounds = %v, want 10x10", img.Bounds())
	}
}

func TestDecode_RejectsGarbage(t *testing.T) {
	if _, err := Decode(bytes.NewReader([]byte("not an image"))); err == nil {
		t.Error("Decode() with garbage input should return an error")
	}
}

func TestResizeCropFill_WideSourceToSquareTarget(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	out := ResizeCropFill(src, 400, 400)
	if out.Bounds().Dx() != 400 || out.Bounds().Dy() != 400 {
		t.Errorf("ResizeCropFill() bounds = %v, want 400x400", out.Bounds())
	}
}

func TestResizeCropFill_TallSourceToWideTarget(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 800))
	out := ResizeCropFill(src, 1200, 630)
	if out.Bounds().Dx() != 1200 || out.Bounds().Dy() != 630 {
		t.Errorf("ResizeCropFill() bounds = %v, want 1200x630", out.Bounds())
	}
}

func TestResizeCropFill_PreservesSolidColor(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fillColor := color.RGBA{R: 10, G: 200, B: 30, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, fillColor)
		}
	}
	out := ResizeCropFill(src, 50, 50)
	r, g, b, a := out.At(25, 25).RGBA()
	if uint8(r>>8) != fillColor.R || uint8(g>>8) != fillColor.G || uint8(b>>8) != fillColor.B || uint8(a>>8) != fillColor.A {
		t.Errorf("center pixel = (%d,%d,%d,%d), want (%d,%d,%d,%d)", r>>8, g>>8, b>>8, a>>8, fillColor.R, fillColor.G, fillColor.B, fillColor.A)
	}
}

func TestResizeCropFill_CropsCorrectRegion(t *testing.T) {
	// 800x400 source, left half red, right half blue. Center-cropping to a
	// 400x400 (square) target should crop to x∈[200,600), i.e. keep the
	// right portion of the red half and the left portion of the blue half —
	// so the output's left edge should sample red-ish and right edge blue-ish.
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			if x < 400 {
				src.Set(x, y, red)
			} else {
				src.Set(x, y, blue)
			}
		}
	}

	out := ResizeCropFill(src, 400, 400)

	leftR, _, leftB, _ := out.At(10, 200).RGBA()
	rightR, _, rightB, _ := out.At(390, 200).RGBA()

	if leftR == 0 || leftB != 0 {
		t.Errorf("left edge of cropped output should sample red (from the retained middle-left portion), got r=%d b=%d", leftR, leftB)
	}
	if rightB == 0 || rightR != 0 {
		t.Errorf("right edge of cropped output should sample blue (from the retained middle-right portion), got r=%d b=%d", rightR, rightB)
	}
}

func TestDecode_RejectsOversizedUpload(t *testing.T) {
	oversized := make([]byte, maxUploadBytes+1)
	_, err := Decode(bytes.NewReader(oversized))
	if err == nil {
		t.Fatal("Decode() with an oversized input should return an error")
	}
	// The size cap must trip before any decode attempt, so the error has to
	// name the limit rather than being a generic "corrupt image".
	if !strings.Contains(err.Error(), "upload limit") {
		t.Errorf("Decode() error = %v, want it to report the upload size limit", err)
	}
}

func TestDecode_RejectsOversizedDimensions(t *testing.T) {
	data := solidPNG(t, maxImageDimension+1, 10, color.RGBA{R: 255, A: 255})
	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Error("Decode() with oversized dimensions should return an error")
	}
}

func TestEncodeWebP_ProducesDecodableOutput(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	data, err := EncodeWebP(src)
	if err != nil {
		t.Fatalf("EncodeWebP() error = %v", err)
	}
	decoded, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("webp.Decode() on EncodeWebP's output error = %v", err)
	}
	if decoded.Bounds().Dx() != 20 || decoded.Bounds().Dy() != 20 {
		t.Errorf("decoded bounds = %v, want 20x20", decoded.Bounds())
	}
}

func TestProcessUpload_EndToEnd(t *testing.T) {
	data := solidPNG(t, 800, 400, color.RGBA{G: 255, A: 255})
	webpBytes, err := ProcessUpload(bytes.NewReader(data), 400, 400)
	if err != nil {
		t.Fatalf("ProcessUpload() error = %v", err)
	}
	if len(webpBytes) == 0 {
		t.Error("ProcessUpload() returned no bytes")
	}
}

// An award can be a trophy, a certificate or a rosette. Cropping those to a
// square cuts the award in half, so the picture keeps its own shape and is
// only capped in size.
func TestResizeFit_KeepsTheSourceAspectRatio(t *testing.T) {
	wide := image.NewRGBA(image.Rect(0, 0, 3000, 1000))
	got := ResizeFit(wide, 1280, 1280)

	if got.Bounds().Dx() != 1280 {
		t.Errorf("width = %d, want it capped at 1280", got.Bounds().Dx())
	}
	if h := got.Bounds().Dy(); h < 420 || h > 430 {
		t.Errorf("height = %d, want about 427 so the 3:1 shape survives", h)
	}
}

func TestResizeFit_CapsTheTallerSideToo(t *testing.T) {
	tall := image.NewRGBA(image.Rect(0, 0, 1000, 4000))
	got := ResizeFit(tall, 1280, 1280)

	if got.Bounds().Dy() != 1280 {
		t.Errorf("height = %d, want it capped at 1280", got.Bounds().Dy())
	}
	if w := got.Bounds().Dx(); w < 315 || w > 325 {
		t.Errorf("width = %d, want about 320", w)
	}
}

// Scaling a small picture up would only invent detail.
func TestResizeFit_LeavesSmallImagesAlone(t *testing.T) {
	small := image.NewRGBA(image.Rect(0, 0, 300, 200))
	got := ResizeFit(small, 1280, 1280)

	if got.Bounds().Dx() != 300 || got.Bounds().Dy() != 200 {
		t.Errorf("size = %dx%d, want it untouched", got.Bounds().Dx(), got.Bounds().Dy())
	}
}
