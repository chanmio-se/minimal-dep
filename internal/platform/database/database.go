// Package database provides database/sql setup, health checks and migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Open opens a database when a driver is configured. With no driver it returns
// (nil, nil) so the application can run with in-memory repositories.
func Open(driver, dsn string) (*sql.DB, error) {
	if driver == "" {
		return nil, nil
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// Ping reports whether the database is reachable. A nil database is healthy
// because the application is running without persistent storage.
func Ping(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.PingContext(ctx)
}
