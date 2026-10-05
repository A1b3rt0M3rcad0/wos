package application_test

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestIndependentReviewUsesPrincipalRatherThanActorAlias(t *testing.T) {
	ctx := context.Background()
	s, _ := newWave09Service(t)
	s.SetIndependentReviewer(true)
	cc := commandContext()
	outcome, err := s.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("0199d110-0000-7000-8000-000000000001"), Title: "Independent proof", DesiredState: "Execution and approval are separate", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: outcome.Value.Ref(), ExpectedVersion: outcome.Value.Version, Title: "Reviewer approval", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	active, err := s.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: outcome.Value.Scope(), ExpectedVersion: outcome.Value.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	work, err := s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: outcome.Value.Scope(), Title: "Execution", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	if err != nil {
		t.Fatal(err)
	}
	executor := cc
	executor.PrincipalID = "executor"
	executor.Actor = domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "executor-agent"}
	if _, err = s.ClaimWorkItem(ctx, executor, application.ClaimWorkItemCommand{Scope: outcome.Value.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: work.Value.Version, TTL: domain.DefaultLeaseTTL}); err != nil {
		t.Fatal(err)
	}
	alias := executor
	alias.Actor = domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "human-alias"}
	assessment := application.AttestCriterionCommand{Owner: outcome.Value.Ref(), ExpectedVersion: active.Value.Version, CriterionID: criterion.Value.ID, CriterionRevision: criterion.Value.Revision, Result: domain.AssessmentResultMet, Rationale: "Verified"}
	if _, err = s.AttestCriterion(ctx, alias, assessment); err == nil {
		t.Fatal("actor alias bypassed independent principal policy")
	}
	reviewer := cc
	reviewer.PrincipalID = "reviewer"
	reviewer.Actor = domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "reviewer"}
	if _, err = s.AttestCriterion(ctx, reviewer, assessment); err != nil {
		t.Fatal("independent reviewer denied", err)
	}
}
