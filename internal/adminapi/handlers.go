package adminapi

import (
	"encoding/json"
	"net/http"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/httpapi"
)

type AuthHandlers struct {
	admins   *auth.AdminRepo
	sessions *auth.SessionStore
}

func NewAuthHandlers(admins *auth.AdminRepo, sessions *auth.SessionStore) *AuthHandlers {
	return &AuthHandlers{admins: admins, sessions: sessions}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10) // 64 KiB
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "missing_fields", "username and password are required")
		return
	}

	admin, err := h.admins.FindByUsername(req.Username)
	if err != nil || !auth.VerifyPassword(admin.PasswordHash, req.Password) {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		return
	}

	token, expiresAt, err := h.sessions.Create(admin.ID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
	})
	httpapi.WriteJSON(w, http.StatusOK, loginResponse{Username: admin.Username})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.sessions.Delete(cookie.Value); err != nil {
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not log out")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	adminID, ok := AdminIDFromContext(r.Context())
	if !ok {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10) // 64 KiB
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}

	admin, err := h.admins.FindByID(adminID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load admin")
		return
	}
	if !auth.VerifyPassword(admin.PasswordHash, req.CurrentPassword) {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not hash password")
		return
	}
	if err := h.admins.UpdatePasswordHash(adminID, newHash); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not update password")
		return
	}
	if err := h.sessions.DeleteAllForAdmin(adminID); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not invalidate sessions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type whoamiResponse struct {
	Username string `json:"username"`
}

func (h *AuthHandlers) Whoami(w http.ResponseWriter, r *http.Request) {
	adminID, ok := AdminIDFromContext(r.Context())
	if !ok {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}

	admin, err := h.admins.FindByID(adminID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load admin")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, whoamiResponse{Username: admin.Username})
}
