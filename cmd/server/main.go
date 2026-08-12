package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pixabros/internal/auth"
	"pixabros/internal/awards"
	"pixabros/internal/config"
	"pixabros/internal/db"
	"pixabros/internal/games"
	"pixabros/internal/httpserver"
	"pixabros/internal/media"
	"pixabros/internal/members"
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

	for _, dir := range []string{
		cfg.DataDir + "/admin-dist",
		cfg.DataDir + "/games",
		cfg.DataDir + "/assets",
		cfg.DataDir + "/media",
		cfg.DataDir + "/rendered-store",
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	renderedFiles := storage.NewLocalDisk(cfg.DataDir+"/rendered-store", "/rendered")
	// mediaapi's storage keys already begin with "media/", so the root is the
	// bare data dir and the base URL is empty: that yields both the on-disk
	// path <data>/media/<target>/<name>.webp and the public URL
	// /media/<target>/<name>.webp with no duplicated segment.
	mediaFiles := storage.NewLocalDisk(cfg.DataDir, "")
	mediaRepo := media.NewRepo(conn)
	membersRepo := members.NewRepo(conn)
	awardsRepo := awards.NewRepo(conn)
	store := render.NewStore(conn, renderedFiles)
	registry := render.NewRegistry()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// All registry population must happen before this point: Registry's mutex
	// makes concurrent access safe, but registration stays a startup-only
	// phase by convention.
	worker := render.NewWorker(conn, registry, store, 2*time.Second, render.WithErrorLogger(func(err error) {
		log.Printf("render worker: %v", err)
	}))
	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()

	handler := httpserver.New(httpserver.Dependencies{
		Admins:     auth.NewAdminRepo(conn),
		Sessions:   auth.NewSessionStore(conn),
		Store:      store,
		Files:      renderedFiles,
		DB:         conn,
		Games:      games.NewRepo(conn),
		Members:    membersRepo,
		Awards:     awardsRepo,
		Media:      mediaRepo,
		MediaFiles: mediaFiles,
		MediaDir:   cfg.DataDir + "/media",
		AdminUIDir: cfg.DataDir + "/admin-dist",
		PlayDir:    cfg.DataDir + "/games",
		AssetsDir:  cfg.DataDir + "/assets",
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}

	// Join the worker so no render is still running against conn when the
	// deferred conn.Close() fires.
	<-workerDone
}
