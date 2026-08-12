// Package id generates the public identifiers used by every content row.
//
// Rows are keyed by CUID2 rather than an autoincrementing integer because
// these ids appear in admin URLs and API payloads: sequential integers would
// expose how many games (or contact submissions) exist and let anyone walk
// the whole table by counting up. CUID2 ids are fixed-length, collision
// resistant, URL-safe and carry no embedded timestamp.
package id

import (
	"fmt"

	"github.com/nrednav/cuid2"
)

// Length is the generated id length. It matches the 24 characters produced
// by lower(hex(randomblob(12))), which migration 0015 used to backfill ids
// for rows that predate this package -- both are 24 lowercase alphanumeric
// characters, so nothing downstream can tell them apart.
const Length = 24

var generate func() string

func init() {
	g, err := cuid2.Init(cuid2.WithLength(Length))
	if err != nil {
		panic(fmt.Sprintf("id: initialise cuid2 generator: %v", err))
	}
	generate = g
}

// New returns a fresh identifier.
func New() string {
	return generate()
}

// IsValid reports whether s has the shape of an id this package hands out.
// It is a cheap sanity check for path segments, not an existence check.
func IsValid(s string) bool {
	if len(s) != Length {
		return false
	}
	for _, r := range s {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if !isLower && !isDigit {
			return false
		}
	}
	return true
}
