// Package slug turns titles into the URL segments the public site uses.
package slug

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var invalidChars = regexp.MustCompile(`[^a-z0-9]+`)

// transliterations maps characters common in the languages this project's
// content is actually written in (starting with Turkish) to a plain-ASCII
// equivalent, applied before non-ASCII characters are simply discarded, so a
// non-English title still produces a readable slug.
var transliterations = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i",
	"İ", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
)

// Make builds a slug from a title. It never returns an empty string, since a
// slug is a URL segment: a title with nothing slug-able in it falls back to
// the given placeholder.
func Make(title, fallback string) string {
	lower := strings.ToLower(transliterations.Replace(title))
	trimmed := strings.Trim(invalidChars.ReplaceAllString(lower, "-"), "-")
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// Unique finds a slug in table that no other row is using, appending -2, -3,
// ... as needed. excludeID leaves one row out of the collision check: pass the
// row's own id on an update so keeping its title also keeps its slug instead
// of colliding with itself and picking up a suffix; pass "" on create.
//
// table is interpolated into the query, so it must be a constant supplied by
// the calling package -- never a value that reached the process from outside.
func Unique(db *sql.DB, table, base, excludeID string) (string, error) {
	candidate := base
	for suffix := 2; ; suffix++ {
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM `+table+` WHERE slug = ? AND id != ?;`, candidate, excludeID,
		).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}
