package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/httpserver"
	"pixabros/internal/render"
	"pixabros/internal/storage"
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

	files := storage.NewLocalDisk(cfg.DataDir+"/media", "/media")
	renderedFiles := storage.NewLocalDisk(cfg.DataDir+"/rendered-store", "/rendered")
	store := render.NewStore(conn, renderedFiles)
	registry := render.NewRegistry()

	worker := render.NewWorker(conn, registry, store, 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		AssetsDir:  cfg.DataDir + "/assets",
	})

	_ = files // wired for media/game uploads by the per-module phases that register their handlers here

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
