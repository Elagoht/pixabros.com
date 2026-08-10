package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return nil, err
	}
	// foreign_keys is a per-connection pragma; pin pool to this connection to ensure it applies to all queries.
	conn.SetMaxOpenConns(1)
	return conn, nil
}
