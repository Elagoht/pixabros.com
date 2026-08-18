package devlogapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"pixabros/internal/devlog"
	"pixabros/internal/games"
	"pixabros/internal/httpapi"
	"pixabros/internal/media"
)

// PublicHandlers serve the devlog index the page's search and pagination read.
// Nothing here requires a session: it is the same data the rendered page
// already carries, in the shape a script can append to.
//
// Only our own origin may read it. The search box lives on the rendered page,
// so every legitimate request is same-origin; anything that names a different
// origin is either a scraper or a third-party embed, and both are refused.
type PublicHandlers struct {
	repo   *devlog.Repo
	games  *games.Repo
	media  *media.Repo
	origin string
	cache  *searchCache
}

func NewPublicHandlers(repo *devlog.Repo, games *games.Repo, media *media.Repo, origin string, cache *searchCache) *PublicHandlers {
	return &PublicHandlers{repo: repo, games: games, media: media, origin: origin, cache: cache}
}

// searchPost is one row of the public index, light enough for the client to
// rebuild the row markup it already renders. No body is carried: a search is
// for titles, and shipping every post's markdown with each page would defeat
// the cache it sits behind.
type searchPost struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Date     string `json:"date"`
	Year     string `json:"year"`
	Game     string `json:"game,omitempty"`
	GameSlug string `json:"game_slug,omitempty"`
	Image    string `json:"image,omitempty"`
}

type searchResponse struct {
	Posts   []searchPost `json:"posts"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Total   int          `json:"total"`
	HasMore bool         `json:"has_more"`
}

// maxPerPage caps how many posts one response may ask for. The page's own
// "load more" steps a page at a time, so a larger request is a caller mistake
// rather than a feature anyone needs.
const maxPerPage = 50

// SearchPosts answers /api/devlog/posts.
//
// q, game and year narrow the results, page (1-based) and per_page bound them.
// Responses are cached by the exact query, so repeated requests -- every
// keystroke of a debounced search does hit the API again -- are answered from
// memory until a post changes.
func (h *PublicHandlers) SearchPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	if origin := r.Header.Get("Origin"); origin != "" && h.origin != "" && origin != h.origin {
		httpapi.WriteError(w, http.StatusForbidden, "forbidden", "cross-origin requests are not allowed")
		return
	}

	query := r.URL.Query()
	q := strings.ToLower(strings.TrimSpace(query.Get("q")))
	game := strings.TrimSpace(query.Get("game"))
	year := strings.TrimSpace(query.Get("year"))
	page := positiveInt(query.Get("page"), 1)
	perPage := positiveInt(query.Get("per_page"), 10)
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	key := strings.Join([]string{q, game, year, strconv.Itoa(page), strconv.Itoa(perPage)}, "\x00")
	if body, ok := h.cache.Get(key); ok {
		h.writeBody(w, body)
		return
	}

	result, err := h.repo.Search(devlog.SearchInput{
		Query:  q,
		Game:   game,
		Year:   year,
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	})
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not search devlog posts")
		return
	}

	gameNames, err := h.gameNames()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not search devlog posts")
		return
	}
	images, err := h.mediaByID()
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not search devlog posts")
		return
	}

	response := searchResponse{
		Page:    page,
		PerPage: perPage,
		Total:   result.Total,
		HasMore: page*perPage < result.Total,
		Posts:   make([]searchPost, 0, len(result.Posts)),
	}
	for _, post := range result.Posts {
		item := searchPost{
			Slug:  post.Slug,
			Title: post.Title,
			Date:  post.PublishedAt,
			Year:  yearOf(post.PublishedAt),
		}
		if post.GameID != nil {
			if game, ok := gameNames[*post.GameID]; ok {
				item.Game = game.Title
				item.GameSlug = game.Slug
			}
		}
		if post.OGImageID != nil {
			if image, ok := images[*post.OGImageID]; ok {
				item.Image = "/" + image.Path
			}
		}
		response.Posts = append(response.Posts, item)
	}

	body, err := json.Marshal(response)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not search devlog posts")
		return
	}
	h.cache.Put(key, body)
	h.writeBody(w, body)
}

func (h *PublicHandlers) writeBody(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// gameNames resolves the game a post is about, in one query rather than one
// per post. Only published games are listed: a draft's row would name a game
// whose page does not exist.
func (h *PublicHandlers) gameNames() (map[string]struct {
	Title string
	Slug  string
}, error) {
	all, err := h.games.List("display_order", false)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]struct {
		Title string
		Slug  string
	}, len(all))
	for _, game := range all {
		if !game.IsPublished {
			continue
		}
		byID[game.ID] = struct {
			Title string
			Slug  string
		}{Title: game.Title, Slug: game.Slug}
	}
	return byID, nil
}

// mediaByID indexes every image by id, for resolving a post's preview.
func (h *PublicHandlers) mediaByID() (map[string]media.Media, error) {
	all, err := h.media.List()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]media.Media, len(all))
	for _, item := range all {
		byID[item.ID] = item.Media
	}
	return byID, nil
}

// positiveInt parses a query parameter as a positive integer, falling back to
// fallback when it is empty or not a number. A zero or negative page is the
// caller asking for nonsense, and clamps to the fallback rather than echoing
// the mistake back as a page number.
func positiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

// yearOf is the four-digit year of a YYYY-MM-DD publication date.
func yearOf(date string) string {
	if len(date) < 4 {
		return ""
	}
	return date[:4]
}

// AllowedOriginOf derives the origin a request must carry to be answered: the
// scheme and host of the site's own URL. An empty or unparseable URL yields an
// empty origin, which the handler treats as "no restriction": the endpoint is
// never handed a URL it could not turn into a check.
func AllowedOriginOf(siteURL string) (string, error) {
	if siteURL == "" {
		return "", nil
	}
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", &url.Error{Op: "parse", URL: siteURL, Err: err}
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}