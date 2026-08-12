package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

// Migrate applies every pending migration.
//
// All of it runs on one pinned *sql.Conn rather than the pool, because the
// PRAGMAs below are per-connection: issued against the pool they could land
// on a different connection than the migration they are meant to guard.
//
// Foreign keys are disabled for the duration. A migration that rebuilds a
// table has to drop the original, and DROP TABLE performs an implicit
// DELETE -- which fires ON DELETE CASCADE and would silently empty every
// child table. PRAGMA foreign_keys is a no-op inside a transaction, so it
// must be set out here, around them. Once the migrations are in,
// foreign_key_check verifies that none of them left a dangling reference.
func Migrate(db *sql.DB) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied := map[int]bool{}
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations;`)
	if err != nil {
		return fmt.Errorf("query schema_migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	pending := false
	for _, m := range migrations {
		if !applied[m.version] {
			pending = true
			break
		}
	}
	if !pending {
		return nil
	}

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF;`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}

	migrateErr := applyPending(ctx, conn, migrations, applied)

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil && migrateErr == nil {
		return fmt.Errorf("re-enable foreign keys: %w", err)
	}
	if migrateErr != nil {
		return migrateErr
	}

	return checkForeignKeys(ctx, conn)
}

func applyPending(ctx context.Context, conn *sql.Conn, migrations []migration, applied map[int]bool) error {
	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?);`, m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.name, err)
		}
	}
	return nil
}

// checkForeignKeys fails loudly if the migrations just applied left a row
// pointing at a parent that no longer exists -- the exact damage disabling
// foreign keys makes possible.
func checkForeignKeys(ctx context.Context, conn *sql.Conn) error {
	rows, err := conn.QueryContext(ctx, `PRAGMA foreign_key_check;`)
	if err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		var table, parent sql.NullString
		var rowid sql.NullInt64
		var fkid sql.NullInt64
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return fmt.Errorf("scan foreign_key_check: %w", err)
		}
		violations = append(violations, fmt.Sprintf("%s -> %s", table.String, parent.String))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(violations) > 0 {
		return fmt.Errorf("migrations left %d dangling foreign key reference(s): %s",
			len(violations), strings.Join(violations, ", "))
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var migrations []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		versionStr := strings.SplitN(e.Name(), "_", 2)[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename %q: %w", e.Name(), err)
		}
		content, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration{version: version, name: e.Name(), sql: string(content)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	return migrations, nil
}
