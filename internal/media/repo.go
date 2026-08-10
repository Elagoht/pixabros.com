package media

import (
	"database/sql"
	"errors"
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
