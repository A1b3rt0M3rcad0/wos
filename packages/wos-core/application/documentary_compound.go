package application

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func compoundDocumentaryEvents[T any](
	s *Service,
	cc domain.CommandContext,
	meta commandMetadata,
	value T,
	revision domain.OutcomeRevision,
) ([]domain.DomainEvent, bool, error) {
	if revision == 0 {
		return nil, false, nil
	}
	result, ok := any(value).(DecisionSupersessionResult)
	if !ok {
		return nil, false, nil
	}
	before, err := previousVersionPtr(result.Predecessor.Version)
	if err != nil {
		return nil, true, err
	}
	specs := []compoundEventSpec{
		{eventType: "decision.accepted", ref: result.Successor.Ref(), after: versionPtr(result.Successor.Version)},
		{eventType: "decision.superseded", ref: result.Predecessor.Ref(), before: before, after: versionPtr(result.Predecessor.Version)},
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	return events, true, err
}
