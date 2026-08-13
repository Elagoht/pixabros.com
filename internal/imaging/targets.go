package imaging

// Target is a size an upload is processed to.
//
// Most are a fixed shape and are cropped to fill it: a cartridge label has to
// be a cartridge label whatever was uploaded. An award's picture is different
// -- a trophy, a certificate, a rosette all have their own shape, and cropping
// them to a square cuts the award in half. Those are fitted instead: the
// source's aspect ratio is kept and only its size is capped.
type Target struct {
	Name   string
	Width  int
	Height int
	// Fit keeps the source's aspect ratio, treating Width and Height as
	// maximums rather than as the output size.
	Fit bool
}

var Targets = map[string]Target{
	"avatar":        {Name: "avatar", Width: 400, Height: 400},
	"cd_cover_art":  {Name: "cd_cover_art", Width: 600, Height: 600},
	"cartridge_art": {Name: "cartridge_art", Width: 400, Height: 560},
	"og_image":      {Name: "og_image", Width: 1200, Height: 630},
	"screenshot":    {Name: "screenshot", Width: 1280, Height: 720},
	"award_picture": {Name: "award_picture", Width: 1280, Height: 1280, Fit: true},
	"org_logo":      {Name: "org_logo", Width: 512, Height: 512},
}

func LookupTarget(name string) (Target, bool) {
	t, ok := Targets[name]
	return t, ok
}
