package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestHTTPDeadlineAfterCommitPreservesReceiptAndExactReplay(t *testing.T) {
	cfg := transportConfig(t)
	cfg.HTTP.RequestTimeout = time.Second
	cfg.HTTP.ReadTimeout = 2 * time.Second
	cfg.HTTP.WriteTimeout = 3 * time.Second
	var posts atomic.Int64
	var drop atomic.Bool
	drop.Store(true)
	committed := make(chan sdk.CommandResult[d.Outcome], 1)
	_, base := serveTransportWithHandler(t, cfg, func(r *Runtime) {
		original := r.http.Handler
		r.http.Handler = admissionHTTP(cfg, r.observer, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.Method != "POST" || !strings.HasSuffix(request.URL.Path, "/commands/create_outcome") {
				original.ServeHTTP(w, request)
				return
			}
			posts.Add(1)
			if !drop.Swap(false) {
				original.ServeHTTP(w, request)
				return
			}
			capture := httptest.NewRecorder()
			original.ServeHTTP(capture, request)
			var result sdk.CommandResult[d.Outcome]
			if capture.Code != 200 || json.Unmarshal(capture.Body.Bytes(), &result) != nil || result.CommandID.IsZero() {
				t.Errorf("actual mutation did not commit: %d %s", capture.Code, capture.Body.String())
				w.WriteHeader(500)
				return
			}
			committed <- result
			<-request.Context().Done()
			conn, _, e := http.NewResponseController(w).Hijack()
			if e != nil {
				t.Error(e)
				return
			}
			conn.Close()
		}))
	})
	client, e := sdk.New(base, "", &http.Client{Timeout: 4 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	// Prime a reused HTTP connection; the SDK must not interpret the idempotency
	// header as permission for an implicit transport-level mutation replay.
	if _, e = client.Capabilities(context.Background()); e != nil {
		t.Fatal(e)
	}
	cmd := a.CreateOutcomeCommand{NamespaceID: d.ID(cfg.Auth.LocalNamespaceID), Title: "deadline after commit", DesiredState: "one durable outcome", Priority: d.PriorityNormal}
	key := "http-deadline-original-intention"
	if _, e = client.CreateOutcome(context.Background(), key, cmd); e == nil {
		t.Fatal("lost response was falsely reported as confirmed")
	}
	if posts.Load() != 1 {
		t.Fatalf("SDK automatically replayed mutation: %d posts", posts.Load())
	}
	var first sdk.CommandResult[d.Outcome]
	select {
	case first = <-committed:
	case <-time.After(time.Second):
		t.Fatal("no committed result was captured")
	}
	receipt, e := client.GetCommandReceiptObservation(context.Background(), cmd.NamespaceID, first.CommandID)
	if e != nil {
		t.Fatal(e)
	}
	if receipt.CommandID != first.CommandID || receipt.OutcomeRevision != first.OutcomeRevision {
		t.Fatal("deadline lost durable command identity")
	}
	replayed, e := client.CreateOutcome(context.Background(), key, cmd)
	if e != nil {
		t.Fatal(e)
	}
	if !replayed.IdempotentReplay || replayed.Value.ID != first.Value.ID || replayed.CommandID != first.CommandID || replayed.OutcomeRevision != first.OutcomeRevision || posts.Load() != 2 {
		t.Fatal("explicit exact retry duplicated or replaced the accepted outcome")
	}
}

func TestHTTPShutdownForcesConnectionsAfterGrace(t *testing.T) {
	cfg := transportConfig(t)
	cfg.Server.ShutdownTimeout = 50 * time.Millisecond
	r, e := OpenRuntime(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	entered := make(chan struct{})
	handlerDone := make(chan struct{})
	r.http.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		close(entered)
		<-request.Context().Done()
		close(handlerDone)
	})
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverDone := make(chan error, 1)
	go func() { serverDone <- r.serve(ctx, listener) }()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		response, e := http.Get("http://" + listener.Addr().String())
		if e == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	}()
	<-entered
	started := time.Now()
	cancel()
	select {
	case e := <-serverDone:
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("forced shutdown did not preserve grace error: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown remained blocked")
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("forced close did not cancel handler")
	}
	select {
	case <-clientDone:
	case <-time.After(time.Second):
		t.Fatal("client connection remained open")
	}
	if time.Since(started) > time.Second {
		t.Fatal("shutdown exceeded bounded grace")
	}
}
