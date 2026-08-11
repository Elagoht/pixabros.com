package games

type Screenshot struct {
	ID           int64
	GameID       int64
	MediaID      int64
	DisplayOrder int
}

func (r *Repo) AddScreenshot(gameID, mediaID int64, displayOrder int) (Screenshot, error) {
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

func (r *Repo) RemoveScreenshot(screenshotID int64) error {
	_, err := r.db.Exec(`DELETE FROM game_screenshots WHERE id = ?;`, screenshotID)
	return err
}
