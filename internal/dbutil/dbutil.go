// Package dbutil holds the small SQL helpers every content module needs.
// They live here rather than being copied into each module so that the rules
// they encode -- what counts as NULL, how a missing row is reported, how
// SQLite's timestamps are parsed -- stay identical across modules.
package dbutil

import (
	"database/sql"
	"strings"
	"time"
)

// NullableString stores an empty string as SQL NULL. Columns that are
// optional are nullable rather than "" so that "unset" and "deliberately
// blank" do not collapse into the same value.
func NullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// NullableID stores a nil id pointer as SQL NULL, for optional foreign keys.
func NullableID(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

// RequireRows converts "the statement matched nothing" into notFound. An
// UPDATE or DELETE against a missing id succeeds at the SQL level, so without
// this a caller cannot tell a no-op apart from a real change.
func RequireRows(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

// ParseTimestamp reads the timestamps SQLite's strftime writes. Those carry
// fractional seconds ("2026-08-12T09:15:04.123Z") which time.RFC3339 will not
// parse, so the fraction is trimmed before parsing.
func ParseTimestamp(s string) (time.Time, error) {
	normalized := s
	if whole, _, found := strings.Cut(s, "."); found {
		normalized = whole + "Z"
	}
	return time.Parse(time.RFC3339, normalized)
}
