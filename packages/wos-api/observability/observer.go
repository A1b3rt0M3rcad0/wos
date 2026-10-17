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
	httpInflight        atomic.Int64
	httpRejected        atomic.Uint64
	httpTimeouts        atomic.Uint64
	httpCanceled        atomic.Uint64
	connectionsInflight atomic.Int64
	connectionsRejected atomic.Uint64
	connectionTimeouts  atomic.Uint64
	Logger              *slog.Logger
	DeliveryDiagnostics ports.DeliveryDiagnostics
	Storage             ports.StoragePoolObserver
	mu                  sync.Mutex
	commands            map[string]uint64
	commandTiming       map[string]timing
	queryTiming         map[string]timing
	deliveryTiming      map[string]timing
	httpTiming          timing
	replays             atomic.Uint64
	conflicts           atomic.Uint64
	requests            atomic.Uint64
	queries             atomic.Uint64
	errors              atomic.Uint64
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
		o.Logger.Info("command", "command", c.Name, "tenant_id", string(c.TenantID), "outcome_id", string(c.OutcomeID), "principal_id", c.PrincipalID, "actor_ref", c.Actor, "command_id", string(c.CommandID), "correlation_id", c.CorrelationID, "outcome_revision", c.Revision, "duration_ms", float64(c.Duration)/float64(time.Millisecond), "transaction_wait_ms", float64(c.TransactionWait)/float64(time.Millisecond), "commit_ms", float64(c.CommitDuration)/float64(time.Millisecond), "guard_ms", float64(c.GuardDuration)/float64(time.Millisecond), "error_code", c.ErrorCode, "idempotent_replay", c.Replay)
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
	var worker *ports.DeliveryWorkerStatus
	if o.DeliveryDiagnostics != nil {
		status := o.DeliveryDiagnostics.DeliveryWorkerStatus()
		worker = &status
	}
	var pool *ports.StoragePoolStats
	if o.Storage != nil {
		stats := o.Storage.ConnectionStats()
		pool = &stats
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"schema_version": 1, "storage_pool": pool, "duration_bucket_bounds_ns": durationBounds, "delivery_worker": worker, "http_inflight": o.httpInflight.Load(), "http_admission_rejected": o.httpRejected.Load(), "http_timeouts": o.httpTimeouts.Load(), "http_canceled": o.httpCanceled.Load(), "connections_inflight": o.connectionsInflight.Load(), "connections_rejected": o.connectionsRejected.Load(), "connection_timeouts": o.connectionTimeouts.Load(), "commands": commands, "command_timing": commandTiming, "query_timing": queryTiming, "delivery_timing": deliveryTiming, "http_timing": httpTiming, "idempotency_replays": o.replays.Load(), "conflicts": o.conflicts.Load(), "http_requests": o.requests.Load(), "http_get_requests": o.queries.Load(), "http_errors": o.errors.Load()})
}

// Cumulative nanoseconds/counts allow rates and mean latency to be computed
// without storing identifiers or high-cardinality documentary data as labels.
type timing struct {
	// Counts per bucket (not cumulative); bounds are exported once per response.
	Buckets       [15]uint64 `json:"duration_buckets"`
	Count         uint64     `json:"count"`
	Errors        uint64     `json:"errors"`
	DurationNS    uint64     `json:"duration_ns"`
	MaxNS         uint64     `json:"max_ns"`
	WaitNS        uint64     `json:"transaction_wait_ns,omitempty"`
	GuardNS       uint64     `json:"guard_ns,omitempty"`
	CommitNS      uint64     `json:"commit_ns,omitempty"`
	ReturnedItems uint64     `json:"returned_items,omitempty"`
	EmptyPages    uint64     `json:"empty_pages,omitempty"`
	Bytes         uint64     `json:"response_bytes,omitempty"`
}

func (t *timing) add(d time.Duration, failed bool) {
	index := len(durationBounds)
	for i, bound := range durationBounds {
		if d <= bound {
			index = i
			break
		}
	}
	t.Buckets[index]++
	t.Count++
	if failed {
		t.Errors++
	}
	t.DurationNS += uint64(d)
	if uint64(d) > t.MaxNS {
		t.MaxNS = uint64(d)
	}
}

var durationBounds = [...]time.Duration{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2500 * time.Millisecond, 5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second}

func (o *Observer) ObserveQuery(q ports.QueryObservation) {
	o.mu.Lock()
	t := o.queryTiming[q.Name]
	t.add(q.Duration, q.ErrorCode != "")
	if q.ReturnedItems != nil && q.ErrorCode == "" {
		t.ReturnedItems += uint64(*q.ReturnedItems)
		if *q.ReturnedItems == 0 {
			t.EmptyPages++
		}
	}
	o.queryTiming[q.Name] = t
	o.mu.Unlock()
	if o.Logger != nil {
		o.Logger.Info("query", "query", q.Name, "tenant_id", q.Scope.TenantID, "outcome_id", q.Scope.OutcomeID, "principal_id", q.PrincipalID, "actor_ref", q.Actor, "outcome_revision", q.Revision, "duration_ms", float64(q.Duration)/float64(time.Millisecond), "error_code", q.ErrorCode)
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

func (o *Observer) HTTPAdmitted()       { o.httpInflight.Add(1) }
func (o *Observer) HTTPReleased()       { o.httpInflight.Add(-1) }
func (o *Observer) HTTPRejected()       { o.httpRejected.Add(1) }
func (o *Observer) HTTPDeadline()       { o.httpTimeouts.Add(1) }
func (o *Observer) HTTPCanceled()       { o.httpCanceled.Add(1) }
func (o *Observer) ConnectionAdmitted() { o.connectionsInflight.Add(1) }
func (o *Observer) ConnectionReleased() { o.connectionsInflight.Add(-1) }
func (o *Observer) ConnectionRejected() { o.connectionsRejected.Add(1) }

func (o *Observer) ConnectionTimeout() { o.connectionTimeouts.Add(1) }
