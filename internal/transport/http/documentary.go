package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) registerArtifact(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request registerArtifactRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RegisterArtifact(r.Context(), cc, application.RegisterArtifactCommand{
		Scope: scope, ArtifactType: request.ArtifactType, Name: request.Name, URI: request.URI,
		MediaType: request.MediaType, Checksum: request.Checksum, SourceVersion: request.SourceVersion,
		ProducerRef: request.ProducerRef, ProducedAt: request.ProducedAt,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindArtifact, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", documentaryLocation(h, scope, "artifacts", result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getArtifact(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "artifact_id")
	if !ok {
		return
	}
	result, err := h.service.GetArtifact(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindArtifact, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listArtifacts(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListArtifacts(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Artifact]{Items: values, OutcomeRevision: revision})
}

func (h *Handler) withdrawArtifact(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "artifact_id")
	if !ok {
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindArtifact, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.WithdrawArtifact(r.Context(), cc, application.WithdrawArtifactCommand{
		Scope: scope, ArtifactID: id, ExpectedVersion: expected, Reason: request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindArtifact, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) registerEvidence(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request registerEvidenceRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	artifactID, err := optionalPathID(request.ArtifactID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RegisterEvidence(r.Context(), cc, application.RegisterEvidenceCommand{
		Scope: scope, EvidenceType: request.EvidenceType, Description: request.Description,
		SourceRef: request.SourceRef, ProducerRef: request.ProducerRef, CapturedAt: request.CapturedAt,
		ArtifactID: artifactID, Measurement: request.Measurement, SourceVersion: request.SourceVersion,
		Checksum: request.Checksum,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidence, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", documentaryLocation(h, scope, "evidence", result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getEvidence(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "evidence_id")
	if !ok {
		return
	}
	result, err := h.service.GetEvidence(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidence, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listEvidence(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListEvidence(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Evidence]{Items: values, OutcomeRevision: revision})
}

func (h *Handler) retractEvidence(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "evidence_id")
	if !ok {
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindEvidence, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RetractEvidence(r.Context(), cc, application.RetractEvidenceCommand{
		Scope: scope, EvidenceID: id, ExpectedVersion: expected, Reason: request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidence, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) createEvidenceLink(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createEvidenceLinkRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	evidenceID, err := domain.ParseID(request.EvidenceID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	target, err := relationEndpoint(scope, request.TargetRef)
	if err != nil {
		writeError(w, r, err)
		return
	}
	criterionID, err := optionalPathID(request.CriterionID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateEvidenceLink(r.Context(), cc, application.CreateEvidenceLinkCommand{
		Scope: scope, EvidenceID: evidenceID, TargetRef: target, CriterionID: criterionID,
		Stance: request.Stance, Rationale: request.Rationale,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidenceLink, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", documentaryLocation(h, scope, "evidence-links", result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getEvidenceLink(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "evidence_link_id")
	if !ok {
		return
	}
	result, err := h.service.GetEvidenceLink(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidenceLink, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listEvidenceLinks(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListEvidenceLinks(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.EvidenceLink]{Items: values, OutcomeRevision: revision})
}

func (h *Handler) retractEvidenceLink(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "evidence_link_id")
	if !ok {
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindEvidenceLink, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RetractEvidenceLink(r.Context(), cc, application.RetractEvidenceLinkCommand{
		Scope: scope, EvidenceLinkID: id, ExpectedVersion: expected,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindEvidenceLink, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) proposeDecision(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request proposeDecisionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ProposeDecision(r.Context(), cc, application.ProposeDecisionCommand{
		Scope: scope, Title: request.Title, Proposal: request.Proposal,
		Rationale: request.Rationale, Alternatives: request.Alternatives,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindDecision, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", documentaryLocation(h, scope, "decisions", result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getDecision(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "decision_id")
	if !ok {
		return
	}
	result, err := h.service.GetDecision(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindDecision, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listDecisions(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListDecisions(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Decision]{Items: values, OutcomeRevision: revision})
}

func (h *Handler) reviseDecision(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "decision_id")
	if !ok {
		return
	}
	var request reviseDecisionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindDecision, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ReviseDecision(r.Context(), cc, application.ReviseDecisionCommand{
		Scope: scope, DecisionID: id, ExpectedVersion: expected,
		Title: request.Title, Proposal: request.Proposal, Rationale: request.Rationale,
		Alternatives: request.Alternatives,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindDecision, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) acceptDecision(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "decision_id")
	if !ok {
		return
	}
	var request acceptDecisionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindDecision, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.AcceptDecision(r.Context(), cc, application.AcceptDecisionCommand{
		Scope: scope, DecisionID: id, ExpectedVersion: expected,
		ChosenAlternative: request.ChosenAlternative, Rationale: request.Rationale,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindDecision, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) rejectDecision(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "decision_id")
	if !ok {
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindDecision, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RejectDecision(r.Context(), cc, application.RejectDecisionCommand{
		Scope: scope, DecisionID: id, ExpectedVersion: expected, Reason: request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindDecision, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) supersedeDecision(w http.ResponseWriter, r *http.Request) {
	scope, id, ok := h.documentaryRequestScope(w, r, "decision_id")
	if !ok {
		return
	}
	var request supersedeDecisionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindDecision, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.SupersedeDecision(r.Context(), cc, application.SupersedeDecisionCommand{
		Scope: scope, DecisionID: id, ExpectedVersion: expected,
		Title: request.Title, Proposal: request.Proposal, Alternatives: request.Alternatives,
		ChosenAlternative: request.ChosenAlternative, Rationale: request.Rationale,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Location", documentaryLocation(h, scope, "decisions", result.Value.Successor.ID))
	setETag(w, domain.EntityKindDecision, result.Value.Successor.ID, result.Value.Successor.Version)
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) documentaryRequestScope(
	w http.ResponseWriter,
	r *http.Request,
	pathName string,
) (domain.Scope, domain.ID, bool) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", false
	}
	id, err := parsePathID(r, pathName)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", false
	}
	return scope, id, true
}

func documentaryLocation(h *Handler, scope domain.Scope, collection string, id domain.ID) string {
	return h.prefix + "/namespaces/" + scope.NamespaceID.String() +
		"/outcomes/" + scope.OutcomeID.String() + "/" + collection + "/" + id.String()
}
