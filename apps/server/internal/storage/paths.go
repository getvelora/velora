package storage

import (
	"os"

	"github.com/mdelle/velora/apps/server/internal/env"
)

type RuntimePaths struct {
	Config string
	Cache  string
	Media  string
}

func RuntimePathsFromEnv() RuntimePaths {
	return RuntimePaths{
		Config: env.OrDefault("VELORA_CONFIG_DIR", "/config"),
		Cache:  env.OrDefault("VELORA_CACHE_DIR", "/cache"),
		Media:  env.OrDefault("VELORA_MEDIA_DIR", "/media"),
	}
}

func EnsureRuntimeDirectories(paths RuntimePaths) error {
	for _, path := range []string{paths.Config, paths.Cache, paths.Media} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return nil
}
