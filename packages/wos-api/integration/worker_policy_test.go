package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func eligibleSecrets(count int) map[string]string {
	secrets := make(map[string]string, count)
	for i := 0; i < count; i++ {
		secrets[fmt.Sprintf("reference-%03d", i)] = strings.Repeat("s", 32)
	}
	return secrets
}

func TestWorkerPolicyRejectsOverflowBeforeConstructionOrPoolStartup(t *testing.T) {
	secrets := eligibleSecrets(256)
	worker, err := New(failingDeliveryStore{}, diagnosticClock{}, diagnosticIDs{}, secrets, Policy{})
	if err != nil {
		t.Fatal("256 eligible references rejected", err)
	}
	secrets["overflow"] = strings.Repeat("s", 32)
	if _, err = New(failingDeliveryStore{}, diagnosticClock{}, diagnosticIDs{}, secrets, Policy{}); err == nil {
		t.Fatal("constructor accepted unsupported eligibility without truncation")
	}
	worker.Diagnostics = NewDiagnostics(true)
	if err = worker.Run(context.Background()); err == nil {
		t.Fatal("pool started with eligibility changed beyond its supported bound")
	}
	if status := worker.Diagnostics.DeliveryWorkerStatus(); status.Started || status.Running || status.Starts != 0 || status.LastTick != nil {
		t.Fatal("invalid worker became healthy or claimed a first delivery", status)
	}
}

func TestWorkerPolicyPreservesOpaqueLongReferencesAndCountsOnlyEligibleSecrets(t *testing.T) {
	longReference := strings.Repeat("opaque-reference/", 80)
	secrets := map[string]string{longReference: strings.Repeat("s", 32)}
	for i := 0; i < 300; i++ {
		secrets[fmt.Sprintf("ineligible-%03d", i)] = "short"
	}
	policy, err := workerClaimPolicy(secrets, ports.DeliveryClaimPolicy{})
	if err != nil || len(policy.SecretRefs) != 1 || policy.SecretRefs[0] != longReference {
		t.Fatal("eligible reference was truncated or opaque identifier rejected", err)
	}
	if _, err = New(failingDeliveryStore{}, diagnosticClock{}, diagnosticIDs{}, secrets, Policy{}); err != nil {
		t.Fatal("supported opaque identifier rejected during construction", err)
	}
	secrets[""] = strings.Repeat("s", 32)
	if err = ValidateWorkerPolicy(secrets, ports.DeliveryClaimPolicy{}); err == nil {
		t.Fatal("empty eligible reference accepted")
	}
}
