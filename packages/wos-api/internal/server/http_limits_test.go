package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/observability"
)

func transportConfig(t *testing.T) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "transport.db")
	cfg.HTTP.ReadHeaderTimeout = 80 * time.Millisecond
	cfg.HTTP.ReadTimeout = 240 * time.Millisecond
	cfg.HTTP.RequestTimeout = 120 * time.Millisecond
	cfg.HTTP.WriteTimeout = 400 * time.Millisecond
	cfg.HTTP.IdleTimeout = 200 * time.Millisecond
	return cfg
}
func serveTransport(t *testing.T, cfg Config) (*Runtime, string) {
	return serveTransportWithHandler(t, cfg, nil)
}
func serveTransportWithHandler(t *testing.T, cfg Config, configure func(*Runtime)) (*Runtime, string) {
	t.Helper()
	r, e := OpenRuntime(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if configure != nil {
		configure(r)
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		r.Close()
		t.Fatal(e)
	}
	listener := &admissionListener{Listener: l, slots: make(chan struct{}, cfg.Server.MaxConnections), observer: r.observer}
	done := make(chan error, 1)
	go func() { done <- r.http.Serve(listener) }()
	t.Cleanup(func() {
		r.http.Close()
		e := <-done
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			t.Error(e)
		}
		r.Close()
	})
	return r, "http://" + l.Addr().String()
}

func TestHTTPBoundsRejectInvalidConfiguration(t *testing.T) {
	for _, mutate := range []func(*Config){func(c *Config) { c.HTTP.ReadTimeout = 0 }, func(c *Config) { c.HTTP.ReadHeaderTimeout = c.HTTP.ReadTimeout + time.Second }, func(c *Config) { c.HTTP.WriteTimeout = c.HTTP.RequestTimeout }, func(c *Config) { c.HTTP.MaxHeaderBytes = 1 }, func(c *Config) { c.HTTP.MaxInflightRequests = 0 }, func(c *Config) { c.Server.MaxConnections = 1 }} {
		cfg := DefaultConfig()
		mutate(&cfg)
		if cfg.Validate() == nil {
			t.Fatal("invalid transport bound accepted")
		}
	}
	for _, value := range []string{"-1s", "0", "999999999999999999999s", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WOS_HTTP_READ_TIMEOUT", value)
			if ConfigFromEnv().Validate() == nil {
				t.Fatal("invalid environment deadline ignored")
			}
		})
	}
}

func TestHTTPRequestAdmissionReservesHealthAndReleasesAfterPanicCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTP.MaxInflightRequests = 1
	observer := observability.New(nil)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var mutations atomic.Int64
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			w.WriteHeader(200)
			return
		}
		if r.URL.Path == "/panic" {
			panic("controlled request panic")
		}
		mutations.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	h := admissionHTTP(cfg, observer, next)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "http://wos.test/mutate", nil).WithContext(ctx))
	}()
	<-entered
	overloaded := httptest.NewRecorder()
	h.ServeHTTP(overloaded, httptest.NewRequest("POST", "http://wos.test/mutate", nil))
	if overloaded.Code != 503 || overloaded.Header().Get("Retry-After") != "1" || mutations.Load() != 1 {
		t.Fatal("saturation started another mutation or omitted retry guidance")
	}
	for _, path := range []string{"/livez", "/readyz"} {
		probe := httptest.NewRecorder()
		h.ServeHTTP(probe, httptest.NewRequest("GET", "http://wos.test"+path, nil))
		if probe.Code != 200 {
			t.Fatal("API saturation consumed reserved health capacity")
		}
	}
	cancel()
	<-done
	func() {
		defer func() {
			if recover() == nil {
				t.Error("controlled panic not propagated")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "http://wos.test/panic", nil))
	}()
	close(release)
	free := httptest.NewRecorder()
	h.ServeHTTP(free, httptest.NewRequest("POST", "http://wos.test/mutate", nil))
	if free.Code == 503 || mutations.Load() != 2 {
		t.Fatal("cancel/panic leaked admission slot")
	}
	metrics := httptest.NewRecorder()
	observer.ServeHTTP(metrics, httptest.NewRequest("GET", "http://wos.test/metrics", nil))
	var counters map[string]any
	if e := json.Unmarshal(metrics.Body.Bytes(), &counters); e != nil {
		t.Fatal(e)
	}
	if counters["http_inflight"] != float64(0) || counters["http_admission_rejected"] != float64(1) || counters["http_canceled"] != float64(1) {
		t.Fatalf("incorrect bounded counters: %s", metrics.Body.String())
	}
}

func TestHTTPSlowHeadersBodyIdleAndOversizedHeaders(t *testing.T) {
	cfg := transportConfig(t)
	cfg.HTTP.MaxHeaderBytes = 1024
	_, base := serveTransport(t, cfg)
	address := strings.TrimPrefix(base, "http://")
	for _, mode := range []string{"headers", "body", "headers-too-large", "idle"} {
		t.Run(mode, func(t *testing.T) {
			conn, e := net.Dial("tcp", address)
			if e != nil {
				t.Fatal(e)
			}
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			switch mode {
			case "headers":
				_, e = io.WriteString(conn, "GET /livez HTTP/1.1\r\nHost: ")
			case "body":
				_, e = fmt.Fprintf(conn, "POST /api/v1/tenants/%s/outcomes HTTP/1.1\r\nHost: %s\r\nContent-Type: application/json\r\nIdempotency-Key: slow-body-not-a-complete-intention\r\nContent-Length: 100\r\n\r\n{\"title\":", cfg.Auth.LocalTenantID, address)
			case "headers-too-large":
				_, e = fmt.Fprintf(conn, "GET /livez HTTP/1.1\r\nHost: %s\r\nX-Large: %s\r\n\r\n", address, strings.Repeat("x", 16<<10))
			case "idle":
				_, e = fmt.Fprintf(conn, "GET /livez HTTP/1.1\r\nHost: %s\r\n\r\n", address)
				if e != nil {
					t.Fatal(e)
				}
				response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "GET"})
				if err != nil {
					t.Fatal(err)
				}
				_, err = io.Copy(io.Discard, response.Body)
				response.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			started := time.Now()
			response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "GET"})
			if response != nil {
				defer response.Body.Close()
			}
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				t.Fatalf("server did not bound %s: %v", mode, err)
			}
			if time.Since(started) > time.Second {
				t.Fatal("connection retained beyond configured bound")
			}
			if mode == "body" && (response == nil || response.StatusCode != 503) {
				t.Fatalf("slow body did not return preserved deadline classification: response=%v error=%v", response, err)
			}
			if mode == "headers-too-large" && (response == nil || response.StatusCode != 431) {
				t.Fatalf("oversized headers accepted: response=%v error=%v", response, err)
			}
		})
	}
}

func TestHTTPRejectsBodyBeforeMutationAndBoundsConnections(t *testing.T) {
	cfg := transportConfig(t)
	cfg.HTTP.MaxInflightRequests = 1
	cfg.Server.MaxConnections = 3
	r, base := serveTransport(t, cfg)
	request, e := http.NewRequest("POST", base+"/api/v1/commands/create_outcome", strings.NewReader("{}"))
	if e != nil {
		t.Fatal(e)
	}
	request.ContentLength = maxRequestBodyBytes + 1
	// A real socket advertises a body without sending it. Rejection must happen
	// at admission, before reading payload or invoking a domain command.
	conn, e := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if e != nil {
		t.Fatal(e)
	}
	fmt.Fprintf(conn, "POST /api/v1/commands/create_outcome HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\n\r\n", strings.TrimPrefix(base, "http://"), request.ContentLength)
	response, e := http.ReadResponse(bufio.NewReader(conn), request)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	conn.Close()
	if response.StatusCode != 413 {
		t.Fatal("oversized Content-Length was not rejected before body read")
	}
	// Exercise connection admission directly with real TCP sockets. HTTP request
	// slots cannot bound clients which have not sent complete headers yet.
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	limited := &admissionListener{Listener: l, slots: make(chan struct{}, 1), observer: r.observer}
	first, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	accepted, e := limited.Accept()
	if e != nil {
		t.Fatal(e)
	}
	second, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	done := make(chan struct{})
	go func() { defer close(done); _, _ = limited.Accept() }()
	_ = second.SetReadDeadline(time.Now().Add(time.Second))
	_, e = second.Read(make([]byte, 1))
	if e == nil {
		t.Fatal("extra connection admitted")
	}
	if timeout, ok := e.(net.Error); ok && timeout.Timeout() {
		t.Fatal("extra connection left pending indefinitely")
	}
	accepted.Close()
	accepted.Close()
	l.Close()
	<-done
}
