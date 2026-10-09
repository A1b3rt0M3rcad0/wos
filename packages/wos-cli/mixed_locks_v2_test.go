package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"testing"
	"time"
)

// This bounded native workspace experiment exercises actual local intention,
// maintenance and verified-receipt paths. SQL/network contention is measured
// separately; these profiles/receipts are explicit unit fixtures.
func TestSignedMixedLocalOperationsDoNotShareGlobalLock(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, cleanupProfile, token, cleanupFile, cleanupIntent, receipt := returnFixtureV2(t)
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.CreateV2(".wos/project.yaml", project))
	must(w.CreateV2(profilePathV2(cleanupProfile.Name), cleanupProfile))
	must(w.CreateV2(contractPathV2(cleanupProfile.Name, *cleanupIntent.ContractID), cleanupFile))
	_, acquisition, _ := profileFixtureV2(t, "acquire_worker")
	_, maintenance, _ := profileFixtureV2(t, "maintenance_worker")
	_, blocked, _ := profileFixtureV2(t, "blocked_worker")
	for _, p := range []ProfileV2{acquisition, maintenance, blocked} {
		must(w.CreateV2(profilePathV2(p.Name), p))
	}
	_, maintenanceFile := contractFixtureV2(t)
	maintenanceFile.Local.Profile = maintenance.Name
	maintenanceID := d.ID(maintenanceFile.Execution.Material.Result.ContractID)
	must(w.CreateV2(contractPathV2(maintenance.Name, maintenanceID), maintenanceFile))
	// A second receipt fixture in a distinct profile permits cleanup and recovery
	// to overlap without racing the same logical intention.
	_, recovery, _, recoveryFile, recoveryIntent, recoveryReceipt := returnFixtureV2(t)
	frozen, binding, e := decodeReturnIntentV2(recovery, token, recoveryIntent)
	must(e)
	recovery.Name = "recover_worker"
	recovery.Local.PendingOperations = nil
	must(recovery.SealBinding(token))
	must(sealFrozenV2(recovery, token, &frozen))
	recoveryFile.Local.Profile = recovery.Name
	recoveryFile.Local.Pending = &frozen
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}
	recoveryIntent, e = returnIntentV2(recovery, frozen, binding, actor)
	must(e)
	recoveryIntent.State = "accepted_confirmed"
	raw, e := json.Marshal(recoveryReceipt)
	must(e)
	recoveryIntent.Response = base64.StdEncoding.EncodeToString(raw)
	recovery.Local.PendingOperations = []PendingOperationV2{recoveryIntent}
	must(recovery.SealBinding(token))
	must(w.CreateV2(profilePathV2(recovery.Name), recovery))
	must(w.CreateV2(contractPathV2(recovery.Name, *recoveryIntent.ContractID), recoveryFile))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	held, e := w.LockV2(ctx, blocked.Name, "")
	must(e)
	defer held()
	start := make(chan struct{})
	results := make(chan error, 5)
	scope := d.Scope{NamespaceID: acquisition.Binding.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}
	acquire := a.AcquireNextSignedWorkContractCommand{Scope: scope, SignerKeyID: acquisition.Signing.KeyID, Limit: 25, TTLSeconds: 300}
	jobs := []func() error{
		func() error {
			_, e := prepareAcquisitionBatchV2(ctx, w, acquisition, token, "AcquireNextSignedWorkContract", scope, actor, acquire, 3)
			return e
		},
		func() error {
			_, e := prepareLeaseV2(ctx, w, maintenance, token, actor, maintenanceID, "execution", false, 300)
			return e
		},
		func() error { return cleanupReturnV2(ctx, w, cleanupProfile, token, cleanupIntent, receipt) },
		func() error {
			_, committed, e := recoverReturnV2(ctx, w, recovery, nil, token, recoveryIntent, false)
			if e == nil && !committed {
				return fmt.Errorf("confirmed receipt recovery lost acceptance")
			}
			return e
		},
	}
	for _, job := range jobs {
		go func(job func() error) { <-start; results <- job() }(job)
	}
	// This held profile must block only its own acquisition. Other profiles must
	// finish before it is released, proving absence of a project-wide lock.
	blockedResult := make(chan error, 1)
	go func() {
		<-start
		own, stop := context.WithTimeout(ctx, 200*time.Millisecond)
		defer stop()
		_, e := prepareAcquisitionBatchV2(own, w, blocked, token, "AcquireNextSignedWorkContract", scope, actor, acquire, 1)
		blockedResult <- e
	}()
	close(start)
	for range jobs {
		select {
		case e := <-results:
			must(e)
		case <-ctx.Done():
			t.Fatal("mixed independent operations deadlocked behind held profile")
		}
	}
	if e := <-blockedResult; e == nil {
		t.Fatal("held profile was bypassed")
	}
	held()
	current, e := w.LoadProfileV2(acquisition.Name)
	must(e)
	if len(current.Local.PendingOperations) != 3 {
		t.Fatal("concurrent acquisition lost its exact batch")
	}
	current, e = w.LoadProfileV2(maintenance.Name)
	must(e)
	if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].ContractID == nil || *current.Local.PendingOperations[0].ContractID != maintenanceID {
		t.Fatal("maintenance intention lost original CID")
	}
	for _, p := range []ProfileV2{cleanupProfile, recovery} {
		current, e := w.LoadProfileV2(p.Name)
		must(e)
		if len(current.Local.PendingOperations) != 0 {
			t.Fatal("confirmed cleanup left pending intention")
		}
		if _, e = w.ReadV2(contractPathV2(p.Name, *cleanupIntent.ContractID)); !os.IsNotExist(e) {
			t.Fatal("confirmed original contract remains", e)
		}
	}
	expected, e := EncodeV2Document(maintenanceFile)
	must(e)
	if _, _, _, bytes, e := loadContractV2(w, maintenance, maintenanceID); e != nil || signing.Digest(bytes) != signing.Digest(expected) {
		t.Fatal("unrelated original maintenance contract lost", e)
	}
}
