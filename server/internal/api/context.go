package api

import (
	"context"

	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Who the current request belongs to.

The acting user is derived from the session and put here by requireSession. It
is never read from anything the client sent — a user id in a body or a header
would be a request to act as somebody, which is not the same as being them. */

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxToken
)

func withUser(ctx context.Context, user store.User, token string) context.Context {
	ctx = context.WithValue(ctx, ctxUser, user)
	return context.WithValue(ctx, ctxToken, token)
}

// userFrom returns the authenticated user and the raw session token that
// authenticated them. Only meaningful behind requireSession.
func userFrom(ctx context.Context) (store.User, string) {
	user, _ := ctx.Value(ctxUser).(store.User)
	token, _ := ctx.Value(ctxToken).(string)
	return user, token
}
