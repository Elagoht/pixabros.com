package games

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrGameNotFound = errors.New("game not found")

type Game struct {
	ID                int64
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
	CartridgeArtID    *int64
	CDCoverArtID      *int64
	OGImageID         *int64
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
	CartridgeArtID    *int64
	CDCoverArtID      *int64
	OGImageID         *int64
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
	var cartridgeArtID, cdCoverArtID, ogImageID sql.NullInt64
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
		id := cartridgeArtID.Int64
		g.CartridgeArtID = &id
	}
	if cdCoverArtID.Valid {
		id := cdCoverArtID.Int64
		g.CDCoverArtID = &id
	}
	if ogImageID.Valid {
		id := ogImageID.Int64
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

func nullableInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func (r *Repo) Create(input CreateInput) (Game, error) {
	slug, err := uniqueSlug(r.db, Slugify(input.Title))
	if err != nil {
		return Game{}, err
	}
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}

	res, err := r.db.Exec(
		`INSERT INTO games (
			slug, title, short_description, full_description, tags,
			is_browser_playable, is_downloadable, is_for_sale,
			price_display, external_links_json, display_order, is_published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		slug, input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks, input.DisplayOrder, input.IsPublished,
	)
	if err != nil {
		return Game{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Game{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) FindByID(id int64) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE id = ?;`, id)
	return scanGame(row)
}

func (r *Repo) FindBySlug(slug string) (Game, error) {
	row := r.db.QueryRow(`SELECT `+gameColumns+` FROM games WHERE slug = ?;`, slug)
	return scanGame(row)
}

func (r *Repo) Update(id int64, input UpdateInput) (Game, error) {
	externalLinks := input.ExternalLinksJSON
	if externalLinks == "" {
		externalLinks = "[]"
	}

	res, err := r.db.Exec(
		`UPDATE games SET
			title = ?, short_description = ?, full_description = ?, tags = ?,
			is_browser_playable = ?, is_downloadable = ?, is_for_sale = ?,
			price_display = ?, external_links_json = ?,
			cartridge_art_id = ?, cd_cover_art_id = ?, og_image_id = ?,
			display_order = ?, is_published = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		input.Title, input.ShortDescription, input.FullDescription, input.Tags,
		input.IsBrowserPlayable, input.IsDownloadable, input.IsForSale,
		nullableString(input.PriceDisplay), externalLinks,
		nullableInt64(input.CartridgeArtID), nullableInt64(input.CDCoverArtID), nullableInt64(input.OGImageID),
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

func (r *Repo) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM games WHERE id = ?;`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
}

func (r *Repo) SetWebExportPath(id int64, path string) error {
	res, err := r.db.Exec(
		`UPDATE games SET web_export_path = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
		path, id,
	)
	if err != nil {
		return err
	}
	return requireRowsAffected(res)
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
