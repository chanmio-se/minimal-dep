// Command migrate applies embedded database migrations and exits.
package main

import (
	"context"
	"fmt"
	"os"

	"minimal-dep/internal/platform/config"
	"minimal-dep/internal/platform/database"
	"minimal-dep/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseDriver == "" {
		return fmt.Errorf("DB_DRIVER and DB_DSN must be configured to migrate")
	}
	db, err := database.Open(cfg.DatabaseDriver, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return database.Migrate(context.Background(), db, migrations.Files, "*.sql")
}
