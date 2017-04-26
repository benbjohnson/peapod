package peapod

import "context"

// NewContext returns a new Context that carries the authenticated user.
func NewContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, valueKey, contextValue{
		user: user,
	})
}

// FromContext returns the user stored in ctx, if any.
func FromContext(ctx context.Context) *User {
	v, _ := ctx.Value(valueKey).(contextValue)
	return v.user
}

// contextValue is the set of data passed with Context.
type contextValue struct {
	user *User
}

// contextKey is an unexported type for preventing context key collisions.
type contextKey int

// valueKey is the key used to store the context value.
const valueKey contextKey = 0
