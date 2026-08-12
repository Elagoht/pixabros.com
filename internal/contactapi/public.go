package contactapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"pixabros/internal/contact"
	"pixabros/internal/httpapi"
)

// MinMessageLength is the floor the architecture spec sets for a submission.
// It exists to make drive-by spam more effort than it is worth.
const MinMessageLength = 100

// HoneypotField is a form input a human never sees and never fills in. A bot
// that fills every field gives itself away. It is exported so the page that
// renders the form and the handler that checks it cannot drift apart.
const HoneypotField = "website"

// RateLimit is how often one address may submit. The spec asks for roughly one
// a minute, which is far more than a person needs and far less than a script
// wants.
const RateLimit = time.Minute

// PublicHandlers serves the contact form. It is the only public write endpoint
// on the site, which is why it carries a rate limit and a honeypot of its own
// rather than relying on the session middleware everything else sits behind.
type PublicHandlers struct {
	repo    *contact.Repo
	limiter *rateLimiter
	// now is injectable so the rate limit can be tested without sleeping.
	now func() time.Time
}

func NewPublicHandlers(repo *contact.Repo) *PublicHandlers {
	return &PublicHandlers{
		repo:    repo,
		limiter: newRateLimiter(),
		now:     time.Now,
	}
}

type submitRequest struct {
	Name          string
	Subject       string
	Phone         string
	Email         string
	Message       string
	WantsCallback bool
	Honeypot      string
}

// Submit accepts a contact form submission.
//
// It answers both a JSON fetch and a plain form post, so the page works with
// and without JavaScript.
func (h *PublicHandlers) Submit(w http.ResponseWriter, r *http.Request) {
	req, wantsHTML, err := parseSubmission(r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "could not read the form")
		return
	}

	// A filled honeypot is answered exactly like a success. Telling a bot it
	// was caught only teaches whoever wrote it.
	if strings.TrimSpace(req.Honeypot) != "" {
		h.respondOK(w, r, wantsHTML)
		return
	}

	if code, message, ok := validateSubmission(req); !ok {
		if wantsHTML {
			// Without JavaScript there is nowhere to show a field error, so the
			// visitor goes back to the form rather than landing on raw JSON.
			http.Redirect(w, r, "/contact", http.StatusSeeOther)
			return
		}
		httpapi.WriteError(w, http.StatusBadRequest, code, message)
		return
	}

	ip := clientIP(r)
	if !h.limiter.allow(ip, h.now(), RateLimit) {
		httpapi.WriteError(w, http.StatusTooManyRequests, "rate_limited",
			"please wait a moment before sending another message")
		return
	}

	if _, err := h.repo.Create(contact.CreateInput{
		Name:          strings.TrimSpace(req.Name),
		Subject:       strings.TrimSpace(req.Subject),
		Phone:         strings.TrimSpace(req.Phone),
		Email:         strings.TrimSpace(req.Email),
		Message:       strings.TrimSpace(req.Message),
		WantsCallback: req.WantsCallback,
		IPAddress:     ip,
	}); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not store the message")
		return
	}

	h.respondOK(w, r, wantsHTML)
}

func (h *PublicHandlers) respondOK(w http.ResponseWriter, r *http.Request, wantsHTML bool) {
	if wantsHTML {
		http.Redirect(w, r, "/contact/sent", http.StatusSeeOther)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// parseSubmission reads either a JSON body or a form post. The second return
// value reports whether the caller is a browser navigating a form, which
// decides whether the response is JSON or a redirect.
func parseSubmission(r *http.Request) (submitRequest, bool, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var body struct {
			Name          string `json:"name"`
			Subject       string `json:"subject"`
			Phone         string `json:"phone"`
			Email         string `json:"email"`
			Message       string `json:"message"`
			WantsCallback bool   `json:"wants_callback"`
			Honeypot      string `json:"website"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return submitRequest{}, false, err
		}
		return submitRequest(body), false, nil
	}

	if err := r.ParseForm(); err != nil {
		return submitRequest{}, true, err
	}
	return submitRequest{
		Name:          r.PostFormValue("name"),
		Subject:       r.PostFormValue("subject"),
		Phone:         r.PostFormValue("phone"),
		Email:         r.PostFormValue("email"),
		Message:       r.PostFormValue("message"),
		WantsCallback: r.PostFormValue("wants_callback") != "",
		Honeypot:      r.PostFormValue(HoneypotField),
	}, true, nil
}

func validateSubmission(req submitRequest) (code, message string, ok bool) {
	if strings.TrimSpace(req.Subject) == "" {
		return "missing_subject", "a subject is required", false
	}
	if len([]rune(strings.TrimSpace(req.Message))) < MinMessageLength {
		return "message_too_short", "the message must be at least 100 characters", false
	}
	// Asking for a call back with no way to be reached cannot be honoured.
	if req.WantsCallback &&
		strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Email) == "" {
		return "missing_contact", "a phone number or email address is needed for a call back", false
	}
	return "", "", true
}

// clientIP takes the address Cloudflare forwards, falling back to the socket's
// own. It is used for rate limiting, not for trust.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("CF-Connecting-IP"); forwarded != "" {
		return forwarded
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first, _, found := strings.Cut(forwarded, ","); found {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimiter remembers when each address last got through. It is in-memory on
// purpose: the limit only needs to survive as long as the process, and a
// database write per rejected bot would defeat the point.
type rateLimiter struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{last: map[string]time.Time{}}
}

func (l *rateLimiter) allow(key string, now time.Time, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if seen, ok := l.last[key]; ok && now.Sub(seen) < window {
		return false
	}

	// Expired entries are dropped as we go, so a long-running process does not
	// accumulate one map entry per address that ever visited.
	for addr, seen := range l.last {
		if now.Sub(seen) >= window {
			delete(l.last, addr)
		}
	}

	l.last[key] = now
	return true
}
