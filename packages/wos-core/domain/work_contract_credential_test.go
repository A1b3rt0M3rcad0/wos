package domain_test

import (
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkContractCredentialBindingV1UsesRootCredentialAndFencing(t *testing.T) {
	w, unbound, now := contractFixture(t)
	holder := d.MustParseID("0199f207-0000-7000-8000-000000000001")
	session := d.MustParseID("0199f207-0000-7000-8000-000000000002")
	otherCredential := d.MustParseID("0199f207-0000-7000-8000-000000000003")
	otherPrincipal := "different-principal"
	c, err := d.NewWorkContractForCredential(unbound.ID, unbound.ExecutionID, w, unbound.HolderPrincipalID, holder, &session, unbound.Actor, unbound.LeasePolicy, 30, unbound.Spec, unbound.OutcomeRevisionAtAcquire, now)
	if err != nil {
		t.Fatal(err)
	}
	if c.CredentialBindingVersion != d.WorkContractCredentialBindingV1 || c.HolderCredentialID != holder || c.SessionCredentialID == nil || *c.SessionCredentialID != session {
		t.Fatalf("credential binding = version %d holder %s session %v", c.CredentialBindingVersion, c.HolderCredentialID, c.SessionCredentialID)
	}
	if err := c.Authorize(c.HolderPrincipalID, c.ExecutionID, c.FencingToken, now); err == nil {
		t.Fatal("principal-only authorization accepted a credential-bound contract")
	}
	if err := c.RenewForCredential(c.HolderPrincipalID, otherCredential, c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, now); err == nil {
		t.Fatal("different credential under the same principal renewed the contract")
	}
	if err := c.RenewForCredential(otherPrincipal, holder, c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, now); err == nil {
		t.Fatal("different principal with the holder credential renewed the contract")
	}
	if c.LeaseVersion != unbound.LeaseVersion {
		t.Fatal("rejected renewal changed the lease version")
	}
	if err := c.RenewForCredential(c.HolderPrincipalID, holder, c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, now); err != nil {
		t.Fatalf("holder credential renewal failed: %v", err)
	}

	priorExecution, priorFence := c.ExecutionID, c.FencingToken
	newExecution := d.MustParseID("0199f207-0000-7000-8000-000000000004")
	if err := c.ResumeForCredential(c.HolderPrincipalID, holder, priorExecution, priorFence, c.LeaseVersion, newExecution, uint64(priorFence), now); err != nil {
		t.Fatalf("holder credential could not resume: %v", err)
	}
	if c.ExecutionID != newExecution || c.FencingToken <= priorFence {
		t.Fatal("resume did not advance execution and fencing")
	}
	if err := c.AuthorizeForCredential(c.HolderPrincipalID, holder, priorExecution, priorFence, now); err == nil {
		t.Fatal("stale execution/fencing pair remained authorized")
	}
}

func TestWorkContractCredentialBindingValidationFailsClosed(t *testing.T) {
	_, contract, now := contractFixture(t)
	holder := d.MustParseID("0199f208-0000-7000-8000-000000000001")
	session := d.MustParseID("0199f208-0000-7000-8000-000000000002")
	valid, err := d.NewWorkContractForCredential(contract.ID, contract.ExecutionID, workForContractFixture(t, contract, now), contract.HolderPrincipalID, holder, &session, contract.Actor, contract.LeasePolicy, 30, contract.Spec, contract.OutcomeRevisionAtAcquire, now)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		change func(*d.WorkContract)
	}{
		{name: "missing holder", change: func(c *d.WorkContract) { c.HolderCredentialID = "" }},
		{name: "malformed holder", change: func(c *d.WorkContract) { c.HolderCredentialID = "not-an-id" }},
		{name: "malformed session", change: func(c *d.WorkContract) { bad := d.ID("not-an-id"); c.SessionCredentialID = &bad }},
		{name: "future version", change: func(c *d.WorkContract) { c.CredentialBindingVersion++ }},
		{name: "unversioned holder", change: func(c *d.WorkContract) { c.CredentialBindingVersion = 0 }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			candidate := valid
			tc.change(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("invalid credential binding validated")
			}
			if err := candidate.AuthorizeForCredential(candidate.HolderPrincipalID, holder, candidate.ExecutionID, candidate.FencingToken, now); err == nil {
				t.Fatal("invalid credential binding authorized execution")
			}
		})
	}
}

func workForContractFixture(t *testing.T, c d.WorkContract, now time.Time) d.WorkItem {
	t.Helper()
	w, err := d.NewWorkItem(c.WorkItemID, c.Scope, c.Spec.Title, "contract binding fixture", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	w.ContractsEnabled = true
	w.LastFencingToken = uint64(c.FencingToken - 1)
	return w
}
