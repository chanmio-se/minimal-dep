package repository

import (
	"context"
	"sort"
	"sync"

	"minimal-dep/internal/modules/tasks/domain"
)

// Memory is an in-memory Repository for development and tests.
type Memory struct {
	mu    sync.RWMutex
	tasks map[string]domain.Task
}

// NewMemory builds an empty in-memory repository.
func NewMemory() *Memory { return &Memory{tasks: make(map[string]domain.Task)} }

// Create stores a task.
func (m *Memory) Create(_ context.Context, task domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
	return nil
}

// ListByOwner returns the owner's tasks in chronological order.
func (m *Memory) ListByOwner(_ context.Context, ownerID string) ([]domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tasks := make([]domain.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		if task.OwnerID == ownerID {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}

// FindByID returns a task by ID.
func (m *Memory) FindByID(_ context.Context, id string) (domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return task, nil
}

// Save replaces a stored task.
func (m *Memory) Save(ctx context.Context, task domain.Task) error {
	return m.Create(ctx, task)
}
