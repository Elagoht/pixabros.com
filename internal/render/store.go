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
// (content-addressed by its ETag), and replaces rendered_pages/page_tags
// bookkeeping for pageKey with the renderer's freshly declared dependencies.
//
// Writing under an etag-derived key keeps the file store and the database
// consistent even if the transaction below fails: the new file is simply an
// orphan and rendered_pages still points at the previous etag, whose file is
// untouched. It also means page keys that are path prefixes of one another
// (e.g. "games" and "games/pixel-quest") can never collide on disk.
func (s *Store) RenderAndPersist(pageKey string, renderer Renderer) (string, error) {
	html, tags, err := renderer(pageKey)
	if err != nil {
		return "", err
	}

	etag := computeETag(html)

	if err := s.files.Put(renderedFileKey(etag), bytes.NewReader(html)); err != nil {
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
	rows, err := s.db.Query(`SELECT page_key FROM page_tags WHERE tag = ? ORDER BY page_key;`, tag)
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

func renderedFileKey(etag string) string {
	return "rendered/" + etag + ".html"
}

func computeETag(html []byte) string {
	sum := sha256.Sum256(html)
	return hex.EncodeToString(sum[:])
}

// HasPage reports whether a page is actually servable: its row exists *and*
// the file that row points at is still in the store.
//
// Checking both matters. The row and the file can drift apart -- a restored
// database backup, a deploy onto a fresh volume, a wiped store directory --
// and a row whose body is gone serves a 404 forever, because a reconciler that
// only looked at the row would decide the page was already there and never
// build it again.
func (s *Store) HasPage(pageKey string) (bool, error) {
	etag, found, err := s.ETag(pageKey)
	if err != nil || !found {
		return false, err
	}

	body, err := s.files.Get(renderedFileKey(etag))
	if err != nil {
		return false, nil
	}
	body.Close()
	return true, nil
}

// PageKeys lists every page currently rendered, so the reconciler can spot the
// ones that should no longer exist.
func (s *Store) PageKeys() ([]string, error) {
	rows, err := s.db.Query(`SELECT page_key FROM rendered_pages ORDER BY page_key;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Forget stops a page being served. Its page_tags rows go with it by cascade.
//
// The stored file is left alone on purpose: files are keyed by content hash
// and two pages with identical content share one, so deleting here could pull
// a file out from under a page that still needs it.
func (s *Store) Forget(pageKey string) error {
	_, err := s.db.Exec(`DELETE FROM rendered_pages WHERE page_key = ?;`, pageKey)
	return err
}
