package integration

import (
	"context"
	"errors"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"testing"
	"time"
)

type diagnosticClock struct{}

func (diagnosticClock) Now() time.Time { return time.Now().UTC() }

type diagnosticIDs struct{}

func (diagnosticIDs) NewID() (d.ID, error) {
	return d.MustParseID("0199d128-0000-7000-8000-000000000001"), nil
}

type failingDeliveryStore struct{ mode string }

func (f failingDeliveryStore) ClaimDelivery(ctx context.Context, _ time.Time, _ d.ID, _ time.Duration) (*ports.Delivery, error) {
	switch f.mode {
	case "panic":
		panic("private receiver / secret")
	case "cancel":
		<-ctx.Done()
		return nil, ctx.Err()
	default:
		return nil, errors.New("private database DSN / secret")
	}
}
func (failingDeliveryStore) FinishDelivery(context.Context, ports.Delivery, time.Time, string, bool, time.Time) error {
	return nil
}
func TestWorkerSupervisionDistinguishesFatalPanicAndCancellation(t *testing.T) {
	for _, mode := range []string{"fatal", "panic", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			diag := NewDiagnostics(true)
			w, e := New(failingDeliveryStore{mode}, diagnosticClock{}, diagnosticIDs{}, nil, Policy{})
			if e != nil {
				t.Fatal(e)
			}
			w.Diagnostics = diag
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- w.Run(ctx) }()
			if mode == "cancel" {
				time.AfterFunc(1200*time.Millisecond, cancel)
			}
			e = <-done
			status := diag.DeliveryWorkerStatus()
			if status.Running || !status.Started || status.Starts != 1 || status.LastTick == nil {
				t.Fatalf("worker exit not observed: %+v", status)
			}
			if mode == "cancel" {
				if e != nil || status.LastFailureClass != "" {
					t.Fatalf("intentional stop classified failure: %v %+v", e, status)
				}
			} else if e == nil || status.LastFailureClass == "" || strings.Contains(status.LastFailureClass, "secret") {
				t.Fatalf("fatal silently lost or leaked: %v %+v", e, status)
			}
			if mode == "panic" && !errors.Is(e, ErrWorkerPanic) {
				t.Fatal("panic not classified", e)
			}
		})
	}
}
