package http

import (
	"context"
	"io"
)

// NewContext returns a new Context that carries the log output.
func NewContext(ctx context.Context, logOutput io.Writer) context.Context {
	return context.WithValue(ctx, valueKey, contextValue{
		logOutput: logOutput,
	})
}

// FromContext returns the log output stored in ctx, if any.
func FromContext(ctx context.Context) io.Writer {
	v, _ := ctx.Value(valueKey).(contextValue)
	return v.logOutput
}

// contextValue is the set of data passed with Context.
type contextValue struct {
	logOutput io.Writer
}

// contextKey is an unexported type for preventing context key collisions.
type contextKey int

// valueKey is the key used to store the context value.
const valueKey contextKey = 0
