package storage

import "os"

type RuntimePaths struct {
	Config string
	Cache  string
	Media  string
}

func RuntimePathsFromEnv() RuntimePaths {
	return RuntimePaths{
		Config: envOrDefault("VELORA_CONFIG_DIR", "/config"),
		Cache:  envOrDefault("VELORA_CACHE_DIR", "/cache"),
		Media:  envOrDefault("VELORA_MEDIA_DIR", "/media"),
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

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

