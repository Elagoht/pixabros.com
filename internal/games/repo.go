package games

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"pixabros/internal/id"
	"pixabros/internal/slug"
)

var ErrGameNotFound = errors.New("game not found")

type Game struct {
	ID                string
	Slug              string
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsBrowserPlayable bool
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	CartridgeArtID    *string
	CDCoverArtID      *string
	OGImageID         *string
	WebExportPath     string
	DisplayOrder      int
	IsPublished       bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateInput struct {
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	DisplayOrder      int
	IsPublished       bool
}

type UpdateInput struct {
	Title             string
	ShortDescription  string
	FullDescription   string
	Tags              string
	IsForSale         bool
	PriceDisplay      string
	ExternalLinksJSON string
	CartridgeArtID    *string
	CDCoverArtID      *string
	OGImageID         *string
	DisplayOrder      int
	IsPublished       bool
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const gameColumns = `id, slug, title, short_description, full_description, tags,
	is_browser_playable, is_for_sale,
	price_display, external_links_json,
	cartridge_art_id, cd_cover_art_id, og_image_id,
	web_export_path, display_order, is_published, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanGame(row rowScanner) (Game, error) {
	var g Game
	var priceDisplay, webExportPath sql.NullString
	var cartridgeArtID, cdCoverArtID, ogImageID sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&g.ID, &g.Slug, &g.Title, &g.ShortDescription, &g.FullDescription, &g.Tags,
		&g.IsBrowserPlayable, &g.IsForSale,
		&priceDisplay, &g.ExternalLinksJSON,
		&cartridgeArtID, &cdCoverArtID, &ogImageID,
		&webExportPath, &g.DisplayOrder, &g.IsPublished,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrGameNotFound
	}
	if err != nil {
		return Game{}, err
	}

	if priceDisplay.Valid {
		g.PriceDisplay = priceDisplay.String
	}
	if webExportPath.Valid {
		g.WebExportPath = webExportPath.String
	}
	if cartridgeArtID.Valid {
		id := cartridgeArtID.String
		g.CartridgeArtID = &id
	}
	if cdCoverArtID.Valid {
		id := cdCoverArtID.String
		g.CDCoverArtID = &id
	}
	if ogImageID.Valid {
		id := ogImageID.String
		g.OGImageID = &id
	}

	createdAt, err := parseTimestamp(createdAtStr)
	if err != nil {
		return Game{}, err
	}
	updatedAt, err := parseTimestamp(updatedAtStr)
	if err != nil {
		return Game{}, err
	}
	g.CreatedAt = createdAt
	g.UpdatedAt = updatedAt

	return g, nil
}

func parseTimestamp(s string) (time.Time, error) {
	normalized := s
	if i := strings.Index(s, "."); i != -1 {
		normalized = s[:i] + "Z"
	}
	return time.Parse(time.RFC3339, normalized)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableID(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func (r *Repo) Create(input CreateInput) (Game, error) {
	newSlug, err := slug.Unique(r.db, "games", slug.Make(input.Title, "game"), "")
	if err != nil {
		return Game{}, err
	}
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}

	newID := id.New()
	if _, err := r.db.Exec(
		`INSERT INTO games (
			id, slug, title, short_description, full_description, tags,
			is_for_sale, price_display, external_links_json, display_order, is_published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		newID, newSlug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsForSale, nullableString(input.PriceDisplay), externalLinks,
		input.DisplayOrder, input.IsPublished,
	); err != nil {
		return Game{}, err
	}
	return r.FindByID(newID)
}

func (r *Repo) FindByID(id string) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE id = ?;`, id)
	return scanGame(row)
}

func (r *Repo) FindBySlug(slug string) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE slug = ?;`, slug)
	return scanGame(row)
}

// Update regenerates the slug from the (possibly changed) title on every
// call. excludeID=id in the uniqueSlug lookup means an unchanged title
// resolves back to the same slug rather than colliding with itself and
// picking up a spurious -2 suffix.
func (r *Repo) Update(id string, input UpdateInput) (Game, error) {
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}
	newSlug, err := slug.Unique(r.db, "games", slug.Make(input.Title, "game"), id)
	if err != nil {
		return Game{}, err
	}

	res, err := r.db.Exec(
		`UPDATE games SET
			slug = ?, title = ?, short_description = ?, full_description = ?, tags = ?,
			is_for_sale = ?,
			price_display = ?, external_links_json = ?,
			cartridge_art_id = ?, cd_cover_art_id = ?, og_image_id = ?,
			display_order = ?, is_published = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		newSlug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks,
		nullableID(input.CartridgeArtID), nullableID(input.CDCoverArtID), nullableID(input.OGImageID),
		input.DisplayOrder, input.IsPublished, id,
	)
	if err != nil {
		return Game{}, err
	}
	if err := requireRowsAffected(res); err != nil {
		return Game{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM games WHERE id = ?;`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}

// SetBuild records where a game's playable build was extracted to, and
// derives is_browser_playable from it: a game is playable in the browser
// exactly when a build exists. Passing an empty path clears both, which is
// what removing a build does. Nothing else may write is_browser_playable --
// the admin form does not offer it as a field.
func (r *Repo) SetBuild(id string, path string) error {
	res, err := r.db.Exec(
		`UPDATE games SET web_export_path = ?, is_browser_playable = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
		nullableString(path), path != "", id,
	)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}

// Reorder sets display_order to each id's index in ids, in one transaction:
// the admin UI's reorder control always sends the complete ordered id list
// (it already has every game loaded), so a partial or unknown id is a bug in
// the caller, not a normal case -- the whole reorder is rolled back rather
// than silently applying half of it.
func (r *Repo) Reorder(ids []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE games SET display_order = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range ids {
		res, err := stmt.Exec(i, id)
		if err != nil {
			return err
		}
		if err := requireRowsAffected(res); err != nil {
			return err
		}
	}

	return tx.Commit()
}

var ErrInvalidSort = errors.New("unknown sort field or direction")

// sortableColumns whitelists what the admin list may be ordered by, mapping
// the API's field names to SQL. Interpolating anything into ORDER BY has to
// come from this map and nowhere else -- a caller-supplied column name would
// be an injection point that no amount of quoting fixes.
//
// Text columns sort case-insensitively so "Zelda" and "asteroids" interleave
// the way a person reading the list expects, rather than all upper-case
// titles sorting ahead of all lower-case ones.
var sortableColumns = map[string]string{
	"title":         "title COLLATE NOCASE",
	"slug":          "slug COLLATE NOCASE",
	"is_published":  "is_published",
	"display_order": "display_order",
	"created_at":    "created_at",
	"updated_at":    "updated_at",
}

// SortableFields lists the accepted sort fields, for error messages.
func SortableFields() []string {
	fields := make([]string, 0, len(sortableColumns))
	for field := range sortableColumns {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

// List returns every game ordered by the given field and direction. An empty
// field means the manual display order, which is what the drag-to-reorder
// control writes. id is always the final tiebreaker so that rows sharing a
// value (every game on display_order 0, say) keep a stable order between
// requests instead of shuffling.
func (r *Repo) List(field string, descending bool) ([]Game, error) {
	if field == "" {
		field = "display_order"
	}
	column, ok := sortableColumns[field]
	if !ok {
		return nil, ErrInvalidSort
	}
	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	rows, err := r.db.Query(
		`SELECT ` + gameColumns + ` FROM games ORDER BY ` + column + ` ` + direction + `, id ASC;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Game
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func requireRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGameNotFound
	}
	return nil
}
