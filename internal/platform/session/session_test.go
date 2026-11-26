package session

import (
	"errors"
	"testing"
	"time"
)

func TestCreateResolveAndDelete(t *testing.T) {
	next := 0
	store := New(time.Hour, WithTokenFunc(func() string {
		next++
		return "token" + string(rune('0'+next))
	}))
	token, err := store.Create("u1")
	if err != nil || token != "token1" {
		t.Fatalf("Create() = %q, %v", token, err)
	}
	subject, err := store.Resolve(token)
	if err != nil || subject != "u1" {
		t.Fatalf("Resolve() = %q, %v", subject, err)
	}
	store.Delete(token)
	if _, err := store.Resolve(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Resolve() after Delete() error = %v", err)
	}
}

func TestRotateRevokesPreviousToken(t *testing.T) {
	tokens := []string{"old", "new"}
	next := 0
	store := New(time.Hour, WithTokenFunc(func() string {
		token := tokens[next]
		next++
		return token
	}))
	old, err := store.Create("u1")
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := store.Rotate(old, "u1")
	if err != nil || rotated != "new" {
		t.Fatalf("Rotate() = %q, %v", rotated, err)
	}
	if _, err := store.Resolve(old); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("old token still resolvable: %v", err)
	}
	if subject, err := store.Resolve(rotated); err != nil || subject != "u1" {
		t.Fatalf("Resolve(rotated) = %q, %v", subject, err)
	}
}

func TestResolveRejectsExpiredToken(t *testing.T) {
	now := time.Now()
	store := New(time.Hour,
		WithTokenFunc(func() string { return "token" }),
		WithClock(func() time.Time { return now }),
	)
	token, err := store.Create("u1")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour + time.Second)
	if _, err := store.Resolve(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Resolve(expired) error = %v, want ErrInvalidToken", err)
	}
}

func TestCreateRejectsEmptySubject(t *testing.T) {
	if _, err := New(time.Hour).Create(""); err == nil {
		t.Fatal("Create() with empty subject must fail")
	}
}
