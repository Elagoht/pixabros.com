// Package stats answers the dashboard's question: what is in here, and is
// anything waiting for me?
//
// Every number is counted live. There is no stats table and no cache: the
// content tables hold hundreds of rows, not millions, so a handful of COUNTs
// is cheaper than the bookkeeping needed to keep a denormalised total honest
// through every create, update and delete.
package stats

import (
	"database/sql"
)

// Stats is the whole dashboard payload.
type Stats struct {
	Games   GameStats    `json:"games"`
	Devlog  DevlogStats  `json:"devlog"`
	Awards  int          `json:"awards"`
	Members int          `json:"members"`
	Contact ContactStats `json:"contact"`
	Media   int          `json:"media"`
}

type GameStats struct {
	Total int `json:"total"`
	// Published is what the public site would show; the rest are drafts.
	Published int `json:"published"`
	// Playable counts games with a working web export. It is derived from the
	// uploaded build, never set by hand -- see games.Repo.SetBuild.
	Playable int `json:"playable"`
	ForSale  int `json:"for_sale"`
}

type DevlogStats struct {
	Total     int `json:"total"`
	Published int `json:"published"`
}

type ContactStats struct {
	Total int `json:"total"`
	// Unread is the only number here that asks the admin to do something, so
	// the dashboard gives it the prominence the others do not get.
	Unread int `json:"unread"`
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// Get collects every dashboard figure.
//
// The counts are grouped into one query per table rather than one query per
// number: SUM(condition) over a single scan gives every flag breakdown a table
// needs without re-scanning it once per flag.
func (r *Repo) Get() (Stats, error) {
	var s Stats

	if err := r.db.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(SUM(is_published), 0),
		        COALESCE(SUM(is_browser_playable), 0),
		        COALESCE(SUM(is_for_sale), 0)
		 FROM games;`,
	).Scan(&s.Games.Total, &s.Games.Published, &s.Games.Playable, &s.Games.ForSale); err != nil {
		return Stats{}, err
	}

	if err := r.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(is_published), 0) FROM devlog_posts;`,
	).Scan(&s.Devlog.Total, &s.Devlog.Published); err != nil {
		return Stats{}, err
	}

	if err := r.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END), 0)
		 FROM contact_submissions;`,
	).Scan(&s.Contact.Total, &s.Contact.Unread); err != nil {
		return Stats{}, err
	}

	if err := r.db.QueryRow(`SELECT COUNT(*) FROM awards;`).Scan(&s.Awards); err != nil {
		return Stats{}, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM members;`).Scan(&s.Members); err != nil {
		return Stats{}, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM media;`).Scan(&s.Media); err != nil {
		return Stats{}, err
	}

	return s, nil
}
