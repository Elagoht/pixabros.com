package media

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrMediaNotFound = errors.New("media not found")

type Media struct {
	ID      int64
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
	res, err := r.db.Exec(
		`INSERT INTO media (path, width, height, format, alt_text) VALUES (?, ?, ?, 'webp', '');`,
		path, width, height,
	)
	if err != nil {
		return Media{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Media{}, err
	}
	return r.FindByID(id)
}

func (r *Repo) FindByID(id int64) (Media, error) {
	var m Media
	err := r.db.QueryRow(
		`SELECT id, path, width, height, format, alt_text FROM media WHERE id = ?;`, id,
	).Scan(&m.ID, &m.Path, &m.Width, &m.Height, &m.Format, &m.AltText)
	if errors.Is(err, sql.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, err
	}
	return m, nil
}

func (r *Repo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM media WHERE id = ?;`, id)
	return err
}

func (r *Repo) AllIDs() ([]int64, error) {
	rows, err := r.db.Query(`SELECT id FROM media;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
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
