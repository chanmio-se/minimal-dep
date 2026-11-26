// Package domain owns the identity business rules.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidInput is returned when user input violates a business rule.
var ErrInvalidInput = errors.New("invalid input")

// User is an identity aggregate.
type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt string
}

// Principal is the identity exposed to other modules and middleware.
type Principal struct {
	ID    string
	Name  string
	Email string
}

// NewUser creates a validated user.
func NewUser(id, email, name string) (User, error) {
	id = strings.TrimSpace(id)
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	switch {
	case id == "":
		return User{}, fmt.Errorf("%w: id is required", ErrInvalidInput)
	case !strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@"):
		return User{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	case name == "" || len(name) > 100:
		return User{}, fmt.Errorf("%w: invalid name", ErrInvalidInput)
	}
	return User{ID: id, Email: email, Name: name}, nil
}

// Principal converts the user into the public identity.
func (u User) Principal() Principal {
	return Principal{ID: u.ID, Name: u.Name, Email: u.Email}
}
