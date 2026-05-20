package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mdelle/velora/apps/server/internal/database"
	"github.com/mdelle/velora/apps/server/internal/health"
	"github.com/mdelle/velora/apps/server/internal/storage"
)

func main() {
	ctx := context.Background()

	runtimePaths := storage.RuntimePathsFromEnv()
	if err := storage.EnsureRuntimeDirectories(runtimePaths); err != nil {
		log.Fatalf("create runtime directories: %v", err)
	}

	databaseConfig := database.ConfigFromEnv()
	db, err := database.Open(databaseConfig)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := waitForDatabase(ctx, db.PingContext); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/health", health.NewHandler(databaseConfig.Driver, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.PingContext(ctx)
	}))
	mux.Handle("/", http.FileServer(http.Dir(envOrDefault("WEB_DIST_DIR", "/app/web"))))

	addr := ":" + envOrDefault("PORT", "8080")
	log.Printf("velora server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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

		if deadline.Err() != nil {
			return lastErr
		}

		time.Sleep(500 * time.Millisecond)
	}
}
