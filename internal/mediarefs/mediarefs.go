// Package mediarefs answers one question: what is using this image?
//
// It is the single place that knows every column pointing at media. Two things
// depend on that being complete:
//
//   - the media library, to show where an image is used and to refuse to
//     delete one that is still in use;
//   - the orphan sweep, which deletes unreferenced images and their files.
//
// A media column that exists but is missing from `sources` below would make
// the sweep delete an image the site is still showing. TestSourcesCoverEvery
// MediaForeignKey guards against exactly that by reading the schema.
package mediarefs

import (
	"database/sql"
	"fmt"
)

// Usage is one place an image is referenced.
type Usage struct {
	// Module names the admin section, so the UI can say where to go and look.
	Module string `json:"module"`
	// Label identifies the row within it, e.g. a game's title.
	Label string `json:"label"`
	// RowID is that row's own id, which is what lets a caller rebuild the one
	// page showing this image instead of every page in the module. Not sent to
	// the admin UI, which only ever displays the label.
	RowID string `json:"-"`
}

// source is one column that can hold a media id, together with a query
// returning (media_id, row_id, label) for every row that fills it in.
//
// The row id is there so a caller can name the one page that has to be
// rebuilt when an image changes, rather than rebuilding the whole module.
type source struct {
	module string
	query  string
}

// sources lists every media reference in the database. Table and column names
// are constants here, never values from outside.
var sources = []source{
	{"games", `SELECT cartridge_art_id, id, title FROM games WHERE cartridge_art_id IS NOT NULL`},
	{"games", `SELECT cd_cover_art_id, id, title FROM games WHERE cd_cover_art_id IS NOT NULL`},
	{"games", `SELECT og_image_id, id, title FROM games WHERE og_image_id IS NOT NULL`},
	{"games", `SELECT s.media_id, g.id, g.title FROM game_screenshots s JOIN games g ON g.id = s.game_id`},
	{"members", `SELECT avatar_id, id, name FROM members WHERE avatar_id IS NOT NULL`},
	{"devlog", `SELECT og_image_id, id, title FROM devlog_posts WHERE og_image_id IS NOT NULL`},
	{"awards", `SELECT picture_id, id, title FROM awards WHERE picture_id IS NOT NULL`},
	// The settings tables have no foreign key -- they store ids as plain text
	// tagged with value_type -- so they cannot be discovered from the schema
	// and have to be listed explicitly. Their "row id" is the setting's key.
	{"site-settings", `SELECT value, key, key FROM site_settings WHERE value_type = 'media' AND value != ''`},
	{"homepage", `SELECT value, key, key FROM homepage_settings WHERE value_type = 'media' AND value != ''`},
}

// Lookup returns every usage of every referenced image, keyed by media id.
func Lookup(db *sql.DB) (map[string][]Usage, error) {
	usages := map[string][]Usage{}

	for _, src := range sources {
		rows, err := db.Query(src.query)
		if err != nil {
			return nil, fmt.Errorf("media references for %s: %w", src.module, err)
		}
		for rows.Next() {
			var mediaID, rowID, label string
			if err := rows.Scan(&mediaID, &rowID, &label); err != nil {
				rows.Close()
				return nil, err
			}
			usages[mediaID] = append(usages[mediaID], Usage{
				Module: src.module, Label: label, RowID: rowID,
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return usages, nil
}

// ReferencedIDs is what the orphan sweep needs: the set of images something
// still points at.
func ReferencedIDs(db *sql.DB) (map[string]bool, error) {
	usages, err := Lookup(db)
	if err != nil {
		return nil, err
	}
	referenced := make(map[string]bool, len(usages))
	for mediaID := range usages {
		referenced[mediaID] = true
	}
	return referenced, nil
}
