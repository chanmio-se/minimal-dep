package quint

import (
	"context"
	"errors"
	"testing"

	"minimal-dep/internal/modules/tasks/repository"
	"minimal-dep/internal/modules/tasks/usecase"
)

// TestAuthzTrace replays ownership/completion transitions against the task
// usecase. Task ids from the model are injected via WithIDFunc.
func TestAuthzTrace(t *testing.T) {
	tr := loadTrace(t, "authz")
	repo := repository.NewMemory()
	var nextID string
	uc := usecase.New(repo, usecase.WithIDFunc(func() string { return nextID }))
	ctx := context.Background()

	for step, s := range tr.States {
		got := ""
		switch s.Action {
		case "init":
			got = "init"
		case "create":
			id := pickString(t, step, s, "id")
			owner := pickString(t, step, s, "owner")
			nextID = id
			if _, err := uc.Create(ctx, owner, "task "+id); err != nil {
				t.Fatalf("step %d: Create(%s): %v", step, id, err)
			}
			got = "created"
		case "completeOwner":
			id := pickString(t, step, s, "id")
			if err := uc.Complete(ctx, s.Tasks.get(id), id); err != nil {
				t.Fatalf("step %d: Complete(owner, %s): %v", step, id, err)
			}
			got = "completed"
		case "completeForeign":
			id := pickString(t, step, s, "id")
			actor := pickString(t, step, s, "actor")
			if err := uc.Complete(ctx, actor, id); !errors.Is(err, usecase.ErrForbidden) {
				t.Fatalf("step %d: Complete(%s, %s) error = %v, want ErrForbidden", step, actor, id, err)
			}
			got = "forbidden"
		case "completeUnknown":
			id := pickString(t, step, s, "id")
			if err := uc.Complete(ctx, "alice", id); !errors.Is(err, usecase.ErrNotFound) {
				t.Fatalf("step %d: Complete(unknown, %s) error = %v, want ErrNotFound", step, id, err)
			}
			got = "not_found"
		case "completeAgain":
			id := pickString(t, step, s, "id")
			if err := uc.Complete(ctx, s.Tasks.get(id), id); err != nil {
				t.Fatalf("step %d: repeated Complete(%s): %v", step, id, err)
			}
			got = "completed"
		default:
			t.Fatalf("step %d: unexpected action %q", step, s.Action)
		}
		assertCode(t, step, s.Last.Code, got)

		// Compare the full modeled task state after every step.
		for _, pair := range s.Tasks.Pairs {
			id, wantOwner := pair[0], pair[1]
			task, err := repo.FindByID(ctx, id)
			if wantOwner == "" {
				if !errors.Is(err, repository.ErrNotFound) {
					t.Fatalf("step %d: task %s exists, model says absent (err=%v)", step, id, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("step %d: FindByID(%s): %v", step, id, err)
			}
			if task.OwnerID != wantOwner {
				t.Fatalf("step %d: task %s owner = %q, want %q", step, id, task.OwnerID, wantOwner)
			}
			if wantDone := s.Done.contains(id); task.Done != wantDone {
				t.Fatalf("step %d: task %s done = %t, want %t", step, id, task.Done, wantDone)
			}
		}
	}
}
