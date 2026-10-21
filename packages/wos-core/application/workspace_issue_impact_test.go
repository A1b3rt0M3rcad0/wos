package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func workspaceIssueImpactSnapshot(f *issueCompletionFixture) []byte {
	f.t.Helper()
	state, err := f.s.GetOutcomeState(f.ctx, f.scope)
	f.must(err)
	raw, err := json.Marshal(struct {
		State  OutcomeState
		Events []d.DomainEvent
	}{state, f.store.SnapshotDomainEvents(f.scope)})
	f.must(err)
	return raw
}

func workspaceIssueImpactView(f *issueCompletionFixture, subject WorkspaceSubject, limit int) EntityWorkspaceView {
	f.t.Helper()
	includeActivity := false
	view, err := f.s.GetEntityWorkspaceView(f.ctx, f.scope, WorkspaceViewQuery{Subject: subject, Limit: limit, IncludeActivity: &includeActivity})
	f.must(err)
	return view
}

func workspaceIssueAchieveCapability(t *testing.T, view EntityWorkspaceView) WorkspaceCapability {
	t.Helper()
	for _, capability := range view.Capabilities {
		if capability.ActionID == "AchieveObjective" {
			return capability
		}
	}
	t.Fatal("Objective achievement capability is missing")
	return WorkspaceCapability{}
}

// Execute the real command against the observed state. A preview must preserve
// its exact availability, first failed precondition and related references.
func workspaceIssueAssertCommandParity(f *issueCompletionFixture, owner d.Objective, capability WorkspaceCapability, allowed bool) {
	f.t.Helper()
	err := f.achieve(owner)
	if allowed {
		f.must(err)
		if capability.Availability != "available" || len(capability.ReasonCodes) != 0 {
			f.t.Fatalf("accepted command was not available: %+v", capability)
		}
		return
	}
	var coded *d.Error
	if !errors.As(err, &coded) {
		f.t.Fatalf("blocked capability did not produce a coded command failure: %v", err)
	}
	reason := string(coded.Code)
	var related []d.EntityRef
	if coded.Details != nil {
		if coded.Details.ReasonCode != "" {
			reason = coded.Details.ReasonCode
		}
		related = coded.Details.RelatedRefs
	}
	if capability.Availability != "blocked" || !reflect.DeepEqual(capability.ReasonCodes, []string{reason}) {
		f.t.Fatalf("capability/command precondition diverged: capability=%+v command=%v", capability, err)
	}
	if len(related) != len(capability.RelatedRefs) || (len(related) > 0 && !reflect.DeepEqual(related, capability.RelatedRefs)) {
		f.t.Fatalf("capability/command related references diverged: capability=%v command=%v", capability.RelatedRefs, related)
	}
}

func TestWorkspaceIssueImpactBoundsSamplesAndUsesStructuralOwner(t *testing.T) {
	for _, limit := range []int{0, 3, 20, 100} {
		t.Run(fmt.Sprintf("view_limit_%d", limit), func(t *testing.T) {
			f := newIssueCompletionFixture(t)
			owner := f.objective("Selected owner", nil, true)
			other := f.objective("Independent owner", nil, true)
			child := f.objective("Child owner", &owner.ID, true)
			owned := map[d.EntityRef]bool{}
			for n := 0; n < 26; n++ {
				issue := f.issue(owner.ID, other.Ref())
				owned[issue.Ref()] = true
				if n%2 == 0 {
					_, err := f.s.StartIssueInvestigation(f.ctx, f.cc(), StartIssueInvestigationCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version})
					f.must(err)
				}
			}
			// Affects is documentary; neither an independent owner's Issue nor
			// a descendant's Issue becomes structurally owned by this Objective.
			f.issue(other.ID, owner.Ref())
			f.issue(child.ID, owner.Ref())
			before := workspaceIssueImpactSnapshot(f)
			view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, limit)
			if view.ConclusionImpact == nil || view.ConclusionImpact.PendingIssues == nil || *view.ConclusionImpact.PendingIssues != 26 {
				t.Fatalf("preview did not count all structurally owned open/investigating Issues: %+v", view.ConclusionImpact)
			}
			sampleLimit := 20
			if limit > 0 {
				sampleLimit = min(limit, 20)
			}
			refs := view.ConclusionImpact.PendingIssueRefs
			coverage, exists := view.Coverage["conclusion_impact_issues"]
			if !exists || coverage.Status != "partial" || coverage.TotalCount == nil || *coverage.TotalCount != 26 || coverage.ReturnedCount == nil || *coverage.ReturnedCount != sampleLimit || len(refs) != sampleLimit || coverage.Reason == "" {
				t.Fatalf("bounded samples masqueraded as a complete Issue collection: refs=%v coverage=%+v", refs, coverage)
			}
			seen := map[d.EntityRef]bool{}
			for _, ref := range refs {
				if !owned[ref] || seen[ref] {
					t.Fatalf("preview sample included another owner or repeated a source: %v", ref)
				}
				seen[ref] = true
			}
			capability := workspaceIssueAchieveCapability(t, view)
			if !reflect.DeepEqual(capability.ReasonCodes, []string{"objective_issue_pending"}) {
				t.Fatalf("valid owned Issues did not explain the achievement gate: %+v", capability)
			}
			workspaceIssueAssertCommandParity(f, owner, capability, false)
			if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
				t.Fatal("Issue preview or rejected achievement changed state, versions, revision or events")
			}
		})
	}
}

func TestWorkspaceIssueImpactCountsDuplicateSourcesWithoutCountingCanonicalOwner(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Duplicate source owner", nil, true)
	other := f.objective("Canonical owner", nil, true)
	canonical := f.issue(other.ID, owner.Ref())
	middle := f.issue(other.ID)
	_, err := f.s.MarkIssueDuplicate(f.ctx, f.cc(), MarkIssueDuplicateCommand{Scope: f.scope, IssueID: middle.ID, ExpectedVersion: middle.Version, DuplicateOfID: canonical.ID})
	f.must(err)
	sources := map[d.EntityRef]bool{}
	for n := 0; n < 3; n++ {
		issue := f.issue(owner.ID, other.Ref())
		sources[issue.Ref()] = true
		_, err := f.s.MarkIssueDuplicate(f.ctx, f.cc(), MarkIssueDuplicateCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version, DuplicateOfID: middle.ID})
		f.must(err)
	}
	closedCanonical := f.close(f.issue(other.ID), d.IssueLifecycleWontFix)
	closedSource := f.issue(owner.ID)
	_, err = f.s.MarkIssueDuplicate(f.ctx, f.cc(), MarkIssueDuplicateCommand{Scope: f.scope, IssueID: closedSource.ID, ExpectedVersion: closedSource.Version, DuplicateOfID: closedCanonical.ID})
	f.must(err)
	before := workspaceIssueImpactSnapshot(f)
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 2)
	if view.ConclusionImpact == nil || view.ConclusionImpact.PendingIssues == nil || *view.ConclusionImpact.PendingIssues != 3 || len(view.ConclusionImpact.PendingIssueRefs) != 2 {
		t.Fatalf("duplicate sources were dropped, counted by target, or confused with canonical ownership: %+v", view.ConclusionImpact)
	}
	for _, ref := range view.ConclusionImpact.PendingIssueRefs {
		if !sources[ref] {
			t.Fatalf("canonical/intermediate/closed source leaked into pending source sample: %v", ref)
		}
	}
	coverage := view.Coverage["conclusion_impact_issues"]
	if coverage.Status != "partial" || coverage.TotalCount == nil || *coverage.TotalCount != 3 || coverage.ReturnedCount == nil || *coverage.ReturnedCount != 2 {
		t.Fatalf("duplicate preview reported a sample count as the total: %+v", coverage)
	}
	rawView, err := json.Marshal(view)
	f.must(err)
	var serialized struct {
		ConclusionImpact struct {
			PendingIssues    *int          `json:"pending_issues"`
			PendingIssueRefs []d.EntityRef `json:"pending_issue_refs"`
		} `json:"conclusion_impact"`
	}
	f.must(json.Unmarshal(rawView, &serialized))
	if serialized.ConclusionImpact.PendingIssues == nil || *serialized.ConclusionImpact.PendingIssues != 3 || !reflect.DeepEqual(serialized.ConclusionImpact.PendingIssueRefs, view.ConclusionImpact.PendingIssueRefs) {
		t.Fatal("EntityWorkspaceView JSON changed numeric Issue count or scoped source references")
	}
	capability := workspaceIssueAchieveCapability(t, view)
	if len(capability.RelatedRefs) != 2 || !sources[capability.RelatedRefs[0]] || capability.RelatedRefs[1] != canonical.Ref() {
		t.Fatalf("duplicate capability did not identify the owned source and canonical terminal: %+v", capability)
	}
	workspaceIssueAssertCommandParity(f, owner, capability, false)
	if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
		t.Fatal("duplicate preview or rejected achievement mutated documentary state")
	}
	// Resolving the canonical terminal closes all three owned duplicate chains;
	// neither the duplicate records nor the selected Objective are rewritten.
	f.close(canonical, d.IssueLifecycleResolved)
	before = workspaceIssueImpactSnapshot(f)
	closedView := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 2)
	if closedView.ConclusionImpact.PendingIssues == nil || *closedView.ConclusionImpact.PendingIssues != 0 || len(closedView.ConclusionImpact.PendingIssueRefs) != 0 || closedView.Coverage["conclusion_impact_issues"].Status != "complete" {
		t.Fatalf("closed canonical chains did not produce exact zero: %+v", closedView.ConclusionImpact)
	}
	if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
		t.Fatal("closed-chain preview changed state")
	}
	workspaceIssueAssertCommandParity(f, owner, workspaceIssueAchieveCapability(t, closedView), true)
}

func TestWorkspaceIssueImpactZeroIsExplicitAndOnlyObjectiveScoped(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Closed owned Issues", nil, true)
	other := f.objective("Open unrelated Issues", nil, true)
	f.close(f.issue(owner.ID), d.IssueLifecycleResolved)
	f.close(f.issue(owner.ID), d.IssueLifecycleWontFix)
	f.issue(other.ID, owner.Ref())
	before := workspaceIssueImpactSnapshot(f)
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 1)
	coverage := view.Coverage["conclusion_impact_issues"]
	if view.ConclusionImpact == nil || view.ConclusionImpact.PendingIssues == nil || *view.ConclusionImpact.PendingIssues != 0 || len(view.ConclusionImpact.PendingIssueRefs) != 0 || coverage.Status != "complete" || coverage.TotalCount == nil || *coverage.TotalCount != 0 || coverage.ReturnedCount == nil || *coverage.ReturnedCount != 0 {
		t.Fatalf("exact zero was confused with absent classification: impact=%+v coverage=%+v", view.ConclusionImpact, coverage)
	}
	raw, err := json.Marshal(view)
	f.must(err)
	var envelope map[string]json.RawMessage
	f.must(json.Unmarshal(raw, &envelope))
	var fields map[string]json.RawMessage
	f.must(json.Unmarshal(envelope["conclusion_impact"], &fields))
	if !bytes.Equal(fields["pending_issues"], []byte("0")) {
		t.Fatalf("exact zero disappeared from JSON: %s", raw)
	}
	outcomeRef := d.EntityRef{Scope: f.scope, Kind: d.EntityKindOutcome, ID: f.scope.OutcomeID}
	outcome := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: outcomeRef}, 1)
	if outcome.ConclusionImpact == nil || outcome.ConclusionImpact.PendingIssues != nil || len(outcome.ConclusionImpact.PendingIssueRefs) != 0 {
		t.Fatal("Objective-owned Issue gate was incorrectly projected as an Outcome-wide gate")
	}
	criterionID := owner.Criteria.Items[0].ID
	criterion := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "criterion", Ref: owner.Ref(), CriterionID: &criterionID}, 1)
	if criterion.ConclusionImpact != nil {
		t.Fatal("criterion subject unexpectedly acquired an Objective conclusion preview")
	}
	if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
		t.Fatal("zero or non-Objective preview mutated state")
	}
	workspaceIssueAssertCommandParity(f, owner, workspaceIssueAchieveCapability(t, view), true)
}

func TestWorkspaceIssueImpactCorruptPersistedCycleHasUnknownCountAndBlocks(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Owner with corrupt legacy chain", nil, true)
	other := f.objective("Other legacy owner", nil, true)
	first, second := f.issue(owner.ID), f.issue(other.ID)
	f.must(first.MarkDuplicate(second.ID, first.UpdatedAt))
	f.must(second.MarkDuplicate(first.ID, second.UpdatedAt))
	u, err := f.store.Begin(f.ctx)
	f.must(err)
	issues := u.(ports.IssueBlockerUnitOfWork).Issues()
	f.must(issues.Save(f.ctx, first, first.Version-1))
	f.must(issues.Save(f.ctx, second, second.Version-1))
	f.must(u.Commit())
	before := workspaceIssueImpactSnapshot(f)
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 3)
	coverage, exists := view.Coverage["conclusion_impact_issues"]
	if view.ConclusionImpact == nil || view.ConclusionImpact.PendingIssues != nil || !exists || coverage.Status != "partial" || coverage.TotalCount != nil || coverage.Reason == "" || len(view.ConclusionImpact.PendingIssueRefs) > 3 {
		t.Fatalf("corrupt chain fabricated a complete count or closure: impact=%+v coverage=%+v", view.ConclusionImpact, coverage)
	}
	raw, err := json.Marshal(view)
	f.must(err)
	var envelope map[string]json.RawMessage
	f.must(json.Unmarshal(raw, &envelope))
	var fields map[string]json.RawMessage
	f.must(json.Unmarshal(envelope["conclusion_impact"], &fields))
	if _, exists := fields["pending_issues"]; exists {
		t.Fatalf("unknown Issue count was serialized as an exact number: %s", raw)
	}
	workspaceIssueAssertCommandParity(f, owner, workspaceIssueAchieveCapability(t, view), false)
	if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
		t.Fatal("corrupt-chain preview or blocked command changed state, versions, revision or events")
	}
}

func TestWorkspaceIssueImpactPreservesActiveOutcomePreconditionPrecedence(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Owner in an inactive Outcome", nil, true)
	f.issue(owner.ID)
	outcome, err := f.s.GetOutcome(f.ctx, f.scope)
	f.must(err)
	_, err = f.s.AbandonOutcome(f.ctx, f.cc(), AbandonOutcomeCommand{Scope: f.scope, ExpectedVersion: outcome.Value.Version, Reason: "Explicitly stop this operational scope"})
	f.must(err)
	before := workspaceIssueImpactSnapshot(f)
	view := workspaceIssueImpactView(f, WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, 1)
	if view.ConclusionImpact == nil || view.ConclusionImpact.PendingIssues == nil || *view.ConclusionImpact.PendingIssues != 1 {
		t.Fatal("inactive Outcome removed useful owned-Issue diagnostics")
	}
	capability := workspaceIssueAchieveCapability(t, view)
	if reflect.DeepEqual(capability.ReasonCodes, []string{"objective_issue_pending"}) {
		t.Fatal("Issue gate hid the command's earlier active-Outcome precondition")
	}
	workspaceIssueAssertCommandParity(f, owner, capability, false)
	if !bytes.Equal(before, workspaceIssueImpactSnapshot(f)) {
		t.Fatal("inactive-Outcome preview or rejection mutated state")
	}
}
