package app

import (
	"context"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

type ctxKey int

const userKey ctxKey = iota

// WithUser stores the authenticated user on the context (set by the HTTP
// auth middleware, read by handlers and the WebSocket adapter).
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFrom returns the authenticated user, or nil.
func UserFrom(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userKey).(*domain.User)
	return u
}
