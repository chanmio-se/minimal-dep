// Package api is the public contract of the tasks module.
package api

import (
	"context"

	"minimal-dep/internal/modules/tasks/domain"
)

// Task is the task representation exposed to other modules.
type Task = domain.Task

// Service is the functionality exposed to other modules.
type Service interface {
	Create(ctx context.Context, ownerID, title string) (Task, error)
	List(ctx context.Context, ownerID string) ([]Task, error)
	Complete(ctx context.Context, ownerID, id string) error
}
