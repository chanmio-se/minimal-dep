// Package session provides the shared session lifecycle: generation,
// resolution, rotation and revocation. It is independent of business modules.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// ErrInvalidToken is returned for unknown, expired or revoked tokens.
var ErrInvalidToken = errors.New("invalid or expired session token")

type entry struct {
	subject   string
	expiresAt time.Time
}

// Option customizes a Store, mainly for tests.
type Option func(*Store)

// WithTokenFunc overrides token generation.
func WithTokenFunc(f func() string) Option {
	return func(s *Store) { s.newToken = f }
}

// WithClock overrides the clock used for expiry checks.
func WithClock(now func() time.Time) Option {
	return func(s *Store) { s.now = now }
}

// Store keeps sessions in memory and enforces a fixed lifetime.
type Store struct {
	mu       sync.RWMutex
	entries  map[string]entry
	ttl      time.Duration
	newToken func() string
	now      func() time.Time
}

// New builds a session store with the given lifetime.
func New(ttl time.Duration, opts ...Option) *Store {
	s := &Store{
		entries:  make(map[string]entry),
		ttl:      ttl,
		newToken: randomToken,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create issues a new session for subject.
func (s *Store) Create(subject string) (string, error) {
	if subject == "" {
		return "", errors.New("session subject is required")
	}
	token := s.newToken()
	if token == "" {
		return "", errors.New("generated session token is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entries[token]; exists {
		return "", errors.New("session token collision")
	}
	s.entries[token] = entry{subject: subject, expiresAt: s.now().Add(s.ttl)}
	return token, nil
}

// Rotate revokes the previous token and issues a new one for subject.
func (s *Store) Rotate(previous, subject string) (string, error) {
	token, err := s.Create(subject)
	if err != nil {
		return "", err
	}
	if previous != "" {
		s.Delete(previous)
	}
	return token, nil
}

// Resolve returns the subject of a live session.
func (s *Store) Resolve(token string) (string, error) {
	if token == "" {
		return "", ErrInvalidToken
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[token]
	if !ok {
		return "", ErrInvalidToken
	}
	if !s.now().Before(e.expiresAt) {
		delete(s.entries, token)
		return "", ErrInvalidToken
	}
	return e.subject, nil
}

// Delete revokes a token. Deleting an unknown token is a no-op.
func (s *Store) Delete(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.entries, token)
	s.mu.Unlock()
}

func randomToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
