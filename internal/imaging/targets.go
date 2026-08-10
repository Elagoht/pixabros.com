package imaging

type Target struct {
	Name   string
	Width  int
	Height int
}

var Targets = map[string]Target{
	"avatar":        {Name: "avatar", Width: 400, Height: 400},
	"cd_cover_art":  {Name: "cd_cover_art", Width: 600, Height: 600},
	"cartridge_art": {Name: "cartridge_art", Width: 400, Height: 560},
	"og_image":      {Name: "og_image", Width: 1200, Height: 630},
	"screenshot":    {Name: "screenshot", Width: 1280, Height: 720},
	"award_picture": {Name: "award_picture", Width: 320, Height: 320},
	"org_logo":      {Name: "org_logo", Width: 512, Height: 512},
}

func LookupTarget(name string) (Target, bool) {
	t, ok := Targets[name]
	return t, ok
}
