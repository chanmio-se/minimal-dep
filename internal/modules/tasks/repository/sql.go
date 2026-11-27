package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"minimal-dep/internal/modules/tasks/domain"
)

// SQL persists tasks with database/sql. The driver is chosen by the
// composition root; queries use portable '?' placeholders.
type SQL struct {
	queries *Queries
}

// NewSQL builds a SQL-backed repository.
func NewSQL(db *sql.DB) *SQL { return &SQL{queries: New(db)} }

// Create inserts a task.
func (r *SQL) Create(ctx context.Context, task domain.Task) error {
	err := r.queries.CreateTask(ctx, CreateTaskParams{
		ID: task.ID, OwnerID: task.OwnerID, Title: task.Title,
		Done: boolToInt64(task.Done), CreatedAt: task.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// ListByOwner returns the owner's tasks in chronological order.
func (r *SQL) ListByOwner(ctx context.Context, ownerID string) ([]domain.Task, error) {
	rows, err := r.queries.ListTasksByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	tasks := make([]domain.Task, 0)
	for _, row := range rows {
		tasks = append(tasks, toDomainTask(row))
	}
	return tasks, nil
}

// FindByID returns a task by ID.
func (r *SQL) FindByID(ctx context.Context, id string) (domain.Task, error) {
	task, err := r.queries.FindTaskByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("find task: %w", err)
	}
	return toDomainTask(task), nil
}

// Save updates a stored task.
func (r *SQL) Save(ctx context.Context, task domain.Task) error {
	err := r.queries.SaveTask(ctx, SaveTaskParams{
		Title: task.Title, Done: boolToInt64(task.Done),
		CreatedAt: task.CreatedAt, ID: task.ID,
	})
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func toDomainTask(task Task) domain.Task {
	return domain.Task{
		ID: task.ID, OwnerID: task.OwnerID, Title: task.Title,
		Done: task.Done != 0, CreatedAt: task.CreatedAt,
	}
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
