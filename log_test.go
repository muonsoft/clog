package clog

import (
	"context"
	"log/slog"
	"testing"
)

func TestInfo_WithLoggerInContext_RecordsLevelAndMessage(t *testing.T) {
	capture := &captureHandler{}
	logger := slog.New(capture)
	ctx := NewContext(context.Background(), logger)

	Info(ctx, "test message", "key", "value")

	if len(capture.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(capture.records))
	}
	r := capture.records[0]
	if r.Level != slog.LevelInfo {
		t.Errorf("expected level Info, got %v", r.Level)
	}
	if r.Message != "test message" {
		t.Errorf("expected message %q, got %q", "test message", r.Message)
	}
	var found bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "key" && a.Value.String() == "value" {
			found = true
		}
		return true
	})
	if !found {
		t.Error("expected key=value attr in record")
	}
}

func TestDebug_WhenDisabled_DoesNotCallHandler(t *testing.T) {
	capture := &captureHandler{}
	inner := &levelHandler{level: slog.LevelInfo, h: capture}
	logger := slog.New(inner)
	ctx := NewContext(context.Background(), logger)

	Debug(ctx, "should not appear")

	if len(capture.records) != 0 {
		t.Errorf("expected 0 records when Debug disabled, got %d", len(capture.records))
	}
}

func TestLogAttrs_WithLoggerInContext_RecordsAttrs(t *testing.T) {
	capture := &captureHandler{}
	logger := slog.New(capture)
	ctx := NewContext(context.Background(), logger)

	LogAttrs(ctx, slog.LevelWarn, "warn msg", slog.Int("n", 42), slog.String("s", "x"))

	if len(capture.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(capture.records))
	}
	r := capture.records[0]
	if r.Level != slog.LevelWarn {
		t.Errorf("expected level Warn, got %v", r.Level)
	}
	if r.Message != "warn msg" {
		t.Errorf("expected message %q, got %q", "warn msg", r.Message)
	}
	var nFound, sFound bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "n" && a.Value.Int64() == 42 {
			nFound = true
		}
		if a.Key == "s" && a.Value.String() == "x" {
			sFound = true
		}
		return true
	})
	if !nFound || !sFound {
		t.Error("expected n=42 and s=x attrs in record")
	}
}

func TestInfo_NilContext_UsesDefaultLogger(t *testing.T) {
	// Should not panic; uses slog.Default()
	Info(nil, "no panic")
}

// levelHandler wraps a handler and filters by level for Enabled.
type levelHandler struct {
	level slog.Level
	h     slog.Handler
}

func (h *levelHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.h.Handle(ctx, r)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{level: h.level, h: h.h.WithAttrs(attrs)}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{level: h.level, h: h.h.WithGroup(name)}
}
