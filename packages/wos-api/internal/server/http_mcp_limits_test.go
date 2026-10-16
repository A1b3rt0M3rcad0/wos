package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMCPJSONAndStandaloneStreamHaveBoundedDistinctDeadlines(t *testing.T) {
	cfg := transportConfig(t)
	cfg.MCP.Enabled = true
	cfg.MCP.Stateless = false
	cfg.HTTP.IdleTimeout = 600 * time.Millisecond
	_, base := serveTransport(t, cfg)
	client := &http.Client{Timeout: 3 * time.Second}
	post := func(body, session, version string) *http.Response {
		t.Helper()
		r, e := http.NewRequest("POST", base+cfg.MCP.Path, strings.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		if session != "" {
			r.Header.Set("Mcp-Session-Id", session)
		}
		if version != "" {
			r.Header.Set("Mcp-Protocol-Version", version)
		}
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		return response
	}
	initialized := post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"deadline-fixture","version":"1"}}}`, "", "")
	var hello struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if e := json.NewDecoder(initialized.Body).Decode(&hello); e != nil {
		initialized.Body.Close()
		t.Fatal(e)
	}
	initialized.Body.Close()
	if initialized.StatusCode != 200 || hello.Result.ProtocolVersion == "" {
		t.Fatal("MCP JSON initialize failed")
	}
	session := initialized.Header.Get("Mcp-Session-Id")
	if session == "" {
		t.Fatal("stateful transport did not issue protocol session")
	}
	notified := post(`{"jsonrpc":"2.0","method":"notifications/initialized"}`, session, hello.Result.ProtocolVersion)
	_, _ = io.Copy(io.Discard, notified.Body)
	notified.Body.Close()
	if notified.StatusCode != 202 {
		t.Fatalf("MCP initialize notification failed: %d", notified.StatusCode)
	}
	get, e := http.NewRequest("GET", base+cfg.MCP.Path, nil)
	if e != nil {
		t.Fatal(e)
	}
	get.Header.Set("Accept", "text/event-stream")
	get.Header.Set("Mcp-Session-Id", session)
	get.Header.Set("Mcp-Protocol-Version", hello.Result.ProtocolVersion)
	started := time.Now()
	stream, e := client.Do(get)
	if e != nil {
		t.Fatal(e)
	}
	if !stream.Close || stream.StatusCode != 200 || !strings.HasPrefix(stream.Header.Get("Content-Type"), "text/event-stream") {
		stream.Body.Close()
		t.Fatalf("MCP standalone stream unavailable: %d", stream.StatusCode)
	}
	_, _ = io.Copy(io.Discard, stream.Body)
	stream.Body.Close()
	elapsed := time.Since(started)
	if elapsed < cfg.HTTP.WriteTimeout || elapsed > 2*time.Second {
		t.Fatalf("stream used ordinary write/request timeout or remained unbounded: %s", elapsed)
	}
	ping := post(`{"jsonrpc":"2.0","id":2,"method":"ping"}`, session, hello.Result.ProtocolVersion)
	body, e := io.ReadAll(ping.Body)
	ping.Body.Close()
	if e != nil || ping.StatusCode != 200 || !strings.Contains(string(body), `"id":2`) {
		t.Fatalf("MCP JSON failed after bounded stream reconnect: %d %s %v", ping.StatusCode, body, e)
	}
}
