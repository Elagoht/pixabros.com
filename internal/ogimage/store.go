package ogimage

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"image"
	"strings"
	"time"

	"pixabros/internal/imaging"
	"pixabros/internal/media"
	"pixabros/internal/storage"
)

// generatedPrefix marks an image this package drew, as opposed to one an admin
// uploaded. It is what lets a regenerated title replace its old picture
// without ever touching artwork a person chose.
const generatedPrefix = "media/og_generated/"

// Store draws a post's social preview and keeps it in the media library like
// any other image, so the orphan sweep and the media screens treat it the same.
type Store struct {
	repo  *media.Repo
	files storage.Storage
	// db is read for the studio's logo, which the card is built around.
	db *sql.DB
}

func NewStore(repo *media.Repo, files storage.Storage, db *sql.DB) *Store {
	return &Store{repo: repo, files: files, db: db}
}

// logo loads the studio's mark. A card without it is still a card, so a
// missing or unreadable logo is not an error -- it just leaves the band bare.
func (s *Store) logo() image.Image {
	if s.db == nil {
		return nil
	}

	var mediaID string
	if err := s.db.QueryRow(
		`SELECT value FROM site_settings WHERE key = 'org_logo' AND value != '';`,
	).Scan(&mediaID); err != nil {
		return nil
	}

	image, err := s.repo.FindByID(mediaID)
	if err != nil {
		return nil
	}
	file, err := s.files.Get(image.Path)
	if err != nil {
		return nil
	}
	defer file.Close()

	decoded, err := imaging.Decode(file)
	if err != nil {
		return nil
	}
	return decoded
}

// IsGenerated reports whether a stored path is one of ours.
func IsGenerated(path string) bool {
	return strings.HasPrefix(path, generatedPrefix)
}

// Create draws the title and stores the result.
func (s *Store) Create(title string) (media.Media, error) {
	encoded, err := GenerateWebP(title, s.logo())
	if err != nil {
		return media.Media{}, fmt.Errorf("draw og image: %w", err)
	}

	key, err := generatedKey()
	if err != nil {
		return media.Media{}, err
	}
	if err := s.files.Put(key, bytes.NewReader(encoded)); err != nil {
		return media.Media{}, fmt.Errorf("store og image: %w", err)
	}

	image, err := s.repo.Create(key, Width, Height)
	if err != nil {
		// Best-effort cleanup, so a failed insert does not leave a file behind
		// that nothing points at.
		_ = s.files.Delete(key)
		return media.Media{}, fmt.Errorf("record og image: %w", err)
	}
	return image, nil
}

// Refresh returns the image a post should carry for the given title.
//
// It leaves an uploaded picture alone: an admin who chose one has overridden
// the generated default, and a title edit must not throw that away. Otherwise
// it draws a new one and deletes the old, so a renamed post never leaves its
// previous preview behind.
func (s *Store) Refresh(currentID *string, title string) (*string, error) {
	if currentID != nil {
		existing, err := s.repo.FindByID(*currentID)
		if err == nil && !IsGenerated(existing.Path) {
			return currentID, nil
		}
	}

	image, err := s.Create(title)
	if err != nil {
		return nil, err
	}

	if currentID != nil {
		// Deleting the row first: the sweep would reclaim the file anyway, but
		// leaving the record would show a stale preview in the media library.
		if old, err := s.repo.FindByID(*currentID); err == nil && IsGenerated(old.Path) {
			_ = s.repo.Delete(old.ID)
			_ = s.files.Delete(old.Path)
		}
	}

	return &image.ID, nil
}

func generatedKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d-%s.webp", generatedPrefix, time.Now().Year(), hex.EncodeToString(b)), nil
}
