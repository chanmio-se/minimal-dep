package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"minimal-dep/internal/modules/identity/domain"
)

// SQL persists users with database/sql. The driver is chosen by the
// composition root; queries use portable '?' placeholders.
type SQL struct {
	db *sql.DB
}

// NewSQL builds a SQL-backed repository.
func NewSQL(db *sql.DB) *SQL { return &SQL{db: db} }

// Create inserts a user.
func (r *SQL) Create(ctx context.Context, user domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO identity_users (id, email, name, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Email, user.Name, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert identity user: %w", err)
	}
	return nil
}

// FindByEmail returns the user with the given email.
func (r *SQL) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.findOne(ctx, `SELECT id, email, name, created_at FROM identity_users WHERE email = ?`, email)
}

// FindByID returns the user with the given ID.
func (r *SQL) FindByID(ctx context.Context, id string) (domain.User, error) {
	return r.findOne(ctx, `SELECT id, email, name, created_at FROM identity_users WHERE id = ?`, id)
}

func (r *SQL) findOne(ctx context.Context, query string, arg string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find identity user: %w", err)
	}
	return user, nil
}
