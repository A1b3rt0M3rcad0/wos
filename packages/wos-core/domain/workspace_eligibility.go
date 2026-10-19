package domain

// EvidenceLinkTargetKinds is the canonical set for clients, capability views
// and link validation. A Criterion is addressed through one of its owners.
func EvidenceLinkTargetKinds() []EntityKind {
	return []EntityKind{EntityKindOutcome, EntityKindObjective, EntityKindWorkItem, EntityKindIssue, EntityKindDecision}
}

func EvidenceLinkTargetAllowed(kind EntityKind) bool {
	for _, allowed := range EvidenceLinkTargetKinds() {
		if kind == allowed {
			return true
		}
	}
	return false
}

// BlockerPropagationOptions describes actual scope semantics, rather than
// treating the valid propagation enum as valid for every target kind.
func BlockerPropagationOptions(kind EntityKind) []BlockerPropagation {
	switch kind {
	case EntityKindOutcome:
		return []BlockerPropagation{BlockerPropagationSubtree}
	case EntityKindObjective:
		return []BlockerPropagation{BlockerPropagationDirect, BlockerPropagationSubtree}
	case EntityKindWorkItem:
		return []BlockerPropagation{BlockerPropagationDirect}
	default:
		return nil
	}
}

func BlockerPropagationAllowed(kind EntityKind, propagation BlockerPropagation) bool {
	for _, allowed := range BlockerPropagationOptions(kind) {
		if propagation == allowed {
			return true
		}
	}
	return false
}
