package clog

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestFromContext_NoLogger_ReturnsDefault(t *testing.T) {
	ctx := context.Background()
	l := FromContext(ctx)
	if l == nil {
		t.Fatal("FromContext returned nil")
	}
	if l != slog.Default() {
		t.Error("FromContext without stored logger should return slog.Default()")
	}
}

func TestFromContext_NilContext_ReturnsDefault(t *testing.T) {
	l := FromContext(nil)
	if l == nil {
		t.Fatal("FromContext(nil) returned nil")
	}
	if l != slog.Default() {
		t.Error("FromContext(nil) should return slog.Default()")
	}
}

func TestNewContext_FromContext_ReturnsStoredLogger(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	ctx = NewContext(ctx, custom)
	l := FromContext(ctx)
	if l != custom {
		t.Error("FromContext should return the logger stored by NewContext")
	}
}

func TestWith_StoresLoggerWithAttrs(t *testing.T) {
	base := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := NewContext(context.Background(), base)
	ctx = With(ctx, "request_id", "req-123", "path", "/api")
	l := FromContext(ctx)
	if l == nil {
		t.Fatal("FromContext returned nil")
	}
	// Logger should be different from base (it has With applied)
	if l == base {
		t.Error("With should produce a new logger with attrs, not the same pointer")
	}
}

func TestWithGroup_StoresLoggerWithGroup(t *testing.T) {
	base := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := NewContext(context.Background(), base)
	ctx = WithGroup(ctx, "http")
	l := FromContext(ctx)
	if l == nil {
		t.Fatal("FromContext returned nil")
	}
	if l == base {
		t.Error("WithGroup should produce a new logger with group, not the same pointer")
	}
}

func TestNewContext_NilContext_UsesBackground(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := NewContext(nil, custom)
	if ctx == nil {
		t.Fatal("NewContext(nil, logger) returned nil context")
	}
	l := FromContext(ctx)
	if l != custom {
		t.Error("NewContext(nil, logger) should still store the logger in a background context")
	}
}
