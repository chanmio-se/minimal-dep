// Package repository persists identity data.
package repository

import (
	"context"
	"errors"

	"minimal-dep/internal/modules/identity/domain"
)

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("identity user not found")

// Repository is the persistence contract used by the usecase layer.
type Repository interface {
	Create(ctx context.Context, user domain.User) error
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}
