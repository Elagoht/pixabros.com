package games

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(title string) string {
	lower := strings.ToLower(title)
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
