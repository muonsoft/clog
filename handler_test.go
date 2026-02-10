package clog

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestContextHandler_Handle_AddsAttrsFromContext(t *testing.T) {
	cap := &captureHandler{}
	keys := []ContextKey{"trace_id", "user_id"}
	h := NewContextHandler(cap, keys)
	logger := slog.New(h)

	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("trace_id"), "trace-abc")
	ctx = context.WithValue(ctx, ContextKey("user_id"), "user-42")

	logger.InfoContext(ctx, "test message", "extra", "value")

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	r := cap.records[0]
	var foundTrace, foundUser, foundExtra bool
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "trace_id":
			foundTrace = a.Value.String() == "trace-abc"
		case "user_id":
			foundUser = a.Value.String() == "user-42"
		case "extra":
			foundExtra = a.Value.String() == "value"
		}
		return true
	})
	if !foundTrace {
		t.Error("expected trace_id from context in record")
	}
	if !foundUser {
		t.Error("expected user_id from context in record")
	}
	if !foundExtra {
		t.Error("expected extra from log call in record")
	}
}

func TestContextHandler_Handle_IgnoresNilValues(t *testing.T) {
	cap := &captureHandler{}
	keys := []ContextKey{"missing"}
	h := NewContextHandler(cap, keys)
	logger := slog.New(h)

	ctx := context.Background()
	logger.InfoContext(ctx, "test")

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	r := cap.records[0]
	count := 0
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "missing" {
			count++
		}
		return true
	})
	if count > 0 {
		t.Error("missing key should not add an attr when value is nil")
	}
}

func TestContextHandler_WithAttrs_PreservesContextExtraction(t *testing.T) {
	cap := &captureHandler{}
	h := NewContextHandler(cap, []ContextKey{"ctx_key"})
	wrapped := h.WithAttrs([]slog.Attr{slog.String("fixed", "attr")})
	logger := slog.New(wrapped)

	ctx := context.WithValue(context.Background(), ContextKey("ctx_key"), "ctx_value")
	logger.InfoContext(ctx, "msg")

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	r := cap.records[0]
	var foundCtx bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "ctx_key" && a.Value.String() == "ctx_value" {
			foundCtx = true
		}
		return true
	})
	if !foundCtx {
		t.Error("WithAttrs wrapper should still add context attrs to the record")
	}
}

func TestContextHandler_WithGroup_PreservesContextExtraction(t *testing.T) {
	var buf strings.Builder
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := NewContextHandler(inner, []ContextKey{"ctx_key"})
	wrapped := h.WithGroup("g")
	logger := slog.New(wrapped)

	ctx := context.WithValue(context.Background(), ContextKey("ctx_key"), "v")
	logger.InfoContext(ctx, "msg")

	out := buf.String()
	if !strings.Contains(out, "ctx_key=v") {
		t.Errorf("WithGroup wrapper should still add context attr, got: %s", out)
	}
	if !strings.Contains(out, "g.") || !strings.Contains(out, "msg=msg") {
		t.Errorf("group should be applied, got: %s", out)
	}
}

func TestNewContextHandler_NilInner_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewContextHandler(nil, keys) should panic")
		}
	}()
	NewContextHandler(nil, []ContextKey{"k"})
}
