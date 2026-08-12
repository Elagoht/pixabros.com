// Package adminui embeds the built admin panel into the server binary.
//
// The panel used to be served from a directory on disk, which meant a deploy
// was two artefacts that could drift: a binary and a folder someone had to
// remember to copy. Embedding makes the binary the whole deployment.
//
// The build output is not committed: `make build` copies admin-ui/dist in here
// before compiling. Building with plain `go build` therefore embeds the
// placeholder below, which says so rather than serving a blank page.
package adminui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// FS returns the panel's files, rooted at what the browser sees as
// /I-am-a-pixabro/.
func FS() (fs.FS, error) {
	return fs.Sub(files, "dist")
}
