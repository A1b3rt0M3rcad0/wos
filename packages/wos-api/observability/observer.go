package observability

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Observer struct {
	Logger    *slog.Logger
	mu        sync.Mutex
	commands  map[string]uint64
	replays   atomic.Uint64
	conflicts atomic.Uint64
	requests  atomic.Uint64
	queries   atomic.Uint64
	errors    atomic.Uint64
}

func New(logger *slog.Logger) *Observer {
	return &Observer{Logger: logger, commands: map[string]uint64{}}
}
func (o *Observer) ObserveCommand(c ports.CommandObservation) {
	o.mu.Lock()
	o.commands[c.Name]++
	o.mu.Unlock()
	if c.Replay {
		o.replays.Add(1)
	}
	if c.ErrorCode == "version_conflict" || c.ErrorCode == "transaction_conflict" || c.ErrorCode == "idempotency_conflict" {
		o.conflicts.Add(1)
	}
	if o.Logger != nil {
		o.Logger.Info("command", "command", c.Name, "namespace_id", string(c.NamespaceID), "outcome_id", string(c.OutcomeID), "principal_id", c.PrincipalID, "actor_ref", c.Actor, "command_id", string(c.CommandID), "correlation_id", c.CorrelationID, "outcome_revision", c.Revision, "duration_ms", float64(c.Duration)/float64(time.Millisecond), "transaction_wait_ms", float64(c.TransactionWait)/float64(time.Millisecond), "commit_ms", float64(c.CommitDuration)/float64(time.Millisecond), "error_code", c.ErrorCode, "idempotent_replay", c.Replay)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (o *Observer) HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		observed := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(observed, r)
		if observed.status == 0 {
			observed.status = 200
		}
		o.requests.Add(1)
		if r.Method == "GET" {
			o.queries.Add(1)
		}
		if observed.status >= 400 {
			o.errors.Add(1)
		}
		if o.Logger != nil {
			o.Logger.Info("request", "method", r.Method, "route", r.Pattern, "path", r.URL.Path, "status", observed.status, "bytes", observed.bytes, "correlation_id", observed.Header().Get("X-Correlation-ID"), "duration_ms", float64(time.Since(start))/float64(time.Millisecond))
		}
	})
}
func (o *Observer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	commands := map[string]uint64{}
	for name, count := range o.commands {
		commands[name] = count
	}
	o.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"schema_version": 1, "commands": commands, "idempotency_replays": o.replays.Load(), "conflicts": o.conflicts.Load(), "http_requests": o.requests.Load(), "http_get_requests": o.queries.Load(), "http_errors": o.errors.Load()})
}
