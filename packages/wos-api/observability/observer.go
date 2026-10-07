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
	Logger         *slog.Logger
	mu             sync.Mutex
	commands       map[string]uint64
	commandTiming  map[string]timing
	queryTiming    map[string]timing
	deliveryTiming map[string]timing
	httpTiming     timing
	replays        atomic.Uint64
	conflicts      atomic.Uint64
	requests       atomic.Uint64
	queries        atomic.Uint64
	errors         atomic.Uint64
}

func New(logger *slog.Logger) *Observer {
	return &Observer{Logger: logger, commands: map[string]uint64{}, commandTiming: map[string]timing{}, queryTiming: map[string]timing{}, deliveryTiming: map[string]timing{}}
}
func (o *Observer) ObserveCommand(c ports.CommandObservation) {
	o.mu.Lock()
	o.commands[c.Name]++
	t := o.commandTiming[c.Name]
	t.add(c.Duration, c.ErrorCode != "")
	t.WaitNS += uint64(c.TransactionWait)
	t.CommitNS += uint64(c.CommitDuration)
	t.GuardNS += uint64(c.GuardDuration)
	o.commandTiming[c.Name] = t
	o.mu.Unlock()
	if c.Replay {
		o.replays.Add(1)
	}
	if c.ErrorCode == "version_conflict" || c.ErrorCode == "transaction_conflict" || c.ErrorCode == "idempotency_conflict" {
		o.conflicts.Add(1)
	}
	if o.Logger != nil {
		o.Logger.Info("command", "command", c.Name, "namespace_id", string(c.NamespaceID), "outcome_id", string(c.OutcomeID), "principal_id", c.PrincipalID, "actor_ref", c.Actor, "command_id", string(c.CommandID), "correlation_id", c.CorrelationID, "outcome_revision", c.Revision, "duration_ms", float64(c.Duration)/float64(time.Millisecond), "transaction_wait_ms", float64(c.TransactionWait)/float64(time.Millisecond), "commit_ms", float64(c.CommitDuration)/float64(time.Millisecond), "guard_ms", float64(c.GuardDuration)/float64(time.Millisecond), "error_code", c.ErrorCode, "idempotent_replay", c.Replay)
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
		o.mu.Lock()
		o.httpTiming.add(time.Since(start), observed.status >= 400)
		o.httpTiming.Bytes += uint64(observed.bytes)
		o.mu.Unlock()
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
	queryTiming := make(map[string]timing, len(o.queryTiming))
	for k, v := range o.queryTiming {
		queryTiming[k] = v
	}
	commandTiming := make(map[string]timing, len(o.commandTiming))
	for k, v := range o.commandTiming {
		commandTiming[k] = v
	}
	deliveryTiming := make(map[string]timing, len(o.deliveryTiming))
	for k, v := range o.deliveryTiming {
		deliveryTiming[k] = v
	}
	httpTiming := o.httpTiming
	o.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"schema_version": 1, "commands": commands, "command_timing": commandTiming, "query_timing": queryTiming, "delivery_timing": deliveryTiming, "http_timing": httpTiming, "idempotency_replays": o.replays.Load(), "conflicts": o.conflicts.Load(), "http_requests": o.requests.Load(), "http_get_requests": o.queries.Load(), "http_errors": o.errors.Load()})
}

// Cumulative nanoseconds/counts allow rates and mean latency to be computed
// without storing identifiers or high-cardinality documentary data as labels.
type timing struct {
	Count      uint64 `json:"count"`
	Errors     uint64 `json:"errors"`
	DurationNS uint64 `json:"duration_ns"`
	MaxNS      uint64 `json:"max_ns"`
	WaitNS     uint64 `json:"transaction_wait_ns,omitempty"`
	GuardNS    uint64 `json:"guard_ns,omitempty"`
	CommitNS   uint64 `json:"commit_ns,omitempty"`
	Bytes      uint64 `json:"response_bytes,omitempty"`
}

func (t *timing) add(d time.Duration, failed bool) {
	t.Count++
	if failed {
		t.Errors++
	}
	t.DurationNS += uint64(d)
	if uint64(d) > t.MaxNS {
		t.MaxNS = uint64(d)
	}
}
func (o *Observer) ObserveQuery(q ports.QueryObservation) {
	o.mu.Lock()
	t := o.queryTiming[q.Name]
	t.add(q.Duration, q.ErrorCode != "")
	o.queryTiming[q.Name] = t
	o.mu.Unlock()
	if o.Logger != nil {
		o.Logger.Info("query", "query", q.Name, "namespace_id", q.Scope.NamespaceID, "outcome_id", q.Scope.OutcomeID, "principal_id", q.PrincipalID, "actor_ref", q.Actor, "outcome_revision", q.Revision, "duration_ms", float64(q.Duration)/float64(time.Millisecond), "error_code", q.ErrorCode)
	}
}

func (o *Observer) ObserveDelivery(result string, duration time.Duration, acknowledged bool) {
	o.mu.Lock()
	t := o.deliveryTiming[result]
	t.add(duration, !acknowledged)
	o.deliveryTiming[result] = t
	o.mu.Unlock()
	if o.Logger != nil {
		o.Logger.Info("delivery", "result", result, "duration_ms", float64(duration)/float64(time.Millisecond), "acknowledged", acknowledged)
	}
}
