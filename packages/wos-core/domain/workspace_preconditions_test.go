package domain_test

import (
	"errors"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
	"time"
)

func TestWorkspacePreconditionDetailsPreserveStableCode(t *testing.T) {
	id := d.ID("01992000-0000-7000-8000-000000000001")
	ns := d.ID("01992000-0000-7000-8000-000000000002")
	o, err := d.NewOutcome(id, ns, "A verifiable result", "", "A documented delivery", d.PriorityNormal, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	err = o.Activate(time.Now().UTC())
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodePreconditionFailed {
		t.Fatalf("code=%s error=%v", code, err)
	}
	var typed *d.Error
	if !errors.As(fmt.Errorf("transport: %w", err), &typed) || typed.Details == nil {
		t.Fatal("wrapped structured error missing")
	}
	if typed.Details.ReasonCode != "missing_required_criterion" || typed.Details.SubjectRef == nil || *typed.Details.SubjectRef != o.Ref() {
		t.Fatalf("details=%+v", typed.Details)
	}
	if len(typed.Details.Remedies) != 1 || typed.Details.Remedies[0].ActionID != "add_criterion" {
		t.Fatalf("remedies=%+v", typed.Details.Remedies)
	}
	if o.Lifecycle != d.OutcomeLifecycleDraft {
		t.Fatal("a preparation error changed the lifecycle")
	}
}
