package games

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugTransliterations maps characters common in the languages this
// project's content is actually written in (starting with Turkish) to a
// plain-ASCII equivalent, applied before non-ASCII characters are simply
// discarded. The slug regenerates from the title on every update (see
// Update), so this only has to look right, not be permanent -- but it should
// still look right for a non-English title on the very first save.
var slugTransliterations = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i",
	"İ", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
)

func Slugify(title string) string {
	transliterated := slugTransliterations.Replace(title)
	lower := strings.ToLower(transliterated)
	slug := slugInvalidChars.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "game"
	}
	return slug
}

// uniqueSlug finds a slug not used by any other game, appending -2, -3, ...
// as needed. excludeID excludes a game from that collision check -- pass the
// game's own id on an update so keeping its title also keeps its slug
// instead of colliding with itself and getting suffixed; pass 0 (no real
// game has this id) on create, where there is no "own row" yet.
func uniqueSlug(db *sql.DB, base string, excludeID int64) (string, error) {
	candidate := base
	for suffix := 2; ; suffix++ {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM games WHERE slug = ? AND id != ?;`, candidate, excludeID).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}
