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
	"pixabros/internal/contact"
	"pixabros/internal/db"
	"pixabros/internal/devlog"
	"pixabros/internal/games"
	"pixabros/internal/httpserver"
	"pixabros/internal/media"
	"pixabros/internal/mediarefs"
	"pixabros/internal/members"
	"pixabros/internal/render"
	"pixabros/internal/settings"
	"pixabros/internal/site"
	"pixabros/internal/stats"
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
	devlogRepo := devlog.NewRepo(conn)
	contactRepo := contact.NewRepo(conn)
	settingsRepo := settings.NewRepo(conn)
	store := render.NewStore(conn, renderedFiles)
	registry := render.NewRegistry()

	// The public site: publish its stylesheet, register a renderer per page,
	// and render the 404 body once. All three are startup misconfigurations if
	// they fail -- a server running without CSS or without any public page is
	// worse than one that refuses to start.
	assets, err := site.Build(cfg.DataDir + "/assets")
	if err != nil {
		log.Fatalf("build site assets: %v", err)
	}
	publicSite, err := site.New(conn, assets)
	if err != nil {
		log.Fatalf("build public site: %v", err)
	}
	publicSite.Register(registry)

	notFoundBody, err := publicSite.NotFoundBody()
	if err != nil {
		log.Fatalf("render 404 page: %v", err)
	}

	// Nothing else creates a page for the first time: the regen worker only
	// resolves a tag to pages that already declared it, so without this every
	// public URL would stay 404 forever.
	reconciler := site.NewReconciler(publicSite.DesiredPages, store, registry,
		site.WithReconcileErrorLogger(func(err error) {
			log.Printf("site reconcile: %v", err)
		}),
	)
	if rendered, removed, err := reconciler.Reconcile(); err != nil {
		log.Printf("site reconcile: %v", err)
	} else if rendered > 0 || removed > 0 {
		log.Printf("site reconcile: rendered %d page(s), removed %d", rendered, removed)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// All registry population must happen before this point: Registry's mutex
	// makes concurrent access safe, but registration stays a startup-only
	// phase by convention.
	// Reconciliation also runs after each batch of jobs: a content change always
	// enqueues a tag, which is the signal that a page may need creating or
	// retiring.
	worker := render.NewWorker(conn, registry, store, 2*time.Second,
		render.WithErrorLogger(func(err error) {
			log.Printf("render worker: %v", err)
		}),
		render.WithAfterBatch(func(processed int) {
			if processed == 0 {
				return
			}
			if rendered, removed, err := reconciler.Reconcile(); err != nil {
				log.Printf("site reconcile: %v", err)
			} else if rendered > 0 || removed > 0 {
				log.Printf("site reconcile: rendered %d page(s), removed %d", rendered, removed)
			}
		}),
	)
	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()

	// Nothing reclaimed orphaned media before this: images left behind by a
	// deleted game or replaced artwork accumulated on disk forever. The lookup
	// comes from mediarefs, the one place that knows every media reference.
	sweeper := media.NewSweeper(
		mediaRepo,
		mediaFiles,
		func() (map[string]bool, error) { return mediarefs.ReferencedIDs(conn) },
		6*time.Hour,
		media.WithSweepLogger(func(deleted int) {
			log.Printf("media sweep: removed %d orphaned image(s)", deleted)
		}),
		media.WithSweepErrorLogger(func(err error) {
			log.Printf("media sweep: %v", err)
		}),
	)
	sweeperDone := make(chan struct{})
	go func() {
		sweeper.Run(ctx)
		close(sweeperDone)
	}()

	handler := httpserver.New(httpserver.Dependencies{
		NotFoundBody: notFoundBody,
		Admins:       auth.NewAdminRepo(conn),
		Sessions:     auth.NewSessionStore(conn),
		Store:        store,
		Files:        renderedFiles,
		DB:           conn,
		Games:        games.NewRepo(conn),
		Members:      membersRepo,
		Awards:       awardsRepo,
		Devlog:       devlogRepo,
		Contact:      contactRepo,
		Stats:        stats.NewRepo(conn),
		Settings:     settingsRepo,
		Media:        mediaRepo,
		MediaFiles:   mediaFiles,
		MediaDir:     cfg.DataDir + "/media",
		AdminUIDir:   cfg.DataDir + "/admin-dist",
		PlayDir:      cfg.DataDir + "/games",
		AssetsDir:    cfg.DataDir + "/assets",
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

	// Join both background goroutines so neither is still using conn when the
	// deferred conn.Close() fires. The sweeper deletes rows and files, so
	// letting it be cut off mid-sweep is exactly what should not happen.
	<-workerDone
	<-sweeperDone
}
