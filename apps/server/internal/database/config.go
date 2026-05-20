package database

import "github.com/mdelle/velora/apps/server/internal/env"

type Config struct {
	Driver string
	URL    string
}

func ConfigFromEnv() Config {
	driver := env.OrDefault("VELORA_DATABASE_DRIVER", "sqlite")
	url := env.OrDefault("VELORA_DATABASE_URL", "/config/velora.db")

	return Config{
		Driver: driver,
		URL:    url,
	}
}
