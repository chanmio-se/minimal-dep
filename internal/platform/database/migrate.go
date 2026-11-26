package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"time"
)

// Migrate applies SQL files from fsys in lexical order. Each file is applied
// once and recorded in schema_migrations.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS, pattern string) error {
	if db == nil {
		return fmt.Errorf("migration requires a database")
	}
	files, err := fs.Glob(fsys, pattern)
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for _, name := range files {
		applied, err := isApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := apply(ctx, db, name, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func isApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return count > 0, nil
}

func apply(ctx context.Context, db *sql.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, body); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE name = ?`, name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`,
		name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}
