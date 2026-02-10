package clog

import (
	"context"
	"log/slog"
)

type contextKey struct{}

var loggerKey = &contextKey{}

// FromContext returns the Logger stored in ctx, or slog.Default() if none.
// Use this to obtain the request-scoped logger when one was set via NewContext.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// NewContext returns a copy of ctx that stores the given Logger.
// Retrieve it later with FromContext. Use this in middleware to attach
// a request-scoped logger (e.g. with request_id, path) to the context.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// With returns a new context that stores a Logger with the given attributes
// attached. The Logger is obtained from ctx (or the default), then With(args...)
// is called on it, and the result is stored in a child context.
func With(ctx context.Context, args ...any) context.Context {
	return NewContext(ctx, FromContext(ctx).With(args...))
}

// WithGroup returns a new context that stores a Logger with the given group
// name. The Logger is obtained from ctx (or the default), then WithGroup(name)
// is called on it, and the result is stored in a child context.
func WithGroup(ctx context.Context, name string) context.Context {
	return NewContext(ctx, FromContext(ctx).WithGroup(name))
}
