package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
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
