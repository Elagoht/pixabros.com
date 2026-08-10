package render

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	"pixabros/internal/storage"
)

type Store struct {
	db    *sql.DB
	files storage.Storage
}

func NewStore(db *sql.DB, files storage.Storage) *Store {
	return &Store{db: db, files: files}
}

// RenderAndPersist calls renderer, writes the resulting HTML via Storage
// (keyed by pageKey), and replaces rendered_pages/page_tags bookkeeping for
// pageKey with the renderer's freshly declared dependencies.
func (s *Store) RenderAndPersist(pageKey string, renderer Renderer) (string, error) {
	html, tags, err := renderer(pageKey)
	if err != nil {
		return "", err
	}

	etag := computeETag(html)

	if err := s.files.Put(renderedFileKey(pageKey), bytes.NewReader(html)); err != nil {
		return "", err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO rendered_pages (page_key, etag) VALUES (?, ?)
		 ON CONFLICT(page_key) DO UPDATE SET etag = excluded.etag, rendered_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');`,
		pageKey, etag,
	); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_key = ?;`, pageKey); err != nil {
		return "", err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT INTO page_tags (page_key, tag) VALUES (?, ?);`, pageKey, tag); err != nil {
			return "", err
		}
	}

	return etag, tx.Commit()
}

func (s *Store) ETag(pageKey string) (string, bool, error) {
	var etag string
	err := s.db.QueryRow(`SELECT etag FROM rendered_pages WHERE page_key = ?;`, pageKey).Scan(&etag)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return etag, true, nil
}

func (s *Store) PageKeysForTag(tag string) ([]string, error) {
	rows, err := s.db.Query(`SELECT page_key FROM page_tags WHERE tag = ?;`, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pageKeys []string
	for rows.Next() {
		var pk string
		if err := rows.Scan(&pk); err != nil {
			return nil, err
		}
		pageKeys = append(pageKeys, pk)
	}
	return pageKeys, rows.Err()
}

func renderedFileKey(pageKey string) string {
	return "rendered/" + pageKey
}

func computeETag(html []byte) string {
	sum := sha256.Sum256(html)
	return hex.EncodeToString(sum[:])
}
