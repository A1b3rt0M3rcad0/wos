package application_test

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestContractResultsRequireCurrentReviewedMaterial(t *testing.T) {
	s, _, _, w := contractServiceFixture(t)
	ctx := context.Background()
	cc := commandContext()
	cc.IdempotencyKey = "contract-criterion-1"
	_, err := s.AddCriterion(ctx, cc, a.AddCriterionCommand{Owner: w.Ref(), ExpectedVersion: w.Version, Title: "Review exact delivery", Required: true, VerificationMode: d.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "contract-acquire-results"
	got, err := s.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	authority := a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}
	cc.IdempotencyKey = "contract-checkpoint-results"
	syncCmd := a.SyncWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: c.Version, Checkpoint: a.ContractCheckpointInput{Summary: "Partial implementation", Pending: []string{"Review"}}}
	checkpoint, err := s.SyncWorkContract(ctx, cc, syncCmd)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.SyncWorkContract(ctx, cc, syncCmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Checkpoint.ID != checkpoint.Value.Checkpoint.ID {
		t.Fatalf("checkpoint replay %v", err)
	}
	if checkpoint.Value.WorkItem.Lifecycle != d.WorkItemLifecycleInProgress || checkpoint.Value.Contract.Status != d.ContractActive {
		t.Fatal("checkpoint completed work")
	}
	cc.IdempotencyKey = "contract-submit-results"
	submitCmd := a.SubmitWorkResultCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: checkpoint.Value.Contract.Version, Material: d.WorkResultMaterial{ContractID: c.ID, WorkItemID: w.ID, SpecDigest: c.SpecDigest, Summary: "Delivery one"}}
	submitted, err := s.SubmitWorkResult(ctx, cc, submitCmd)
	if err != nil {
		t.Fatal(err)
	}
	finalCmd := a.FinalizeWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: submitted.Value.Contract.Version, ExpectedWorkItemVersion: got.Value.WorkItem.Version, SubmissionID: submitted.Value.Submission.ID, Reason: "Reviewed delivery"}
	cc.IdempotencyKey = "contract-finalize-pending"
	if _, err = s.FinalizeWorkContract(ctx, cc, finalCmd); err == nil {
		t.Fatal("pending review accepted")
	}
	criterion := got.Value.WorkItem.Criteria.Items[0]
	assessmentCmd := a.AttestCriterionCommand{Owner: w.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision, ExpectedVersion: got.Value.WorkItem.Version, Result: d.AssessmentResultMet, Rationale: "Checked exact material", SubmissionID: &submitted.Value.Submission.ID}
	cc.IdempotencyKey = "contract-assess-results"
	assessed, err := s.AttestCriterion(ctx, cc, assessmentCmd)
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Value.SubmissionDigest != submitted.Value.Submission.Digest {
		t.Fatal("assessment binding missing")
	}
	submitCmd.ExpectedContractVersion = submitted.Value.Contract.Version
	submitCmd.Material.Summary = "Delivery two"
	submitCmd.SupersedesSubmissionID = &submitted.Value.Submission.ID
	cc.IdempotencyKey = "contract-submit-revised"
	revised, err := s.SubmitWorkResult(ctx, cc, submitCmd)
	if err != nil {
		t.Fatal(err)
	}
	finalCmd.SubmissionID = revised.Value.Submission.ID
	finalCmd.ExpectedContractVersion = revised.Value.Contract.Version
	finalCmd.ExpectedWorkItemVersion++
	cc.IdempotencyKey = "contract-finalize-obsolete"
	if _, err = s.FinalizeWorkContract(ctx, cc, finalCmd); err == nil {
		t.Fatal("obsolete assessment accepted")
	}
	assessmentCmd.SubmissionID = &revised.Value.Submission.ID
	assessmentCmd.ExpectedVersion = finalCmd.ExpectedWorkItemVersion
	cc.IdempotencyKey = "contract-assess-revised"
	if _, err = s.AttestCriterion(ctx, cc, assessmentCmd); err != nil {
		t.Fatal(err)
	}
	finalCmd.ExpectedWorkItemVersion++
	cc.IdempotencyKey = "contract-finalize-results"
	finished, err := s.FinalizeWorkContract(ctx, cc, finalCmd)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Value.Contract.Status != d.ContractCompleted || finished.Value.WorkItem.Lifecycle != d.WorkItemLifecycleDone || finished.Value.WorkItem.CurrentConclusion.SubmissionDigest != revised.Value.Submission.Digest {
		t.Fatal("completion binding incorrect")
	}
	replay, err = s.FinalizeWorkContract(ctx, cc, finalCmd)
	if err != nil || !replay.IdempotentReplay {
		t.Fatalf("finalize replay %v", err)
	}
}

func TestContractSyncRollbackOwnershipAndMaterialGuards(t *testing.T) {
	s, _, _, w := contractServiceFixture(t)
	ctx := context.Background()
	cc := commandContext()
	cc.IdempotencyKey = "sync-atomic-acquire"
	got, err := s.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	authority := a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}
	cc.IdempotencyKey = "sync-atomic-failure"
	artifact := a.RegisterArtifactCommand{Scope: w.Scope, ArtifactType: "file", Name: "deliverable", URI: "file:///external/deliverable", MediaType: "text/plain", Checksum: "sha256:exact", SourceVersion: "abc123"}
	cmd := a.SyncWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: authority, ExpectedContractVersion: c.Version, Checkpoint: a.ContractCheckpointInput{Summary: "Atomic progress"}, Artifacts: []a.SyncArtifactInput{{LocalKey: "artifact", Artifact: artifact}, {LocalKey: "artifact", Artifact: artifact}}}
	if _, err = s.SyncWorkContract(ctx, cc, cmd); err == nil {
		t.Fatal("duplicate local key accepted")
	}
	// Reusing the uncommitted intent proves that reservation and documentary writes rolled back.
	cmd.Artifacts = cmd.Artifacts[:1]
	synced, err := s.SyncWorkContract(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(synced.Value.Artifacts) != 1 || len(synced.Value.LocalKeys) != 1 {
		t.Fatal("atomic mapping incomplete")
	}
	replay, err := s.SyncWorkContract(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.LocalKeys["artifact"] != synced.Value.LocalKeys["artifact"] {
		t.Fatal("local key mapping not replay stable")
	}
	view, err := s.GetWorkContract(ctx, w.Scope, c.ID)
	if err != nil || view.Value.LatestCheckpoint == nil || view.Value.LatestCheckpoint.ID != synced.Value.Checkpoint.ID {
		t.Fatal("latest checkpoint missing")
	}
	cc.IdempotencyKey = "sync-other-principal"
	cc.PrincipalID = "other"
	cmd.ExpectedContractVersion = synced.Value.Contract.Version
	if _, err = s.SyncWorkContract(ctx, cc, cmd); err == nil {
		t.Fatal("nonholder sync accepted")
	}
	cc = commandContext()
	cc.IdempotencyKey = "guard-contract-update"
	title := "Changed obligation"
	if _, err = s.UpdateWorkItem(ctx, cc, a.UpdateWorkItemCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedVersion: got.Value.WorkItem.Version, Title: &title}); err == nil {
		t.Fatal("material update bypassed contract")
	}
	cc.IdempotencyKey = "guard-contract-archive"
	if _, err = s.ArchiveOutcome(ctx, cc, a.ArchiveOutcomeCommand{Scope: w.Scope, ExpectedVersion: 1}); err == nil {
		t.Fatal("outcome archive bypassed contract")
	}
}
