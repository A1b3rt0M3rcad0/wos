package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type verificationContractReadRepository struct {
	ports.WorkContractRepository
	submission d.WorkSubmission
	reads      int
}

func (r *verificationContractReadRepository) GetSubmission(_ context.Context, scope d.Scope, id d.ID) (d.WorkSubmission, error) {
	r.reads++
	if r.submission.Scope != scope || r.submission.ID != id {
		return d.WorkSubmission{}, d.NewError(d.ErrorCodeNotFound, "submission not found")
	}
	return r.submission, nil
}

type verificationContractReadUnit struct {
	ports.UnitOfWork
	repository ports.WorkContractRepository
}

func (u verificationContractReadUnit) WorkContracts() ports.WorkContractRepository {
	return u.repository
}
func (u verificationContractReadUnit) SignedWorkContracts() ports.WorkContractRepository {
	return u.repository
}
func (u verificationContractReadUnit) SignedContracts() ports.SignedContractRepository { return nil }

func TestTaskVerificationDeliveredSignedContractCannotRecoverOldAuthority(t *testing.T) {
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name        string
		status      d.ContractStatus
		lifecycle   d.WorkItemLifecycle
		retained    bool
		expiresAt   time.Time
		recoverable bool
		executable  bool
	}{
		{"delivered_changes_requested", d.ContractDelivered, d.WorkItemLifecycleInProgress, true, at.Add(-time.Second), false, false},
		{"delivered_history_after_replanning", d.ContractDelivered, d.WorkItemLifecycleTodo, false, at.Add(-time.Second), false, false},
		{"actual_expired_execution", d.ContractActive, d.WorkItemLifecycleInProgress, true, at, true, false},
		{"actual_live_execution", d.ContractActive, d.WorkItemLifecycleInProgress, true, at.Add(time.Minute), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := d.WorkItem{ID: verificationTestID(3), Scope: verificationTestScope(), Version: 7, Lifecycle: tc.lifecycle, ContractsEnabled: true, Criteria: d.NewCriterionSet()}
			c, err := d.NewWorkContract(verificationTestID(8), verificationTestID(17), w, "executor", verificationTestActor(), d.DefaultContractLeasePolicy(), 30, d.WorkContractSpec{Title: "Signed execution source"}, 98, at.Add(-2*time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			keyID := verificationTestID(16)
			c.SignedBinding = &d.SignedContractBinding{ProtocolVersion: 2, ServerID: "server", CredentialID: verificationTestID(15), SignerKeyID: keyID, PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, AllowedSigningKeyIDs: []d.ID{keyID}}
			c.Status, c.ExpiresAt = tc.status, tc.expiresAt
			if tc.retained {
				w.CurrentContractID = &c.ID
			}
			submission := d.WorkSubmission{ID: verificationTestID(9), Scope: w.Scope, Material: d.WorkResultMaterial{ContractID: c.ID, WorkItemID: w.ID, SpecDigest: c.SubmissionSpecDigest(), Summary: "Submitted result"}, Digest: verificationTestDigest("a"), PrincipalID: "executor", Actor: verificationTestActor(), SubmittedAt: at.Add(-time.Minute)}
			if tc.status == d.ContractDelivered {
				closed, caseID := at.Add(-time.Minute), verificationTestID(7)
				c.ClosedAt, c.ClosedBy, c.CloseReason, c.LatestSubmissionID = &closed, "executor", "Submitted for review", &submission.ID
				w.LatestReviewCaseID, w.CorrectionReviewCaseID = &caseID, &caseID
			}
			if err := c.Validate(); err != nil {
				t.Fatal(err)
			}
			beforeWork, beforeContract := w, c
			repo := &verificationContractReadRepository{submission: submission}
			result, err := renderWorkContractView(context.Background(), verificationContractReadUnit{repository: repo}, w.Scope, c, w, 99, at, d.Readiness{Ready: true})
			if err != nil {
				t.Fatal(err)
			}
			if result.Value.Recoverable != tc.recoverable || result.Value.ExecutionAllowed != tc.executable {
				t.Fatalf("wrong recovery/execution authority: %+v", result.Value)
			}
			if tc.status == d.ContractDelivered && (result.Value.EffectiveStatus != d.ContractDelivered || result.Value.LatestSubmission == nil || result.Value.LatestSubmission.ID != submission.ID || repo.reads != 1) {
				t.Fatal("delivered history must preserve the exact submission without acquiring authority")
			}
			if result.OutcomeRevision != 99 || result.Value.EvaluatedAt != at || !reflect.DeepEqual(w, beforeWork) || !reflect.DeepEqual(c, beforeContract) {
				t.Fatal("read projection mutated history or changed its snapshot")
			}
		})
	}
}
