package settings

import (
	"database/sql"
	"fmt"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// Values reads every stored value in the group. Keys the registry defines but
// nobody has filled in yet come back as empty strings rather than being
// absent, so callers never have to distinguish "unset" from "blank".
func (r *Repo) Values(group Group) (map[string]string, error) {
	values := make(map[string]string, len(group.Definitions))
	for _, definition := range group.Definitions {
		values[definition.Key] = ""
	}

	rows, err := r.db.Query(`SELECT key, value FROM ` + group.Table + `;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		// A key left over from an earlier version of the registry is ignored
		// rather than surfaced: nothing reads it, and showing it would invite
		// editing something with no effect.
		if _, ok := values[key]; ok {
			values[key] = value
		}
	}
	return values, rows.Err()
}

// Replace writes the given values, which must all be keys this group defines.
// It is one transaction: a half-applied settings save would leave the site
// describing itself inconsistently.
func (r *Repo) Replace(group Group, values map[string]string) error {
	for key := range values {
		if _, err := group.Define(key); err != nil {
			return err
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO ` + group.Table + ` (key, value, value_type) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, value_type = excluded.value_type;`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range values {
		definition, err := group.Define(key)
		if err != nil {
			return err
		}
		// value_type mirrors the registry rather than being supplied by the
		// caller, so the two can never drift apart.
		if _, err := stmt.Exec(key, value, string(definition.Kind)); err != nil {
			return fmt.Errorf("store %q: %w", key, err)
		}
	}

	return tx.Commit()
}

// MediaValues returns the media ids referenced by this group, which is what
// the orphan-media sweep needs in order not to delete an image the site is
// still using. It reads value_type rather than the registry so it keeps
// working for a key the registry no longer defines.
func (r *Repo) MediaValues(group Group) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT value FROM ` + group.Table + ` WHERE value_type = 'media' AND value != '';`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		ids = append(ids, value)
	}
	return ids, rows.Err()
}

// MediaExists reports whether a media row is really there. The settings tables
// store ids as plain text with no foreign key, so this is the only thing
// standing between a typo and a setting pointing at nothing.
func (r *Repo) MediaExists(mediaID string) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM media WHERE id = ?;`, mediaID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
