package memory

import (
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestCloneWorkItemDeepCopiesSessionCredentialProvenance(t *testing.T) {
	sessionID := domain.MustParseID("0199fa42-0000-7000-8000-000000000009")
	item := domain.WorkItem{CurrentLease: &domain.WorkLease{SessionCredentialID: &sessionID}}
	cloned := cloneWorkItem(item)
	if cloned.CurrentLease == item.CurrentLease || cloned.CurrentLease.SessionCredentialID == item.CurrentLease.SessionCredentialID {
		t.Fatal("lease and session provenance pointers must be isolated")
	}
	*cloned.CurrentLease.SessionCredentialID = domain.MustParseID("0199fa42-0000-7000-8000-000000000010")
	if *item.CurrentLease.SessionCredentialID != sessionID {
		t.Fatal("mutating a cloned session credential changed the stored lease")
	}
}
