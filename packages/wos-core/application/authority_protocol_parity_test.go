package application_test

import (
	"context"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestContractsV1CredentialBindingMatchesSignedAuthorityOwnership(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	root, firstSession := f.addCredentialTree(t,
		"0199f209-0000-7000-8000-000000000001", "0199f20a-0000-7000-8000-000000000001",
		"contract-root", "contract-session-a", f.cc.PrincipalID, f.cc.Actor)
	secondSession := f.addSession(t, "0199f20a-0000-7000-8000-000000000002", "contract-session-b", root, f.cc.PrincipalID, f.cc.Actor)
	otherRoot := f.addRoot(t, "0199f209-0000-7000-8000-000000000003", "contract-other-root", f.cc.PrincipalID, f.cc.Actor)

	acquireContext := f.identityContext(firstSession)
	acquireCC := f.cc
	acquireCC.IdempotencyKey = "contracts-v1-acquire-binding"
	acquired, err := f.service.AcquireWorkContract(acquireContext, acquireCC, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 30,
	})
	if err != nil {
		t.Fatalf("AcquireWorkContract() error = %v", err)
	}
	contract := acquired.Value.Contract
	if contract.SignedBinding != nil || contract.CredentialBindingVersion != d.WorkContractCredentialBindingV1 || contract.HolderCredentialID != root.ID || contract.SessionCredentialID == nil || *contract.SessionCredentialID != firstSession.ID {
		t.Fatalf("contracts_v1 authority binding = %+v", contract)
	}
	acquireReplay, err := f.service.AcquireWorkContract(f.identityContext(secondSession), acquireCC, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 30,
	})
	if err != nil || !acquireReplay.IdempotentReplay || acquireReplay.Value.Contract.ID != contract.ID {
		t.Fatalf("same-root session could not reconcile acquisition: replay=%v err=%v", acquireReplay.IdempotentReplay, err)
	}
	_, err = f.service.AcquireWorkContract(f.identityContext(otherRoot), acquireCC, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 30,
	})
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeIdempotencyConflict {
		t.Fatalf("different root replayed a contracts_v1 acquisition receipt: code=%s err=%v", code, err)
	}

	// A different authenticated session under the same canonical credential
	// continues the original authority without rewriting its acquisition origin.
	renewCC := f.cc
	renewCC.IdempotencyKey = "contracts-v1-renew-same-root"
	renewed, err := f.service.RenewWorkContract(f.identityContext(secondSession), renewCC, a.RenewWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID,
		Authority:            a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedLeaseVersion: contract.LeaseVersion, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatalf("same-root session could not renew contracts_v1 authority: %v", err)
	}
	contract = renewed.Value.Contract
	if contract.HolderCredentialID != root.ID || contract.SessionCredentialID == nil || *contract.SessionCredentialID != firstSession.ID {
		t.Fatalf("renewal rewrote immutable credential provenance: %+v", contract)
	}

	wrongRenew := f.cc
	wrongRenew.IdempotencyKey = "contracts-v1-renew-other-root"
	_, err = f.service.RenewWorkContract(f.identityContext(otherRoot), wrongRenew, a.RenewWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID,
		Authority:            a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedLeaseVersion: contract.LeaseVersion, TTLSeconds: 60,
	})
	if code, _ := d.ErrorCodeOf(err); err == nil || code != d.ErrorCodeLease {
		t.Fatalf("same Principal with another credential renewed the contract: code=%s err=%v", code, err)
	}
	_, err = f.service.RenewWorkContract(context.Background(), f.cc, a.RenewWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID,
		Authority:            a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedLeaseVersion: contract.LeaseVersion, TTLSeconds: 60,
	})
	if code, _ := d.ErrorCodeOf(err); err == nil || code == "" {
		t.Fatalf("principal-only embedded caller used credential-bound contract: code=%s err=%v", code, err)
	}

	result := d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: contract.WorkItemID, SpecDigest: contract.SpecDigest, Summary: "Result with an auditable handoff"}
	submitCC := f.cc
	submitCC.IdempotencyKey = "contracts-v1-submit-bound-result"
	submit := a.SubmitWorkResultCommand{Scope: f.scope, ContractID: contract.ID, Authority: a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest}, ExpectedContractVersion: contract.Version, Material: result}
	wrongReplayCC := submitCC
	// The Principal, command ID, idempotency key, and payload are identical;
	// only the trusted credential differs. The E25 receipt guard must keep this
	// replay from disclosing or reusing another credential's accepted result.
	_, err = f.service.SubmitWorkResult(f.identityContext(secondSession), submitCC, submit)
	if err != nil {
		t.Fatalf("same-root session could not submit result: %v", err)
	}
	replayed, err := f.service.SubmitWorkResult(f.identityContext(firstSession), submitCC, submit)
	if err != nil || !replayed.IdempotentReplay {
		t.Fatalf("same root could not reconcile its frozen submit: replay=%v err=%v", replayed.IdempotentReplay, err)
	}
	_, err = f.service.SubmitWorkResult(f.identityContext(otherRoot), wrongReplayCC, submit)
	if err == nil {
		// The receipt guard intentionally rejects the other credential before
		// the command closure, so it cannot disclose the accepted result.
		t.Fatal("same Principal with another credential replayed the contract submission")
	}

	contract = replayed.Value.Contract
	wrongFinalizeCC := f.cc
	wrongFinalizeCC.IdempotencyKey = "contracts-v1-finalize-other-root"
	_, err = f.service.FinalizeWorkContract(f.identityContext(otherRoot), wrongFinalizeCC, a.FinalizeWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID, Authority: a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedContractVersion: contract.Version, ExpectedWorkItemVersion: replayed.Value.WorkItem.Version,
		SubmissionID: replayed.Value.Submission.ID, Reason: "must remain credential-bound",
	})
	if err == nil {
		t.Fatal("same Principal with another credential finalized the contract")
	}
	finalizeCC := f.cc
	finalizeCC.IdempotencyKey = "contracts-v1-finalize-same-root"
	finalized, err := f.service.FinalizeWorkContract(f.identityContext(secondSession), finalizeCC, a.FinalizeWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID, Authority: a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedContractVersion: contract.Version, ExpectedWorkItemVersion: replayed.Value.WorkItem.Version,
		SubmissionID: replayed.Value.Submission.ID, Reason: "the bound executor finalized the result",
	})
	if err != nil {
		t.Fatalf("same-root session could not finalize its submitted result: %v", err)
	}
	if finalized.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone {
		t.Fatal("valid contracts_v1 handoff did not complete the work item")
	}
}

func TestHistoricalUnboundContractsV1RemainEmbeddedOnly(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	root := f.addRoot(t, "0199f209-0000-7000-8000-000000000010", "contract-historical-root", f.cc.PrincipalID, f.cc.Actor)
	acquireCC := f.cc
	acquireCC.IdempotencyKey = "contracts-v1-historical-acquire"
	acquired, err := f.service.AcquireWorkContract(context.Background(), acquireCC, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := acquired.Value.Contract
	if contract.CredentialBindingVersion != 0 || !contract.HolderCredentialID.IsZero() {
		t.Fatalf("legacy embedded contract was silently rebound: %+v", contract)
	}
	renew := a.RenewWorkContractCommand{
		Scope: f.scope, ContractID: contract.ID,
		Authority:            a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedLeaseVersion: contract.LeaseVersion, TTLSeconds: 60,
	}
	authenticatedCC := f.cc
	authenticatedCC.IdempotencyKey = "contracts-v1-historical-auth-renew"
	_, err = f.service.RenewWorkContract(f.identityContext(root), authenticatedCC, renew)
	if code, _ := d.ErrorCodeOf(err); err == nil || code == "" {
		t.Fatalf("authenticated credential inherited an unbound historical contract: code=%s err=%v", code, err)
	}
	embeddedCC := f.cc
	embeddedCC.IdempotencyKey = "contracts-v1-historical-embedded-renew"
	if _, err := f.service.RenewWorkContract(context.Background(), embeddedCC, renew); err != nil {
		t.Fatalf("trusted embedded caller lost historical contract compatibility: %v", err)
	}
}

func markWorkContractEnabled(t *testing.T, f *credentialLeaseFixture) {
	t.Helper()
	ctx := context.Background()
	uow, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	work, err := uow.WorkItems().Get(ctx, f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := work.Version
	next, err := expected.Next()
	if err != nil {
		t.Fatal(err)
	}
	work.ContractsEnabled = true
	work.Version = next
	work.UpdatedAt = f.now
	if err := uow.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	f.work = work
}
