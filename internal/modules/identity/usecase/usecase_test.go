package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"minimal-dep/internal/modules/identity/repository"
	"minimal-dep/internal/platform/session"
)

const testTTL = 8 * time.Hour

func TestLoginAuthenticateAndLogout(t *testing.T) {
	ctx := context.Background()
	tokens := []string{"token1", "token2"}
	next := 0
	store := session.New(testTTL, session.WithTokenFunc(func() string {
		token := tokens[next]
		next++
		return token
	}))
	uc := New(repository.NewMemory(), store)
	uc.newID = func() string { return "u1" }

	token, err := uc.Login(ctx, " User@Example.com ", "Alice")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "token1" {
		t.Fatalf("token = %q, want token1", token)
	}
	principal, err := uc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.ID != "u1" || principal.Email != "user@example.com" || principal.Name != "Alice" {
		t.Fatalf("principal = %+v", principal)
	}

	if err := uc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := uc.Authenticate(ctx, token); !errors.Is(err, session.ErrInvalidToken) {
		t.Fatalf("Authenticate() after Logout() error = %v, want ErrInvalidToken", err)
	}
}

func TestLoginRotatesExistingSession(t *testing.T) {
	ctx := context.Background()
	tokens := []string{"old", "new"}
	next := 0
	store := session.New(testTTL, session.WithTokenFunc(func() string {
		token := tokens[next]
		next++
		return token
	}))
	uc := New(repository.NewMemory(), store)
	uc.newID = func() string { return "u1" }

	old, err := uc.Login(ctx, "u@example.com", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := uc.LoginWithSession(ctx, "u@example.com", "Alice", old)
	if err != nil {
		t.Fatalf("LoginWithSession() error = %v", err)
	}
	if rotated != "new" {
		t.Fatalf("rotated token = %q, want new", rotated)
	}
	if _, err := uc.Authenticate(ctx, old); !errors.Is(err, session.ErrInvalidToken) {
		t.Fatalf("old token still valid: %v", err)
	}
	if _, err := uc.Authenticate(ctx, rotated); err != nil {
		t.Fatalf("new token invalid: %v", err)
	}
}

func TestLoginReusesUser(t *testing.T) {
	tokens := []string{"token1", "token2"}
	next := 0
	store := session.New(testTTL, session.WithTokenFunc(func() string {
		token := tokens[next]
		next++
		return token
	}))
	uc := New(repository.NewMemory(), store)
	uc.newID = func() string { return "u1" }
	for i := 0; i < 2; i++ {
		if _, err := uc.Login(context.Background(), "u@example.com", "Alice"); err != nil {
			t.Fatalf("Login() #%d error = %v", i, err)
		}
	}
	principal, err := uc.Authenticate(context.Background(), "token2")
	if err != nil || principal.ID != "u1" {
		t.Fatalf("Authenticate() = %+v, %v", principal, err)
	}
}

func TestLoginRejectsInvalidInput(t *testing.T) {
	uc := New(repository.NewMemory(), session.New(testTTL))
	if _, err := uc.Login(context.Background(), "not-an-email", "Alice"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Login() error = %v, want ErrInvalidInput", err)
	}
}
