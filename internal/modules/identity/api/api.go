// Package api is the public contract of the identity module.
// Other packages must depend on this package instead of internal layers.
package api

import (
	"context"

	"minimal-dep/internal/modules/identity/domain"
)

// SessionCookieName is the cookie used to carry a session token.
const SessionCookieName = "session"

// Principal is the authenticated caller.
type Principal = domain.Principal

// Service is the functionality exposed to other modules.
type Service interface {
	Login(ctx context.Context, email, name string) (token string, err error)
	Authenticate(ctx context.Context, token string) (Principal, error)
	Logout(ctx context.Context, token string) error
}
