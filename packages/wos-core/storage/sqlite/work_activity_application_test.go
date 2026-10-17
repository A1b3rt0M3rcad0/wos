package sqlite

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

// Run inside the authenticated acquisition fixture while its first authority is
// still live, so these are application invariants rather than fabricated grants.
func assertSignedWorkActivityApplication(t *testing.T, service *a.Service, store ports.TransactionManager, agentCtx context.Context, acquisitionCC d.CommandContext, acquired a.MutationResult[a.WorkContractResult], signer d.ID, clock sqliteFixedClock, ids *sqliteSequenceIDs) {
	t.Helper()
	ctx := context.Background()
	contract := acquired.Value.Contract
	scope := contract.Scope
	criterion := contract.Spec.Criteria[0]
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	id := func(n int) d.ID { return d.MustParseID(fmt.Sprintf("019a0002-0000-7000-8000-%012x", n)) }
	begin := func() ports.UnitOfWork {
		t.Helper()
		u, err := store.Begin(ctx)
		must(err)
		return u
	}
	commandContext := func(key string) d.CommandContext {
		t.Helper()
		cc := acquisitionCC
		var err error
		cc.CommandID, err = ids.NewID()
		must(err)
		cc.IdempotencyKey = key
		return cc
	}
	foreignScope := d.Scope{TenantID: scope.TenantID, OutcomeID: id(1)}
	foreignOutcome, err := d.NewOutcome(foreignScope.OutcomeID, scope.TenantID, "Isolated activity records", "", "Scope isolation", d.PriorityNormal, clock.now)
	must(err)
	foreignArtifact, err := d.NewArtifact(id(2), foreignScope, "source", "Foreign source", "git:foreign-commit", "", "", "foreign-commit", contract.Actor, nil, clock.now)
	must(err)
	foreignEvidence, err := d.NewEvidence(id(3), foreignScope, d.EvidenceTypeTestResult, "Foreign test run", d.SourceReference{Provider: "test", ID: "foreign-run"}, contract.Actor, clock.now, nil, nil, "foreign-commit", "", clock.now)
	must(err)
	withdrawn, err := d.NewArtifact(id(4), scope, "source", "Withdrawn source", "git:withdrawn-commit", "", "", "withdrawn-commit", contract.Actor, nil, clock.now)
	must(err)
	must(withdrawn.Withdraw("Superseded outside this activity", clock.now))
	retracted, err := d.NewEvidence(id(5), scope, d.EvidenceTypeTestResult, "Retracted test run", d.SourceReference{Provider: "test", ID: "retracted-run"}, contract.Actor, clock.now, nil, nil, "retracted-commit", "", clock.now)
	must(err)
	must(retracted.Retract("Invalid external observation", clock.now))
	u := begin()
	must(u.Outcomes().Insert(ctx, foreignOutcome))
	documents := u.(ports.DocumentaryUnitOfWork)
	must(documents.Artifacts().Insert(ctx, foreignArtifact))
	must(documents.Evidence().Insert(ctx, foreignEvidence))
	must(documents.Artifacts().Insert(ctx, withdrawn))
	must(documents.Evidence().Insert(ctx, retracted))
	// The new key is valid for this Principal and current credential policy,
	// but was never in the immutable authority issued to this contract.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	fingerprint, err := signing.Fingerprint(pub)
	must(err)
	foreignSigner := id(6)
	registry := u.(ports.SigningIdentityUnitOfWork).SigningIdentity()
	must(registry.SaveKey(ctx, d.SigningKey{ID: foreignSigner, TenantID: scope.TenantID, PrincipalID: contract.HolderPrincipalID, Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fingerprint, Version: 1, Status: "active", CreatedAt: clock.now, UpdatedAt: clock.now}, 0))
	policy, err := registry.CredentialPolicy(ctx, scope.TenantID, contract.SignedBinding.CredentialID)
	must(err)
	oldPolicyVersion := policy.Version
	policy.Version++
	policy.AllowedSigningKeyIDs = append(policy.AllowedSigningKeyIDs, foreignSigner)
	must(registry.SaveCredentialPolicy(ctx, policy, oldPolicyVersion))
	must(u.Commit())

	type stateSnapshot struct {
		Revision         d.OutcomeRevision
		Work             d.WorkItem
		Contract         d.WorkContract
		Activity         []d.WorkActivity
		Artifacts        []d.Artifact
		Evidence         []d.Evidence
		Links            []d.EvidenceLink
		ForeignArtifacts []d.Artifact
		ForeignEvidence  []d.Evidence
		Events           []d.DomainEvent
	}
	snapshot := func() stateSnapshot {
		t.Helper()
		var state stateSnapshot
		u := begin()
		coordination, err := u.Coordination().LockOutcome(ctx, scope)
		must(err)
		state.Revision = coordination.Revision
		state.Work, err = u.WorkItems().Get(ctx, scope, contract.WorkItemID)
		must(err)
		state.Contract, err = u.(ports.SignedContractUnitOfWork).SignedWorkContracts().Get(ctx, scope, contract.ID)
		must(err)
		state.Activity, err = u.(ports.WorkActivityUnitOfWork).WorkActivities().List(ctx, scope, ports.WorkActivityFilter{ThroughSequence: math.MaxUint64, Limit: 101})
		must(err)
		documents := u.(ports.DocumentaryUnitOfWork)
		state.Artifacts, err = documents.Artifacts().ListByOutcome(ctx, scope)
		must(err)
		state.Evidence, err = documents.Evidence().ListByOutcome(ctx, scope)
		must(err)
		state.Links, err = documents.EvidenceLinks().ListByOutcome(ctx, scope)
		must(err)
		state.ForeignArtifacts, err = documents.Artifacts().ListByOutcome(ctx, foreignScope)
		must(err)
		state.ForeignEvidence, err = documents.Evidence().ListByOutcome(ctx, foreignScope)
		must(err)
		must(u.Rollback())
		switch adapter := store.(type) {
		case *Store:
			state.Events, err = adapter.SnapshotDomainEvents(ctx, scope)
			must(err)
		case *memory.Store:
			state.Events = adapter.SnapshotDomainEvents(scope)
		default:
			t.Fatalf("activity fixture does not support event snapshots: %T", store)
		}
		return state
	}
	const sourceVersion = "0123456789abcdef0123456789abcdef01234567"
	const checksum = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	command := func() a.AppendWorkActivityCommand {
		return a.AppendWorkActivityCommand{
			Scope: scope, ContractID: contract.ID, SignerKeyID: signer,
			Authority:       a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SignedBinding.SpecificationDigest},
			AuthorityDigest: signing.Digest(acquired.Value.IssuedAuthority.Payload), ExpectedContractVersion: contract.Version,
			Type: d.WorkActivityValidation, Description: "External tests passed for the exact source revision; independent review is still pending.",
			Artifacts:     []a.SyncArtifactInput{{LocalKey: "build", Artifact: a.RegisterArtifactCommand{Scope: scope, ArtifactType: "source", Name: "Exact tested source", URI: "git:" + sourceVersion, SourceVersion: sourceVersion, Checksum: checksum}}},
			Evidence:      []a.SyncEvidenceInput{{LocalKey: "tests", ArtifactLocalKey: "build", Evidence: a.RegisterEvidenceCommand{Scope: scope, EvidenceType: d.EvidenceTypeTestResult, Description: "External test results", SourceRef: d.SourceReference{Provider: "test", ID: "activity-test-run"}, CapturedAt: clock.now, SourceVersion: sourceVersion}}},
			EvidenceLinks: []a.SyncEvidenceLinkInput{{LocalKey: "criterion-link", EvidenceLocalKey: "tests", Link: a.SyncEvidenceLinkRecord{Scope: scope, TargetRef: acquired.Value.WorkItem.Ref(), CriterionID: &criterion.ID, Stance: d.EvidenceStanceSupports, Rationale: "Observed tests support this frozen criterion; this is not an assessment"}}},
		}
	}
	before := snapshot()
	cc := commandContext("work-activity-inline-records")
	cmd := command()
	result, err := service.AppendWorkActivity(agentCtx, cc, cmd)
	must(err)
	if len(result.Value.LocalKeys) != 3 || len(result.Value.Artifacts) != 1 || len(result.Value.Evidence) != 1 || len(result.Value.EvidenceLinks) != 1 {
		t.Fatal("activity did not atomically register all local-key records")
	}
	artifact, evidence, link := result.Value.Artifacts[0], result.Value.Evidence[0], result.Value.EvidenceLinks[0]
	activity := result.Value.Activity
	if result.Value.LocalKeys["build"] != artifact.ID || result.Value.LocalKeys["tests"] != evidence.ID || result.Value.LocalKeys["criterion-link"] != link.ID || evidence.ArtifactID == nil || *evidence.ArtifactID != artifact.ID || link.EvidenceID != evidence.ID || link.TargetRef != acquired.Value.WorkItem.Ref() {
		t.Fatal("inline documentary local-key references were not bound together")
	}
	if !reflect.DeepEqual(activity.ArtifactRefs, []d.ID{artifact.ID}) || !reflect.DeepEqual(activity.EvidenceRefs, []d.ID{evidence.ID}) || !reflect.DeepEqual(activity.CriterionRefs, []d.WorkActivityCriterionRef{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision}}) || !reflect.DeepEqual(activity.ArtifactVersions, []d.SubmissionArtifact{{ArtifactID: artifact.ID, SourceVersion: sourceVersion, Checksum: checksum}}) {
		t.Fatalf("activity lost evidence, frozen criteria, or exact artifact version: %#v", activity)
	}
	if result.IdempotentReplay || result.OutcomeRevision != before.Revision+1 || activity.Sequence != uint64(result.OutcomeRevision) || activity.PrincipalID != contract.HolderPrincipalID || activity.Actor != contract.Actor || activity.RecordedAt.IsZero() {
		t.Fatal("activity chronology/provenance differs from its committed command")
	}
	baseline := snapshot()
	if !reflect.DeepEqual(before.Work, baseline.Work) || !reflect.DeepEqual(before.Contract, baseline.Contract) || baseline.Work.CurrentConclusion != nil || len(baseline.Work.Criteria.Assessments) != 0 || len(baseline.Work.Criteria.CurrentAssessments) != 0 {
		t.Fatal("activity altered task/criterion approval, contract versions, execution, or lease")
	}
	if len(baseline.Events) != len(before.Events)+4 {
		t.Fatal("activity and inline documents were not recorded in the same event batch")
	}
	var retryCC d.CommandContext
	probes := []struct {
		name   string
		code   d.ErrorCode
		mutate func(*a.AppendWorkActivityCommand)
	}{
		{"authority digest", d.ErrorCodeContractSpecMismatch, func(c *a.AppendWorkActivityCommand) {
			c.AuthorityDigest = signing.Digest([]byte("different issued authority"))
		}},
		{"execution", d.ErrorCodeContractSpecMismatch, func(c *a.AppendWorkActivityCommand) { c.Authority.ExecutionID = id(10) }},
		{"fencing", d.ErrorCodeContractSpecMismatch, func(c *a.AppendWorkActivityCommand) { c.Authority.FencingToken++ }},
		{"contract version", d.ErrorCodeVersionConflict, func(c *a.AppendWorkActivityCommand) { c.ExpectedContractVersion++ }},
		{"unissued signer", d.ErrorCodeForbidden, func(c *a.AppendWorkActivityCommand) { c.SignerKeyID = foreignSigner }},
		{"criterion revision", d.ErrorCodeCriterion, func(c *a.AppendWorkActivityCommand) {
			c.CriterionRefs = []d.WorkActivityCriterionRef{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision + 1}}
		}},
		{"foreign artifact reference", d.ErrorCodeNotFound, func(c *a.AppendWorkActivityCommand) { c.ArtifactRefs = []d.ID{foreignArtifact.ID} }},
		{"foreign evidence reference", d.ErrorCodeNotFound, func(c *a.AppendWorkActivityCommand) { c.EvidenceRefs = []d.ID{foreignEvidence.ID} }},
		{"inline artifact scope", d.ErrorCodeInvalidScope, func(c *a.AppendWorkActivityCommand) { c.Artifacts[0].Artifact.Scope = foreignScope }},
		{"inline evidence scope", d.ErrorCodeInvalidScope, func(c *a.AppendWorkActivityCommand) { c.Evidence[0].Evidence.Scope = foreignScope }},
		{"link target", d.ErrorCodeInvalidScope, func(c *a.AppendWorkActivityCommand) {
			c.EvidenceLinks[0].Link.TargetRef = d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
		}},
		{"missing link evidence", d.ErrorCodeNotFound, func(c *a.AppendWorkActivityCommand) {
			missing := id(11)
			c.EvidenceLinks[0].EvidenceLocalKey = ""
			c.EvidenceLinks[0].Link.EvidenceID = &missing
		}},
		{"withdrawn artifact", d.ErrorCodePreconditionFailed, func(c *a.AppendWorkActivityCommand) { c.ArtifactRefs = []d.ID{withdrawn.ID} }},
		{"retracted evidence", d.ErrorCodePreconditionFailed, func(c *a.AppendWorkActivityCommand) { c.EvidenceRefs = []d.ID{retracted.ID} }},
		{"retracted linked evidence", d.ErrorCodePreconditionFailed, func(c *a.AppendWorkActivityCommand) {
			c.EvidenceLinks[0].EvidenceLocalKey = ""
			c.EvidenceLinks[0].Link.EvidenceID = &retracted.ID
		}},
	}
	for i, probe := range probes {
		t.Run("activity guard "+probe.name, func(t *testing.T) {
			invalid := command()
			probe.mutate(&invalid)
			failedCC := commandContext(fmt.Sprintf("work-activity-rejected-%d", i))
			_, err := service.AppendWorkActivity(agentCtx, failedCC, invalid)
			if code, _ := d.ErrorCodeOf(err); code != probe.code {
				t.Fatalf("got %v; want %v", err, probe.code)
			}
			if got := snapshot(); !reflect.DeepEqual(got, baseline) {
				t.Fatalf("rejected %s activity changed records, revision, events or authority", probe.name)
			}
			u := begin()
			_, err = u.(ports.SignedOperationUnitOfWork).SignedOperations().Get(ctx, scope.TenantID, contract.HolderPrincipalID, failedCC.IdempotencyKey)
			must(u.Rollback())
			if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
				t.Fatalf("rejected activity persisted a durable operation: %v", err)
			}
			if probe.name == "criterion revision" {
				retryCC = failedCC
			}
		})
	}
	// A small references-only request may expand huge pre-existing metadata.
	// The committed history must remain readable within its record byte budget.
	oversizedArtifact, err := d.NewArtifact(id(7), scope, "source", "Oversized legacy metadata", "git:oversized", "", "", strings.Repeat("a", 300<<10), contract.Actor, nil, clock.now)
	must(err)
	u = begin()
	must(u.(ports.DocumentaryUnitOfWork).Artifacts().Insert(ctx, oversizedArtifact))
	must(u.Commit())
	beforeOversized := snapshot()
	oversizedCommand := command()
	oversizedCommand.Artifacts, oversizedCommand.Evidence, oversizedCommand.EvidenceLinks = nil, nil, nil
	oversizedCommand.ArtifactRefs = []d.ID{oversizedArtifact.ID}
	smallRequest, err := json.Marshal(oversizedCommand)
	must(err)
	if len(smallRequest) > 4096 {
		t.Fatalf("oversized metadata regression must use a tiny request, got %d bytes", len(smallRequest))
	}
	oversizedCC := commandContext("work-activity-reject-expanded-metadata")
	_, err = service.AppendWorkActivity(agentCtx, oversizedCC, oversizedCommand)
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeInvalidArgument || !strings.Contains(err.Error(), "128 KiB") {
		t.Fatalf("expanded activity metadata bypassed the 128 KiB record cap: %v", err)
	}
	if !reflect.DeepEqual(snapshot(), beforeOversized) {
		t.Fatal("oversized expanded metadata changed activity records, revision, events or authority")
	}
	u = begin()
	_, err = u.(ports.SignedOperationUnitOfWork).SignedOperations().Get(ctx, scope.TenantID, contract.HolderPrincipalID, oversizedCC.IdempotencyKey)
	must(u.Rollback())
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
		t.Fatalf("oversized activity persisted an operation result: %v", err)
	}
	// The criterion probe failed after inline records had been inserted. Reusing
	// exactly that intent verifies rollback released the reservation as well.
	retried, err := service.AppendWorkActivity(agentCtx, retryCC, command())
	must(err)
	if retried.IdempotentReplay || retried.Value.Activity.ID == activity.ID || retried.OutcomeRevision != result.OutcomeRevision+1 {
		t.Fatal("failed activity left a reserved intent or reused documentary records")
	}
	stable := snapshot()
	if sqlStore, ok := store.(*Store); ok {
		_, err = sqlStore.db.ExecContext(ctx, `DELETE FROM idempotency_records WHERE command_name='AppendWorkActivity' AND idempotency_key=?`, cc.IdempotencyKey)
		must(err)
	}
	withoutCache, err := a.NewService(signedNoCacheManager{store}, clock, ids)
	must(err)
	replay, err := withoutCache.AppendWorkActivity(agentCtx, cc, cmd)
	must(err)
	if !replay.IdempotentReplay || replay.CommandID != result.CommandID || replay.OutcomeRevision != result.OutcomeRevision {
		t.Fatal("cache-free replay changed original command identity or revision")
	}
	replay.IdempotentReplay = false
	expectedJSON, err := json.Marshal(result)
	must(err)
	replayedJSON, err := json.Marshal(replay)
	must(err)
	if !bytes.Equal(expectedJSON, replayedJSON) || !reflect.DeepEqual(snapshot(), stable) {
		t.Fatal("cache-free replay changed original activity/local-key payload or wrote new state")
	}
	changed := cmd
	changed.Description = "A different semantic intention"
	if _, err := withoutCache.AppendWorkActivity(agentCtx, cc, changed); err == nil {
		t.Fatal("a changed activity reused an accepted durable intent")
	}
	operation, err := service.ReadSignedState(agentCtx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: cc.IdempotencyKey})
	must(err)
	payload, err := base64.StdEncoding.Strict().DecodeString(operation.OperationPayload)
	must(err)
	if operation.OperationName != "AppendWorkActivity" || operation.OperationCommandID == nil || *operation.OperationCommandID != result.CommandID || operation.PayloadDigest != signing.Digest(expectedJSON) || !bytes.Equal(payload, expectedJSON) {
		t.Fatal("durable operation expansion did not preserve the original activity response")
	}
	if !reflect.DeepEqual(snapshot(), stable) {
		t.Fatal("operation recovery or changed-intent rejection mutated activity state")
	}
}
