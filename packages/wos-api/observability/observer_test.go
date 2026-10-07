package observability

import (
	"bytes"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentMetricsAndLogsExcludeRequestContent(t *testing.T) {
	var output bytes.Buffer
	o := New(slog.New(slog.NewJSONHandler(&output, nil)))
	handler := o.HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); w.Write([]byte("private document")) }))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "http://localhost/api/v1/commands/create_outcome?secret=unlogged", strings.NewReader("private document"))
			req.Header.Set("Authorization", "Bearer unlogged-token")
			handler.ServeHTTP(httptest.NewRecorder(), req)
			o.ObserveCommand(ports.CommandObservation{Name: "CreateOutcome", Duration: time.Millisecond, TransactionWait: time.Microsecond})
			o.ObserveQuery(ports.QueryObservation{Name: "continuity", Duration: 2 * time.Millisecond})
			o.ObserveDelivery("interrupted", time.Millisecond, true)
		}()
	}
	wg.Wait()
	response := httptest.NewRecorder()
	o.ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	var v struct {
		Requests   uint64            `json:"http_requests"`
		Commands   map[string]timing `json:"command_timing"`
		Queries    map[string]timing `json:"query_timing"`
		Deliveries map[string]timing `json:"delivery_timing"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Requests != 20 || v.Commands["CreateOutcome"].Count != 20 || v.Queries["continuity"].Count != 20 || v.Deliveries["interrupted"].Count != 20 {
		t.Fatalf("lost concurrent observations: %+v", v)
	}
	for _, private := range []string{"private document", "unlogged-token", "secret=unlogged"} {
		if strings.Contains(output.String(), private) || strings.Contains(response.Body.String(), private) {
			t.Fatalf("observability exposed %q", private)
		}
	}
}
