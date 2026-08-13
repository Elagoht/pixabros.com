package admincmd

import (
	"database/sql"
	"fmt"

	"pixabros/internal/config"
	"pixabros/internal/media"
	"pixabros/internal/ogimage"
	"pixabros/internal/render"
	"pixabros/internal/storage"
)

// ogPost is the little a redraw needs to know about a post.
type ogPost struct {
	id      string
	title   string
	gameID  *string
	imageID *string
}

// redrawOGImages redraws every generated devlog card.
//
// A card is normally only redrawn when its post is renamed, which is right for
// editing but leaves every existing card behind whenever the drawing itself
// changes. This is the pass that catches them up. Cards an admin uploaded are
// left alone: Refresh treats a chosen picture as an override, and a redesign is
// no reason to throw one away.
func redrawOGImages() error {
	conn, err := openDB()
	if err != nil {
		return err
	}
	defer conn.Close()

	cfg := config.Load()
	store := ogimage.NewStore(media.NewRepo(conn), storage.NewLocalDisk(cfg.DataDir, ""), conn)

	posts, err := listOGPosts(conn)
	if err != nil {
		return err
	}

	redrawn := 0
	for _, p := range posts {
		refreshed, err := store.Refresh(p.imageID, p.title, p.gameID)
		if err != nil {
			return fmt.Errorf("redraw %q: %w", p.title, err)
		}
		// An unchanged id means the post carries an uploaded picture, which stays.
		if p.imageID != nil && refreshed != nil && *refreshed == *p.imageID {
			continue
		}

		if _, err := conn.Exec(
			`UPDATE devlog_posts SET og_image_id = ? WHERE id = ?;`, refreshed, p.id,
		); err != nil {
			return fmt.Errorf("attach card to %q: %w", p.title, err)
		}
		if err := render.EnqueueRegen(conn, "devlog:"+p.id); err != nil {
			return fmt.Errorf("queue regen for %q: %w", p.title, err)
		}
		redrawn++
	}

	if redrawn > 0 {
		// The index shows the cards as thumbnails, so it goes stale too.
		if err := render.EnqueueRegen(conn, "devlog:list"); err != nil {
			return fmt.Errorf("queue index regen: %w", err)
		}
	}
	fmt.Printf("redrawn: %d of %d post(s)\n", redrawn, len(posts))
	return nil
}

// listOGPosts reads every post up front rather than redrawing while the rows
// are still open, because each redraw writes to the same database.
func listOGPosts(conn *sql.DB) ([]ogPost, error) {
	rows, err := conn.Query(`SELECT id, title, game_id, og_image_id FROM devlog_posts;`)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	var posts []ogPost
	for rows.Next() {
		var p ogPost
		if err := rows.Scan(&p.id, &p.title, &p.gameID, &p.imageID); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read posts: %w", err)
	}
	return posts, nil
}
