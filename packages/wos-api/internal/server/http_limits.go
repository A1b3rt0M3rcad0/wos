package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/observability"
)

// Space for JSON-RPC/REST framing around the existing 256 KiB command bound.
// Individual command/session decoders retain their narrower semantic limits.
const maxRequestBodyBytes = (256 + 16) << 10

func admissionHTTP(cfg Config, observer *observability.Observer, next http.Handler) http.Handler {
	slots := make(chan struct{}, cfg.HTTP.MaxInflightRequests)
	health := make(chan struct{}, 2)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pool := slots
		probe := r.Method == http.MethodGet && (r.URL.Path == "/livez" || r.URL.Path == "/readyz")
		if probe {
			pool = health
		}
		select {
		case pool <- struct{}{}:
			observer.HTTPAdmitted()
			defer func() { <-pool; observer.HTTPReleased() }()
		default:
			observer.HTTPRejected()
			w.Header().Set("Connection", "close")
			w.Header().Set("Retry-After", "1")
			writeTransportError(w, http.StatusServiceUnavailable, "overloaded", "request admission is saturated; reconcile existing mutations before retrying")
			return
		}
		if r.ContentLength > maxRequestBodyBytes {
			w.Header().Set("Connection", "close")
			writeTransportError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request exceeds body limit")
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		timeout := cfg.HTTP.RequestTimeout
		if probe {
			timeout = min(timeout, 2*time.Second)
		}
		// Stateful Streamable HTTP can expose a standalone GET event stream.
		// Give that stream a dedicated bounded lifetime/socket deadline, rather
		// than cutting it at the ordinary mutation response deadline. Reconnect
		// carries only protocol state; no domain write is implicitly repeated.
		if cfg.MCP.Enabled && !cfg.MCP.Stateless && r.Method == http.MethodGet && r.URL.Path == cfg.MCP.Path {
			timeout = cfg.HTTP.IdleTimeout
			if r.ContentLength != 0 {
				writeTransportError(w, 400, "invalid_stream_request", "standalone stream GET requires an empty body")
				return
			}
			if e := http.NewResponseController(w).SetReadDeadline(time.Now().Add(timeout)); e != nil && !errors.Is(e, http.ErrNotSupported) {
				writeTransportError(w, 503, "stream_unavailable", "stream read deadline cannot be established")
				return
			}
			if e := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(timeout)); e != nil && !errors.Is(e, http.ErrNotSupported) {
				writeTransportError(w, 503, "stream_unavailable", "stream deadline cannot be established")
				return
			}
		}
		// A context deadline alone cannot interrupt a socket body read. Allow
		// a small timer propagation margin, while retaining the stricter server
		// read limit. Test recorders may not implement socket deadline control.
		if r.ContentLength != 0 {
			if e := http.NewResponseController(w).SetReadDeadline(time.Now().Add(min(cfg.HTTP.ReadTimeout, timeout+25*time.Millisecond))); e != nil && !errors.Is(e, http.ErrNotSupported) {
				writeTransportError(w, 503, "read_unavailable", "request read deadline cannot be established")
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		defer func() {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				observer.HTTPDeadline()
			} else if errors.Is(ctx.Err(), context.Canceled) {
				observer.HTTPCanceled()
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func writeTransportError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "message": message, "retryable": false}})
}

type admissionListener struct {
	net.Listener
	slots    chan struct{}
	observer *observability.Observer
}

func (l *admissionListener) Accept() (net.Conn, error) {
	for {
		conn, e := l.Listener.Accept()
		if e != nil {
			return nil, e
		}
		select {
		case l.slots <- struct{}{}:
			l.observer.ConnectionAdmitted()
			return &admissionConn{Conn: conn, observer: l.observer, release: func() { <-l.slots; l.observer.ConnectionReleased() }}, nil
		default:
			l.observer.ConnectionRejected()
			_ = conn.Close()
		}
	}
}

type admissionConn struct {
	net.Conn
	once        sync.Once
	release     func()
	timeoutOnce sync.Once
	observer    *observability.Observer
}

func (c *admissionConn) Close() error { e := c.Conn.Close(); c.once.Do(c.release); return e }

func (c *admissionConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.observeTimeout(e)
	return n, e
}
func (c *admissionConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.observeTimeout(e)
	return n, e
}
func (c *admissionConn) observeTimeout(e error) {
	var timeout net.Error
	if errors.As(e, &timeout) && timeout.Timeout() {
		c.timeoutOnce.Do(c.observer.ConnectionTimeout)
	}
}
