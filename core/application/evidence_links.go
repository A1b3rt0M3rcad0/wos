package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

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
}

func (s *Service) CreateEvidenceLink(
	ctx context.Context,
	cc domain.CommandContext,
	cmd CreateEvidenceLinkCommand,
) (MutationResult[domain.EvidenceLink], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	if cmd.TargetRef.Scope != cmd.Scope {
		return MutationResult[domain.EvidenceLink]{}, domain.NewError(
			domain.ErrorCodeEvidenceLink,
			"target_ref must match command scope",
		)
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	now := s.clock.Now().UTC()
	link, err := domain.NewEvidenceLink(
		id,
		cmd.Scope,
		cmd.EvidenceID,
		cmd.TargetRef,
		cmd.CriterionID,
		cmd.Stance,
		cmd.Rationale,
		now,
	)
	if err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}

	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.EvidenceLink, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		evidence, err := repositories.evidence.Get(ctx, cmd.Scope, cmd.EvidenceID)
		if err != nil {
			return domain.EvidenceLink{}, 0, domain.WrapError(
				domain.ErrorCodeEvidenceLink,
				"evidence_id does not resolve to persisted evidence",
				err,
			)
		}
		if evidence.Lifecycle != domain.EvidenceLifecycleRegistered {
			return domain.EvidenceLink{}, 0, domain.NewError(
				domain.ErrorCodePreconditionFailed,
				"retracted evidence cannot receive an active evidence link",
			)
		}
		if err := validateEvidenceLinkTarget(ctx, uow, repositories, cmd.TargetRef, cmd.CriterionID); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := repositories.evidenceLinks.Insert(ctx, link); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return link, revision, err
	})
}

func (s *Service) RetractEvidenceLink(
	ctx context.Context,
	cc domain.CommandContext,
	cmd RetractEvidenceLinkCommand,
) (MutationResult[domain.EvidenceLink], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.EvidenceLink]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.EvidenceLink, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		link, err := repositories.evidenceLinks.Get(ctx, cmd.Scope, cmd.EvidenceLinkID)
		if err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := link.Retract(now); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		if err := repositories.evidenceLinks.Save(ctx, link, cmd.ExpectedVersion); err != nil {
			return domain.EvidenceLink{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return link, revision, err
	})
}

func (s *Service) GetEvidenceLink(
	ctx context.Context,
	scope domain.Scope,
	id domain.ID,
) (ReadResult[domain.EvidenceLink], error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	defer uow.Rollback()
	value, err := repositories.evidenceLinks.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.EvidenceLink]{}, err
	}
	return ReadResult[domain.EvidenceLink]{Value: value, OutcomeRevision: revision}, nil
}

func validateEvidenceLinkTarget(
	ctx context.Context,
	uow ports.UnitOfWork,
	repositories documentaryRepositorySet,
	target domain.EntityRef,
	criterionID *domain.ID,
) error {
	var criteria *domain.CriterionSet
	switch target.Kind {
	case domain.EntityKindOutcome:
		if target.ID != target.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeEvidenceLink, "outcome target id must equal outcome scope id")
		}
		value, err := uow.Outcomes().Get(ctx, target.Scope.NamespaceID, target.Scope.OutcomeID)
		if err != nil {
			return domain.WrapError(domain.ErrorCodeEvidenceLink, "target_ref does not resolve", err)
		}
		criteria = &value.Criteria
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, target.Scope, target.ID)
		if err != nil {
			return domain.WrapError(domain.ErrorCodeEvidenceLink, "target_ref does not resolve", err)
		}
		criteria = &value.Criteria
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, target.Scope, target.ID)
		if err != nil {
			return domain.WrapError(domain.ErrorCodeEvidenceLink, "target_ref does not resolve", err)
		}
		criteria = &value.Criteria
	case domain.EntityKindIssue:
		issues, _, err := issueBlockerRepositories(uow)
		if err != nil {
			return err
		}
		if _, err := issues.Get(ctx, target.Scope, target.ID); err != nil {
			return domain.WrapError(domain.ErrorCodeEvidenceLink, "target_ref does not resolve", err)
		}
	case domain.EntityKindDecision:
		if _, err := repositories.decisions.Get(ctx, target.Scope, target.ID); err != nil {
			return domain.WrapError(domain.ErrorCodeEvidenceLink, "target_ref does not resolve", err)
		}
	default:
		return domain.NewError(domain.ErrorCodeEvidenceLink, "target_ref kind is not supported")
	}

	if criterionID == nil {
		return nil
	}
	if criteria == nil {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "criterion_id requires a criterion-owning target")
	}
	criterion, err := criteria.Find(*criterionID)
	if err != nil {
		return domain.WrapError(domain.ErrorCodeEvidenceLink, "criterion_id does not belong to target_ref", err)
	}
	if criterion.OwnerRef != target {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "criterion_id owner does not match target_ref")
	}
	return nil
}
