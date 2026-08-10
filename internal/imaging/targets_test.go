package imaging

import "testing"

func TestLookupTarget_KnownAndUnknown(t *testing.T) {
	cases := []struct {
		name          string
		wantW, wantH  int
	}{
		{"avatar", 400, 400},
		{"cd_cover_art", 600, 600},
		{"cartridge_art", 400, 560},
		{"og_image", 1200, 630},
		{"screenshot", 1280, 720},
		{"award_picture", 320, 320},
		{"org_logo", 512, 512},
	}
	for _, c := range cases {
		target, ok := LookupTarget(c.name)
		if !ok {
			t.Errorf("LookupTarget(%q) not found", c.name)
			continue
		}
		if target.Width != c.wantW || target.Height != c.wantH {
			t.Errorf("LookupTarget(%q) = %dx%d, want %dx%d", c.name, target.Width, target.Height, c.wantW, c.wantH)
		}
	}

	if _, ok := LookupTarget("does_not_exist"); ok {
		t.Error("LookupTarget() should return ok=false for an unknown name")
	}
}
