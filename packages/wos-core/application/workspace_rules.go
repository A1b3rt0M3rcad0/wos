package application

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

// WorkspaceRules describes canonical typed eligibility, never authorization.
// Every mutation still validates its actual target and current scoped state.
type WorkspaceRules struct {
	Version                 int                                               `json:"version"`
	EvidenceLinkTargetKinds []domain.EntityKind                               `json:"evidence_link_target_kinds"`
	BlockerPropagation      map[domain.EntityKind][]domain.BlockerPropagation `json:"blocker_propagation"`
}

func CanonicalWorkspaceRules() WorkspaceRules {
	propagation := make(map[domain.EntityKind][]domain.BlockerPropagation)
	for _, kind := range []domain.EntityKind{domain.EntityKindOutcome, domain.EntityKindObjective, domain.EntityKindWorkItem} {
		propagation[kind] = domain.BlockerPropagationOptions(kind)
	}
	return WorkspaceRules{Version: 1, EvidenceLinkTargetKinds: domain.EvidenceLinkTargetKinds(), BlockerPropagation: propagation}
}
