package clog

import (
	"context"
	"log/slog"

	"github.com/muonsoft/errors"
)

// Errorf creates an error with errors.Errorf(msg, args...) and logs it at Error
// level using the logger from ctx (or the default). The error is logged with
// all structured attributes and stack trace from the muonsoft/errors chain.
func Errorf(ctx context.Context, msg string, args ...any) {
	err := errors.Errorf(msg, args...)
	errors.Log(ctx, FromContext(ctx), err)
}

// ErrorLevel logs err at the given slog level using the logger from ctx
// (or the default). The error is logged with all structured attributes and
// stack trace from the muonsoft/errors chain. If err is nil, nothing is logged.
func ErrorLevel(ctx context.Context, err error, level slog.Level) {
	errors.LogLevel(ctx, FromContext(ctx), level, err)
}
