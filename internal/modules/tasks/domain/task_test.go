package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewTask(t *testing.T) {
	task, err := NewTask("t1", "u1", "  write docs  ")
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Title != "write docs" {
		t.Fatalf("Title = %q, want %q", task.Title, "write docs")
	}
	if task.Done {
		t.Fatal("new task must not be done")
	}
}

func TestNewTaskRejectsInvalidInput(t *testing.T) {
	tests := []struct{ id, owner, title string }{
		{"", "u1", "title"},
		{"t1", "", "title"},
		{"t1", "u1", " "},
		{"t1", "u1", strings.Repeat("x", 201)},
	}
	for _, tc := range tests {
		if _, err := NewTask(tc.id, tc.owner, tc.title); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("NewTask(%q, %q, %q) error = %v, want ErrInvalidInput", tc.id, tc.owner, tc.title, err)
		}
	}
}

func TestComplete(t *testing.T) {
	task, err := NewTask("t1", "u1", "title")
	if err != nil {
		t.Fatal(err)
	}
	done := task.Complete()
	if !done.Done || task.Done {
		t.Fatalf("Complete() = %+v, original = %+v", done, task)
	}
}
