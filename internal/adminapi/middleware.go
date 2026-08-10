package adminapi

import (
	"context"
	"net/http"

	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
)

const sessionCookieName = "pixabros_session"

type contextKey string

const adminIDContextKey contextKey = "adminID"

func RequireSession(sessions *auth.SessionStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
			return
		}
		session, err := sessions.Validate(cookie.Value)
		if err != nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "session expired or invalid")
			return
		}
		next(w, r.WithContext(withAdminID(r.Context(), session.AdminID)))
	}
}

func AdminIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(adminIDContextKey).(int64)
	return id, ok
}

func withAdminID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, adminIDContextKey, id)
}
