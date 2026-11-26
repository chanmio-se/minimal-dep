package quint

import (
	"context"
	"strings"
	"testing"
	"time"

	"minimal-dep/internal/modules/identity/repository"
	"minimal-dep/internal/modules/identity/usecase"
	"minimal-dep/internal/platform/session"
)

// TestSessionTrace replays a Quint trace through the public identity API.
// Token names in the model are injected into the session store, so they are
// also the real tokens in Go.
func TestSessionTrace(t *testing.T) {
	tr := loadTrace(t, "session")
	nextToken := ""
	store := session.New(time.Hour, session.WithTokenFunc(func() string { return nextToken }))
	uc := usecase.New(repository.NewMemory(), store)
	ctx := context.Background()

	for step, s := range tr.States {
		got := ""
		switch s.Action {
		case "init":
			got = "init"
		case "issue":
			alias := pickString(t, step, s, "token")
			subject := pickString(t, step, s, "subject")
			nextToken = alias
			if _, err := uc.Login(ctx, subject+"@example.com", subject); err != nil {
				t.Fatalf("step %d: Login(%s): %v", step, alias, err)
			}
			got = "issued"
		case "rotate":
			previous := pickString(t, step, s, "previous")
			alias := pickString(t, step, s, "token")
			subject := pickString(t, step, s, "subject")
			nextToken = alias
			if _, err := uc.LoginWithSession(ctx, subject+"@example.com", subject, previous); err != nil {
				t.Fatalf("step %d: LoginWithSession(%s -> %s): %v", step, previous, alias, err)
			}
			got = "rotated"
		case "authenticate":
			alias := pickString(t, step, s, "token")
			principal, err := uc.Authenticate(ctx, alias)
			if err != nil {
				t.Fatalf("step %d: Authenticate(%s): %v", step, alias, err)
			}
			if subject := subjectOf(principal.Email); subject != s.Last.Subject {
				t.Fatalf("step %d: subject = %q, want %q", step, subject, s.Last.Subject)
			}
			got = "authenticated"
		case "authenticateUnknown":
			alias := pickString(t, step, s, "token")
			if _, err := uc.Authenticate(ctx, alias); err == nil {
				t.Fatalf("step %d: Authenticate(%s) unexpectedly succeeded", step, alias)
			}
			got = "unknown"
		case "expire", "logout":
			// The trace models expiry and logout as the same invalidation event;
			// the Go driver uses the public revocation API for both.
			alias := pickString(t, step, s, "token")
			if err := uc.Logout(ctx, alias); err != nil {
				t.Fatalf("step %d: Logout(%s): %v", step, alias, err)
			}
			got = map[string]string{"expire": "expired", "logout": "logged_out"}[s.Action]
		default:
			t.Fatalf("step %d: unexpected action %q", step, s.Action)
		}
		assertCode(t, step, s.Last.Code, got)

		// Compare the full modeled session state after every step.
		for _, pair := range s.Sessions.Pairs {
			alias, want := pair[0], pair[1]
			principal, err := uc.Authenticate(ctx, alias)
			subject := ""
			if err == nil {
				subject = subjectOf(principal.Email)
			}
			if subject != want {
				t.Fatalf("step %d: session %s = %q, want %q", step, alias, subject, want)
			}
		}
	}
}

func subjectOf(email string) string {
	return strings.TrimSuffix(email, "@example.com")
}
