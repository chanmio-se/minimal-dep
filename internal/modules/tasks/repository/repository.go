// Package repository persists task data.
package repository

import (
	"context"
	"errors"

	"minimal-dep/internal/modules/tasks/domain"
)

// ErrNotFound is returned when a task does not exist.
var ErrNotFound = errors.New("task not found")

// Repository is the persistence contract used by the usecase layer.
type Repository interface {
	Create(ctx context.Context, task domain.Task) error
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Task, error)
	FindByID(ctx context.Context, id string) (domain.Task, error)
	Save(ctx context.Context, task domain.Task) error
}
