// Package contact reads the submissions left by the public contact form.
//
// Nothing here creates a submission: they arrive from the public site. The
// admin side can only read them, mark them read or unread, and delete them.
// Submissions never appear on the public site either, so unlike every other
// content module a change here invalidates no rendered page.
package contact

import (
	"database/sql"
	"errors"
	"sort"
	"time"

	"pixabros/internal/dbutil"
)

var ErrSubmissionNotFound = errors.New("contact submission not found")

var ErrInvalidSort = errors.New("unknown sort field or direction")

type Submission struct {
	ID string
	// Name is empty for anything submitted through the public form, which does
	// not ask for one.
	Name          string
	Subject       string
	Phone         string
	Email         string
	Message       string
	WantsCallback bool
	IsRead        bool
	IPAddress     string
	CreatedAt     time.Time
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const submissionColumns = `id, name, subject, phone, email, message,
	wants_callback, is_read, ip_address, created_at`

// sortableColumns whitelists what the admin list may be ordered by. Anything
// interpolated into ORDER BY must come from this map and nowhere else.
var sortableColumns = map[string]string{
	"name":       "name COLLATE NOCASE",
	"subject":    "subject COLLATE NOCASE",
	"email":      "email COLLATE NOCASE",
	"is_read":    "is_read",
	"created_at": "created_at",
}

// SortableFields lists the accepted sort fields, for error messages.
func SortableFields() []string {
	fields := make([]string, 0, len(sortableColumns))
	for field := range sortableColumns {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubmission(row rowScanner) (Submission, error) {
	var s Submission
	var name, phone, email sql.NullString
	var createdAtStr string

	err := row.Scan(
		&s.ID, &name, &s.Subject, &phone, &email, &s.Message,
		&s.WantsCallback, &s.IsRead, &s.IPAddress, &createdAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Submission{}, ErrSubmissionNotFound
	}
	if err != nil {
		return Submission{}, err
	}

	if name.Valid {
		s.Name = name.String
	}
	if phone.Valid {
		s.Phone = phone.String
	}
	if email.Valid {
		s.Email = email.String
	}

	if s.CreatedAt, err = dbutil.ParseTimestamp(createdAtStr); err != nil {
		return Submission{}, err
	}
	return s, nil
}

func (r *Repo) FindByID(submissionID string) (Submission, error) {
	row := r.db.QueryRow(
		`SELECT `+submissionColumns+` FROM contact_submissions WHERE id = ?;`, submissionID,
	)
	return scanSubmission(row)
}

// SetRead marks a submission read or unread. It is the only field the admin
// can change: everything else is what the sender wrote.
func (r *Repo) SetRead(submissionID string, isRead bool) (Submission, error) {
	res, err := r.db.Exec(
		`UPDATE contact_submissions SET is_read = ? WHERE id = ?;`, isRead, submissionID,
	)
	if err != nil {
		return Submission{}, err
	}
	if err := dbutil.RequireRows(res, ErrSubmissionNotFound); err != nil {
		return Submission{}, err
	}
	return r.FindByID(submissionID)
}

func (r *Repo) Delete(submissionID string) error {
	res, err := r.db.Exec(`DELETE FROM contact_submissions WHERE id = ?;`, submissionID)
	if err != nil {
		return err
	}
	return dbutil.RequireRows(res, ErrSubmissionNotFound)
}

// UnreadCount is what an inbox badge would show.
func (r *Repo) UnreadCount() (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM contact_submissions WHERE is_read = 0;`,
	).Scan(&count)
	return count, err
}

// List returns every submission ordered by the given field and direction.
// With no field chosen the newest arrival comes first, which is the only
// order an inbox is useful in.
func (r *Repo) List(field string, descending bool) ([]Submission, error) {
	if field == "" {
		return r.query(`ORDER BY created_at DESC, id ASC`)
	}
	column, ok := sortableColumns[field]
	if !ok {
		return nil, ErrInvalidSort
	}
	direction := "ASC"
	if descending {
		direction = "DESC"
	}
	return r.query(`ORDER BY ` + column + ` ` + direction + `, id ASC`)
}

func (r *Repo) query(orderBy string) ([]Submission, error) {
	rows, err := r.db.Query(
		`SELECT ` + submissionColumns + ` FROM contact_submissions ` + orderBy + `;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Submission
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}
