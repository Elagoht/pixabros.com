package media

import (
	"time"

	"pixabros/internal/storage"
)

type ReferenceLookup func() (map[int64]bool, error)

// SweepOrphans deletes media rows (and their backing files) that are not
// referenced by any module table and were created more than olderThan ago.
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
	for _, m := range all {
		if referencedIDs[m.ID] {
			continue
		}
		if now.Sub(m.CreatedAt) < olderThan {
			continue
		}
		if err := files.Delete(m.Path); err != nil {
			return deleted, err
		}
		if err := repo.Delete(m.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
