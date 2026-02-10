// Package clog provides contextual logging compatible with log/slog.
//
// It combines two mechanisms:
//
//  1. Context Handler — wrap any slog.Handler with NewContextHandler(inner, keys).
//     For each log record, values for the given context keys are read from
//     context.Context and added as attributes. Use context.WithValue to set
//     values (e.g. trace_id, user_id) and pass the same context to logging
//     calls.
//
//  2. Logger in context — store a *slog.Logger in the context with NewContext,
//     retrieve it with FromContext. In middleware, create a request-scoped
//     logger (e.g. With("request_id", id)) and attach it to the context so
//     handlers can call Info(ctx, "message") without passing attributes every time.
//
// Example setup:
//
//	h := clog.NewContextHandler(slog.NewJSONHandler(os.Stdout, nil),
//	    []clog.ContextKey{"trace_id", "user_id"})
//	slog.SetDefault(slog.New(h))
//
// Example in HTTP middleware:
//
//	ctx = clog.NewContext(ctx, clog.FromContext(ctx).With("request_id", id, "path", r.URL.Path))
//
// Example in a handler:
//
//	clog.Info(ctx, "request started")
//	clog.Error(ctx, "operation failed", "error", err)
package clog
