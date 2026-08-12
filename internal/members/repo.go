package members

import (
	"database/sql"
	"errors"
	"sort"
	"time"

	"pixabros/internal/dbutil"
	"pixabros/internal/id"
)

var ErrMemberNotFound = errors.New("member not found")

var ErrInvalidSort = errors.New("unknown sort field or direction")

type Member struct {
	Name         string
	Description  string
	Tags         string
	LinksJSON    string
	ID           string
	AvatarID     *string
	DisplayOrder int
	IsPublished  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateInput struct {
	Name         string
	Description  string
	Tags         string
	LinksJSON    string
	DisplayOrder int
	IsPublished  bool
}

type UpdateInput struct {
	Name         string
	Description  string
	Tags         string
	LinksJSON    string
	AvatarID     *string
	DisplayOrder int
	IsPublished  bool
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const memberColumns = `id, name, avatar_id, tags, description, links_json,
	display_order, is_published, created_at, updated_at`

// sortableColumns whitelists what the admin list may be ordered by. Anything
// interpolated into ORDER BY must come from this map and nowhere else.
var sortableColumns = map[string]string{
	"name":          "name COLLATE NOCASE",
	"tags":          "tags COLLATE NOCASE",
	"is_published":  "is_published",
	"display_order": "display_order",
	"created_at":    "created_at",
	"updated_at":    "updated_at",
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

func scanMember(row rowScanner) (Member, error) {
	var m Member
	var avatarID sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&m.ID, &m.Name, &avatarID, &m.Tags, &m.Description, &m.LinksJSON,
		&m.DisplayOrder, &m.IsPublished, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrMemberNotFound
	}
	if err != nil {
		return Member{}, err
	}
	if avatarID.Valid {
		value := avatarID.String
		m.AvatarID = &value
	}

	if m.CreatedAt, err = dbutil.ParseTimestamp(createdAtStr); err != nil {
		return Member{}, err
	}
	if m.UpdatedAt, err = dbutil.ParseTimestamp(updatedAtStr); err != nil {
		return Member{}, err
	}
	return m, nil
}

func (r *Repo) Create(input CreateInput) (Member, error) {
	links := input.LinksJSON
	if links == "" {
		links = "[]"
	}

	newID := id.New()
	if _, err := r.db.Exec(
		`INSERT INTO members (id, name, tags, description, links_json, display_order, is_published)
		 VALUES (?, ?, ?, ?, ?, ?, ?);`,
		newID, input.Name, input.Tags, input.Description, links,
		input.DisplayOrder, input.IsPublished,
	); err != nil {
		return Member{}, err
	}
	return r.FindByID(newID)
}

func (r *Repo) FindByID(memberID string) (Member, error) {
	row := r.db.QueryRow(`SELECT `+memberColumns+` FROM members WHERE id = ?;`, memberID)
	return scanMember(row)
}

func (r *Repo) Update(memberID string, input UpdateInput) (Member, error) {
	links := input.LinksJSON
	if links == "" {
		links = "[]"
	}

	res, err := r.db.Exec(
		`UPDATE members SET
			name = ?, avatar_id = ?, tags = ?, description = ?, links_json = ?,
			display_order = ?, is_published = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		input.Name, dbutil.NullableID(input.AvatarID), input.Tags, input.Description, links,
		input.DisplayOrder, input.IsPublished, memberID,
	)
	if err != nil {
		return Member{}, err
	}
	if err := dbutil.RequireRows(res, ErrMemberNotFound); err != nil {
		return Member{}, err
	}
	return r.FindByID(memberID)
}

func (r *Repo) Delete(memberID string) error {
	res, err := r.db.Exec(`DELETE FROM members WHERE id = ?;`, memberID)
	if err != nil {
		return err
	}
	return dbutil.RequireRows(res, ErrMemberNotFound)
}

// List returns every member ordered by the given field and direction. An
// empty field means the manual display order that the drag-to-reorder
// control writes. id is always the final tiebreaker so rows sharing a value
// keep a stable order between requests.
func (r *Repo) List(field string, descending bool) ([]Member, error) {
	if field == "" {
		field = "display_order"
	}
	column, ok := sortableColumns[field]
	if !ok {
		return nil, ErrInvalidSort
	}
	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	rows, err := r.db.Query(
		`SELECT ` + memberColumns + ` FROM members ORDER BY ` + column + ` ` + direction + `, id ASC;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// Reorder sets display_order to each id's index in ids, in one transaction.
// The admin UI always sends the complete ordered list, so an unknown id is a
// caller bug -- the whole reorder is rolled back rather than half-applied.
func (r *Repo) Reorder(ids []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`UPDATE members SET display_order = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for index, memberID := range ids {
		res, err := stmt.Exec(index, memberID)
		if err != nil {
			return err
		}
		if err := dbutil.RequireRows(res, ErrMemberNotFound); err != nil {
			return err
		}
	}

	return tx.Commit()
}
