# Changelog

## [v0.1.0] - 2025-02-10

First release. Contextual logging for Go compatible with `log/slog`.

### clog

- **Context handler** — `NewContextHandler(inner, keys)` wraps any `slog.Handler` and injects attributes from `context.Context` (by key) into every log record.
- **Logger in context** — `FromContext(ctx)`, `NewContext(ctx, logger)`, `With(ctx, args...)`, `WithGroup(ctx, name)` to attach and retrieve a request-scoped logger.
- **Logging** — `Debug`, `Info`, `Warn`, `Error`, `Log`, `LogAttrs` using the logger from context; caller source location is preserved.
- **Error logging** — `Errorf(ctx, msg, args...)` and `ErrorLevel(ctx, err, level)` for [muonsoft/errors](https://github.com/muonsoft/errors) (optional dependency).

### clog/http

- **Middleware** — Injects a request-scoped logger with `request_id`, `http.method`, `http.path` (optional `remote_addr`). Reads or generates request ID; supports `X-Request-Id` header.
- **Time-sortable request IDs** — 6 bytes timestamp (ms) + 2 bytes random, 16 hex chars; lexicographic order matches time order.
- **Options** — `LogStart` / `LogFinish` for request lifecycle logs; `AddRemoteAddr`; configurable request/response ID headers.
- **ResponseWriter** — Wrapper preserves `http.Flusher` (SSE), `http.Hijacker` (WebSocket), `io.ReaderFrom` (sendfile), and `Unwrap()` for `http.ResponseController`.

### Requirements

- Go 1.21+
- Optional: `github.com/muonsoft/errors` for `Errorf` / `ErrorLevel`

[v0.1.0]: https://github.com/muonsoft/clog/releases/tag/v0.1.0
