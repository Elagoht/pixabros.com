package httpserver

import (
	"net/http"

	"pixabros/internal/adminapi"
	"pixabros/internal/auth"
)

type Dependencies struct {
	Admins     *auth.AdminRepo
	Sessions   *auth.SessionStore
	AdminUIDir string
	PlayDir    string
	PublicDir  string
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))

	mux.Handle("/I-am-a-pixabro/", http.StripPrefix("/I-am-a-pixabro/", http.FileServer(http.Dir(deps.AdminUIDir))))
	mux.Handle("/play/", http.StripPrefix("/play/", http.FileServer(http.Dir(deps.PlayDir))))
	mux.Handle("/", http.FileServer(http.Dir(deps.PublicDir)))

	return mux
}
