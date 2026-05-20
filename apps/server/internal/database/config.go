package database

import "os"

type Config struct {
	Driver string
	URL    string
}

func ConfigFromEnv() Config {
	driver := envOrDefault("VELORA_DATABASE_DRIVER", "sqlite")
	url := envOrDefault("VELORA_DATABASE_URL", "/config/velora.db")

	return Config{
		Driver: driver,
		URL:    url,
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

