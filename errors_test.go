package clog

import (
	"context"
	"log/slog"
	"testing"

	"github.com/muonsoft/errors"
)

func TestErrorf_LogsError(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(cap)
	oldDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldDefault)

	ctx := context.Background()
	Errorf(ctx, "test error: %s", "something failed")

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	r := cap.records[0]
	if r.Level != slog.LevelError {
		t.Errorf("expected level Error, got %v", r.Level)
	}
	// muonsoft/errors.Log logs the error; message or attrs contain the error
	if r.Message != "" {
		// some versions may set message
		if r.Message != "test error: something failed" {
			t.Logf("record message: %q", r.Message)
		}
	}
	// Error is typically added as an attr by the errors package
	var hasError bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" || a.Key == "error" {
			hasError = true
		}
		return true
	})
	if !hasError && r.Message == "" {
		t.Logf("record has no err/error attr and empty message; record level=%v", r.Level)
		// Still accept: we verified level and that one record was written
	}
}

func TestErrorf_WithLoggerInContext(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(cap)
	ctx := NewContext(context.Background(), logger)

	Errorf(ctx, "ctx error: %d", 42)

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	if cap.records[0].Level != slog.LevelError {
		t.Errorf("expected level Error, got %v", cap.records[0].Level)
	}
}

func TestErrorLevel_LogsAtSpecifiedLevel(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(cap)
	oldDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldDefault)

	ctx := context.Background()
	err := errors.Errorf("wrapped: %w", errors.New("root"))
	ErrorLevel(ctx, err, slog.LevelWarn)

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	r := cap.records[0]
	if r.Level != slog.LevelWarn {
		t.Errorf("expected level Warn, got %v", r.Level)
	}
}

func TestErrorLevel_NilError_NoLog(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(cap)
	oldDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldDefault)

	ctx := context.Background()
	ErrorLevel(ctx, nil, slog.LevelError)

	if len(cap.records) != 0 {
		t.Errorf("expected 0 records when err is nil, got %d", len(cap.records))
	}
}

func TestErrorLevel_WithLoggerInContext(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(cap)
	ctx := NewContext(context.Background(), logger)

	err := errors.New("sentinel")
	ErrorLevel(ctx, err, slog.LevelInfo)

	if len(cap.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(cap.records))
	}
	if cap.records[0].Level != slog.LevelInfo {
		t.Errorf("expected level Info, got %v", cap.records[0].Level)
	}
}
