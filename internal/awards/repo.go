package awards

import (
	"database/sql"
	"errors"
	"sort"
	"time"

	"pixabros/internal/dbutil"
	"pixabros/internal/id"
)

var ErrAwardNotFound = errors.New("award not found")

var ErrInvalidSort = errors.New("unknown sort field or direction")

type Award struct {
	ID        string
	Title     string
	Issuer    string
	Date      string
	PictureID *string
	GameID    *string
	Link      string
	CreatedAt time.Time
}

type CreateInput struct {
	Title  string
	Issuer string
	Date   string
	Link   string
}

type UpdateInput struct {
	Title     string
	Issuer    string
	Date      string
	Link      string
	PictureID *string
	GameID    *string
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const awardColumns = `id, title, issuer, date, picture_id, game_id, link, created_at`

// sortableColumns whitelists what the admin list may be ordered by. Anything
// interpolated into ORDER BY must come from this map and nowhere else.
var sortableColumns = map[string]string{
	"title":      "title COLLATE NOCASE",
	"issuer":     "issuer COLLATE NOCASE",
	"date":       "date",
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

func scanAward(row rowScanner) (Award, error) {
	var a Award
	var pictureID, gameID, link sql.NullString
	var createdAtStr string

	err := row.Scan(
		&a.ID, &a.Title, &a.Issuer, &a.Date,
		&pictureID, &gameID, &link, &createdAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Award{}, ErrAwardNotFound
	}
	if err != nil {
		return Award{}, err
	}

	if pictureID.Valid {
		value := pictureID.String
		a.PictureID = &value
	}
	if gameID.Valid {
		value := gameID.String
		a.GameID = &value
	}
	if link.Valid {
		a.Link = link.String
	}

	if a.CreatedAt, err = dbutil.ParseTimestamp(createdAtStr); err != nil {
		return Award{}, err
	}
	return a, nil
}

// Create takes no picture or game: like a game's artwork, media is attached
// once the row exists.
func (r *Repo) Create(input CreateInput) (Award, error) {
	newID := id.New()
	if _, err := r.db.Exec(
		`INSERT INTO awards (id, title, issuer, date, link) VALUES (?, ?, ?, ?, ?);`,
		newID, input.Title, input.Issuer, input.Date, dbutil.NullableString(input.Link),
	); err != nil {
		return Award{}, err
	}
	return r.FindByID(newID)
}

func (r *Repo) FindByID(awardID string) (Award, error) {
	row := r.db.QueryRow(`SELECT `+awardColumns+` FROM awards WHERE id = ?;`, awardID)
	return scanAward(row)
}

func (r *Repo) Update(awardID string, input UpdateInput) (Award, error) {
	res, err := r.db.Exec(
		`UPDATE awards SET title = ?, issuer = ?, date = ?, link = ?,
			picture_id = ?, game_id = ?
		WHERE id = ?;`,
		input.Title, input.Issuer, input.Date, dbutil.NullableString(input.Link),
		dbutil.NullableID(input.PictureID), dbutil.NullableID(input.GameID), awardID,
	)
	if err != nil {
		return Award{}, err
	}
	if err := dbutil.RequireRows(res, ErrAwardNotFound); err != nil {
		return Award{}, err
	}
	return r.FindByID(awardID)
}

func (r *Repo) Delete(awardID string) error {
	res, err := r.db.Exec(`DELETE FROM awards WHERE id = ?;`, awardID)
	if err != nil {
		return err
	}
	return dbutil.RequireRows(res, ErrAwardNotFound)
}

// List returns every award ordered by the given field and direction. Awards
// have no manual display order -- they are shown as a timeline -- so the
// default is the most recent first. id is the final tiebreaker so awards
// sharing a date keep a stable order between requests.
func (r *Repo) List(field string, descending bool) ([]Award, error) {
	if field == "" {
		return r.query(`ORDER BY date DESC, id ASC`)
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

func (r *Repo) query(orderBy string) ([]Award, error) {
	rows, err := r.db.Query(`SELECT ` + awardColumns + ` FROM awards ` + orderBy + `;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Award
	for rows.Next() {
		a, err := scanAward(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
