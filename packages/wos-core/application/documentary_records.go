package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type RegisterArtifactCommand struct {
	Scope         domain.Scope
	ArtifactType  string
	Name          string
	URI           string
	MediaType     string
	Checksum      string
	SourceVersion string
	ProducedAt    *time.Time
}

type WithdrawArtifactCommand struct {
	Scope           domain.Scope
	ArtifactID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type RegisterEvidenceCommand struct {
	Scope         domain.Scope
	EvidenceType  domain.EvidenceType
	Description   string
	SourceRef     domain.SourceReference
	CapturedAt    time.Time
	ArtifactID    *domain.ID
	Measurement   *domain.Measurement
	SourceVersion string
	Checksum      string
}

type RetractEvidenceCommand struct {
	Scope           domain.Scope
	EvidenceID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type CreateEvidenceLinkCommand struct {
	Scope       domain.Scope
	EvidenceID  domain.ID
	TargetRef   domain.EntityRef
	CriterionID *domain.ID
	Stance      domain.EvidenceStance
	Rationale   string
}

type RetractEvidenceLinkCommand struct {
	Scope           domain.Scope
	EvidenceLinkID  domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type ProposeDecisionCommand struct {
	Scope             domain.Scope
	Title             string
	Proposal          string
	Alternatives      []string
	ChosenAlternative string
	Rationale         string
}

type UpdateDecisionCommand struct {
	Scope             domain.Scope
	DecisionID        domain.ID
	ExpectedVersion   domain.Version
	Title             *string
	Proposal          *string
	Alternatives      *[]string
	ChosenAlternative *string
	Rationale         *string
}

type AcceptDecisionCommand struct {
	Scope           domain.Scope
	DecisionID      domain.ID
	ExpectedVersion domain.Version
}

type RejectDecisionCommand struct {
	Scope           domain.Scope
	DecisionID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type SupersedeDecisionCommand struct {
	Scope             domain.Scope
	DecisionID        domain.ID
	ExpectedVersion   domain.Version
	Title             string
	Proposal          string
	Alternatives      []string
	ChosenAlternative string
	Rationale         string
}

type DecisionSupersessionResult struct {
	Successor   domain.Decision `json:"successor"`
	Predecessor domain.Decision `json:"predecessor"`
}

func (s *Service) RegisterArtifact(ctx context.Context, cc domain.CommandContext, cmd RegisterArtifactCommand) (MutationResult[domain.Artifact], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	value, err := domain.NewArtifact(id, cmd.Scope, cmd.ArtifactType, cmd.Name, cmd.URI, cmd.MediaType, cmd.Checksum, cmd.SourceVersion, cc.Actor, cmd.ProducedAt, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Artifact, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Artifact{}, 0, err
		}
		artifacts, _, _, _, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := artifacts.Insert(ctx, value); err != nil {
			return domain.Artifact{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) WithdrawArtifact(ctx context.Context, cc domain.CommandContext, cmd WithdrawArtifactCommand) (MutationResult[domain.Artifact], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Artifact, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Artifact{}, 0, err
		}
		artifacts, _, _, _, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		value, err := artifacts.Get(ctx, cmd.Scope, cmd.ArtifactID)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := value.Withdraw(cmd.Reason, s.clock.Now().UTC()); err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := artifacts.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Artifact{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RegisterEvidence(ctx context.Context, cc domain.CommandContext, cmd RegisterEvidenceCommand) (MutationResult[domain.Evidence], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	value, err := domain.NewEvidence(id, cmd.Scope, cmd.EvidenceType, cmd.Description, cmd.SourceRef, cc.Actor, cmd.CapturedAt, cmd.ArtifactID, cmd.Measurement, cmd.SourceVersion, cmd.Checksum, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Evidence, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Evidence{}, 0, err
		}
		artifacts, evidence, _, _, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		if value.ArtifactID != nil {
			artifact, err := artifacts.Get(ctx, cmd.Scope, *value.ArtifactID)
			if err != nil {
				return domain.Evidence{}, 0, err
			}
			if artifact.Lifecycle != domain.ArtifactLifecycleRegistered {
				return domain.Evidence{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "evidence cannot reference a withdrawn artifact")
			}
		}
		if err := evidence.Insert(ctx, value); err != nil {
			return domain.Evidence{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RetractEvidence(ctx context.Context, cc domain.CommandContext, cmd RetractEvidenceCommand) (MutationResult[domain.Evidence], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Evidence, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Evidence{}, 0, err
		}
		_, evidence, _, _, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		value, err := evidence.Get(ctx, cmd.Scope, cmd.EvidenceID)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := value.Retract(cmd.Reason, s.clock.Now().UTC()); err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := evidence.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Evidence{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) CreateEvidenceLink(ctx context.Context, cc domain.CommandContext, cmd CreateEvidenceLinkCommand) (MutationResult[domain.EvidenceLink], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	if cmd.TargetRef.Scope != cmd.Scope {
		return MutationResult[domain.EvidenceLink]{}, domain.NewError(domain.ErrorCodeEvidenceLink, "target_ref must match command scope")
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	value, err := domain.NewEvidenceLink(id, cmd.Scope, cmd.EvidenceID, cmd.TargetRef, cmd.CriterionID, cmd.Stance, cmd.Rationale, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.EvidenceLink, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		_, evidence, links, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		ev, err := evidence.Get(ctx, cmd.Scope, cmd.EvidenceID)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if ev.Lifecycle != domain.EvidenceLifecycleRegistered {
			return domain.EvidenceLink{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "retracted evidence cannot receive a new link")
		}
		if err := validateEvidenceLinkTarget(ctx, uow, decisions, value.TargetRef, value.CriterionID); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := links.Insert(ctx, value); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RetractEvidenceLink(ctx context.Context, cc domain.CommandContext, cmd RetractEvidenceLinkCommand) (MutationResult[domain.EvidenceLink], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.EvidenceLink, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		_, _, links, _, err := documentaryRepositories(uow)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		value, err := links.Get(ctx, cmd.Scope, cmd.EvidenceLinkID)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := value.Retract(cmd.Reason, s.clock.Now().UTC()); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := links.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) ProposeDecision(ctx context.Context, cc domain.CommandContext, cmd ProposeDecisionCommand) (MutationResult[domain.Decision], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	value, err := domain.NewDecision(id, cmd.Scope, cmd.Title, cmd.Proposal, cmd.Alternatives, cmd.ChosenAlternative, cmd.Rationale, cc.Actor, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		_, _, _, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := decisions.Insert(ctx, value); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) UpdateDecision(ctx context.Context, cc domain.CommandContext, cmd UpdateDecisionCommand) (MutationResult[domain.Decision], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	if cmd.Title == nil && cmd.Proposal == nil && cmd.Alternatives == nil && cmd.ChosenAlternative == nil && cmd.Rationale == nil {
		return MutationResult[domain.Decision]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "decision patch has no editable fields")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		_, _, _, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		value, err := decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if value.Version != cmd.ExpectedVersion {
			return domain.Decision{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "decision expected_version is stale")
		}
		changed, err := value.Update(cmd.Title, cmd.Proposal, cmd.ChosenAlternative, cmd.Rationale, cmd.Alternatives, s.clock.Now().UTC())
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if !changed {
			return value, coordination.Revision, nil
		}
		if err := decisions.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) AcceptDecision(ctx context.Context, cc domain.CommandContext, cmd AcceptDecisionCommand) (MutationResult[domain.Decision], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	return s.mutateDecision(ctx, cc, cmd, cmd.Scope, cmd.DecisionID, cmd.ExpectedVersion, func(value *domain.Decision) error {
		return value.Accept(cc.Actor, s.clock.Now().UTC())
	})
}

func (s *Service) RejectDecision(ctx context.Context, cc domain.CommandContext, cmd RejectDecisionCommand) (MutationResult[domain.Decision], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	return s.mutateDecision(ctx, cc, cmd, cmd.Scope, cmd.DecisionID, cmd.ExpectedVersion, func(value *domain.Decision) error {
		return value.Reject(cmd.Reason, cc.Actor, s.clock.Now().UTC())
	})
}

func (s *Service) mutateDecision(ctx context.Context, cc domain.CommandContext, command any, scope domain.Scope, id domain.ID, expected domain.Version, mutate func(*domain.Decision) error) (MutationResult[domain.Decision], error) {
	return transactCommand(ctx, s, cc, command, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, scope); err != nil {
			return domain.Decision{}, 0, err
		}
		_, _, _, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		value, err := decisions.Get(ctx, scope, id)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := mutate(&value); err != nil {
			return domain.Decision{}, 0, err
		}
		if err := decisions.Save(ctx, value, expected); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, scope)
		return value, revision, err
	})
}

func (s *Service) SupersedeDecision(ctx context.Context, cc domain.CommandContext, cmd SupersedeDecisionCommand) (MutationResult[DecisionSupersessionResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[DecisionSupersessionResult]{}, err
	}
	successorID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[DecisionSupersessionResult]{}, err
	}
	now := s.clock.Now().UTC()
	successor, err := domain.NewSupersedingDecision(successorID, cmd.Scope, cmd.DecisionID, cmd.Title, cmd.Proposal, cmd.Alternatives, cmd.ChosenAlternative, cmd.Rationale, cc.Actor, now)
	if err != nil {
		return MutationResult[DecisionSupersessionResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (DecisionSupersessionResult, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		_, _, _, decisions, err := documentaryRepositories(uow)
		if err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		predecessor, err := decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		if predecessor.Version != cmd.ExpectedVersion {
			return DecisionSupersessionResult{}, 0, domain.NewError(domain.ErrorCodeVersionConflict, "decision expected_version is stale")
		}
		if predecessor.Lifecycle != domain.DecisionLifecycleAccepted {
			return DecisionSupersessionResult{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "only accepted decision can be superseded")
		}
		all, err := decisions.ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		for _, existing := range all {
			if existing.SupersedesDecisionID != nil && *existing.SupersedesDecisionID == predecessor.ID {
				return DecisionSupersessionResult{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "accepted decision already has a direct successor")
			}
		}
		if err := predecessor.MarkSuperseded(now); err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		if err := decisions.Save(ctx, predecessor, cmd.ExpectedVersion); err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		if err := decisions.Insert(ctx, successor); err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return DecisionSupersessionResult{}, 0, err
		}
		return DecisionSupersessionResult{Successor: successor, Predecessor: predecessor}, revision, nil
	})
}

func (s *Service) GetArtifact(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Artifact], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	artifacts, _, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	value, err := artifacts.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.Artifact]{Value: value, OutcomeRevision: c.Revision}, err
}

func (s *Service) ListArtifacts(ctx context.Context, scope domain.Scope) ([]domain.Artifact, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	artifacts, _, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	values, err := artifacts.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return values, c.Revision, err
}

func (s *Service) GetEvidence(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Evidence], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	_, evidence, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	value, err := evidence.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.Evidence]{Value: value, OutcomeRevision: c.Revision}, err
}

func (s *Service) ListEvidence(ctx context.Context, scope domain.Scope) ([]domain.Evidence, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	_, evidence, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	values, err := evidence.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return values, c.Revision, err
}

func (s *Service) GetEvidenceLink(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.EvidenceLink], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	_, _, links, _, err := documentaryRepositories(uow)
	if err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	value, err := links.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.EvidenceLink]{Value: value, OutcomeRevision: c.Revision}, err
}

func (s *Service) ListEvidenceLinks(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	_, _, links, _, err := documentaryRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	values, err := links.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return values, c.Revision, err
}

func (s *Service) GetDecision(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Decision], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	_, _, _, decisions, err := documentaryRepositories(uow)
	if err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	value, err := decisions.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return ReadResult[domain.Decision]{Value: value, OutcomeRevision: c.Revision}, err
}

func (s *Service) ListDecisions(ctx context.Context, scope domain.Scope) ([]domain.Decision, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	_, _, _, decisions, err := documentaryRepositories(uow)
	if err != nil {
		return nil, 0, err
	}
	values, err := decisions.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	c, err := uow.Coordination().LockOutcome(ctx, scope)
	return values, c.Revision, err
}

func documentaryRepositories(uow ports.UnitOfWork) (ports.ArtifactRepository, ports.EvidenceRepository, ports.EvidenceLinkRepository, ports.DecisionRepository, error) {
	value, ok := uow.(ports.DocumentaryUnitOfWork)
	if !ok {
		return nil, nil, nil, nil, domain.NewError(domain.ErrorCodeInvalidConfig, "transaction adapter does not support documentary records")
	}
	return value.Artifacts(), value.Evidence(), value.EvidenceLinks(), value.Decisions(), nil
}

func validateEvidenceLinkTarget(ctx context.Context, uow ports.UnitOfWork, decisions ports.DecisionRepository, ref domain.EntityRef, criterionID *domain.ID) error {
	switch ref.Kind {
	case domain.EntityKindOutcome:
		if ref.ID != ref.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeEvidenceLink, "outcome target id must equal outcome scope id")
		}
		value, err := uow.Outcomes().Get(ctx, ref.Scope.NamespaceID, ref.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if criterionID != nil {
			_, err = value.Criteria.Find(*criterionID)
			return err
		}
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		if criterionID != nil {
			_, err = value.Criteria.Find(*criterionID)
			return err
		}
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return err
		}
		if criterionID != nil {
			_, err = value.Criteria.Find(*criterionID)
			return err
		}
	case domain.EntityKindIssue:
		if criterionID != nil {
			return domain.NewError(domain.ErrorCodeEvidenceLink, "issue target cannot own criterion_id")
		}
		value, ok := uow.(ports.IssueBlockerUnitOfWork)
		if !ok {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "transaction adapter does not support issues")
		}
		_, err := value.Issues().Get(ctx, ref.Scope, ref.ID)
		return err
	case domain.EntityKindDecision:
		if criterionID != nil {
			return domain.NewError(domain.ErrorCodeEvidenceLink, "decision target cannot own criterion_id")
		}
		_, err := decisions.Get(ctx, ref.Scope, ref.ID)
		return err
	default:
		return domain.NewError(domain.ErrorCodeEvidenceLink, "unsupported evidence target kind")
	}
	return nil
}
