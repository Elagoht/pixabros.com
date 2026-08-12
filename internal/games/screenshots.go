package games

import (
	"errors"

	"pixabros/internal/id"
)

var ErrScreenshotNotFound = errors.New("screenshot not found")

type Screenshot struct {
	ID           string
	GameID       string
	MediaID      string
	DisplayOrder int
}

// AddScreenshot checks the game exists first so that a screenshot added
// against an unknown game surfaces as ErrGameNotFound rather than as a raw
// foreign-key violation the caller cannot classify.
func (r *Repo) AddScreenshot(gameID, mediaID string, displayOrder int) (Screenshot, error) {
	if _, err := r.FindByID(gameID); err != nil {
		return Screenshot{}, err
	}
	newID := id.New()
	if _, err := r.db.Exec(
		`INSERT INTO game_screenshots (id, game_id, media_id, display_order) VALUES (?, ?, ?, ?);`,
		newID, gameID, mediaID, displayOrder,
	); err != nil {
		return Screenshot{}, err
	}
	return Screenshot{ID: newID, GameID: gameID, MediaID: mediaID, DisplayOrder: displayOrder}, nil
}

func (r *Repo) ListScreenshots(gameID string) ([]Screenshot, error) {
	rows, err := r.db.Query(
		`SELECT id, game_id, media_id, display_order FROM game_screenshots
		 WHERE game_id = ? ORDER BY display_order ASC, id ASC;`,
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Screenshot
	for rows.Next() {
		var s Screenshot
		if err := rows.Scan(&s.ID, &s.GameID, &s.MediaID, &s.DisplayOrder); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ReorderScreenshots sets display_order to each id's index in screenshotIDs,
// scoped to gameID the same way RemoveScreenshot is: a screenshot id
// belonging to a different game must not be reorderable through another
// game's URL. The whole reorder is one transaction, rolled back if any id
// doesn't belong to this game.
func (r *Repo) ReorderScreenshots(gameID string, screenshotIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE game_screenshots SET display_order = ? WHERE id = ? AND game_id = ?;`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, screenshotID := range screenshotIDs {
		res, err := stmt.Exec(i, screenshotID, gameID)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrScreenshotNotFound
		}
	}

	return tx.Commit()
}

// RemoveScreenshot scopes the delete to gameID as well as screenshotID: a
// screenshot ID belonging to a different game must not be deletable through
// another game's URL, since the caller only invalidates the rendered page of
// the game it was called for.
func (r *Repo) RemoveScreenshot(gameID, screenshotID string) error {
	res, err := r.db.Exec(`DELETE FROM game_screenshots WHERE id = ? AND game_id = ?;`, screenshotID, gameID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrScreenshotNotFound
	}
	return nil
}
