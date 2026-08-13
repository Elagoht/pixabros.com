package httpserver

import (
	"net/http"
	"strings"
)

// adminUIPrefix is where the panel is mounted. It is a constant because both
// the route and the policy that covers it have to agree on it.
const adminUIPrefix = "/I-am-a-pixabro/"

// The four areas of this origin need different policies, so one header for
// everything would have to be as loose as the loosest of them. Splitting them
// keeps the public site, which is the part strangers reach, strict.

// publicCSP covers the rendered site. It can afford to be strict because the
// pages carry no inline script or style at all: everything is a hashed file
// under /assets, and the only third party in the whole site is the video
// player a devlog post can embed.
//
// frame-src names youtube-nocookie for that player and 'self' for the game
// console, which frames a build from /play on the same origin.
const publicCSP = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"frame-src 'self' https://www.youtube-nocookie.com; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'"

// panelCSP covers the admin SPA, which needs two things the public site does
// not: Google's font service, and inline style attributes, because React
// components set them for measured widths and positions. 'unsafe-inline' for
// style is a real loosening; it is confined to a screen that already requires
// a session, and it does not extend to script.
const panelCSP = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"img-src 'self'; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"connect-src 'self'; " +
	"frame-src 'self'; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'"

// apiCSP covers JSON. Nothing here is ever a document, so the policy only has
// to say that.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// playCSP covers uploaded game builds, and deliberately restricts only who may
// frame them.
//
// A build is a third-party bundle: engine exports run inline script, compile
// WebAssembly, and start workers from blob URLs. A content policy written here
// would break games rather than protect anyone, and the code inside a build is
// already trusted at upload time. What is worth pinning is that only this site
// may put a game in an iframe.
const playCSP = "frame-ancestors 'self'"

// withSecurityHeaders states what each response is allowed to load.
//
// nosniff rides along because it answers the same question from the other
// side: the policy says where content may come from, and nosniff says the
// browser must believe the type we gave it rather than guessing from bytes.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policyFor(r.URL.Path))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func policyFor(path string) string {
	switch {
	case strings.HasPrefix(path, "/play/"):
		return playCSP
	case strings.HasPrefix(path, adminUIPrefix):
		return panelCSP
	case strings.HasPrefix(path, "/api/"):
		return apiCSP
	default:
		return publicCSP
	}
}
