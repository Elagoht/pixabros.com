package ogimage

import "testing"

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
