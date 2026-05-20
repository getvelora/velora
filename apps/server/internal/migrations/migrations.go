// Package migrations applies Velora's database schema migrations using Goose.
// SQL files are embedded per dialect so the binary needs no migration files at
// runtime; the driver name in the database config selects the embedded set.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed sql/sqlite/*.sql
var sqliteFS embed.FS

//go:embed sql/postgres/*.sql
var postgresFS embed.FS

// Run applies any pending migrations against db. driver selects the SQL
// dialect and the embedded migration set ("sqlite" or "postgres"). It is safe
// to call repeatedly; Goose tracks applied versions in `goose_db_version`.
func Run(ctx context.Context, db *sql.DB, driver string) error {
	dialect, fsys, err := selectMigrationSet(driver)
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(dialect, db, fsys)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func selectMigrationSet(driver string) (goose.Dialect, fs.FS, error) {
	switch driver {
	case "sqlite":
		sub, err := fs.Sub(sqliteFS, "sql/sqlite")
		if err != nil {
			return "", nil, fmt.Errorf("locate sqlite migrations: %w", err)
		}
		return goose.DialectSQLite3, sub, nil
	case "postgres":
		sub, err := fs.Sub(postgresFS, "sql/postgres")
		if err != nil {
			return "", nil, fmt.Errorf("locate postgres migrations: %w", err)
		}
		return goose.DialectPostgres, sub, nil
	default:
		return "", nil, fmt.Errorf("unsupported migration driver: %q", driver)
	}
}
