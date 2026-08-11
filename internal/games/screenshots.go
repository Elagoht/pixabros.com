package games

import "errors"

var ErrScreenshotNotFound = errors.New("screenshot not found")

type Screenshot struct {
	ID           int64
	GameID       int64
	MediaID      int64
	DisplayOrder int
}

// AddScreenshot checks the game exists first so that a screenshot added
// against an unknown game surfaces as ErrGameNotFound rather than as a raw
// foreign-key violation the caller cannot classify.
func (r *Repo) AddScreenshot(gameID, mediaID int64, displayOrder int) (Screenshot, error) {
	if _, err := r.FindByID(gameID); err != nil {
		return Screenshot{}, err
	}
	res, err := r.db.Exec(
		`INSERT INTO game_screenshots (game_id, media_id, display_order) VALUES (?, ?, ?);`,
		gameID, mediaID, displayOrder,
	)
	if err != nil {
		return Screenshot{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Screenshot{}, err
	}
	return Screenshot{ID: id, GameID: gameID, MediaID: mediaID, DisplayOrder: displayOrder}, nil
}

func (r *Repo) ListScreenshots(gameID int64) ([]Screenshot, error) {
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
func (r *Repo) ReorderScreenshots(gameID int64, screenshotIDs []int64) error {
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

	for i, id := range screenshotIDs {
		res, err := stmt.Exec(i, id, gameID)
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
func (r *Repo) RemoveScreenshot(gameID, screenshotID int64) error {
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
