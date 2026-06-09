package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // Register the pgx database/sql driver.
	_ "modernc.org/sqlite"             // Register the SQLite database/sql driver.
)

func Open(config Config) (*sql.DB, error) {
	switch config.Driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(config.URL), 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}

		separator := "?"
		if strings.Contains(config.URL, "?") {
			separator = "&"
		}
		dsn := config.URL + separator + "_pragma=foreign_keys(1)"
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite database: %w", err)
		}

		return db, nil
	case "postgres":
		db, err := sql.Open("pgx", config.URL)
		if err != nil {
			return nil, fmt.Errorf("open postgres database: %w", err)
		}

		return db, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Driver)
	}
}
