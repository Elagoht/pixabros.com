package media

import (
	"errors"
	"fmt"
	"time"

	"pixabros/internal/storage"
)

type ReferenceLookup func() (map[string]bool, error)

// SweepOrphans deletes media rows (and their backing files) that are not
// referenced by any module table and were created more than olderThan ago.
// If a single candidate's delete fails, the sweep continues with remaining candidates
// and returns a combined error along with the count of successfully deleted candidates.
func SweepOrphans(repo *Repo, files storage.Storage, referenced ReferenceLookup, olderThan time.Duration, now time.Time) (int, error) {
	referencedIDs, err := referenced()
	if err != nil {
		return 0, err
	}

	all, err := repo.allWithCreatedAt()
	if err != nil {
		return 0, err
	}

	deleted := 0
	var errs []error
	for _, m := range all {
		if referencedIDs[m.ID] {
			continue
		}
		if now.Sub(m.CreatedAt) < olderThan {
			continue
		}
		if err := files.Delete(m.Path); err != nil {
			errs = append(errs, fmt.Errorf("delete file for media %s: %w", m.ID, err))
			continue
		}
		if err := repo.Delete(m.ID); err != nil {
			errs = append(errs, fmt.Errorf("delete media row %s: %w", m.ID, err))
			continue
		}
		deleted++
	}
	return deleted, errors.Join(errs...)
}
