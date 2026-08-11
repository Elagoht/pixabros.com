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
// discarded. Slugs are immutable once assigned (see Update), so silently
// dropping these characters instead of transliterating them would be a
// permanent, unrecoverable defect for any non-English title.
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

func uniqueSlug(db *sql.DB, base string) (string, error) {
	candidate := base
	for suffix := 2; ; suffix++ {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM games WHERE slug = ?;`, candidate).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}
