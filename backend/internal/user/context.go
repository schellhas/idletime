package user

import "context"

type contextKey struct{}

// WithContext returns a context carrying the authenticated user.
func WithContext(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// FromContext returns the authenticated user stored by WithContext.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}
