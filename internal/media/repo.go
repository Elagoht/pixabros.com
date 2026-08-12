package media

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"pixabros/internal/id"
)

var ErrMediaNotFound = errors.New("media not found")

type Media struct {
	ID      string
	Path    string
	Width   int
	Height  int
	Format  string
	AltText string
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(path string, width, height int) (Media, error) {
	newID := id.New()
	if _, err := r.db.Exec(
		`INSERT INTO media (id, path, width, height, format, alt_text) VALUES (?, ?, ?, ?, 'webp', '');`,
		newID, path, width, height,
	); err != nil {
		return Media{}, err
	}
	return r.FindByID(newID)
}

func (r *Repo) FindByID(mediaID string) (Media, error) {
	var m Media
	err := r.db.QueryRow(
		`SELECT id, path, width, height, format, alt_text FROM media WHERE id = ?;`, mediaID,
	).Scan(&m.ID, &m.Path, &m.Width, &m.Height, &m.Format, &m.AltText)
	if errors.Is(err, sql.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, err
	}
	return m, nil
}

func (r *Repo) Delete(mediaID string) error {
	_, err := r.db.Exec(`DELETE FROM media WHERE id = ?;`, mediaID)
	return err
}

func (r *Repo) AllIDs() ([]string, error) {
	rows, err := r.db.Query(`SELECT id FROM media;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var mediaID string
		if err := rows.Scan(&mediaID); err != nil {
			return nil, err
		}
		ids = append(ids, mediaID)
	}
	return ids, rows.Err()
}

type mediaWithCreatedAt struct {
	Media
	CreatedAt time.Time
}

func (r *Repo) allWithCreatedAt() ([]mediaWithCreatedAt, error) {
	rows, err := r.db.Query(`SELECT id, path, width, height, format, alt_text, created_at FROM media;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []mediaWithCreatedAt
	for rows.Next() {
		var m mediaWithCreatedAt
		var createdAtStr string
		if err := rows.Scan(&m.ID, &m.Path, &m.Width, &m.Height, &m.Format, &m.AltText, &createdAtStr); err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339, normalizeTimestamp(createdAtStr))
		if err != nil {
			return nil, err
		}
		m.CreatedAt = createdAt
		result = append(result, m)
	}
	return result, rows.Err()
}

func normalizeTimestamp(s string) string {
	if i := strings.Index(s, "."); i != -1 {
		return s[:i] + "Z"
	}
	return s
}
