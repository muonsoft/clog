package clog

import (
	"context"
	"log/slog"
)

// ContextKey is the type for context keys that ContextHandler reads
// to add attributes to each log record. Use it with context.WithValue.
type ContextKey string

// ContextHandler wraps a slog.Handler and adds attributes from context.Context
// for each key in keys. Values are retrieved via ctx.Value(key) and added
// to the record before passing to the inner handler.
type ContextHandler struct {
	inner slog.Handler
	keys  []ContextKey
}

// NewContextHandler returns a Handler that adds context attributes for the
// given keys to every record, then delegates to inner.
func NewContextHandler(inner slog.Handler, keys []ContextKey) *ContextHandler {
	if inner == nil {
		panic("clog: inner handler is nil")
	}
	keysCopy := make([]ContextKey, len(keys))
	copy(keysCopy, keys)
	return &ContextHandler{inner: inner, keys: keysCopy}
}

// Enabled reports whether the handler handles records at the given level.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle adds attributes from ctx for each configured key, then passes
// the record to the inner handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, k := range h.keys {
		v := ctx.Value(k)
		if v != nil {
			r.Add(slog.Any(string(k), v))
		}
	}
	return h.inner.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler with the same context keys and
// the inner handler wrapped with the given attrs.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		inner: h.inner.WithAttrs(attrs),
		keys:  h.keys,
	}
}

// WithGroup returns a new ContextHandler with the same context keys and
// the inner handler wrapped with the given group name.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		inner: h.inner.WithGroup(name),
		keys:  h.keys,
	}
}
