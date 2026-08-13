// Package youtube reads a video id out of a YouTube link.
//
// It exists so the admin API and the public site agree on what a YouTube URL
// is. The site frames only the id, never the URL, so this is the boundary that
// keeps a stored link from choosing what gets embedded: everything about the
// URL except the id is read and thrown away.
package youtube

import (
	"net/url"
	"regexp"
	"strings"
)

// EmbedHost is where a video is framed from. nocookie is the
// privacy-preserving host, and it is a constant so no caller can vary it.
const EmbedHost = "https://www.youtube-nocookie.com"

// idPattern is the shape of a video id, and the reason no other part of a link
// needs sanitising.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// IsID reports whether s is shaped like a video id.
func IsID(s string) bool {
	return idPattern.MatchString(s)
}

// EmbedURL is where the player for a video id lives.
func EmbedURL(videoID string) string {
	return EmbedHost + "/embed/" + videoID
}

// ID pulls the video id out of any of the forms YouTube hands out when you
// press Share: a youtu.be link, a watch URL, an embed URL, a short.
//
// Anything else, including a link to another site, is not a video.
func ID(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\n\r") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}

	var candidate string
	switch strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.") {
	case "youtu.be":
		candidate = parsed.Path
	case "youtube.com", "m.youtube.com", "youtube-nocookie.com":
		switch {
		case parsed.Path == "/watch":
			candidate = parsed.Query().Get("v")
		case strings.HasPrefix(parsed.Path, "/embed/"):
			candidate = strings.TrimPrefix(parsed.Path, "/embed/")
		case strings.HasPrefix(parsed.Path, "/shorts/"):
			candidate = strings.TrimPrefix(parsed.Path, "/shorts/")
		}
	default:
		return "", false
	}

	candidate = strings.Trim(candidate, "/")
	if !IsID(candidate) {
		return "", false
	}
	return candidate, true
}
