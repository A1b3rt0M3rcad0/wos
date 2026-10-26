package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type ArtifactRegistration struct {
	ArtifactType  string     `json:"artifact_type"`
	Name          string     `json:"name"`
	URI           string     `json:"uri"`
	MediaType     string     `wos:"optional" json:"media_type,omitempty"`
	Checksum      string     `wos:"optional" json:"checksum,omitempty"`
	SourceVersion string     `wos:"optional" json:"source_version,omitempty"`
	ProducedAt    *time.Time `wos:"optional" json:"produced_at,omitempty"`
}

type EvidenceRegistration struct {
	EvidenceType  domain.EvidenceType    `json:"evidence_type"`
	Description   string                 `wos:"required" json:"description"`
	SourceRef     domain.SourceReference `wos:"optional" json:"source_ref"`
	CapturedAt    time.Time              `json:"captured_at"`
	ArtifactID    *domain.ID             `wos:"optional" json:"artifact_id,omitempty"`
	Measurement   *domain.Measurement    `wos:"optional" json:"measurement,omitempty"`
	SourceVersion string                 `wos:"optional" json:"source_version,omitempty"`
	Checksum      string                 `wos:"optional" json:"checksum,omitempty"`
}

type EvidenceObservation struct {
	SubjectRef  domain.EntityRef `json:"subject_ref"`
	CriterionID *domain.ID       `wos:"optional" json:"criterion_id,omitempty"`
	Body        string           `json:"body"`
	OccurredAt  *time.Time       `wos:"optional" json:"occurred_at,omitempty"`
}

type EvidenceLinkInput struct {
	TargetRef             domain.EntityRef          `json:"target_ref"`
	TargetExpectedVersion *domain.Version           `wos:"optional" json:"target_expected_version,string,omitempty"`
	CriterionID           *domain.ID                `wos:"optional" json:"criterion_id,omitempty"`
	CriterionRevision     *domain.CriterionRevision `wos:"optional" json:"criterion_revision,string,omitempty"`
	Stance                domain.EvidenceStance     `json:"stance"`
	Rationale             string                    `json:"rationale"`
}

type RegisterEvidenceWithLinksCommand struct {
	Scope              domain.Scope          `json:"scope"`
	Evidence           *EvidenceRegistration `wos:"optional" json:"evidence,omitempty"`
	ExistingEvidenceID *domain.ID            `wos:"optional" json:"existing_evidence_id,omitempty"`
	Artifact           *ArtifactRegistration `wos:"optional" json:"artifact,omitempty"`
	Observation        *EvidenceObservation  `wos:"optional" json:"observation,omitempty"`
	Links              []EvidenceLinkInput   `wos:"required" json:"links"`
}

type RegisterEvidenceWithLinksResult struct {
	Evidence        domain.Evidence       `json:"evidence"`
	EvidenceCreated bool                  `json:"evidence_created"`
	Artifact        *domain.Artifact      `json:"artifact,omitempty"`
	Observation     *domain.EntityNote    `json:"observation,omitempty"`
	Links           []domain.EvidenceLink `json:"links"`
}

type RegisterArtifactInContextCommand struct {
	Scope        domain.Scope         `json:"scope"`
	SubjectRef   domain.EntityRef     `json:"subject_ref"`
	Relationship domain.RelationType  `json:"relationship"`
	Artifact     ArtifactRegistration `json:"artifact"`
}

type RegisterArtifactInContextResult struct {
	Artifact domain.Artifact `json:"artifact"`
	Relation domain.Relation `json:"relation"`
}

type DecisionProposal struct {
	Title             string   `json:"title"`
	Proposal          string   `json:"proposal"`
	Alternatives      []string `wos:"optional" json:"alternatives,omitempty"`
	ChosenAlternative string   `wos:"optional" json:"chosen_alternative,omitempty"`
	Rationale         string   `json:"rationale"`
}

type ProposeDecisionInContextCommand struct {
	Scope      domain.Scope     `json:"scope"`
	SubjectRef domain.EntityRef `json:"subject_ref"`
	Decision   DecisionProposal `json:"decision"`
}

type ProposeDecisionInContextResult struct {
	Decision domain.Decision `json:"decision"`
	Relation domain.Relation `json:"relation"`
}

func requireDocumentaryContext(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, subject *domain.EntityRef) error {
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	if outcome.IsArchived() {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "archived Outcomes cannot receive contextual records")
	}
	if subject != nil {
		if subject.Scope != scope {
			return domain.NewError(domain.ErrorCodeInvalidScope, "context subject must belong to this Outcome")
		}
		return validateNoteSubject(ctx, uow, *subject, nil)
	}
	return nil
}

func (s *Service) registerArtifactInUnitOfWork(ctx context.Context, uow ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, input ArtifactRegistration, at time.Time) (domain.Artifact, error) {
	id, err := s.ids.NewID()
	if err != nil {
		return domain.Artifact{}, err
	}
	value, err := domain.NewArtifact(id, scope, input.ArtifactType, input.Name, input.URI, input.MediaType, input.Checksum, input.SourceVersion, cc.Actor, input.ProducedAt, at)
	if err != nil {
		return domain.Artifact{}, err
	}
	artifacts, _, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return domain.Artifact{}, err
	}
	if err = artifacts.Insert(ctx, value); err != nil {
		return domain.Artifact{}, err
	}
	return value, nil
}

func validateObservedEvidenceTarget(ctx context.Context, uow ports.UnitOfWork, decisions ports.DecisionRepository, input EvidenceLinkInput, requireObservedCriterion bool) error {
	if err := validateEvidenceLinkTarget(ctx, uow, decisions, input.TargetRef, input.CriterionID); err != nil {
		return err
	}
	if input.CriterionID == nil && input.CriterionRevision != nil {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "criterion_revision requires criterion_id")
	}
	if input.CriterionRevision != nil && *input.CriterionRevision < domain.InitialCriterionRevision {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "criterion_revision must be positive")
	}
	if input.CriterionID != nil && requireObservedCriterion && (input.CriterionRevision == nil || input.TargetExpectedVersion == nil) {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "criterion links require observed criterion_revision and target_expected_version")
	}
	if input.TargetExpectedVersion == nil && input.CriterionRevision == nil {
		return nil
	}
	var version domain.Version
	var criteria domain.CriterionSet
	switch input.TargetRef.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, input.TargetRef.NamespaceID, input.TargetRef.OutcomeID)
		if err != nil {
			return err
		}
		version = value.Version
		criteria = value.Criteria
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, input.TargetRef.Scope, input.TargetRef.ID)
		if err != nil {
			return err
		}
		version = value.Version
		criteria = value.Criteria
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, input.TargetRef.Scope, input.TargetRef.ID)
		if err != nil {
			return err
		}
		version = value.Version
		criteria = value.Criteria
	case domain.EntityKindIssue:
		repo, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return err
		}
		value, err := repo.Get(ctx, input.TargetRef.Scope, input.TargetRef.ID)
		if err != nil {
			return err
		}
		version = value.Version
	case domain.EntityKindDecision:
		value, err := decisions.Get(ctx, input.TargetRef.Scope, input.TargetRef.ID)
		if err != nil {
			return err
		}
		version = value.Version
	}
	if input.TargetExpectedVersion != nil {
		if err := input.TargetExpectedVersion.Validate(); err != nil {
			return err
		}
		if version != *input.TargetExpectedVersion {
			return domain.NewError(domain.ErrorCodeVersionConflict, "evidence target version changed; review the current target")
		}
	}
	if input.CriterionID != nil && input.CriterionRevision != nil {
		criterion, err := criteria.Find(*input.CriterionID)
		if err != nil {
			return err
		}
		if criterion.Status != domain.CriterionStatusActive || criterion.Revision != *input.CriterionRevision {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "criterion definition changed or was retired; choose the current criterion")
		}
	}
	return nil
}

func (s *Service) createEvidenceLinkInUnitOfWork(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, evidenceID domain.ID, input EvidenceLinkInput, at time.Time, requireObservedCriterion bool) (domain.EvidenceLink, error) {
	if input.TargetRef.Scope != scope {
		return domain.EvidenceLink{}, domain.NewError(domain.ErrorCodeEvidenceLink, "target_ref must match command scope")
	}
	_, evidence, links, decisions, err := documentaryRepositories(uow)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	observation, err := evidence.Get(ctx, scope, evidenceID)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	if observation.Lifecycle != domain.EvidenceLifecycleRegistered {
		return domain.EvidenceLink{}, domain.NewError(domain.ErrorCodePreconditionFailed, "retracted evidence cannot receive a new link")
	}
	if _, err := validateEntityNoteSource(ctx, uow, scope, observation.SourceRef); err != nil {
		return domain.EvidenceLink{}, err
	}
	if err := validateObservedEvidenceTarget(ctx, uow, decisions, input, requireObservedCriterion); err != nil {
		return domain.EvidenceLink{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	value, err := domain.NewEvidenceLink(id, scope, evidenceID, input.TargetRef, input.CriterionID, input.Stance, input.Rationale, at)
	if err != nil {
		return domain.EvidenceLink{}, err
	}
	if err := links.Insert(ctx, value); err != nil {
		return domain.EvidenceLink{}, err
	}
	return value, nil
}

func (s *Service) RegisterEvidenceWithLinks(ctx context.Context, cc domain.CommandContext, cmd RegisterEvidenceWithLinksCommand) (MutationResult[RegisterEvidenceWithLinksResult], error) {
	var zero MutationResult[RegisterEvidenceWithLinksResult]
	if err := cc.Validate(); err != nil {
		return zero, err
	}
	if err := cmd.Scope.Validate(); err != nil {
		return zero, err
	}
	if err := requireWorkspaceIntent(cc); err != nil {
		return MutationResult[RegisterEvidenceWithLinksResult]{}, err
	}
	if (cmd.Evidence == nil) == (cmd.ExistingEvidenceID == nil) {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "choose exactly one new or existing Evidence")
	}
	if len(cmd.Links) == 0 || len(cmd.Links) > 100 {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "evidence registration requires 1..100 explicit links")
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return zero, err
	}
	if cmd.ExistingEvidenceID != nil && (cmd.Artifact != nil || cmd.Observation != nil) {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "existing Evidence is immutable; Artifact and observation only accompany new Evidence")
	}
	if cmd.Artifact != nil && cmd.Evidence.ArtifactID != nil {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "choose a new or an existing Artifact")
	}
	if cmd.Observation != nil && (cmd.Evidence.EvidenceType != domain.EvidenceTypeAttestation || cmd.Evidence.Measurement != nil || cmd.Evidence.SourceRef.Provider != "" || cmd.Evidence.SourceRef.ID != "" || cmd.Evidence.SourceRef.URI != "") {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "a human observation creates attestation Evidence with a server-verified source")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (RegisterEvidenceWithLinksResult, domain.OutcomeRevision, error) {
		var empty RegisterEvidenceWithLinksResult
		if err := requireDocumentaryContext(ctx, uow, cmd.Scope, nil); err != nil {
			return empty, 0, err
		}
		at, err := s.transactionTime(ctx, uow)
		if err != nil {
			return empty, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return empty, 0, err
		}
		result := RegisterEvidenceWithLinksResult{EvidenceCreated: cmd.Evidence != nil, Links: make([]domain.EvidenceLink, 0, len(cmd.Links))}
		artifacts, evidence, _, _, err := documentaryRepositories(uow)
		if err != nil {
			return empty, 0, err
		}
		if cmd.Artifact != nil {
			artifact, err := s.registerArtifactInUnitOfWork(ctx, uow, cc, cmd.Scope, *cmd.Artifact, at)
			if err != nil {
				return empty, 0, err
			}
			result.Artifact = &artifact
		}
		if cmd.ExistingEvidenceID != nil {
			result.Evidence, err = evidence.Get(ctx, cmd.Scope, *cmd.ExistingEvidenceID)
			if err != nil {
				return empty, 0, err
			}
		} else {
			input := *cmd.Evidence
			if result.Artifact != nil {
				input.ArtifactID = &result.Artifact.ID
			}
			if cmd.Observation != nil {
				observation := cmd.Observation
				note, err := s.appendEntityNoteInUnitOfWork(ctx, uow, cc, AppendEntityNoteCommand{Scope: cmd.Scope, SubjectRef: observation.SubjectRef, CriterionID: observation.CriterionID, Type: domain.EntityNoteObservation, Body: observation.Body, OccurredAt: observation.OccurredAt}, at, revision)
				if err != nil {
					return empty, 0, err
				}
				result.Observation = &note
				input.SourceRef = domain.SourceReference{Provider: domain.EntityNoteSourceProvider, ID: note.ID.String(), DisplayName: "Recorded participant observation"}
				if input.CapturedAt.IsZero() {
					input.CapturedAt = at
					if note.OccurredAt != nil {
						input.CapturedAt = *note.OccurredAt
					}
				}
			}
			if input.ArtifactID != nil {
				artifact, err := artifacts.Get(ctx, cmd.Scope, *input.ArtifactID)
				if err != nil {
					return empty, 0, err
				}
				if artifact.Lifecycle != domain.ArtifactLifecycleRegistered {
					return empty, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "evidence cannot reference a withdrawn artifact")
				}
			}
			if _, err := validateEntityNoteSource(ctx, uow, cmd.Scope, input.SourceRef); err != nil {
				return empty, 0, err
			}
			id, err := s.ids.NewID()
			if err != nil {
				return empty, 0, err
			}
			value, err := domain.NewEvidence(id, cmd.Scope, input.EvidenceType, input.Description, input.SourceRef, cc.Actor, input.CapturedAt, input.ArtifactID, input.Measurement, input.SourceVersion, input.Checksum, at)
			if err != nil {
				return empty, 0, err
			}
			if err := evidence.Insert(ctx, value); err != nil {
				return empty, 0, err
			}
			result.Evidence = value
		}
		for _, input := range cmd.Links {
			link, err := s.createEvidenceLinkInUnitOfWork(ctx, uow, cmd.Scope, result.Evidence.ID, input, at, true)
			if err != nil {
				return empty, 0, err
			}
			result.Links = append(result.Links, link)
		}
		return result, revision, nil
	})
}

func (s *Service) insertContextualRelation(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, source, target domain.EntityRef, kind domain.RelationType, at time.Time) (domain.Relation, error) {
	if source.Scope != scope || target.Scope != scope {
		return domain.Relation{}, domain.NewError(domain.ErrorCodeInvalidScope, "documentary relation endpoints must belong to this Outcome")
	}
	if kind != domain.RelationTypeRelatesTo && kind != domain.RelationTypeProduces {
		return domain.Relation{}, domain.NewError(domain.ErrorCodeInvalidRelation, "context relationship must be relates_to or produces")
	}
	if kind == domain.RelationTypeProduces && (target.Kind != domain.EntityKindArtifact || source.Kind != domain.EntityKindWorkItem && source.Kind != domain.EntityKindObjective && source.Kind != domain.EntityKindOutcome) {
		return domain.Relation{}, domain.NewError(domain.ErrorCodeInvalidRelation, "produces requires an operational producer and Artifact target")
	}
	if kind == domain.RelationTypeRelatesTo && string(source.Kind)+"/"+source.ID.String() > string(target.Kind)+"/"+target.ID.String() {
		source, target = target, source
	}
	id, err := s.ids.NewID()
	if err != nil {
		return domain.Relation{}, err
	}
	value := domain.Relation{ID: id, Scope: scope, Version: domain.InitialVersion, SourceRef: source, TargetRef: target, RelationType: kind, Lifecycle: domain.RelationLifecycleActive, CreatedAt: at, UpdatedAt: at}
	if err := value.Validate(); err != nil {
		return domain.Relation{}, err
	}
	if err := uow.Relations().Insert(ctx, value); err != nil {
		return domain.Relation{}, err
	}
	return value, nil
}

func (s *Service) RegisterArtifactInContext(ctx context.Context, cc domain.CommandContext, cmd RegisterArtifactInContextCommand) (MutationResult[RegisterArtifactInContextResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[RegisterArtifactInContextResult]{}, err
	}
	if err := requireWorkspaceIntent(cc); err != nil {
		return MutationResult[RegisterArtifactInContextResult]{}, err
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return MutationResult[RegisterArtifactInContextResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (RegisterArtifactInContextResult, domain.OutcomeRevision, error) {
		var zero RegisterArtifactInContextResult
		if err := requireDocumentaryContext(ctx, uow, cmd.Scope, &cmd.SubjectRef); err != nil {
			return zero, 0, err
		}
		at, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
		}
		artifact, err := s.registerArtifactInUnitOfWork(ctx, uow, cc, cmd.Scope, cmd.Artifact, at)
		if err != nil {
			return zero, 0, err
		}
		relation, err := s.insertContextualRelation(ctx, uow, cmd.Scope, cmd.SubjectRef, artifact.Ref(), cmd.Relationship, at)
		if err != nil {
			return zero, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return RegisterArtifactInContextResult{Artifact: artifact, Relation: relation}, revision, err
	})
}

func (s *Service) ProposeDecisionInContext(ctx context.Context, cc domain.CommandContext, cmd ProposeDecisionInContextCommand) (MutationResult[ProposeDecisionInContextResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[ProposeDecisionInContextResult]{}, err
	}
	if err := requireWorkspaceIntent(cc); err != nil {
		return MutationResult[ProposeDecisionInContextResult]{}, err
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return MutationResult[ProposeDecisionInContextResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (ProposeDecisionInContextResult, domain.OutcomeRevision, error) {
		var zero ProposeDecisionInContextResult
		if err := requireDocumentaryContext(ctx, uow, cmd.Scope, &cmd.SubjectRef); err != nil {
			return zero, 0, err
		}
		at, err := s.transactionTime(ctx, uow)
		if err != nil {
			return zero, 0, err
		}
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		input := cmd.Decision
		decision, err := domain.NewDecision(id, cmd.Scope, input.Title, input.Proposal, input.Alternatives, input.ChosenAlternative, input.Rationale, cc.Actor, at)
		if err != nil {
			return zero, 0, err
		}
		_, _, _, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return zero, 0, err
		}
		if err := decisions.Insert(ctx, decision); err != nil {
			return zero, 0, err
		}
		relation, err := s.insertContextualRelation(ctx, uow, cmd.Scope, cmd.SubjectRef, decision.Ref(), domain.RelationTypeRelatesTo, at)
		if err != nil {
			return zero, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return ProposeDecisionInContextResult{Decision: decision, Relation: relation}, revision, err
	})
}

func documentaryContextEvents[T any](s *Service, cc domain.CommandContext, meta commandMetadata, value T, revision domain.OutcomeRevision) ([]domain.DomainEvent, bool, error) {
	specs := []compoundEventSpec{}
	payloads := []any{}
	add := func(eventType string, ref domain.EntityRef, version *domain.Version, payload any) {
		specs = append(specs, compoundEventSpec{eventType: eventType, ref: ref, after: version})
		payloads = append(payloads, payload)
	}
	switch result := any(value).(type) {
	case RegisterEvidenceWithLinksResult:
		if result.Artifact != nil {
			add("artifact.registered", result.Artifact.Ref(), versionPtr(result.Artifact.Version), *result.Artifact)
		}
		if result.Observation != nil {
			add("entity_note.recorded", result.Observation.SubjectRef, nil, *result.Observation)
		}
		if result.EvidenceCreated {
			add("evidence.registered", result.Evidence.Ref(), versionPtr(result.Evidence.Version), result.Evidence)
		}
		for _, link := range result.Links {
			add("evidence.link_created", link.Ref(), versionPtr(link.Version), link)
		}
	case RegisterArtifactInContextResult:
		add("artifact.registered", result.Artifact.Ref(), versionPtr(result.Artifact.Version), result.Artifact)
		add("relation.created", result.Relation.Ref(), versionPtr(result.Relation.Version), result.Relation)
	case ProposeDecisionInContextResult:
		add("decision.proposed", result.Decision.Ref(), versionPtr(result.Decision.Version), result.Decision)
		add("relation.created", result.Relation.Ref(), versionPtr(result.Relation.Version), result.Relation)
	default:
		return nil, false, nil
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, true, err
	}
	for i := range events {
		events[i].Payload, err = json.Marshal(payloads[i])
		if err != nil {
			return nil, true, err
		}
		if err := events[i].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, nil
}
