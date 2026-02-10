package clog

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// Debug logs at LevelDebug using the logger from ctx (or the default).
func Debug(ctx context.Context, msg string, args ...any) {
	log(ctx, FromContext(ctx), slog.LevelDebug, msg, args...)
}

// Info logs at LevelInfo using the logger from ctx (or the default).
func Info(ctx context.Context, msg string, args ...any) {
	log(ctx, FromContext(ctx), slog.LevelInfo, msg, args...)
}

// Warn logs at LevelWarn using the logger from ctx (or the default).
func Warn(ctx context.Context, msg string, args ...any) {
	log(ctx, FromContext(ctx), slog.LevelWarn, msg, args...)
}

// Error logs at LevelError using the logger from ctx (or the default).
func Error(ctx context.Context, msg string, args ...any) {
	log(ctx, FromContext(ctx), slog.LevelError, msg, args...)
}

// Log logs at the given level using the logger from ctx (or the default).
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	log(ctx, FromContext(ctx), level, msg, args...)
}

// LogAttrs logs at the given level with the given attrs using the logger
// from ctx (or the default). It is more efficient than Log when all
// arguments are already Attrs.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, FromContext(ctx), level, msg, attrs...)
}

// log is the internal implementation for variadic args. It must use a fixed
// call depth so the recorded pc points to the caller of Debug/Info/etc.
func log(ctx context.Context, l *slog.Logger, level slog.Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// logAttrs is the internal implementation for Attrs. Same call depth as log.
func logAttrs(ctx context.Context, l *slog.Logger, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
}
