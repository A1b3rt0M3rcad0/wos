package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func documentaryOwner(t *testing.T, s *application.Service) domain.Outcome {
	t.Helper()
	result, err := s.CreateOutcome(context.Background(), commandContext(), application.CreateOutcomeCommand{NamespaceID: id("0199e200-0000-7000-8000-000000000001"), Title: "Reviewed installation", DesiredState: "Installation is independently checked", Priority: domain.PriorityNormal, InitialCriteria: []application.InitialCriterion{initialProof("Installation succeeds")}})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
}

func observedCriterionLink(owner domain.Outcome) application.EvidenceLinkInput {
	criterion := owner.Criteria.Items[0]
	version := owner.Version
	revision := criterion.Revision
	return application.EvidenceLinkInput{TargetRef: owner.Ref(), TargetExpectedVersion: &version, CriterionID: &criterion.ID, CriterionRevision: &revision, Stance: domain.EvidenceStanceSupports, Rationale: "The recorded installation observation relates to this condition"}
}

func TestRegisterEvidenceWithLinksObservationIsAtomicAndDoesNotApprove(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	owner := documentaryOwner(t, service)
	cmd := application.RegisterEvidenceWithLinksCommand{Scope: owner.Scope(), Evidence: &application.EvidenceRegistration{EvidenceType: domain.EvidenceTypeAttestation, Description: "I followed the installation instructions"}, Observation: &application.EvidenceObservation{SubjectRef: owner.Ref(), Body: "I followed the instructions on my local machine"}, Artifact: &application.ArtifactRegistration{ArtifactType: "report", Name: "Installation transcript", URI: "https://example.test/transcript.txt"}, Links: []application.EvidenceLinkInput{observedCriterionLink(owner)}}
	cc := planningIntent(t.Name())
	cc.CorrelationID = "observed-installation"
	result, err := service.RegisterEvidenceWithLinks(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Observation == nil || result.Value.Artifact == nil || !result.Value.EvidenceCreated || len(result.Value.Links) != 1 {
		t.Fatalf("result=%#v", result)
	}
	if result.Value.Observation.PrincipalID != cc.PrincipalID || result.Value.Observation.Actor != cc.Actor || result.Value.Evidence.SourceRef.Provider != domain.EntityNoteSourceProvider || result.Value.Evidence.SourceRef.ID != result.Value.Observation.ID.String() {
		t.Fatalf("provenance=%#v", result.Value)
	}
	current, err := service.GetOutcome(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Value.Criteria.Assessments) != 0 || current.Value.Lifecycle != domain.OutcomeLifecycleDraft || current.Value.Version != owner.Version {
		t.Fatal("documentary registration assessed or activated the Outcome")
	}
	var compound []domain.DomainEvent
	for _, event := range store.SnapshotDomainEvents(owner.Scope()) {
		if event.CommandID == cc.CommandID && event.CorrelationID == cc.CorrelationID {
			compound = append(compound, event)
		}
	}
	if len(compound) != 4 {
		t.Fatalf("events=%#v", compound)
	}
	for i, event := range compound {
		if event.OutcomeRevision != result.OutcomeRevision || event.EventIndex != uint32(i) {
			t.Fatal("compound events do not share ordered revision")
		}
	}
	replay, err := service.RegisterEvidenceWithLinks(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || !sameWorkspaceJSON(result.Value, replay.Value) {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}

func TestRegisterEvidenceWithLinksInvalidLastLinkRollsBackEveryRecord(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	owner := documentaryOwner(t, service)
	before := store.SnapshotDomainEvents(owner.Scope())
	invalid := domain.EntityRef{Scope: owner.Scope(), Kind: domain.EntityKindBlocker, ID: id("0199ed90-0000-7000-8000-999999999999")}
	cmd := application.RegisterEvidenceWithLinksCommand{Scope: owner.Scope(), Evidence: &application.EvidenceRegistration{EvidenceType: domain.EvidenceTypeAttestation, Description: "Human observation"}, Observation: &application.EvidenceObservation{SubjectRef: owner.Ref(), Body: "Observed locally"}, Artifact: &application.ArtifactRegistration{ArtifactType: "report", Name: "Report", URI: "file:///report.txt"}, Links: []application.EvidenceLinkInput{observedCriterionLink(owner), {TargetRef: invalid, Stance: domain.EvidenceStanceContext, Rationale: "Invalid target"}}}
	if _, err := service.RegisterEvidenceWithLinks(ctx, planningIntent(t.Name()), cmd); err == nil {
		t.Fatal("expected unsupported Blocker target")
	}
	state, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Artifacts) != 0 || len(state.Evidence) != 0 || len(state.EvidenceLinks) != 0 || len(store.SnapshotDomainEvents(owner.Scope())) != len(before) {
		t.Fatal("partial documentary records or events persisted")
	}
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	notes, err := uow.(ports.EntityNoteUnitOfWork).EntityNotes().List(ctx, owner.Scope(), ports.EntityNoteFilter{Limit: 10, ThroughRevision: state.OutcomeRevision})
	if err != nil || len(notes) != 0 {
		t.Fatalf("orphan notes=%#v err=%v", notes, err)
	}
}

func TestEvidenceReuseHasLinkLocalStanceAndValidTargets(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	owner := documentaryOwner(t, service)
	external := application.RegisterEvidenceWithLinksCommand{Scope: owner.Scope(), Evidence: &application.EvidenceRegistration{EvidenceType: domain.EvidenceTypeSource, Description: "Installation report", SourceRef: domain.SourceReference{Provider: "external", URI: "https://example.test/report"}, CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)}, Links: []application.EvidenceLinkInput{observedCriterionLink(owner)}}
	first, err := service.RegisterEvidenceWithLinks(ctx, planningIntent(t.Name()+"-new"), external)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := service.ProposeDecision(ctx, commandContext(), application.ProposeDecisionCommand{Scope: owner.Scope(), Title: "Installer", Proposal: "Use package manager", Rationale: "Repeatable"})
	if err != nil {
		t.Fatal(err)
	}
	reuse := application.RegisterEvidenceWithLinksCommand{Scope: owner.Scope(), ExistingEvidenceID: &first.Value.Evidence.ID, Links: []application.EvidenceLinkInput{{TargetRef: decision.Value.Ref(), Stance: domain.EvidenceStanceContradicts, Rationale: "Report shows another supported installation"}}}
	cc := planningIntent(t.Name() + "-reuse")
	cc.CorrelationID = "reuse-existing"
	second, err := service.RegisterEvidenceWithLinks(ctx, cc, reuse)
	if err != nil {
		t.Fatal(err)
	}
	if second.Value.EvidenceCreated || second.Value.Evidence.ID != first.Value.Evidence.ID || second.Value.Links[0].Stance != domain.EvidenceStanceContradicts {
		t.Fatalf("reuse=%#v", second)
	}
	for _, event := range store.SnapshotDomainEvents(owner.Scope()) {
		if event.CorrelationID == cc.CorrelationID && event.EventType == "evidence.registered" {
			t.Fatal("existing Evidence emitted a second registration")
		}
	}
	_, err = service.RetractEvidence(ctx, commandContext(), application.RetractEvidenceCommand{Scope: owner.Scope(), EvidenceID: first.Value.Evidence.ID, ExpectedVersion: first.Value.Evidence.Version, Reason: "Outdated"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RegisterEvidenceWithLinks(ctx, planningIntent(t.Name()+"-retracted"), reuse); err == nil {
		t.Fatal("retracted Evidence received another link")
	}
}

func TestContextualEvidenceRejectsStaleCriterionAndForgedInternalSource(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	owner := documentaryOwner(t, service)
	base := application.RegisterEvidenceWithLinksCommand{Scope: owner.Scope(), Evidence: &application.EvidenceRegistration{EvidenceType: domain.EvidenceTypeSource, Description: "Report", SourceRef: domain.SourceReference{Provider: "external", ID: "report"}, CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)}, Links: []application.EvidenceLinkInput{observedCriterionLink(owner)}}
	link := base.Links[0]
	link.CriterionRevision = nil
	base.Links = []application.EvidenceLinkInput{link}
	if _, err := service.RegisterEvidenceWithLinks(ctx, planningIntent(t.Name()+"-missing"), base); err == nil {
		t.Fatal("criterion revision was not required")
	}
	link = observedCriterionLink(owner)
	revision := domain.CriterionRevision(2)
	link.CriterionRevision = &revision
	base.Links = []application.EvidenceLinkInput{link}
	if _, err := service.RegisterEvidenceWithLinks(ctx, planningIntent(t.Name()+"-stale"), base); err == nil {
		t.Fatal("stale criterion accepted")
	}
	_, err := service.RegisterEvidence(ctx, commandContext(), application.RegisterEvidenceCommand{Scope: owner.Scope(), EvidenceType: domain.EvidenceTypeAttestation, Description: "Forged source", SourceRef: domain.SourceReference{Provider: domain.EntityNoteSourceProvider, ID: id("0199ed90-0000-7000-8000-999999999999").String()}, CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)})
	if err == nil {
		t.Fatal("reserved provider accepted a missing observation")
	}
}

func TestContextualArtifactAndDecisionUseCanonicalRelations(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	owner := documentaryOwner(t, service)
	work, err := service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: owner.Scope(), ObjectiveID: objectiveForWorkFixture(t, service, owner.Scope()), Title: "Write README", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := service.RegisterArtifactInContext(ctx, planningIntent(t.Name()+"-artifact"), application.RegisterArtifactInContextCommand{Scope: owner.Scope(), SubjectRef: work.Value.Ref(), Relationship: domain.RelationTypeProduces, Artifact: application.ArtifactRegistration{ArtifactType: "document", Name: "README", URI: "https://example.test/README.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Value.Relation.RelationType != domain.RelationTypeProduces || artifact.Value.Relation.SourceRef != work.Value.Ref() || artifact.Value.Relation.TargetRef != artifact.Value.Artifact.Ref() {
		t.Fatalf("relation=%#v", artifact.Value.Relation)
	}
	decision, err := service.ProposeDecisionInContext(ctx, planningIntent(t.Name()+"-decision"), application.ProposeDecisionInContextCommand{Scope: owner.Scope(), SubjectRef: work.Value.Ref(), Decision: application.DecisionProposal{Title: "README audience", Proposal: "Write for developers", Rationale: "First integration is developer led"}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Value.Decision.Lifecycle != domain.DecisionLifecycleProposed || decision.Value.Relation.RelationType != domain.RelationTypeRelatesTo {
		t.Fatalf("decision=%#v", decision.Value)
	}
	state, err := service.GetOutcomeState(ctx, owner.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.EvidenceLinks) != 0 || state.WorkItems[0].Lifecycle != domain.WorkItemLifecycleBacklog {
		t.Fatal("documentary context changed operational work or added EvidenceLinks")
	}
}
