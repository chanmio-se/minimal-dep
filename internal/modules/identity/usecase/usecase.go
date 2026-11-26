// Package usecase implements identity application operations.
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"minimal-dep/internal/modules/identity/domain"
	"minimal-dep/internal/modules/identity/repository"
	"minimal-dep/internal/platform/session"
)

// ErrInvalidInput is returned for invalid credentials or profile data.
var ErrInvalidInput = domain.ErrInvalidInput

// IDFunc generates user IDs.
type IDFunc func() string

// Usecase owns identity operations. Session lifecycle is delegated to the
// shared session store so generation, rotation and revocation stay consistent.
type Usecase struct {
	repo     repository.Repository
	sessions *session.Store
	newID    IDFunc
	now      func() time.Time
}

// New builds an identity usecase.
func New(repo repository.Repository, sessions *session.Store) *Usecase {
	return &Usecase{
		repo:     repo,
		sessions: sessions,
		newID:    newRandomID,
		now:      time.Now,
	}
}

// Login finds or creates a user and returns a new session token.
func (u *Usecase) Login(ctx context.Context, email, name string) (string, error) {
	return u.login(ctx, email, name, "")
}

// LoginWithSession rotates an existing session when the caller already has
// one, preventing session fixation across login.
func (u *Usecase) LoginWithSession(ctx context.Context, email, name, previous string) (string, error) {
	return u.login(ctx, email, name, previous)
}

func (u *Usecase) login(ctx context.Context, email, name, previous string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if !strings.Contains(email, "@") || name == "" {
		return "", fmt.Errorf("%w: email and name are required", ErrInvalidInput)
	}

	user, err := u.repo.FindByEmail(ctx, email)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		user, err = domain.NewUser(u.newID(), email, name)
		if err != nil {
			return "", err
		}
		user.CreatedAt = u.now().UTC().Format(time.RFC3339)
		if err := u.repo.Create(ctx, user); err != nil {
			return "", fmt.Errorf("create user: %w", err)
		}
	case err != nil:
		return "", fmt.Errorf("find user: %w", err)
	}

	if previous != "" {
		return u.sessions.Rotate(previous, user.ID)
	}
	return u.sessions.Create(user.ID)
}

// Authenticate resolves a session token to the caller's public identity.
func (u *Usecase) Authenticate(ctx context.Context, token string) (domain.Principal, error) {
	userID, err := u.sessions.Resolve(token)
	if err != nil {
		return domain.Principal{}, err
	}
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("find session user: %w", err)
	}
	return user.Principal(), nil
}

// Logout revokes a session token.
func (u *Usecase) Logout(_ context.Context, token string) error {
	u.sessions.Delete(token)
	return nil
}

func newRandomID() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
