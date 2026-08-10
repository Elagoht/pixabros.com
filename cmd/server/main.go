package main

import (
	"log"
	"net/http"
	"os"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/httpserver"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	for _, dir := range []string{cfg.DataDir + "/admin-dist", cfg.DataDir + "/games", cfg.DataDir + "/rendered"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("create static dir %s: %v", dir, err)
		}
	}

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		PublicDir:  cfg.DataDir + "/rendered",
	})

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
