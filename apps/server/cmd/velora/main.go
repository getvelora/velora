package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/getvelora/velora/apps/server/internal/database"
	"github.com/getvelora/velora/apps/server/internal/env"
	"github.com/getvelora/velora/apps/server/internal/health"
	"github.com/getvelora/velora/apps/server/internal/libraries"
	"github.com/getvelora/velora/apps/server/internal/mediafiles"
	"github.com/getvelora/velora/apps/server/internal/migrations"
	"github.com/getvelora/velora/apps/server/internal/storage"
	"github.com/getvelora/velora/apps/server/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runtimePaths := storage.RuntimePathsFromEnv()
	if err := storage.EnsureRuntimeDirectories(runtimePaths); err != nil {
		log.Fatalf("create runtime directories: %v", err)
	}

	databaseConfig := database.ConfigFromEnv()
	db, err := database.Open(databaseConfig)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := waitForDatabase(ctx, db.PingContext); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	if err := migrations.Run(ctx, db, databaseConfig.Driver); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	libraryStore := libraries.NewRepository(db, databaseConfig.Driver)
	mediaFileStore := mediafiles.NewRepository(db, databaseConfig.Driver)
	mediaFileHandler := mediafiles.NewHandler(
		libraryStore,
		mediaFileStore,
		runtimePaths.Media,
		mediafiles.Discover,
	)

	mux := http.NewServeMux()
	mux.Handle("/api/health", health.NewHandler(databaseConfig.Driver, func() error {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.PingContext(pingCtx)
	}))
	mux.Handle("/api/libraries", libraries.NewHandler(libraryStore, runtimePaths.Media))
	mux.Handle("GET /api/libraries/{id}/files", mediaFileHandler)
	mux.Handle("POST /api/libraries/{id}/scan", mediaFileHandler)
	mux.Handle("/", web.NewSPAHandler(env.OrDefault("WEB_DIST_DIR", "/app/web")))

	addr := ":" + env.OrDefault("PORT", "8080")
	log.Printf("velora server listening on %s", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}

func waitForDatabase(ctx context.Context, ping func(context.Context) error) error {
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var lastErr error
	for {
		pingCtx, pingCancel := context.WithTimeout(deadline, 2*time.Second)
		lastErr = ping(pingCtx)
		pingCancel()
		if lastErr == nil {
			return nil
		}

		select {
		case <-deadline.Done():
			return lastErr
		case <-time.After(500 * time.Millisecond):
		}
	}
}
