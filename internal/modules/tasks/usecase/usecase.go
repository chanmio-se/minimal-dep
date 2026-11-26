// Package usecase implements task application operations.
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"minimal-dep/internal/modules/tasks/domain"
	"minimal-dep/internal/modules/tasks/repository"
)

// Layer errors translated by the handler into HTTP statuses.
var (
	ErrInvalidInput = domain.ErrInvalidInput
	ErrNotFound     = errors.New("task not found")
	ErrForbidden    = errors.New("task belongs to another owner")
)

// IDFunc generates task IDs.
type IDFunc func() string

// Option customizes the usecase, mainly for tests.
type Option func(*Usecase)

// WithIDFunc overrides task ID generation.
func WithIDFunc(f IDFunc) Option {
	return func(u *Usecase) { u.newID = f }
}

// Usecase owns task operations and the transaction boundary.
type Usecase struct {
	repo  repository.Repository
	newID IDFunc
	now   func() time.Time
}

// New builds a task usecase.
func New(repo repository.Repository, opts ...Option) *Usecase {
	u := &Usecase{repo: repo, newID: newRandomID, now: time.Now}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Create adds a task for the owner.
func (u *Usecase) Create(ctx context.Context, ownerID, title string) (domain.Task, error) {
	task, err := domain.NewTask(u.newID(), ownerID, title)
	if err != nil {
		return domain.Task{}, err
	}
	task.CreatedAt = u.now().UTC()
	if err := u.repo.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("persist task: %w", err)
	}
	return task, nil
}

// List returns all tasks owned by the caller.
func (u *Usecase) List(ctx context.Context, ownerID string) ([]domain.Task, error) {
	if ownerID == "" {
		return nil, fmt.Errorf("%w: owner is required", ErrInvalidInput)
	}
	tasks, err := u.repo.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Complete marks the caller's task as done.
func (u *Usecase) Complete(ctx context.Context, ownerID, id string) error {
	task, err := u.repo.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	if task.OwnerID != ownerID {
		return ErrForbidden
	}
	if task.Done {
		return nil
	}
	if err := u.repo.Save(ctx, task.Complete()); err != nil {
		return fmt.Errorf("persist completed task: %w", err)
	}
	return nil
}

func newRandomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
