// Package domain owns the task business rules.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidInput is returned when task input violates a business rule.
var ErrInvalidInput = errors.New("invalid input")

// Task is an owned task entity.
type Task struct {
	ID        string
	OwnerID   string
	Title     string
	Done      bool
	CreatedAt time.Time
}

// NewTask creates a validated task.
func NewTask(id, ownerID, title string) (Task, error) {
	id = strings.TrimSpace(id)
	ownerID = strings.TrimSpace(ownerID)
	title = strings.TrimSpace(title)
	switch {
	case id == "":
		return Task{}, fmt.Errorf("%w: id is required", ErrInvalidInput)
	case ownerID == "":
		return Task{}, fmt.Errorf("%w: owner is required", ErrInvalidInput)
	case title == "" || len(title) > 200:
		return Task{}, fmt.Errorf("%w: title must be 1-200 characters", ErrInvalidInput)
	}
	return Task{ID: id, OwnerID: ownerID, Title: title}, nil
}

// Complete returns the completed task.
func (t Task) Complete() Task {
	t.Done = true
	return t
}
