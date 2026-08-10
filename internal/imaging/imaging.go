package imaging

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"golang.org/x/image/draw"
	// Imported for its side effect of registering the WebP decoder with the
	// standard image package.
	_ "golang.org/x/image/webp"

	webpenc "github.com/mayahiro/go-webp"
)

const (
	// maxUploadBytes bounds how much of an uploaded image Decode will read
	// into memory before giving up.
	maxUploadBytes = 20 << 20 // 20 MiB
	// maxImageDimension bounds decoded width/height, checked via
	// image.DecodeConfig before the full pixel decode allocates anything.
	maxImageDimension = 8192
)

// Decode reads a JPEG, PNG, GIF, or WebP image from r.
func Decode(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) > maxUploadBytes {
		return nil, fmt.Errorf("image exceeds the %d byte upload limit", maxUploadBytes)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	if cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
		return nil, fmt.Errorf("image dimensions %dx%d exceed the maximum of %dx%d", cfg.Width, cfg.Height, maxImageDimension, maxImageDimension)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// ResizeCropFill center-crops src to the target aspect ratio, then scales it
// to exactly targetW x targetH.
func ResizeCropFill(src image.Image, targetW, targetH int) image.Image {
	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	srcAspect := float64(srcW) / float64(srcH)
	targetAspect := float64(targetW) / float64(targetH)

	cropRect := srcBounds
	switch {
	case srcAspect > targetAspect:
		newW := int(float64(srcH) * targetAspect)
		offset := (srcW - newW) / 2
		cropRect = image.Rect(srcBounds.Min.X+offset, srcBounds.Min.Y, srcBounds.Min.X+offset+newW, srcBounds.Max.Y)
	case srcAspect < targetAspect:
		newH := int(float64(srcW) / targetAspect)
		offset := (srcH - newH) / 2
		cropRect = image.Rect(srcBounds.Min.X, srcBounds.Min.Y+offset, srcBounds.Max.X, srcBounds.Min.Y+offset+newH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Over, nil)
	return dst
}

// EncodeWebP encodes img as a lossy WebP image.
func EncodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := webpenc.Encode(&buf, img, &webpenc.Options{
		Compression: webpenc.CompressionLossy,
		Quality:     85,
	})
	if err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}

// ProcessUpload decodes r, resizes it to exactly targetW x targetH, and
// returns the resulting WebP bytes. This is the single entry point the
// upload handler calls.
func ProcessUpload(r io.Reader, targetW, targetH int) ([]byte, error) {
	src, err := Decode(r)
	if err != nil {
		return nil, err
	}
	resized := ResizeCropFill(src, targetW, targetH)
	return EncodeWebP(resized)
}
