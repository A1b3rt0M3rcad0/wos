package application_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func completionModeRequest(t *testing.T, f *operationalProfileFixture, objective domain.Objective, credential ports.Credential, key string, mode domain.ObjectiveCompletionMode) (context.Context, domain.CommandContext, application.SetObjectiveCompletionModeCommand) {
	t.Helper()
	state, err := f.Service.GetOutcomeState(f.Context, f.Scope)
	if err != nil {
		t.Fatal(err)
	}
	cc := f.Command
	cc.PrincipalID = credential.PrincipalID
	cc.Actor = credential.Actor
	cc.IdempotencyKey = key
	identity := application.Identity{
		CredentialID: credential.ID, CredentialDigest: credential.Digest,
		PrincipalID: credential.PrincipalID, NamespaceID: credential.NamespaceID, Actor: credential.Actor,
	}
	return application.WithIdentity(f.Context, identity), cc, application.SetObjectiveCompletionModeCommand{
		Scope: f.Scope, ObjectiveID: objective.ID, Mode: mode, ExpectedVersion: objective.Version,
		ExpectedOutcomeRevision: revisionPointer(state.OutcomeRevision), Reason: "Authorize an explicit Objective completion policy",
	}
}

func grantObjectiveCompletionPermission(t *testing.T, f *operationalProfileFixture, credential ports.Credential, allow bool) {
	t.Helper()
	uow, err := f.Store.Begin(f.Context)
	if err != nil {
		t.Fatal(err)
	}
	registry, ok := uow.(ports.SigningIdentityUnitOfWork)
	if !ok {
		t.Fatal("memory transaction does not expose credential policy storage")
	}
	policy, err := registry.SigningIdentity().CredentialPolicy(f.Context, credential.NamespaceID, credential.ID)
	oldVersion := policy.Version
	if err != nil {
		if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeNotFound {
			t.Fatal(err)
		}
		baseline := f.Credentials["operator_human"]
		policy, err = registry.SigningIdentity().CredentialPolicy(f.Context, baseline.NamespaceID, baseline.ID)
		if err != nil {
			t.Fatal(err)
		}
		policy.CredentialID = credential.ID
		policy.Version = 0
		oldVersion = 0
	}
	filtered := policy.PermittedOperations[:0]
	for _, permission := range policy.PermittedOperations {
		if permission != string(ports.PermissionObjectiveConfigureCompletion) {
			filtered = append(filtered, permission)
		}
	}
	policy.PermittedOperations = filtered
	if allow {
		policy.PermittedOperations = append(policy.PermittedOperations, string(ports.PermissionObjectiveConfigureCompletion))
	}
	policy.Version = oldVersion + 1
	policy.UpdatedAt = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if err := registry.SigningIdentity().SaveCredentialPolicy(f.Context, policy, oldVersion); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
}

func setCompletionOutcomeProfile(t *testing.T, f *operationalProfileFixture, profile domain.OperationalProfile, completion domain.CompletionPolicy) {
	t.Helper()
	uow, err := f.Store.Begin(f.Context)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(f.Context, f.Scope.NamespaceID, f.Scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	previous := outcome.Version
	outcome.Version++
	outcome.OperationalProfile = profile
	outcome.CompletionPolicy = completion
	outcome.UpdatedAt = outcome.UpdatedAt.Add(time.Second)
	if err := uow.Outcomes().Save(f.Context, outcome, previous); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestSetObjectiveCompletionModeRequiresCredentialPermissionAndRecordsImmutableAuthority(t *testing.T) {
	f := newOperationalProfileFixture(t)
	objective := f.PlannedObjective
	credential := f.Credentials["operator_human"]
	grantObjectiveCompletionPermission(t, f, credential, false)
	ctx, cc, cmd := completionModeRequest(t, f, objective, credential, "completion-manual-authority", domain.ObjectiveCompletionModeManual)
	stateBefore, err := f.Service.GetOutcomeState(f.Context, f.Scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.Store.SnapshotDomainEvents(f.Scope)
	receiptsBefore := f.Store.IdempotencyRecordCount()
	if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err == nil {
		t.Fatal("completion mode changed without the dedicated credential permission")
	} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("missing permission error code=%s, want forbidden: %v", code, err)
	}
	stateAfter, err := f.Service.GetOutcomeState(f.Context, f.Scope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stateBefore, stateAfter) || !reflect.DeepEqual(eventsBefore, f.Store.SnapshotDomainEvents(f.Scope)) || receiptsBefore != f.Store.IdempotencyRecordCount() {
		t.Fatal("denied completion-policy change mutated Objective state, events, or receipts")
	}

	grantObjectiveCompletionPermission(t, f, credential, true)
	result, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd)
	if err != nil {
		t.Fatalf("authorized manual completion mode change failed: %v", err)
	}
	if result.Value.Authority.PolicyVersion != 1 || result.Value.Authority.Mode != domain.ObjectiveCompletionModeManual || result.Value.Authority.DelegatorCredentialID == nil || *result.Value.Authority.DelegatorCredentialID != credential.ID || result.Value.Authority.CredentialPolicyRevision == 0 {
		t.Fatalf("manual authority provenance = %+v", result.Value.Authority)
	}
	if result.Value.Authority.CompletionCycle != objective.EffectiveCompletionCycle() || result.Value.Authority.AutomationActor != domain.ObjectiveCompletionAutomationActor() {
		t.Fatalf("manual authority cycle/host actor = %+v", result.Value.Authority)
	}
	if result.Value.Objective.Version != objective.Version+1 || result.OutcomeRevision != stateBefore.OutcomeRevision+1 {
		t.Fatalf("policy mutation versions: Objective=%d OutcomeRevision=%d", result.Value.Objective.Version, result.OutcomeRevision)
	}
	var eventCount int
	for _, event := range f.Store.SnapshotDomainEvents(f.Scope) {
		if event.EventType == "objective.completion_policy_authorized" {
			eventCount++
		}
	}
	if eventCount != 1 {
		t.Fatalf("completion policy authorization events=%d, want exactly one", eventCount)
	}
	history, err := f.Service.ListObjectiveCompletionAuthorities(ctx, f.Scope, objective.ID)
	if err != nil || len(history.Authorities) != 1 || history.Authorities[0].ID != result.Value.Authority.ID || history.Authorities[0].PolicyVersion != result.Value.Authority.PolicyVersion || history.Authorities[0].DelegatorCredentialID == nil || *history.Authorities[0].DelegatorCredentialID != credential.ID {
		t.Fatalf("completion authority history=%+v err=%v", history, err)
	}

	replay, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Authority.ID != result.Value.Authority.ID {
		t.Fatalf("same-credential receipt replay=%+v err=%v", replay, err)
	}
	otherCredential := credential
	otherCredential.ID = id("0199e280-0000-7000-8000-000000000020")
	otherCredential.Digest = "test-credential-digest/operator-human-clone"
	if err := f.Store.PutCredential(f.Context, otherCredential); err != nil {
		t.Fatal(err)
	}
	grantObjectiveCompletionPermission(t, f, otherCredential, true)
	otherCtx, otherCommand, sameIntent := completionModeRequest(t, f, result.Value.Objective, otherCredential, cmd.Reason, domain.ObjectiveCompletionModeManual)
	// Preserve the exact idempotency key and payload. A different root
	// credential for the same Principal must not receive the stored result.
	otherCommand.IdempotencyKey = cc.IdempotencyKey
	sameIntent.ExpectedVersion = cmd.ExpectedVersion
	sameIntent.ExpectedOutcomeRevision = cmd.ExpectedOutcomeRevision
	sameIntent.Reason = cmd.Reason
	if _, err := f.Service.SetObjectiveCompletionMode(otherCtx, otherCommand, sameIntent); err == nil {
		t.Fatal("same-Principal different-credential replay disclosed a protected receipt")
	} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("different-credential replay code=%s, want idempotency conflict: %v", code, err)
	}
}

func TestSetAutomaticObjectiveCompletionRequiresV2CredentialAndIndependentReviewCompatibility(t *testing.T) {
	t.Run("valid scoped delegation", func(t *testing.T) {
		f := newOperationalProfileFixture(t)
		setCompletionOutcomeProfile(t, f, domain.OperationalProfileOperationalV2, domain.CompletionPolicyStrictWork)
		credential := f.Credentials["operator_human"]
		grantObjectiveCompletionPermission(t, f, credential, true)
		ctx, cc, cmd := completionModeRequest(t, f, f.InProgressObjective, credential, "completion-auto-v2", domain.ObjectiveCompletionModeAutomatic)
		result, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd)
		if err != nil {
			t.Fatalf("valid automatic completion delegation failed: %v", err)
		}
		if result.Value.Authority.OperationalProfile != domain.OperationalProfileOperationalV2 || result.Value.Authority.CompletionPolicy != domain.CompletionPolicyStrictWork || result.Value.Authority.DelegatorCredentialID == nil || result.Value.Authority.CredentialPolicyRevision == 0 {
			t.Fatalf("automatic delegation snapshot = %+v", result.Value.Authority)
		}
	})

	t.Run("compatibility policy fails closed", func(t *testing.T) {
		f := newOperationalProfileFixture(t)
		credential := f.Credentials["operator_human"]
		grantObjectiveCompletionPermission(t, f, credential, true)
		ctx, cc, cmd := completionModeRequest(t, f, f.PlannedObjective, credential, "completion-auto-compat", domain.ObjectiveCompletionModeAutomatic)
		before, err := f.Service.GetOutcomeState(f.Context, f.Scope)
		if err != nil {
			t.Fatal(err)
		}
		eventsBefore := f.Store.SnapshotDomainEvents(f.Scope)
		if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err == nil {
			t.Fatal("automatic completion was enabled for compatibility policy")
		}
		after, err := f.Service.GetOutcomeState(f.Context, f.Scope)
		if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(eventsBefore, f.Store.SnapshotDomainEvents(f.Scope)) {
			t.Fatalf("failed compatibility-policy change mutated state or events: err=%v", err)
		}
		history, err := f.Service.ListObjectiveCompletionAuthorities(f.Context, f.Scope, f.PlannedObjective.ID)
		if err != nil || len(history.Authorities) != 0 {
			t.Fatalf("invalid policy stored authority history=%+v err=%v", history, err)
		}
	})

	t.Run("human review policy blocks automatic mode", func(t *testing.T) {
		f := newOperationalProfileFixture(t)
		setCompletionOutcomeProfile(t, f, domain.OperationalProfileOperationalV2, domain.CompletionPolicyStrictWork)
		f.Service.SetIndependentReviewer(true)
		credential := f.Credentials["operator_human"]
		grantObjectiveCompletionPermission(t, f, credential, true)
		ctx, cc, cmd := completionModeRequest(t, f, f.PlannedObjective, credential, "completion-auto-review", domain.ObjectiveCompletionModeAutomatic)
		before, err := f.Service.GetOutcomeState(f.Context, f.Scope)
		if err != nil {
			t.Fatal(err)
		}
		eventsBefore := f.Store.SnapshotDomainEvents(f.Scope)
		if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err == nil {
			t.Fatal("automatic completion was enabled while independent human review is required")
		} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
			t.Fatalf("independent reviewer gate code=%s: %v", code, err)
		}
		after, err := f.Service.GetOutcomeState(f.Context, f.Scope)
		if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(eventsBefore, f.Store.SnapshotDomainEvents(f.Scope)) {
			t.Fatalf("review-gated automatic policy changed shared state: %v", err)
		}
	})
}

func TestSetObjectiveCompletionModeRejectsStaleVersionsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name           string
		key            string
		staleObjective bool
	}{
		{name: "stale objective version", key: "completion-stale-objective-version", staleObjective: true},
		{name: "stale outcome revision", key: "completion-stale-outcome-revision", staleObjective: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newOperationalProfileFixture(t)
			credential := f.Credentials["operator_human"]
			grantObjectiveCompletionPermission(t, f, credential, true)
			ctx, cc, cmd := completionModeRequest(t, f, f.PlannedObjective, credential, tc.key, domain.ObjectiveCompletionModeManual)
			if tc.staleObjective {
				cmd.ExpectedVersion++
			} else {
				stale := *cmd.ExpectedOutcomeRevision + 1
				cmd.ExpectedOutcomeRevision = &stale
			}
			before, err := f.Service.GetOutcomeState(f.Context, f.Scope)
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore := f.Store.SnapshotDomainEvents(f.Scope)
			receiptsBefore := f.Store.IdempotencyRecordCount()
			if _, err := f.Service.SetObjectiveCompletionMode(ctx, cc, cmd); err == nil {
				t.Fatal("stale completion policy command succeeded")
			} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict {
				t.Fatalf("stale version error code=%s: %v", code, err)
			}
			after, err := f.Service.GetOutcomeState(f.Context, f.Scope)
			if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(eventsBefore, f.Store.SnapshotDomainEvents(f.Scope)) || receiptsBefore != f.Store.IdempotencyRecordCount() {
				t.Fatalf("stale completion policy command changed shared state: %v", err)
			}
		})
	}
}

func revisionPointer(revision domain.OutcomeRevision) *domain.OutcomeRevision {
	value := revision
	return &value
}
