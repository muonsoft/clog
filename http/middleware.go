package http

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/muonsoft/clog"
)

// DefaultRequestIDHeader is the header name used to read and set request ID
// when MiddlewareOptions.RequestIDHeader is empty.
const DefaultRequestIDHeader = "X-Request-Id"

// MiddlewareOptions configures the HTTP middleware behavior.
type MiddlewareOptions struct {
	// RequestIDHeader is the header to read request ID from; if present, it is
	// used. If empty, DefaultRequestIDHeader ("X-Request-Id") is used.
	// If no ID is in the request, one is generated.
	RequestIDHeader string
	// ResponseHeader sets the response header for request ID when non-empty.
	// If empty, the same header as RequestIDHeader (or DefaultRequestIDHeader) is used.
	ResponseHeader string
	// AddRemoteAddr adds "remote_addr" to the logger when true.
	AddRemoteAddr bool
	// LogStart logs "request started" at Info when true.
	LogStart bool
	// LogFinish logs "request completed" at Info with duration and status when true.
	LogFinish bool
}

func (o *MiddlewareOptions) requestIDHeader() string {
	if o != nil && o.RequestIDHeader != "" {
		return o.RequestIDHeader
	}
	return DefaultRequestIDHeader
}

func (o *MiddlewareOptions) responseHeader() string {
	if o != nil && o.ResponseHeader != "" {
		return o.ResponseHeader
	}
	return o.requestIDHeader()
}

func (o *MiddlewareOptions) addRemoteAddr() bool {
	return o != nil && o.AddRemoteAddr
}

func (o *MiddlewareOptions) logStart() bool {
	return o == nil || o.LogStart
}

func (o *MiddlewareOptions) logFinish() bool {
	return o == nil || o.LogFinish
}

// Middleware returns an http.Handler that injects a request-scoped logger
// into the request context with request_id, http.method, and http.path
// (and optionally remote_addr). It can log request start/finish and set
// the request ID in the response header.
func Middleware(next http.Handler, opts *MiddlewareOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		reqIDHeader := opts.requestIDHeader()
		requestID := r.Header.Get(reqIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}
		respHeader := opts.responseHeader()
		if respHeader != "" {
			w.Header().Set(respHeader, requestID)
		}

		attrs := []any{
			"request_id", requestID,
			"http.method", r.Method,
			"http.path", r.URL.Path,
		}
		if opts.addRemoteAddr() {
			attrs = append(attrs, "remote_addr", r.RemoteAddr)
		}
		logger := clog.FromContext(ctx).With(attrs...)
		ctx = clog.NewContext(ctx, logger)

		if opts.logStart() {
			clog.Info(ctx, "request started")
		}

		r = r.WithContext(ctx)
		var rw http.ResponseWriter = w
		if opts.logFinish() {
			rw = &responseWriter{ResponseWriter: w, statusCode: 200, written: false}
		}
		next.ServeHTTP(rw, r)

		if opts.logFinish() {
			var statusCode int
			if rw, ok := rw.(*responseWriter); ok {
				statusCode = rw.statusCode
			}
			clog.Info(ctx, "request completed",
				"http.status", statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Flusher, http.Hijacker, and io.ReaderFrom by
// delegating to the underlying ResponseWriter when supported. This ensures
// compatibility with SSE, WebSocket upgrades, sendfile optimization, and
// other advanced HTTP features.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = 200
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher. It delegates to the underlying
// ResponseWriter if it supports flushing (required for SSE).
func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker. It delegates to the underlying
// ResponseWriter if it supports hijacking (required for WebSocket upgrades).
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("clog/http: underlying ResponseWriter does not implement http.Hijacker")
}

// ReadFrom implements io.ReaderFrom. It delegates to the underlying
// ResponseWriter if it supports ReadFrom (preserves sendfile optimization
// when serving files via io.Copy). If the underlying writer does not
// implement io.ReaderFrom, it falls back to a regular io.Copy.
func (w *responseWriter) ReadFrom(r io.Reader) (int64, error) {
	if !w.written {
		w.statusCode = 200
		w.written = true
	}
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	// Wrap in a plain io.Writer to prevent io.Copy from calling
	// this ReadFrom again (which would cause infinite recursion).
	return io.Copy(struct{ io.Writer }{w.ResponseWriter}, r)
}

// Unwrap returns the underlying ResponseWriter. This is used by
// http.ResponseController to access the original writer.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// generateRequestID returns a time-sortable ID (similar to UUID v7):
// 6 bytes = Unix timestamp in milliseconds (big-endian), 2 bytes = random.
// Lexicographic order matches chronological order.
//
// Collision probability depends only on the random part: 2 bytes = 65536
// values per millisecond. Approximate probability of at least one collision
// when n IDs are generated within the same millisecond: P ≈ n²/(2·65536).
// Examples: ~10 req/ms → ~0.08%, ~100 req/ms → ~8%, ~256 req/ms → ~50%.
// For higher throughput, consider increasing the random part (e.g. 4 bytes).
func generateRequestID() string {
	b := make([]byte, 8)
	ms := time.Now().UnixMilli() & 0xFFFFFFFFFFFF // 48 bits
	binary.BigEndian.PutUint64(b, uint64(ms)<<16) // high 6 bytes used
	if _, err := rand.Read(b[6:8]); err != nil {
		// fallback: use nanos mod 65536 for uniqueness
		b[6] = byte(time.Now().UnixNano() >> 8)
		b[7] = byte(time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
