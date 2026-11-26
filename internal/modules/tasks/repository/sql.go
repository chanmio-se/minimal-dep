package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"minimal-dep/internal/modules/tasks/domain"
)

const selectColumns = `SELECT id, owner_id, title, done, created_at FROM tasks`

// SQL persists tasks with database/sql. The driver is chosen by the
// composition root; queries use portable '?' placeholders.
type SQL struct {
	db *sql.DB
}

// NewSQL builds a SQL-backed repository.
func NewSQL(db *sql.DB) *SQL { return &SQL{db: db} }

// Create inserts a task.
func (r *SQL) Create(ctx context.Context, task domain.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (id, owner_id, title, done, created_at) VALUES (?, ?, ?, ?, ?)`,
		task.ID, task.OwnerID, task.Title, task.Done, task.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// ListByOwner returns the owner's tasks in chronological order.
func (r *SQL) ListByOwner(ctx context.Context, ownerID string) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx, selectColumns+` WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

// FindByID returns a task by ID.
func (r *SQL) FindByID(ctx context.Context, id string) (domain.Task, error) {
	task, err := scanTask(r.db.QueryRowContext(ctx, selectColumns+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("find task: %w", err)
	}
	return task, nil
}

// Save updates a stored task.
func (r *SQL) Save(ctx context.Context, task domain.Task) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, done = ?, created_at = ? WHERE id = ?`,
		task.Title, task.Done, task.CreatedAt, task.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(row scanner) (domain.Task, error) {
	var task domain.Task
	var done int
	var createdAt time.Time
	if err := row.Scan(&task.ID, &task.OwnerID, &task.Title, &done, &createdAt); err != nil {
		return domain.Task{}, err
	}
	task.Done = done != 0
	task.CreatedAt = createdAt
	return task, nil
}
