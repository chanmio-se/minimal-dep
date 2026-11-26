// Command server starts the web application.
package main

import (
	"context"
	"fmt"
	"os"

	"minimal-dep/internal/app"
	"minimal-dep/internal/platform/config"
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
	application, err := app.New(cfg)
	if err != nil {
		return err
	}
	if cfg.DatabaseDriver != "" {
		if err := application.Migrate(context.Background(), migrations.Files, "*.sql"); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return application.Run(context.Background())
}
