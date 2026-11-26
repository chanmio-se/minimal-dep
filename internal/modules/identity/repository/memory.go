package repository

import (
	"context"
	"sync"

	"minimal-dep/internal/modules/identity/domain"
)

// Memory is an in-memory Repository for development and tests.
type Memory struct {
	mu     sync.RWMutex
	byID   map[string]domain.User
	byMail map[string]domain.User
}

// NewMemory builds an empty in-memory repository.
func NewMemory() *Memory {
	return &Memory{byID: make(map[string]domain.User), byMail: make(map[string]domain.User)}
}

// Create stores a user.
func (m *Memory) Create(_ context.Context, user domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[user.ID] = user
	m.byMail[user.Email] = user
	return nil
}

// FindByEmail looks up a user by email address.
func (m *Memory) FindByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.byMail[email]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return user, nil
}

// FindByID looks up a user by ID.
func (m *Memory) FindByID(_ context.Context, id string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.byID[id]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return user, nil
}
