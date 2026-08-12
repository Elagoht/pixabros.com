package games

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"pixabros/internal/id"
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
	IsDownloadable    bool
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
	IsBrowserPlayable bool
	IsDownloadable    bool
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
	IsBrowserPlayable bool
	IsDownloadable    bool
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
	is_browser_playable, is_downloadable, is_for_sale,
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
		&g.IsBrowserPlayable, &g.IsDownloadable, &g.IsForSale,
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
	slug, err := uniqueSlug(r.db, Slugify(input.Title), "")
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
			is_browser_playable, is_downloadable, is_for_sale,
			price_display, external_links_json, display_order, is_published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		newID, slug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks, input.DisplayOrder, input.IsPublished,
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
	slug, err := uniqueSlug(r.db, Slugify(input.Title), id)
	if err != nil {
		return Game{}, err
	}

	res, err := r.db.Exec(
		`UPDATE games SET
			slug = ?, title = ?, short_description = ?, full_description = ?, tags = ?,
			is_browser_playable = ?, is_downloadable = ?, is_for_sale = ?,
			price_display = ?, external_links_json = ?,
			cartridge_art_id = ?, cd_cover_art_id = ?, og_image_id = ?,
			display_order = ?, is_published = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		slug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
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

func (r *Repo) SetWebExportPath(id string, path string) error {
	res, err := r.db.Exec(
		`UPDATE games SET web_export_path = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
		path, id,
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

func (r *Repo) List() ([]Game, error) {
	rows, err := r.db.Query(`SELECT ` + gameColumns + ` FROM games ORDER BY display_order ASC, id ASC;`)
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
