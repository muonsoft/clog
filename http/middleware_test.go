package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/muonsoft/clog"
)

func TestMiddleware_InjectsLoggerWithRequestID(t *testing.T) {
	var logBuf bytes.Buffer
	h := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	var gotRequestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := clog.FromContext(r.Context())
		if l == nil {
			t.Error("logger should be in context")
		}
		clog.Info(r.Context(), "in handler")
		// request_id is on the logger, so it will appear in the log
		gotRequestID = r.Header.Get(DefaultRequestIDHeader)
		if gotRequestID == "" {
			// we generated one, check response
			gotRequestID = w.Header().Get(DefaultRequestIDHeader)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(next, nil)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotRequestID == "" && rec.Header().Get(DefaultRequestIDHeader) == "" {
		t.Error("expected request ID in response header when not in request")
	}
	if logBuf.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestMiddleware_UsesRequestIDFromHeader(t *testing.T) {
	wantID := "client-request-id-123"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Middleware(next, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(DefaultRequestIDHeader, wantID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(DefaultRequestIDHeader); got != wantID {
		t.Errorf("expected response header %q, got %q", wantID, got)
	}
}

func TestMiddleware_CapturesStatusCode(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := Middleware(next, &MiddlewareOptions{LogStart: false, LogFinish: true})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("http.status=404")) {
		t.Errorf("expected log to contain http.status=404, got: %s", buf.String())
	}
}

func TestMiddleware_OptionsDisableLogStartAndFinish(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clog.Info(r.Context(), "only this")
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(next, &MiddlewareOptions{LogStart: false, LogFinish: false})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	if bytes.Contains([]byte(out), []byte("request started")) {
		t.Error("LogStart=false should not log request started")
	}
	if bytes.Contains([]byte(out), []byte("request completed")) {
		t.Error("LogFinish=false should not log request completed")
	}
	if !bytes.Contains([]byte(out), []byte("only this")) {
		t.Error("handler log should appear")
	}
}

func TestMiddleware_AddRemoteAddr(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clog.Info(r.Context(), "handler")
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(next, &MiddlewareOptions{AddRemoteAddr: true, LogStart: false, LogFinish: false})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !bytes.Contains(buf.Bytes(), []byte("remote_addr=192.168.1.1:12345")) {
		t.Errorf("expected remote_addr in log, got: %s", buf.String())
	}
}

func TestResponseWriter_Flush_DelegatesToUnderlying(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: 200}

	// httptest.ResponseRecorder implements http.Flusher
	rw.Flush()

	if !rec.Flushed {
		t.Error("Flush should delegate to underlying ResponseWriter")
	}
}

func TestResponseWriter_Flush_NoopWhenNotFlusher(t *testing.T) {
	rw := &responseWriter{ResponseWriter: &nonFlusherWriter{}, statusCode: 200}

	// Should not panic
	rw.Flush()
}

func TestResponseWriter_Hijack_DelegatesToUnderlying(t *testing.T) {
	hw := &fakeHijacker{}
	rw := &responseWriter{ResponseWriter: hw, statusCode: 200}

	conn, brw, err := rw.Hijack()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil || brw == nil {
		t.Error("Hijack should return non-nil conn and bufio.ReadWriter")
	}
}

func TestResponseWriter_Hijack_ErrorWhenNotHijacker(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: 200}

	_, _, err := rw.Hijack()
	if err == nil {
		t.Error("Hijack should return error when underlying writer is not a Hijacker")
	}
}

func TestResponseWriter_Unwrap_ReturnsUnderlying(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: 200}

	if rw.Unwrap() != rec {
		t.Error("Unwrap should return the underlying ResponseWriter")
	}
}

func TestResponseWriter_SSE_FlushInHandler(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(slog.Default())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to implement http.Flusher for SSE")
		}
		_, _ = fmt.Fprintf(w, "data: hello\n\n")
		flusher.Flush()
		clog.Info(r.Context(), "sse event sent")
	})

	handler := Middleware(next, &MiddlewareOptions{LogStart: false, LogFinish: true})
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if body != "data: hello\n\n" {
		t.Errorf("unexpected body: %q", body)
	}
	if !rec.Flushed {
		t.Error("expected recorder to be flushed via SSE handler")
	}
}

// nonFlusherWriter is an http.ResponseWriter that does NOT implement http.Flusher.
type nonFlusherWriter struct{}

func (w *nonFlusherWriter) Header() http.Header         { return http.Header{} }
func (w *nonFlusherWriter) Write(b []byte) (int, error) { return len(b), nil }
func (w *nonFlusherWriter) WriteHeader(int)             {}

// fakeHijacker implements http.ResponseWriter and http.Hijacker for testing.
type fakeHijacker struct {
	nonFlusherWriter
}

func (h *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c1, c2 := net.Pipe()
	_ = c2.Close()
	brw := bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1))
	return c1, brw, nil
}

func TestResponseWriter_ReadFrom_DelegatesToUnderlying(t *testing.T) {
	fr := &fakeReaderFrom{}
	rw := &responseWriter{ResponseWriter: fr, statusCode: 200}

	src := strings.NewReader("hello world")
	n, err := rw.ReadFrom(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 11 {
		t.Errorf("expected 11 bytes, got %d", n)
	}
	if !fr.called {
		t.Error("ReadFrom should delegate to underlying io.ReaderFrom")
	}
	if !rw.written {
		t.Error("ReadFrom should mark response as written")
	}
}

func TestResponseWriter_ReadFrom_FallbackWhenNotReaderFrom(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: 200}

	src := strings.NewReader("fallback data")
	n, err := rw.ReadFrom(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 13 {
		t.Errorf("expected 13 bytes, got %d", n)
	}
	if rec.Body.String() != "fallback data" {
		t.Errorf("expected %q, got %q", "fallback data", rec.Body.String())
	}
}

func TestResponseWriter_ReadFrom_SetsStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: 200}

	src := strings.NewReader("x")
	_, _ = rw.ReadFrom(src)

	if rw.statusCode != 200 {
		t.Errorf("expected status 200, got %d", rw.statusCode)
	}
	if !rw.written {
		t.Error("ReadFrom should mark response as written")
	}
}

// fakeReaderFrom implements http.ResponseWriter and io.ReaderFrom.
type fakeReaderFrom struct {
	nonFlusherWriter
	called bool
	buf    bytes.Buffer
}

func (f *fakeReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	f.called = true
	return io.Copy(&f.buf, r)
}

func TestGenerateRequestID_TimeSortable(t *testing.T) {
	ids := make([]string, 5)
	for i := range ids {
		ids[i] = generateRequestID()
		if i < len(ids)-1 {
			time.Sleep(2 * time.Millisecond) // ensure different ms for sort order
		}
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("IDs should be sortable by time: %q <= %q", ids[i-1], ids[i])
		}
	}
	for i, id := range ids {
		if len(id) != 16 {
			t.Errorf("id[%d] length: want 16, got %d", i, len(id))
		}
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("id[%d] should be hex: %q", i, id)
				break
			}
		}
	}
}
