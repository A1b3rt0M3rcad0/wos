package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (s *Service) ListArtifacts(ctx context.Context, scope domain.Scope) ([]domain.Artifact, domain.OutcomeRevision, error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	values, err := repositories.artifacts.ListByOutcome(ctx, scope)
	return values, revision, err
}

func (s *Service) ListEvidence(ctx context.Context, scope domain.Scope) ([]domain.Evidence, domain.OutcomeRevision, error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	values, err := repositories.evidence.ListByOutcome(ctx, scope)
	return values, revision, err
}

func (s *Service) ListEvidenceLinks(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, domain.OutcomeRevision, error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	values, err := repositories.evidenceLinks.ListByOutcome(ctx, scope)
	return values, revision, err
}

func (s *Service) ListDecisions(ctx context.Context, scope domain.Scope) ([]domain.Decision, domain.OutcomeRevision, error) {
	repositories, uow, revision, err := s.documentaryRead(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	values, err := repositories.decisions.ListByOutcome(ctx, scope)
	return values, revision, err
}
