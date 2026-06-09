package database

import "testing"

func TestConfigDefaultsToSQLiteInConfigDirectory(t *testing.T) {
	t.Setenv("VELORA_DATABASE_DRIVER", "")
	t.Setenv("VELORA_DATABASE_URL", "")

	config := ConfigFromEnv()

	if config.Driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %q", config.Driver)
	}

	if config.URL != "/config/velora.db" {
		t.Fatalf("expected /config/velora.db, got %q", config.URL)
	}
}

func TestConfigUsesExplicitPostgresSettings(t *testing.T) {
	t.Setenv("VELORA_DATABASE_DRIVER", "postgres")
	t.Setenv("VELORA_DATABASE_URL", "postgres://velora:secret@example.com:5432/velora?sslmode=disable")

	config := ConfigFromEnv()

	if config.Driver != "postgres" {
		t.Fatalf("expected postgres driver, got %q", config.Driver)
	}

	if config.URL != "postgres://velora:secret@example.com:5432/velora?sslmode=disable" {
		t.Fatalf("unexpected database URL: %q", config.URL)
	}
}
