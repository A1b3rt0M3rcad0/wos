package application

import "github.com/A1b3rt0M3rcad0/wos/core/domain"

// DocumentaryState keeps complete documentary history alongside explicit
// current projections so a consumer does not need previous chat/session state
// to distinguish historical records from the facts and choices in force now.
type DocumentaryState struct {
	Artifacts           []domain.Artifact     `json:"artifacts"`
	RegisteredArtifacts []domain.Artifact     `json:"registered_artifacts"`
	Evidence            []domain.Evidence     `json:"evidence"`
	RegisteredEvidence  []domain.Evidence     `json:"registered_evidence"`
	EvidenceLinks       []domain.EvidenceLink `json:"evidence_links"`
	ActiveEvidenceLinks []domain.EvidenceLink `json:"active_evidence_links"`
	Decisions           []domain.Decision     `json:"decisions"`
	CurrentDecisions    []domain.Decision     `json:"current_decisions"`
}

func ProjectDocumentaryState(
	artifacts []domain.Artifact,
	evidence []domain.Evidence,
	links []domain.EvidenceLink,
	decisions []domain.Decision,
) DocumentaryState {
	state := DocumentaryState{
		Artifacts:           append([]domain.Artifact(nil), artifacts...),
		RegisteredArtifacts: make([]domain.Artifact, 0, len(artifacts)),
		Evidence:            cloneEvidenceSlice(evidence),
		RegisteredEvidence:  make([]domain.Evidence, 0, len(evidence)),
		EvidenceLinks:       cloneEvidenceLinkSlice(links),
		ActiveEvidenceLinks: make([]domain.EvidenceLink, 0, len(links)),
		Decisions:           cloneDecisionSlice(decisions),
		CurrentDecisions:    make([]domain.Decision, 0, len(decisions)),
	}
	for _, value := range artifacts {
		if value.Lifecycle == domain.ArtifactLifecycleRegistered {
			state.RegisteredArtifacts = append(state.RegisteredArtifacts, value)
		}
	}
	for _, value := range evidence {
		if value.Lifecycle == domain.EvidenceLifecycleRegistered {
			state.RegisteredEvidence = append(state.RegisteredEvidence, cloneEvidenceForProjection(value))
		}
	}
	for _, value := range links {
		if value.Lifecycle == domain.EvidenceLinkLifecycleActive {
			state.ActiveEvidenceLinks = append(state.ActiveEvidenceLinks, cloneEvidenceLinkForProjection(value))
		}
	}
	for _, value := range decisions {
		if value.Lifecycle == domain.DecisionLifecycleAccepted {
			state.CurrentDecisions = append(state.CurrentDecisions, cloneDecisionForProjection(value))
		}
	}
	return state
}

func cloneEvidenceSlice(values []domain.Evidence) []domain.Evidence {
	result := make([]domain.Evidence, len(values))
	for i, value := range values {
		result[i] = cloneEvidenceForProjection(value)
	}
	return result
}

func cloneEvidenceForProjection(value domain.Evidence) domain.Evidence {
	if value.ArtifactID != nil {
		id := *value.ArtifactID
		value.ArtifactID = &id
	}
	if value.Measurement != nil {
		measurement := *value.Measurement
		measurement.Value = append([]byte(nil), value.Measurement.Value...)
		measurement.Conditions = append([]byte(nil), value.Measurement.Conditions...)
		value.Measurement = &measurement
	}
	return value
}

func cloneEvidenceLinkSlice(values []domain.EvidenceLink) []domain.EvidenceLink {
	result := make([]domain.EvidenceLink, len(values))
	for i, value := range values {
		result[i] = cloneEvidenceLinkForProjection(value)
	}
	return result
}

func cloneEvidenceLinkForProjection(value domain.EvidenceLink) domain.EvidenceLink {
	if value.CriterionID != nil {
		id := *value.CriterionID
		value.CriterionID = &id
	}
	return value
}

func cloneDecisionSlice(values []domain.Decision) []domain.Decision {
	result := make([]domain.Decision, len(values))
	for i, value := range values {
		result[i] = cloneDecisionForProjection(value)
	}
	return result
}

func cloneDecisionForProjection(value domain.Decision) domain.Decision {
	value.Alternatives = append([]string(nil), value.Alternatives...)
	if value.DecidedBy != nil {
		actor := *value.DecidedBy
		value.DecidedBy = &actor
	}
	if value.DecidedAt != nil {
		at := *value.DecidedAt
		value.DecidedAt = &at
	}
	if value.SupersedesDecisionID != nil {
		id := *value.SupersedesDecisionID
		value.SupersedesDecisionID = &id
	}
	return value
}
