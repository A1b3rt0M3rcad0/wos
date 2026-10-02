package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type RegisterArtifactCommand struct {
	Scope         domain.Scope
	ArtifactType  string
	Name          string
	URI           string
	MediaType     string
	Checksum      string
	SourceVersion string
	ProducerRef   domain.ActorRef
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
	SourceRef     domain.ExternalReference
	ProducerRef   domain.ActorRef
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

type ProposeDecisionCommand struct {
	Scope        domain.Scope
	Title        string
	Proposal     string
	Rationale    string
	Alternatives []string
}

type ReviseDecisionCommand struct {
	Scope           domain.Scope
	DecisionID      domain.ID
	ExpectedVersion domain.Version
	Title           string
	Proposal        string
	Rationale       string
	Alternatives    []string
}

type AcceptDecisionCommand struct {
	Scope             domain.Scope
	DecisionID        domain.ID
	ExpectedVersion   domain.Version
	ChosenAlternative string
	Rationale         string
}

type RejectDecisionCommand struct {
	Scope           domain.Scope
	DecisionID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

func (s *Service) RegisterArtifact(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RegisterArtifactCommand,
) (MutationResult[domain.Artifact], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	now := s.clock.Now().UTC()
	value, err := domain.NewArtifact(
		id, cmd.Scope, cmd.ArtifactType, cmd.Name, cmd.URI, cmd.MediaType,
		cmd.Checksum, cmd.SourceVersion, cmd.ProducerRef, cmd.ProducedAt, now,
	)
	if err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Artifact, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := repositories.artifacts.Insert(ctx, value); err != nil {
			return domain.Artifact{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) WithdrawArtifact(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd WithdrawArtifactCommand,
) (MutationResult[domain.Artifact], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Artifact]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Artifact, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Artifact{}, 0, err
		}
		value, err := repositories.artifacts.Get(ctx, cmd.Scope, cmd.ArtifactID)
		if err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := value.Withdraw(cmd.Reason, now); err != nil {
			return domain.Artifact{}, 0, err
		}
		if err := repositories.artifacts.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Artifact{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RegisterEvidence(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RegisterEvidenceCommand,
) (MutationResult[domain.Evidence], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	now := s.clock.Now().UTC()
	value, err := domain.NewEvidence(
		id, cmd.Scope, cmd.EvidenceType, cmd.Description, cmd.SourceRef,
		cmd.ProducerRef, cmd.CapturedAt, cmd.ArtifactID, cmd.Measurement,
		cmd.SourceVersion, cmd.Checksum, now,
	)
	if err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Evidence, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Evidence{}, 0, err
		}
		if cmd.ArtifactID != nil {
			artifact, err := repositories.artifacts.Get(ctx, cmd.Scope, *cmd.ArtifactID)
			if err != nil {
				return domain.Evidence{}, 0, err
			}
			if artifact.Lifecycle != domain.ArtifactLifecycleRegistered {
				return domain.Evidence{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "evidence artifact must be registered")
			}
		}
		if err := repositories.evidence.Insert(ctx, value); err != nil {
			return domain.Evidence{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RetractEvidence(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RetractEvidenceCommand,
) (MutationResult[domain.Evidence], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Evidence]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Evidence, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Evidence{}, 0, err
		}
		value, err := repositories.evidence.Get(ctx, cmd.Scope, cmd.EvidenceID)
		if err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := value.Retract(cmd.Reason, now); err != nil {
			return domain.Evidence{}, 0, err
		}
		if err := repositories.evidence.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Evidence{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) ProposeDecision(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd ProposeDecisionCommand,
) (MutationResult[domain.Decision], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	now := s.clock.Now().UTC()
	value, err := domain.NewDecision(
		id, cmd.Scope, cmd.Title, cmd.Proposal, cmd.Rationale, cmd.Alternatives, nil, now,
	)
	if err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		if err := repositories.decisions.Insert(ctx, value); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) ReviseDecision(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd ReviseDecisionCommand,
) (MutationResult[domain.Decision], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		value, err := repositories.decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := value.ReviseProposal(cmd.Title, cmd.Proposal, cmd.Rationale, cmd.Alternatives, now); err != nil {
			return domain.Decision{}, 0, err
		}
		if err := repositories.decisions.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) AcceptDecision(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd AcceptDecisionCommand,
) (MutationResult[domain.Decision], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		value, err := repositories.decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := value.Accept(cmd.ChosenAlternative, cmd.Rationale, commandContext.Actor, now); err != nil {
			return domain.Decision{}, 0, err
		}
		if err := repositories.decisions.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) RejectDecision(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RejectDecisionCommand,
) (MutationResult[domain.Decision], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Decision]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Decision, domain.OutcomeRevision, error) {
		repositories, err := documentaryRepositories(uow)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := requireDocumentaryOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Decision{}, 0, err
		}
		value, err := repositories.decisions.Get(ctx, cmd.Scope, cmd.DecisionID)
		if err != nil {
			return domain.Decision{}, 0, err
		}
		if err := value.Reject(cmd.Reason, commandContext.Actor, now); err != nil {
			return domain.Decision{}, 0, err
		}
		if err := repositories.decisions.Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.Decision{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) GetArtifact(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Artifact], error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	defer uow.Rollback()
	value, err := repositories.artifacts.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Artifact]{}, err
	}
	return ReadResult[domain.Artifact]{Value: value, OutcomeRevision: revision}, nil
}

func (s *Service) GetEvidence(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Evidence], error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	defer uow.Rollback()
	value, err := repositories.evidence.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Evidence]{}, err
	}
	return ReadResult[domain.Evidence]{Value: value, OutcomeRevision: revision}, nil
}

func (s *Service) GetDecision(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Decision], error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	defer uow.Rollback()
	value, err := repositories.decisions.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Decision]{}, err
	}
	return ReadResult[domain.Decision]{Value: value, OutcomeRevision: revision}, nil
}

type documentaryRepositorySet struct {
	artifacts     ports.ArtifactRepository
	evidence      ports.EvidenceRepository
	decisions     ports.DecisionRepository
	evidenceLinks ports.EvidenceLinkRepository
}

func documentaryRepositories(uow ports.UnitOfWork) (documentaryRepositorySet, error) {
	documentary, ok := uow.(ports.DocumentaryUnitOfWork)
	if !ok {
		return documentaryRepositorySet{}, domain.NewError(domain.ErrorCodeInvalidConfig, "transaction manager does not support documentary repositories")
	}
	return documentaryRepositorySet{
		artifacts:     documentary.Artifacts(),
		evidence:      documentary.Evidence(),
		decisions:     documentary.Decisions(),
		evidenceLinks: documentary.EvidenceLinks(),
	}, nil
}

func requireDocumentaryOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	if outcome.IsArchived() {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "archived outcome does not accept documentary mutations")
	}
	return nil
}

func (s *Service) documentaryRead(
	ctx context.Context,
	scope domain.Scope,
) (documentaryRepositorySet, ports.UnitOfWork, domain.OutcomeRevision, error) {
	if err := scope.Validate(); err != nil {
		return documentaryRepositorySet{}, nil, 0, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return documentaryRepositorySet{}, nil, 0, err
	}
	repositories, err := documentaryRepositories(uow)
	if err != nil {
		_ = uow.Rollback()
		return documentaryRepositorySet{}, nil, 0, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		_ = uow.Rollback()
		return documentaryRepositorySet{}, nil, 0, err
	}
	return repositories, uow, coordination.Revision, nil
}
