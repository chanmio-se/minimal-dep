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
	queries *Queries
}

// NewSQL builds a SQL-backed repository.
func NewSQL(db *sql.DB) *SQL { return &SQL{queries: New(db)} }

// Create inserts a user.
func (r *SQL) Create(ctx context.Context, user domain.User) error {
	err := r.queries.CreateUser(ctx, CreateUserParams{
		ID: user.ID, Email: user.Email, Name: user.Name, CreatedAt: user.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert identity user: %w", err)
	}
	return nil
}

// FindByEmail returns the user with the given email.
func (r *SQL) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.findOne(func() (IdentityUser, error) {
		return r.queries.FindUserByEmail(ctx, email)
	})
}

// FindByID returns the user with the given ID.
func (r *SQL) FindByID(ctx context.Context, id string) (domain.User, error) {
	return r.findOne(func() (IdentityUser, error) {
		return r.queries.FindUserByID(ctx, id)
	})
}

func (r *SQL) findOne(find func() (IdentityUser, error)) (domain.User, error) {
	row, err := find()
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find identity user: %w", err)
	}
	return domain.User{ID: row.ID, Email: row.Email, Name: row.Name, CreatedAt: row.CreatedAt}, nil
}
