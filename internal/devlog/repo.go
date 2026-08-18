package devlog

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"pixabros/internal/dbutil"
	"pixabros/internal/id"
	"pixabros/internal/slug"
)

var ErrPostNotFound = errors.New("devlog post not found")

var ErrInvalidSort = errors.New("unknown sort field or direction")

const table = "devlog_posts"

type Post struct {
	ID              string
	Slug            string
	Title           string
	ContentMarkdown string
	GameID          *string
	OGImageID       *string
	IsPublished     bool
	// PublishedAt is empty until the post is first published. It is stored as
	// YYYY-MM-DD, the same shape the date picker submits.
	PublishedAt string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateInput struct {
	Title           string
	ContentMarkdown string
	IsPublished     bool
	PublishedAt     string
}

type UpdateInput struct {
	Title           string
	ContentMarkdown string
	GameID          *string
	OGImageID       *string
	IsPublished     bool
	PublishedAt     string
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

const postColumns = `id, slug, title, content_markdown, game_id, og_image_id,
	is_published, published_at, created_at, updated_at`

// sortableColumns whitelists what the admin list may be ordered by. Anything
// interpolated into ORDER BY must come from this map and nowhere else.
var sortableColumns = map[string]string{
	"title":        "title COLLATE NOCASE",
	"is_published": "is_published",
	"published_at": "published_at",
	"created_at":   "created_at",
	"updated_at":   "updated_at",
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

func scanPost(row rowScanner) (Post, error) {
	var p Post
	var gameID, ogImageID, publishedAt sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&p.ID, &p.Slug, &p.Title, &p.ContentMarkdown, &gameID, &ogImageID,
		&p.IsPublished, &publishedAt, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, ErrPostNotFound
	}
	if err != nil {
		return Post{}, err
	}

	if gameID.Valid {
		value := gameID.String
		p.GameID = &value
	}
	if ogImageID.Valid {
		value := ogImageID.String
		p.OGImageID = &value
	}
	if publishedAt.Valid {
		p.PublishedAt = publishedAt.String
	}

	if p.CreatedAt, err = dbutil.ParseTimestamp(createdAtStr); err != nil {
		return Post{}, err
	}
	if p.UpdatedAt, err = dbutil.ParseTimestamp(updatedAtStr); err != nil {
		return Post{}, err
	}
	return p, nil
}

// today is the publication date stamped when a post is first published. It is
// a var so tests can pin it.
var today = func() string { return time.Now().UTC().Format("2006-01-02") }

// resolvePublishedAt implements "automatic on first publish, editable
// afterwards": an explicit date always wins, otherwise a post being published
// for the first time is stamped with today, and anything else keeps whatever
// it already had. Unpublishing deliberately preserves the date so that
// re-publishing does not silently re-date the post.
func resolvePublishedAt(requested string, isPublished bool, existing string) string {
	if requested != "" {
		return requested
	}
	if isPublished && existing == "" {
		return today()
	}
	return existing
}

func (r *Repo) Create(input CreateInput) (Post, error) {
	newSlug, err := slug.Unique(r.db, table, slug.Make(input.Title, "post"), "")
	if err != nil {
		return Post{}, err
	}

	newID := id.New()
	publishedAt := resolvePublishedAt(input.PublishedAt, input.IsPublished, "")

	if _, err := r.db.Exec(
		`INSERT INTO `+table+` (id, slug, title, content_markdown, is_published, published_at)
		 VALUES (?, ?, ?, ?, ?, ?);`,
		newID, newSlug, input.Title, input.ContentMarkdown, input.IsPublished,
		dbutil.NullableString(publishedAt),
	); err != nil {
		return Post{}, err
	}
	return r.FindByID(newID)
}

func (r *Repo) FindByID(postID string) (Post, error) {
	row := r.db.QueryRow(`SELECT `+postColumns+` FROM `+table+` WHERE id = ?;`, postID)
	return scanPost(row)
}

func (r *Repo) FindBySlug(postSlug string) (Post, error) {
	row := r.db.QueryRow(`SELECT `+postColumns+` FROM `+table+` WHERE slug = ?;`, postSlug)
	return scanPost(row)
}

// Update regenerates the slug from the title, published or not, the same way a
// game's does. A renamed post whose URL still said the old name would be the
// odder outcome, and the admin API resolves a post by id as readily as by slug,
// so nothing inside the panel depends on the slug holding still.
func (r *Repo) Update(postID string, input UpdateInput) (Post, error) {
	existing, err := r.FindByID(postID)
	if err != nil {
		return Post{}, err
	}

	nextSlug, err := slug.Unique(r.db, table, slug.Make(input.Title, "post"), postID)
	if err != nil {
		return Post{}, err
	}

	publishedAt := resolvePublishedAt(input.PublishedAt, input.IsPublished, existing.PublishedAt)

	res, err := r.db.Exec(
		`UPDATE `+table+` SET
			slug = ?, title = ?, content_markdown = ?, game_id = ?, og_image_id = ?,
			is_published = ?, published_at = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?;`,
		nextSlug, input.Title, input.ContentMarkdown,
		dbutil.NullableID(input.GameID), dbutil.NullableID(input.OGImageID),
		input.IsPublished, dbutil.NullableString(publishedAt), postID,
	)
	if err != nil {
		return Post{}, err
	}
	if err := dbutil.RequireRows(res, ErrPostNotFound); err != nil {
		return Post{}, err
	}
	return r.FindByID(postID)
}

func (r *Repo) Delete(postID string) error {
	res, err := r.db.Exec(`DELETE FROM `+table+` WHERE id = ?;`, postID)
	if err != nil {
		return err
	}
	return dbutil.RequireRows(res, ErrPostNotFound)
}

// List returns every post ordered by the given field and direction. With no
// field chosen the newest post comes first, falling back to the creation time
// for drafts so an unpublished post sorts by when it was written rather than
// sinking below everything that has a date.
func (r *Repo) List(field string, descending bool) ([]Post, error) {
	if field == "" {
		return r.query(`ORDER BY COALESCE(published_at, date(created_at)) DESC, id ASC`)
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

// SearchInput narrows the public devlog index. Zero values mean "no filter".
type SearchInput struct {
	// Query is a case-insensitive substring of the title, the same match the
	// old client-side filter made against the rendered row text.
	Query string
	// Game is the slug of the game a post must be about.
	Game string
	// Year is the four-digit year of the publication date.
	Year  string
	Limit int
	// Offset is where the page starts; page N is Offset = (N-1)*Limit.
	Offset int
}

type SearchResult struct {
	Posts []Post
	Total int
}

// Search returns one page of published posts, newest first, and how many match
// the same filters in total so a client can decide whether another page exists.
func (r *Repo) Search(input SearchInput) (SearchResult, error) {
	where, args := searchWhere(input)
	orderBy := `ORDER BY COALESCE(published_at, date(created_at)) DESC, id ASC`

	// A zero limit is the zero value, not "no rows": a caller who only
	// filters still wants the matching page. 50 is a page nobody would
	// actually ask for, so it only ever appears when Limit was forgotten.
	limit := input.Limit
	if limit < 1 {
		limit = 50
	}

	rows, err := r.db.Query(`SELECT `+postColumns+` FROM `+table+` `+where+` `+orderBy+`
		LIMIT ? OFFSET ?;`, append(args, limit, input.Offset)...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return SearchResult{}, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return SearchResult{}, err
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM `+table+` `+where+`;`, args...).Scan(&total); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Posts: posts, Total: total}, nil
}

// searchWhere builds the shared WHERE clause for Search's list and count
// queries. lower() only folds ASCII, which is what the titles of these posts
// are expected to carry; the clientside filter that used to do this work ran
// the same toLowerCase.
func searchWhere(input SearchInput) (string, []any) {
	clauses := []string{"is_published = 1"}
	var args []any
	if input.Query != "" {
		clauses = append(clauses, "instr(lower(title), lower(?)) > 0")
		args = append(args, input.Query)
	}
	if input.Game != "" {
		clauses = append(clauses, "game_id IN (SELECT id FROM games WHERE slug = ?)")
		args = append(args, input.Game)
	}
	if input.Year != "" {
		clauses = append(clauses, "substr(published_at, 1, 4) = ?")
		args = append(args, input.Year)
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (r *Repo) query(orderBy string) ([]Post, error) {
	rows, err := r.db.Query(`SELECT ` + postColumns + ` FROM ` + table + ` ` + orderBy + `;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
