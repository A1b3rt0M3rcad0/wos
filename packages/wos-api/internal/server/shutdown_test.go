package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestGracefulShutdownDrainsInFlightResponse(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = t.TempDir() + "/shutdown.db"
	cfg.Server.ShutdownTimeout = 2 * time.Second
	reserve, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Server.Listen = reserve.Addr().String()
	reserve.Close()
	runtime, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	entered := make(chan struct{})
	release := make(chan struct{})
	original := runtime.http.Handler
	runtime.http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; original.ServeHTTP(w, r) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- runtime.Serve(ctx) }()
	// Wait for the listener without making a request through the slow handler.
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, e := net.DialTimeout("tcp", cfg.Server.Listen, 50*time.Millisecond)
		if e == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(e)
		}
		time.Sleep(5 * time.Millisecond)
	}
	result := make(chan error, 1)
	go func() {
		c := http.Client{Timeout: 3 * time.Second}
		r, e := c.Get("http://" + cfg.Server.Listen + "/readyz")
		if e == nil {
			r.Body.Close()
			if r.StatusCode != 200 {
				e = fmt.Errorf("readiness status %d", r.StatusCode)
			}
		}
		result <- e
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("request never entered")
	}
	cancel()
	select {
	case e := <-served:
		close(release)
		t.Fatalf("server returned before draining: %v", e)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if e := <-result; e != nil {
		t.Fatal("in-flight response lost", e)
	}
	select {
	case e := <-served:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
}
