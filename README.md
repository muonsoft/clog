# clog

Contextual logging for Go, compatible with the standard `log/slog` package.

## Installation

```bash
go get github.com/muonsoft/clog
```

Requires Go 1.21+ (for `log/slog`). Optional: [github.com/muonsoft/errors](https://github.com/muonsoft/errors) for structured error logging.

## Features

- **Context handler** — inject attributes from `context.Context` into every log record (e.g. `trace_id`, `user_id`).
- **Logger in context** — attach a request-scoped logger to the context once (e.g. in middleware), then call `clog.Info(ctx, "message")` without passing attributes every time.
- **HTTP middleware** — subpackage `clog/http` provides middleware that adds `request_id`, `http.method`, `http.path` (and optionally `remote_addr`) to the context logger, with time-sortable request IDs and optional request start/finish logs.
- **Structured error logging** — integration with [muonsoft/errors](https://github.com/muonsoft/errors) for logging errors with attributes and stack traces at configurable levels.

## Quick start

```go
package main

import (
	"context"
	"os"

	"github.com/muonsoft/clog"
	"log/slog"
)

func main() {
	// Wrap your handler to add context attributes to every record
	h := clog.NewContextHandler(
		slog.NewJSONHandler(os.Stdout, nil),
		[]clog.ContextKey{"trace_id", "user_id"},
	)
	slog.SetDefault(slog.New(h))

	ctx := context.Background()
	ctx = context.WithValue(ctx, clog.ContextKey("trace_id"), "abc-123")
	ctx = clog.NewContext(ctx, clog.FromContext(ctx).With("request_id", "req-1"))

	clog.Info(ctx, "request started")
	clog.Info(ctx, "work done", "items", 42)
}
```

## Context API

| Function | Description |
|----------|-------------|
| `FromContext(ctx)` | Returns the logger stored in the context, or `slog.Default()` if none. |
| `NewContext(ctx, logger)` | Returns a copy of `ctx` that stores the given logger. |
| `With(ctx, args...)` | Returns a new context whose logger has the given attributes (e.g. `request_id`, `path`). |
| `WithGroup(ctx, name)` | Returns a new context whose logger starts a group with the given name. |

## Logging

Use the same context in handlers and services so logs carry the same attributes:

- `clog.Debug(ctx, msg, args...)`
- `clog.Info(ctx, msg, args...)`
- `clog.Warn(ctx, msg, args...)`
- `clog.Error(ctx, msg, args...)`
- `clog.Log(ctx, level, msg, args...)`
- `clog.LogAttrs(ctx, level, msg, attrs...)`

All use the logger from `FromContext(ctx)` and record the caller’s source location correctly.

## Error logging (muonsoft/errors)

When using [github.com/muonsoft/errors](https://github.com/muonsoft/errors):

- **`clog.Errorf(ctx, msg, args...)`** — builds an error with `errors.Errorf(msg, args...)` and logs it at Error level with attributes and stack trace.
- **`clog.ErrorLevel(ctx, err, level)`** — logs an existing error at the given `slog.Level`. Does nothing if `err` is nil.

```go
err := errors.Wrap(dbErr, slog.String("query", sql), slog.Int("id", id))
clog.ErrorLevel(ctx, err, slog.LevelWarn)
```

## HTTP middleware

Import the HTTP subpackage and wrap your handler:

```go
import (
	"net/http"
	cloghttp "github.com/muonsoft/clog/http"
)

mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	clog.Info(r.Context(), "handler")
	w.WriteHeader(http.StatusOK)
})

handler := cloghttp.Middleware(mux, &cloghttp.MiddlewareOptions{
	AddRemoteAddr: true,
	LogStart:      true,
	LogFinish:     true,
})
http.ListenAndServe(":8080", handler)
```

The middleware:

- Injects a request-scoped logger with `request_id`, `http.method`, `http.path` (and optionally `remote_addr`).
- Uses a **time-sortable** request ID (similar to UUID v7): 6 bytes timestamp + 2 bytes random, 16 hex chars. Reads or echoes `X-Request-Id` when provided.
- Optionally logs "request started" and "request completed" (with `duration_ms` and `http.status`).
- Wraps `http.ResponseWriter` so **SSE** (Flush), **WebSockets** (Hijack), and **sendfile** (ReadFrom) keep working.

## License

MIT. See [LICENSE](LICENSE).
