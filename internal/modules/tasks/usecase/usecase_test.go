package usecase

import (
	"context"
	"errors"
	"testing"

	"minimal-dep/internal/modules/tasks/repository"
)

func TestCreateListAndComplete(t *testing.T) {
	repo := repository.NewMemory()
	uc := New(repo, WithIDFunc(func() string { return "t1" }))

	created, err := uc.Create(context.Background(), "u1", "buy milk")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "t1" || created.OwnerID != "u1" {
		t.Fatalf("Create() = %+v", created)
	}

	tasks, err := uc.List(context.Background(), "u1")
	if err != nil || len(tasks) != 1 {
		t.Fatalf("List() = %+v, %v", tasks, err)
	}

	if err := uc.Complete(context.Background(), "u1", "t1"); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	tasks, err = uc.List(context.Background(), "u1")
	if err != nil || len(tasks) != 1 || !tasks[0].Done {
		t.Fatalf("List() after Complete() = %+v, %v", tasks, err)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	uc := New(repository.NewMemory())
	if _, err := uc.Create(context.Background(), "u1", " "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create() error = %v, want ErrInvalidInput", err)
	}
}

func TestListScopesToOwner(t *testing.T) {
	repo := repository.NewMemory()
	uc := New(repo, WithIDFunc(func() string { return "t1" }))
	if _, err := uc.Create(context.Background(), "u1", "mine"); err != nil {
		t.Fatal(err)
	}
	tasks, err := uc.List(context.Background(), "u2")
	if err != nil || len(tasks) != 0 {
		t.Fatalf("List(u2) = %+v, %v", tasks, err)
	}
}

func TestCompleteErrors(t *testing.T) {
	repo := repository.NewMemory()
	uc := New(repo, WithIDFunc(func() string { return "t1" }))
	if _, err := uc.Create(context.Background(), "u1", "mine"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Complete(context.Background(), "u2", "t1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Complete(u2) error = %v, want ErrForbidden", err)
	}
	if err := uc.Complete(context.Background(), "u1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Complete(missing) error = %v, want ErrNotFound", err)
	}
}
