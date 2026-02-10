// Package http provides HTTP middleware for clog.
//
// Use Middleware to inject a request-scoped logger into the request context
// with request_id, http.method, and http.path (and optionally remote_addr).
// Handlers can then call clog.Info(ctx, "message") and get these attributes
// in every log line.
//
// Example (with cloghttp "github.com/muonsoft/clog/http"):
//
//	handler := cloghttp.Middleware(mux, &cloghttp.MiddlewareOptions{
//	    AddRemoteAddr: true,
//	    LogStart:      true,
//	    LogFinish:     true,
//	})
package http
